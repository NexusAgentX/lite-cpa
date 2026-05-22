package logging

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	_ "modernc.org/sqlite"
)

type SqliteRequestLogger struct {
	db      *sql.DB
	enabled bool
	mu      sync.RWMutex
}

func NewSqliteRequestLogger(enabled bool, dbPath string) (*SqliteRequestLogger, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := createRequestLogsTable(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	log.Infof("sqlite request logger initialized: %s", dbPath)
	return &SqliteRequestLogger{db: db, enabled: enabled}, nil
}

func createRequestLogsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS request_logs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id      TEXT NOT NULL DEFAULT '',
			timestamp       TEXT NOT NULL DEFAULT '',
			url             TEXT NOT NULL DEFAULT '',
			method          TEXT NOT NULL DEFAULT '',
			status_code     INTEGER NOT NULL DEFAULT 0,
			model           TEXT NOT NULL DEFAULT '',
			protocol        TEXT NOT NULL DEFAULT '',
			user_agent      TEXT NOT NULL DEFAULT '',
			input_tokens    INTEGER NOT NULL DEFAULT 0,
			output_tokens   INTEGER NOT NULL DEFAULT 0,
			cache_read      INTEGER NOT NULL DEFAULT 0,
			cache_create    INTEGER NOT NULL DEFAULT 0,
			duration_ms     INTEGER NOT NULL DEFAULT 0,
			request_body    TEXT NOT NULL DEFAULT '',
			response_body   TEXT NOT NULL DEFAULT '',
			api_request     TEXT NOT NULL DEFAULT '',
			api_response    TEXT NOT NULL DEFAULT '',
			created_at      TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_request_logs_ts ON request_logs(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_request_logs_model ON request_logs(model);
	`)
	return err
}

func (l *SqliteRequestLogger) Close() error {
	return l.db.Close()
}

func (l *SqliteRequestLogger) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled
}

func (l *SqliteRequestLogger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

func (l *SqliteRequestLogger) LogRequest(url, method string, requestHeaders map[string][]string, body []byte, statusCode int, responseHeaders map[string][]string, response, websocketTimeline, apiRequest, apiResponse, apiWebsocketTimeline []byte, apiResponseErrors []*interfaces.ErrorMessage, requestID string, requestTimestamp, apiResponseTimestamp time.Time) error {
	return l.logRequest(url, method, requestHeaders, body, statusCode, responseHeaders, response, websocketTimeline, apiRequest, apiResponse, apiWebsocketTimeline, apiResponseErrors, false, requestID, requestTimestamp, apiResponseTimestamp)
}

func (l *SqliteRequestLogger) LogRequestWithOptions(url, method string, requestHeaders map[string][]string, body []byte, statusCode int, responseHeaders map[string][]string, response, websocketTimeline, apiRequest, apiResponse, apiWebsocketTimeline []byte, apiResponseErrors []*interfaces.ErrorMessage, force bool, requestID string, requestTimestamp, apiResponseTimestamp time.Time) error {
	return l.logRequest(url, method, requestHeaders, body, statusCode, responseHeaders, response, websocketTimeline, apiRequest, apiResponse, apiWebsocketTimeline, apiResponseErrors, force, requestID, requestTimestamp, apiResponseTimestamp)
}

func (l *SqliteRequestLogger) logRequest(url, method string, requestHeaders map[string][]string, body []byte, statusCode int, responseHeaders map[string][]string, response, websocketTimeline, apiRequest, apiResponse, apiWebsocketTimeline []byte, apiResponseErrors []*interfaces.ErrorMessage, force bool, requestID string, requestTimestamp, apiResponseTimestamp time.Time) error {
	log.Infof("sqlite LogRequest called: url=%s method=%s status=%d force=%v enabled=%v", url, method, statusCode, force, l.enabled)
	if !force {
		l.mu.RLock()
		enabled := l.enabled
		l.mu.RUnlock()
		if !enabled {
			log.Infof("sqlite LogRequest: disabled, skipping")
			return nil
		}
	}

	model, protocol := extractModelAndProtocol(url, response, body)
	ua := extractUserAgent(requestHeaders)
	inputTokens, outputTokens, cacheRead, cacheCreate := extractTokens(response)
	if inputTokens == 0 && outputTokens == 0 {
		inputTokens, outputTokens, cacheRead, cacheCreate = extractTokensFromSSE(response)
	}

	durationMs := int(apiResponseTimestamp.Sub(requestTimestamp).Milliseconds())
	if durationMs < 0 {
		durationMs = 0
	}

	_, err := l.db.Exec(`INSERT INTO request_logs
		(request_id, timestamp, url, method, status_code, model, protocol, user_agent,
		 input_tokens, output_tokens, cache_read, cache_create, duration_ms,
		 request_body, response_body, api_request, api_response)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		requestID,
		requestTimestamp.Format(time.RFC3339Nano),
		url, method, statusCode,
		model, protocol, ua,
		inputTokens, outputTokens, cacheRead, cacheCreate, durationMs,
		string(body), string(response),
		string(apiRequest), string(apiResponse),
	)
	if err != nil {
		log.Errorf("sqlite log write failed: %v", err)
	}
	return err
}

func (l *SqliteRequestLogger) LogStreamingRequest(url, method string, headers map[string][]string, body []byte, requestID string) (StreamingLogWriter, error) {
	l.mu.RLock()
	enabled := l.enabled
	l.mu.RUnlock()

	if !enabled {
		return &NoOpStreamingLogWriter{}, nil
	}

	return &SqliteStreamingLogWriter{
		logger:           l,
		requestID:        requestID,
		url:              url,
		method:           method,
		requestHeaders:   headers,
		requestBody:      body,
		requestTimestamp: time.Now(),
	}, nil
}

type SqliteStreamingLogWriter struct {
	logger               *SqliteRequestLogger
	requestID            string
	url                  string
	method               string
	requestHeaders       map[string][]string
	requestBody          []byte
	statusCode           int
	responseHeaders      map[string][]string
	responseBody         bytes.Buffer
	apiRequest           []byte
	apiResponse          []byte
	apiWebsocketTimeline []byte
	requestTimestamp     time.Time
	firstChunkTimestamp  time.Time
	mu                   sync.Mutex
}

func (w *SqliteStreamingLogWriter) WriteChunkAsync(chunk []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.responseBody.Write(chunk)
}

func (w *SqliteStreamingLogWriter) WriteStatus(status int, headers map[string][]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.statusCode = status
	w.responseHeaders = headers
	return nil
}

func (w *SqliteStreamingLogWriter) WriteAPIRequest(apiRequest []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.apiRequest = bytes.Clone(apiRequest)
	return nil
}

func (w *SqliteStreamingLogWriter) WriteAPIResponse(apiResponse []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.apiResponse = bytes.Clone(apiResponse)
	return nil
}

func (w *SqliteStreamingLogWriter) WriteAPIWebsocketTimeline(apiWebsocketTimeline []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.apiWebsocketTimeline = bytes.Clone(apiWebsocketTimeline)
	return nil
}

func (w *SqliteStreamingLogWriter) SetFirstChunkTimestamp(timestamp time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.firstChunkTimestamp = timestamp
}

func (w *SqliteStreamingLogWriter) Close() error {
	log.Infof("sqlite streaming Close: url=%s method=%s status=%d", w.url, w.method, w.statusCode)
	w.mu.Lock()
	defer w.mu.Unlock()

	apiResponse := w.apiResponse
	if len(apiResponse) == 0 {
		apiResponse = w.responseBody.Bytes()
	}

	model, protocol := extractModelAndProtocol(w.url, apiResponse, w.requestBody)
	ua := extractUserAgent(w.requestHeaders)
	inputTokens, outputTokens, cacheRead, cacheCreate := extractTokens(apiResponse)
	if inputTokens == 0 && outputTokens == 0 {
		inputTokens, outputTokens, cacheRead, cacheCreate = extractTokensFromSSE(w.responseBody.Bytes())
	}

	var durationMs int64
	if !w.firstChunkTimestamp.IsZero() {
		durationMs = w.firstChunkTimestamp.Sub(w.requestTimestamp).Milliseconds()
	}
	if durationMs < 0 {
		durationMs = 0
	}

	_, err := w.logger.db.Exec(`INSERT INTO request_logs
		(request_id, timestamp, url, method, status_code, model, protocol, user_agent,
		 input_tokens, output_tokens, cache_read, cache_create, duration_ms,
		 request_body, response_body, api_request, api_response)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		w.requestID,
		w.requestTimestamp.Format(time.RFC3339Nano),
		w.url, w.method, w.statusCode,
		model, protocol, ua,
		inputTokens, outputTokens, cacheRead, cacheCreate, durationMs,
		string(w.requestBody), w.responseBody.String(),
		string(w.apiRequest), string(apiResponse),
	)
	if err != nil {
		log.Errorf("sqlite streaming log write failed: %v", err)
	}
	return err
}

func extractModelAndProtocol(url string, apiResponse, requestBody []byte) (model, protocol string) {
	switch {
	case strings.Contains(url, "/v1/messages"):
		protocol = "anthropic"
	case strings.Contains(url, "/v1/chat/completions"):
		protocol = "openai"
	case strings.Contains(url, "/v1/responses"):
		protocol = "responses"
	case strings.Contains(url, "/v1beta"):
		protocol = "gemini"
	default:
		protocol = "other"
	}

	if len(apiResponse) > 0 {
		model = gjson.GetBytes(apiResponse, "model").String()
		if model == "" {
			model = gjson.GetBytes(apiResponse, "usage.model").String()
		}
	}
	if model == "" && len(requestBody) > 0 {
		model = gjson.GetBytes(requestBody, "model").String()
	}
	return
}

func extractTokens(apiResponse []byte) (input, output, cacheRead, cacheCreate int64) {
	if len(apiResponse) == 0 {
		return
	}

	usage := gjson.GetBytes(apiResponse, "usage")
	if !usage.Exists() {
		return
	}

	input = usage.Get("input_tokens").Int()
	output = usage.Get("output_tokens").Int()
	cacheRead = usage.Get("cache_read_input_tokens").Int()
	cacheCreate = usage.Get("cache_creation_input_tokens").Int()

	if input == 0 && output == 0 {
		input = usage.Get("prompt_tokens").Int()
		output = usage.Get("completion_tokens").Int()
		if cached := usage.Get("prompt_tokens_details.cached_tokens"); cached.Exists() {
			cacheRead = cached.Int()
		}
	}

	return
}

func extractTokensFromSSE(body []byte) (input, output, cacheRead, cacheCreate int64) {
	if len(body) == 0 {
		return
	}
	prefix := []byte("data:")
	for _, line := range bytes.Split(body, []byte("\n")) {
		if !bytes.HasPrefix(line, prefix) {
			continue
		}
		blob := line[len(prefix):]
		if len(blob) > 0 && blob[0] == ' ' {
			blob = blob[1:]
		}
		if !gjson.ValidBytes(blob) {
			continue
		}
		usage := gjson.GetBytes(blob, "usage")
		messageUsage := gjson.GetBytes(blob, "message.usage")
		responseUsage := gjson.GetBytes(blob, "response.usage")
		if !usage.Exists() && !messageUsage.Exists() && !responseUsage.Exists() {
			continue
		}
		if messageUsage.Exists() {
			if v := messageUsage.Get("input_tokens"); v.Exists() {
				input = v.Int()
			}
			if v := messageUsage.Get("cache_read_input_tokens"); v.Exists() {
				cacheRead = v.Int()
			}
			if v := messageUsage.Get("cache_creation_input_tokens"); v.Exists() {
				cacheCreate = v.Int()
			}
		}
		if responseUsage.Exists() {
			if v := responseUsage.Get("input_tokens"); v.Exists() && input == 0 {
				input = v.Int()
			}
			if v := responseUsage.Get("output_tokens"); v.Exists() && output == 0 {
				output = v.Int()
			}
			if v := responseUsage.Get("input_tokens_details.cached_tokens"); v.Exists() && cacheRead == 0 {
				cacheRead = v.Int()
			}
		}
		if usage.Exists() {
			if v := usage.Get("output_tokens"); v.Exists() {
				output = v.Int()
			}
			if v := usage.Get("input_tokens"); v.Exists() && input == 0 {
				input = v.Int()
			}
			if v := usage.Get("prompt_tokens"); v.Exists() && input == 0 {
				input = v.Int()
			}
			if v := usage.Get("completion_tokens"); v.Exists() && output == 0 {
				output = v.Int()
			}
			if v := usage.Get("cache_read_input_tokens"); v.Exists() && cacheRead == 0 {
				cacheRead = v.Int()
			}
			if v := usage.Get("cache_creation_input_tokens"); v.Exists() && cacheCreate == 0 {
				cacheCreate = v.Int()
			}
			if cached := usage.Get("prompt_tokens_details.cached_tokens"); cached.Exists() && cacheRead == 0 {
				cacheRead = cached.Int()
			}
		}
	}
	return
}

func extractUserAgent(headers map[string][]string) string {
	if values, ok := headers["User-Agent"]; ok && len(values) > 0 {
		return values[0]
	}
	if values, ok := headers["user-agent"]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}
