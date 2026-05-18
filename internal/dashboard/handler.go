package dashboard

import (
	"bytes"
	"database/sql"
	_ "embed"
	"encoding/csv"
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

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func buildInClause(values []string) (string, []interface{}) {
	placeholders := make([]string, len(values))
	params := make([]interface{}, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		params[i] = v
	}
	return "(" + strings.Join(placeholders, ",") + ")", params
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
	MethodOptions    []string       `json:"method_options,omitempty"`
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
	modelFilters := splitCSV(q.Get("model"))
	protocolFilters := splitCSV(q.Get("protocol"))
	statusFiltersStr := splitCSV(q.Get("status_code"))
	methodFilters := splitCSV(q.Get("method"))
	timeFrom := strings.TrimSpace(q.Get("time_from"))
	timeTo := strings.TrimSpace(q.Get("time_to"))
	sortBy := q.Get("sort_by")
	sortDir := q.Get("sort_dir")
	inputTokensMinStr := strings.TrimSpace(q.Get("input_tokens_min"))
	inputTokensMaxStr := strings.TrimSpace(q.Get("input_tokens_max"))
	outputTokensMinStr := strings.TrimSpace(q.Get("output_tokens_min"))
	outputTokensMaxStr := strings.TrimSpace(q.Get("output_tokens_max"))
	durationMinStr := strings.TrimSpace(q.Get("duration_min"))
	durationMaxStr := strings.TrimSpace(q.Get("duration_max"))

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
		whereClauses = append(whereClauses, "(model LIKE ? OR url LIKE ? OR method LIKE ? OR user_agent LIKE ?)")
		params = append(params, like, like, like, like)
	}
	if len(modelFilters) > 0 {
		clause, clauseParams := buildInClause(modelFilters)
		whereClauses = append(whereClauses, "model IN "+clause)
		params = append(params, clauseParams...)
	}
	if len(protocolFilters) > 0 {
		clause, clauseParams := buildInClause(protocolFilters)
		whereClauses = append(whereClauses, "protocol IN "+clause)
		params = append(params, clauseParams...)
	}
	if len(statusFiltersStr) > 0 {
		var codes []interface{}
		for _, s := range statusFiltersStr {
			code, err := strconv.Atoi(s)
			if err == nil {
				codes = append(codes, code)
			}
		}
		if len(codes) > 0 {
			placeholders := make([]string, len(codes))
			for i := range codes {
				placeholders[i] = "?"
				params = append(params, codes[i])
			}
			whereClauses = append(whereClauses, "status_code IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if len(methodFilters) > 0 {
		clause, clauseParams := buildInClause(methodFilters)
		whereClauses = append(whereClauses, "method IN "+clause)
		params = append(params, clauseParams...)
	}
	if inputTokensMinStr != "" {
		v, err := strconv.ParseInt(inputTokensMinStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "input_tokens >= ?")
			params = append(params, v)
		}
	}
	if inputTokensMaxStr != "" {
		v, err := strconv.ParseInt(inputTokensMaxStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "input_tokens <= ?")
			params = append(params, v)
		}
	}
	if outputTokensMinStr != "" {
		v, err := strconv.ParseInt(outputTokensMinStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "output_tokens >= ?")
			params = append(params, v)
		}
	}
	if outputTokensMaxStr != "" {
		v, err := strconv.ParseInt(outputTokensMaxStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "output_tokens <= ?")
			params = append(params, v)
		}
	}
	if durationMinStr != "" {
		v, err := strconv.Atoi(durationMinStr)
		if err == nil {
			whereClauses = append(whereClauses, "duration_ms >= ?")
			params = append(params, v)
		}
	}
	if durationMaxStr != "" {
		v, err := strconv.Atoi(durationMaxStr)
		if err == nil {
			whereClauses = append(whereClauses, "duration_ms <= ?")
			params = append(params, v)
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

	hasNoFilters := search == "" && len(modelFilters) == 0 && len(protocolFilters) == 0 &&
		len(statusFiltersStr) == 0 && len(methodFilters) == 0
	if page == 1 && hasNoFilters {
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
		methodRows, err := h.db.Query(`SELECT DISTINCT method FROM request_logs WHERE method != '' ORDER BY method`)
		if err == nil {
			var methods []string
			for methodRows.Next() {
				var m string
				if methodRows.Scan(&m) == nil && m != "" {
					methods = append(methods, m)
				}
			}
			methodRows.Close()
			resp.MethodOptions = methods
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

func (h *Handler) ExportLogs(c *gin.Context) {
	if h.db == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "dashboard database not available"})
		return
	}

	q := c.Request.URL.Query()
	search := strings.TrimSpace(q.Get("search"))
	modelFilters := splitCSV(q.Get("model"))
	protocolFilters := splitCSV(q.Get("protocol"))
	statusFiltersStr := splitCSV(q.Get("status_code"))
	methodFilters := splitCSV(q.Get("method"))
	timeFrom := strings.TrimSpace(q.Get("time_from"))
	timeTo := strings.TrimSpace(q.Get("time_to"))
	sortBy := q.Get("sort_by")
	sortDir := q.Get("sort_dir")
	inputTokensMinStr := strings.TrimSpace(q.Get("input_tokens_min"))
	inputTokensMaxStr := strings.TrimSpace(q.Get("input_tokens_max"))
	outputTokensMinStr := strings.TrimSpace(q.Get("output_tokens_min"))
	outputTokensMaxStr := strings.TrimSpace(q.Get("output_tokens_max"))
	durationMinStr := strings.TrimSpace(q.Get("duration_min"))
	durationMaxStr := strings.TrimSpace(q.Get("duration_max"))

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
		whereClauses = append(whereClauses, "(model LIKE ? OR url LIKE ? OR method LIKE ? OR user_agent LIKE ?)")
		params = append(params, like, like, like, like)
	}
	if len(modelFilters) > 0 {
		clause, clauseParams := buildInClause(modelFilters)
		whereClauses = append(whereClauses, "model IN "+clause)
		params = append(params, clauseParams...)
	}
	if len(protocolFilters) > 0 {
		clause, clauseParams := buildInClause(protocolFilters)
		whereClauses = append(whereClauses, "protocol IN "+clause)
		params = append(params, clauseParams...)
	}
	if len(statusFiltersStr) > 0 {
		var codes []interface{}
		for _, s := range statusFiltersStr {
			code, err := strconv.Atoi(s)
			if err == nil {
				codes = append(codes, code)
			}
		}
		if len(codes) > 0 {
			placeholders := make([]string, len(codes))
			for i := range codes {
				placeholders[i] = "?"
				params = append(params, codes[i])
			}
			whereClauses = append(whereClauses, "status_code IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if len(methodFilters) > 0 {
		clause, clauseParams := buildInClause(methodFilters)
		whereClauses = append(whereClauses, "method IN "+clause)
		params = append(params, clauseParams...)
	}
	if inputTokensMinStr != "" {
		v, err := strconv.ParseInt(inputTokensMinStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "input_tokens >= ?")
			params = append(params, v)
		}
	}
	if inputTokensMaxStr != "" {
		v, err := strconv.ParseInt(inputTokensMaxStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "input_tokens <= ?")
			params = append(params, v)
		}
	}
	if outputTokensMinStr != "" {
		v, err := strconv.ParseInt(outputTokensMinStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "output_tokens >= ?")
			params = append(params, v)
		}
	}
	if outputTokensMaxStr != "" {
		v, err := strconv.ParseInt(outputTokensMaxStr, 10, 64)
		if err == nil {
			whereClauses = append(whereClauses, "output_tokens <= ?")
			params = append(params, v)
		}
	}
	if durationMinStr != "" {
		v, err := strconv.Atoi(durationMinStr)
		if err == nil {
			whereClauses = append(whereClauses, "duration_ms >= ?")
			params = append(params, v)
		}
	}
	if durationMaxStr != "" {
		v, err := strconv.Atoi(durationMaxStr)
		if err == nil {
			whereClauses = append(whereClauses, "duration_ms <= ?")
			params = append(params, v)
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
	orderClause := fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	rows, err := h.db.Query(`SELECT id, timestamp, url, method, status_code, model, protocol,
		user_agent, input_tokens, output_tokens, cache_read, cache_create, duration_ms,
		request_body, response_body
		FROM request_logs`+where+orderClause+` LIMIT 50000`, params...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var buf bytes.Buffer
	buf.Write([]byte{0xef, 0xbb, 0xbf}) // BOM for Excel UTF-8 compatibility
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"ID", "Timestamp", "URL", "Method", "Status", "Model", "Protocol",
		"UserAgent", "InputTokens", "OutputTokens", "CacheRead", "CacheCreate", "DurationMs",
		"RequestBody", "ResponseBody"})

	for rows.Next() {
		var id, status, durationMs int
		var ts, url, method, model, protocol, ua string
		var inputTokens, outputTokens, cacheRead, cacheCreate int64
		var reqBody, respBody string
		if err := rows.Scan(&id, &ts, &url, &method, &status, &model, &protocol,
			&ua, &inputTokens, &outputTokens, &cacheRead, &cacheCreate, &durationMs,
			&reqBody, &respBody); err != nil {
			continue
		}
		_ = writer.Write([]string{
			strconv.Itoa(id), ts, url, method, strconv.Itoa(status), model, protocol,
			ua, strconv.FormatInt(inputTokens, 10), strconv.FormatInt(outputTokens, 10),
			strconv.FormatInt(cacheRead, 10), strconv.FormatInt(cacheCreate, 10),
			strconv.Itoa(durationMs), reqBody, respBody,
		})
	}
	writer.Flush()

	header := c.Writer.Header()
	header.Set("Content-Type", "text/csv; charset=utf-8")
	header.Set("Content-Disposition", "attachment; filename=api-logs.csv")
	header.Set("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}
