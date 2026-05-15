package dashboard

import (
	"database/sql"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging"
	_ "modernc.org/sqlite"
)

//go:embed dashboard.html
var dashboardHTML []byte

type Handler struct {
	dbPath string
	db     *sql.DB
}

func NewHandler(cfg *config.Config) *Handler {
	logsDir := logging.ResolveLogDirectory(cfg)
	dbPath := os.Getenv("SQLITE_LOG_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(logsDir, "requests.db")
	}
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_query_only=true")
	if err != nil {
		return &Handler{dbPath: dbPath}
	}
	if err := ensureIndexes(db); err != nil {
		db.Close()
		return &Handler{dbPath: dbPath}
	}
	return &Handler{dbPath: dbPath, db: db}
}

func ensureIndexes(db *sql.DB) error {
	_, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_model_status_ts ON request_logs(model, status_code, timestamp DESC)`)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_logs_protocol ON request_logs(protocol)`)
	return err
}

func (h *Handler) IsAvailable() bool {
	return h.db != nil
}

func (h *Handler) ServeDashboard(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", dashboardHTML)
}

type logEntryLite struct {
	ID           int    `json:"id"`
	Time         string `json:"time"`
	TimeStr      string `json:"time_str"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	Status       int    `json:"status"`
	Model        string `json:"model"`
	Protocol     string `json:"protocol"`
	UA           string `json:"ua"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	CacheRead    int64  `json:"cache_read"`
	CacheCreate  int64  `json:"cache_create"`
	DurationMs   int    `json:"duration_ms"`
}

type logEntryDetail struct {
	logEntryLite
	RequestBody  string `json:"request_body"`
	ResponseBody string `json:"response_body"`
	APIRequest   string `json:"api_request"`
	APIResponse  string `json:"api_response"`
}

type logsResponse struct {
	Page             int            `json:"page"`
	PerPage          int            `json:"per_page"`
	Total            int            `json:"total"`
	TotalReq         int64          `json:"total_req"`
	TotalInput       int64          `json:"total_input"`
	TotalOutput      int64          `json:"total_output"`
	TotalCacheRead   int64          `json:"total_cache_read"`
	TotalCacheCreate int64          `json:"total_cache_create"`
	Entries          []logEntryLite `json:"entries"`
	ModelOptions     []string       `json:"model_options,omitempty"`
	ProtocolOptions  []string       `json:"protocol_options,omitempty"`
}

func formatTime(t string) string {
	if t == "" {
		return ""
	}
	dt, err := time.Parse(time.RFC3339Nano, t)
	if err != nil {
		if len(t) >= 19 {
			return t[:19]
		}
		return t
	}
	return dt.Format("01-02 15:04:05")
}

func (h *Handler) QueryLogs(c *gin.Context) {
	if h.db == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "dashboard database not available"})
		return
	}

	q := c.Request.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	search := strings.TrimSpace(q.Get("search"))
	modelFilter := strings.TrimSpace(q.Get("model"))
	protocolFilter := strings.TrimSpace(q.Get("protocol"))
	statusFilter := strings.TrimSpace(q.Get("status_code"))
	timeFrom := strings.TrimSpace(q.Get("time_from"))
	timeTo := strings.TrimSpace(q.Get("time_to"))
	sortBy := q.Get("sort_by")
	sortDir := q.Get("sort_dir")

	allowedSorts := map[string]bool{
		"timestamp": true, "model": true, "protocol": true,
		"status_code": true, "duration_ms": true,
		"input_tokens": true, "output_tokens": true,
	}
	if !allowedSorts[sortBy] {
		sortBy = "timestamp"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	var whereClauses []string
	var params []interface{}

	if search != "" {
		like := "%" + search + "%"
		whereClauses = append(whereClauses, "model LIKE ?")
		params = append(params, like)
	}
	if modelFilter != "" {
		whereClauses = append(whereClauses, "model = ?")
		params = append(params, modelFilter)
	}
	if protocolFilter != "" {
		whereClauses = append(whereClauses, "protocol = ?")
		params = append(params, protocolFilter)
	}
	if statusFilter != "" {
		code, err := strconv.Atoi(statusFilter)
		if err == nil {
			whereClauses = append(whereClauses, "status_code = ?")
			params = append(params, code)
		}
	}
	if timeFrom != "" {
		whereClauses = append(whereClauses, "timestamp >= ?")
		params = append(params, timeFrom)
	}
	if timeTo != "" {
		whereClauses = append(whereClauses, "timestamp <= ?")
		params = append(params, timeTo)
	}

	where := ""
	if len(whereClauses) > 0 {
		where = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM request_logs"+where, params...).Scan(&total)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var stats struct {
		TotalReq       int64
		TotalInput     int64
		TotalOutput    int64
		TotalCacheRead int64
		TotalCacheCr   int64
	}
	err = h.db.QueryRow(`SELECT
		COUNT(*),
		COALESCE(SUM(input_tokens),0),
		COALESCE(SUM(output_tokens),0),
		COALESCE(SUM(cache_read),0),
		COALESCE(SUM(cache_create),0)
		FROM request_logs`+where, params...).Scan(
		&stats.TotalReq, &stats.TotalInput, &stats.TotalOutput,
		&stats.TotalCacheRead, &stats.TotalCacheCr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * perPage
	orderClause := fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	rows, err := h.db.Query(`SELECT id, timestamp, url, method, status_code, model, protocol,
		user_agent, input_tokens, output_tokens, cache_read, cache_create, duration_ms
		FROM request_logs`+where+orderClause+` LIMIT ? OFFSET ?`,
		append(params, perPage, offset)...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	entries := make([]logEntryLite, 0, perPage)
	for rows.Next() {
		var e logEntryLite
		var ts string
		err := rows.Scan(&e.ID, &ts, &e.URL, &e.Method, &e.Status, &e.Model, &e.Protocol,
			&e.UA, &e.InputTokens, &e.OutputTokens, &e.CacheRead, &e.CacheCreate, &e.DurationMs)
		if err != nil {
			continue
		}
		e.Time = ts
		e.TimeStr = formatTime(ts)
		entries = append(entries, e)
	}

	resp := logsResponse{
		Page:             page,
		PerPage:          perPage,
		Total:            total,
		TotalReq:         stats.TotalReq,
		TotalInput:       stats.TotalInput,
		TotalOutput:      stats.TotalOutput,
		TotalCacheRead:   stats.TotalCacheRead,
		TotalCacheCreate: stats.TotalCacheCr,
		Entries:          entries,
	}

	if page == 1 && search == "" && modelFilter == "" && protocolFilter == "" && statusFilter == "" {
		modelRows, err := h.db.Query(`SELECT DISTINCT model FROM request_logs WHERE model != '' ORDER BY model`)
		if err == nil {
			var models []string
			for modelRows.Next() {
				var m string
				if modelRows.Scan(&m) == nil && m != "" {
					models = append(models, m)
				}
			}
			modelRows.Close()
			resp.ModelOptions = models
		}
		protoRows, err := h.db.Query(`SELECT DISTINCT protocol FROM request_logs WHERE protocol != '' ORDER BY protocol`)
		if err == nil {
			var protocols []string
			for protoRows.Next() {
				var p string
				if protoRows.Scan(&p) == nil && p != "" {
					protocols = append(protocols, p)
				}
			}
			protoRows.Close()
			resp.ProtocolOptions = protocols
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetLogDetail(c *gin.Context) {
	if h.db == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "dashboard database not available"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var e logEntryDetail
	var ts, reqBody, respBody, apiReq, apiResp string
	err = h.db.QueryRow(`SELECT id, timestamp, url, method, status_code, model, protocol,
		user_agent, input_tokens, output_tokens, cache_read, cache_create, duration_ms,
		request_body, response_body, api_request, api_response
		FROM request_logs WHERE id = ?`, id).Scan(
		&e.ID, &ts, &e.URL, &e.Method, &e.Status, &e.Model, &e.Protocol,
		&e.UA, &e.InputTokens, &e.OutputTokens, &e.CacheRead, &e.CacheCreate, &e.DurationMs,
		&reqBody, &respBody, &apiReq, &apiResp)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	e.Time = ts
	e.TimeStr = formatTime(ts)
	e.RequestBody = reqBody
	e.ResponseBody = respBody
	e.APIRequest = apiReq
	e.APIResponse = apiResp
	c.JSON(http.StatusOK, e)
}
