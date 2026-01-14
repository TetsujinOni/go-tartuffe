package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"
)

//go:embed templates/*.html templates/docs/*.html templates/docs/**/*.html templates/docs/**/**/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

var (
	templates     *template.Template
	templatesOnce sync.Once
	templatesErr  error
)

// getTemplates returns the parsed templates, initializing them if needed
func getTemplates() (*template.Template, error) {
	templatesOnce.Do(func() {
		// Define template functions
		funcMap := template.FuncMap{
			"json": func(v interface{}) string {
				// Simple JSON-like output for displaying in templates
				return ""
			},
		}

		// Parse all templates including nested docs
		patterns := []string{
			"templates/*.html",
			"templates/docs/*.html",
			"templates/docs/api/*.html",
			"templates/docs/api/predicates/*.html",
			"templates/docs/api/behaviors/*.html",
			"templates/docs/api/proxy/*.html",
			"templates/docs/cli/*.html",
			"templates/docs/protocols/*.html",
		}

		templates = template.New("").Funcs(funcMap)
		for _, pattern := range patterns {
			matches, err := fs.Glob(templatesFS, pattern)
			if err != nil {
				templatesErr = err
				return
			}
			for _, match := range matches {
				content, err := fs.ReadFile(templatesFS, match)
				if err != nil {
					templatesErr = err
					return
				}
				// Use just the filename as the template name for root templates,
				// but use full path for docs templates
				name := match
				if strings.HasPrefix(match, "templates/") {
					name = strings.TrimPrefix(match, "templates/")
				}
				_, templatesErr = templates.New(name).Parse(string(content))
				if templatesErr != nil {
					return
				}
			}
		}
	})
	return templates, templatesErr
}

// Render renders a template with the given data
func Render(w http.ResponseWriter, name string, data interface{}) error {
	tmpl, err := getTemplates()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, name, data)
}

// AcceptsHTML checks if the request accepts HTML responses
func AcceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	// Check for text/html in Accept header
	// Also consider requests without Accept header or with */* as potentially wanting HTML
	// when coming from a browser
	if strings.Contains(accept, "text/html") {
		return true
	}
	// Check for browser-like user agents that didn't specify Accept
	if accept == "" || accept == "*/*" {
		ua := r.Header.Get("User-Agent")
		// Common browser indicators
		return strings.Contains(ua, "Mozilla") || strings.Contains(ua, "Chrome") || strings.Contains(ua, "Safari")
	}
	return false
}

// StaticHandler returns an http.Handler that serves static files
func StaticHandler() http.Handler {
	// Strip the "static" prefix from the embedded filesystem
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to create static file sub-filesystem: " + err.Error())
	}
	return http.StripPrefix("/public/", http.FileServer(http.FS(subFS)))
}

// PageData is the base data structure for all page templates
type PageData struct {
	Title       string
	Description string
}

// HomePageData contains data for the home page
type HomePageData struct {
	PageData
	Notices []Notice
}

// Notice represents a release notice
type Notice struct {
	Version string
	When    string
}

// ImpostersPageData contains data for the imposters list page
type ImpostersPageData struct {
	PageData
	Imposters []ImposterSummary
}

// ImposterSummary is a summary of an imposter for the list view
type ImposterSummary struct {
	Port             int
	Protocol         string
	Name             string
	NumberOfRequests int
	SelfHref         string
}

// ImposterPageData contains data for an individual imposter page
type ImposterPageData struct {
	PageData
	Protocol string
	Port     int
	Requests []interface{}
	Stubs    []interface{}
}

// LogsPageData contains data for the logs page
type LogsPageData struct {
	PageData
	Logs      []LogEntry
	LogsCount int
}

// LogEntry represents a single log entry
type LogEntry struct {
	Level   string
	Message string
}

// ConfigPageData contains data for the config page
type ConfigPageData struct {
	PageData
	Version string
	Options map[string]interface{}
	Process ProcessInfo
}

// ProcessInfo contains process information
type ProcessInfo struct {
	GoVersion    string
	Architecture string
	Platform     string
	RSS          int64
	HeapAlloc    int64
	Uptime       int64
	Cwd          string
}

// DocPageData contains data for documentation pages
type DocPageData struct {
	PageData
	Section    string // Current section for navigation highlighting
	Subsection string // Current subsection for navigation highlighting
}

// CodeExample represents a code example with language and content
type CodeExample struct {
	Language string
	Code     string
}

// DocSection represents a documentation section with anchor
type DocSection struct {
	ID      string
	Title   string
	Content template.HTML
}
