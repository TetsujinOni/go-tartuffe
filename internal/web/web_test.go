package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAcceptsHTML(t *testing.T) {
	tests := []struct {
		name       string
		accept     string
		userAgent  string
		wantResult bool
	}{
		{
			name:       "explicit text/html",
			accept:     "text/html",
			wantResult: true,
		},
		{
			name:       "text/html with charset",
			accept:     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			wantResult: true,
		},
		{
			name:       "application/json only",
			accept:     "application/json",
			wantResult: false,
		},
		{
			name:       "empty accept with browser user agent",
			accept:     "",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			wantResult: true,
		},
		{
			name:       "wildcard accept with browser user agent",
			accept:     "*/*",
			userAgent:  "Mozilla/5.0 Chrome/91.0",
			wantResult: true,
		},
		{
			name:       "empty accept with curl user agent",
			accept:     "",
			userAgent:  "curl/7.68.0",
			wantResult: false,
		},
		{
			name:       "wildcard accept with non-browser",
			accept:     "*/*",
			userAgent:  "go-http-client/1.1",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			got := AcceptsHTML(req)
			if got != tt.wantResult {
				t.Errorf("AcceptsHTML() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestRender(t *testing.T) {
	// Test rendering the index template
	w := httptest.NewRecorder()
	data := HomePageData{
		PageData: PageData{
			Title:       "test title",
			Description: "test description",
		},
		Notices: []Notice{},
	}

	err := Render(w, "index.html", data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %v, want text/html", contentType)
	}

	// Check that HTML was rendered
	body := w.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Error("Response should contain <html>")
	}
	if !strings.Contains(body, "test title") {
		t.Error("Response should contain title")
	}
}

func TestRenderImposters(t *testing.T) {
	w := httptest.NewRecorder()
	data := ImpostersPageData{
		PageData: PageData{
			Title:       "running imposters",
			Description: "test",
		},
		Imposters: []ImposterSummary{
			{Port: 3000, Protocol: "http", Name: "test-imposter", NumberOfRequests: 5, SelfHref: "/imposters/3000"},
			{Port: 3001, Protocol: "https", NumberOfRequests: 0, SelfHref: "/imposters/3001"},
		},
	}

	err := Render(w, "imposters.html", data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "test-imposter") {
		t.Error("Response should contain imposter name")
	}
	if !strings.Contains(body, "3000") {
		t.Error("Response should contain port number")
	}
}

func TestRenderLogs(t *testing.T) {
	w := httptest.NewRecorder()
	data := LogsPageData{
		PageData: PageData{
			Title:       "logs",
			Description: "test",
		},
		Logs: []LogEntry{
			{Level: "info", Message: "test message"},
			{Level: "error", Message: "error message"},
		},
		LogsCount: 2,
	}

	err := Render(w, "logs.html", data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "test message") {
		t.Error("Response should contain log message")
	}
	if !strings.Contains(body, "info") {
		t.Error("Response should contain log level")
	}
}

func TestRenderConfig(t *testing.T) {
	w := httptest.NewRecorder()
	data := ConfigPageData{
		PageData: PageData{
			Title:       "configuration",
			Description: "test",
		},
		Version: "1.0.0",
		Options: map[string]interface{}{
			"port":           2525,
			"allowInjection": false,
		},
		Process: ProcessInfo{
			GoVersion:    "go1.21",
			Architecture: "amd64",
			Platform:     "linux",
			RSS:          1000000,
			HeapAlloc:    500000,
			Uptime:       3600,
			Cwd:          "/home/test",
		},
	}

	err := Render(w, "config.html", data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, "1.0.0") {
		t.Error("Response should contain version")
	}
	if !strings.Contains(body, "2525") {
		t.Error("Response should contain port")
	}
}

func TestStaticHandler(t *testing.T) {
	handler := StaticHandler()

	tests := []struct {
		name           string
		path           string
		wantStatusCode int
		wantContains   string
	}{
		{
			name:           "CSS file",
			path:           "/public/css/application.css",
			wantStatusCode: http.StatusOK,
			wantContains:   "body",
		},
		{
			name:           "JS file",
			path:           "/public/js/urlHashHandler.js",
			wantStatusCode: http.StatusOK,
			wantContains:   "toggleExpandedOnSection",
		},
		{
			name:           "non-existent file",
			path:           "/public/nonexistent.txt",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("StaticHandler() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantContains != "" && !strings.Contains(w.Body.String(), tt.wantContains) {
				t.Errorf("StaticHandler() body should contain %q", tt.wantContains)
			}
		})
	}
}
