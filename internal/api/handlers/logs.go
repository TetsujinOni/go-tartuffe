package handlers

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/TetsujinOni/go-tartuffe/internal/response"
	"github.com/TetsujinOni/go-tartuffe/internal/web"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// LogsResponse is the response for GET /logs
type LogsResponse struct {
	Logs []LogEntry `json:"logs"`
}

// LogsHandler handles log operations
type LogsHandler struct {
	logs []LogEntry
	mu   sync.RWMutex
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler() *LogsHandler {
	return &LogsHandler{
		logs: make([]LogEntry, 0),
	}
}

// AddLog adds a log entry
func (h *LogsHandler) AddLog(level, message, timestamp string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logs = append(h.logs, LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: timestamp,
	})
}

// GetLogs handles GET /logs
func (h *LogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Parse pagination
	startIndex := 0
	endIndex := len(h.logs)

	if s := r.URL.Query().Get("startIndex"); s != "" {
		if idx, err := strconv.Atoi(s); err == nil && idx >= 0 {
			startIndex = idx
		}
	}

	if e := r.URL.Query().Get("endIndex"); e != "" {
		if idx, err := strconv.Atoi(e); err == nil && idx > startIndex {
			endIndex = idx
		}
	}

	// Clamp indices
	if startIndex > len(h.logs) {
		startIndex = len(h.logs)
	}
	if endIndex > len(h.logs) {
		endIndex = len(h.logs)
	}

	logs := h.logs[startIndex:endIndex]

	// Content negotiation: HTML for browsers, JSON for API clients
	if web.AcceptsHTML(r) {
		webLogs := make([]web.LogEntry, len(logs))
		for i, log := range logs {
			webLogs[i] = web.LogEntry{
				Level:   log.Level,
				Message: log.Message,
			}
		}
		data := web.LogsPageData{
			PageData: web.PageData{
				Title:       "logs",
				Description: "Placeholder description for logs page.",
			},
			Logs:      webLogs,
			LogsCount: len(h.logs),
		}
		web.Render(w, "logs.html", data)
		return
	}

	response.WriteJSON(w, http.StatusOK, LogsResponse{Logs: logs})
}
