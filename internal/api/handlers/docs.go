package handlers

import (
	"net/http"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/web"
)

// Docs is a handler for documentation pages
type Docs struct{}

// NewDocsHandler creates a new docs handler
func NewDocsHandler() *Docs {
	return &Docs{}
}

// docRoutes maps URL paths to template names and section info
var docRoutes = map[string]struct {
	template   string
	section    string
	subsection string
	title      string
}{
	// Getting started section
	"/docs":                 {"docs/gettingStarted.html", "gettingStarted", "", "getting started"},
	"/docs/gettingStarted":  {"docs/gettingStarted.html", "gettingStarted", "", "getting started"},
	"/docs/mentalModel":     {"docs/mentalModel.html", "mentalModel", "", "mental model"},
	"/docs/commandLine":     {"docs/commandLine.html", "commandLine", "", "command line"},
	"/docs/security":        {"docs/security.html", "security", "", "security"},
	"/docs/faq":             {"docs/faq.html", "faq", "", "faqs"},

	// API section
	"/docs/api/overview":         {"docs/api/overview.html", "api", "overview", "api overview"},
	"/docs/api/contracts":        {"docs/api/contracts.html", "api", "contracts", "contracts"},
	"/docs/api/mocks":            {"docs/api/mocks.html", "api", "mocks", "mock verification"},
	"/docs/api/stubs":            {"docs/api/stubs.html", "api", "stubs", "stub responses"},
	"/docs/api/predicates":       {"docs/api/predicates.html", "api", "predicates", "stub predicates"},
	"/docs/api/injection":        {"docs/api/injection.html", "api", "injection", "injection"},
	"/docs/api/errors":           {"docs/api/errors.html", "api", "errors", "errors"},

	// Behaviors
	"/docs/api/behaviors":          {"docs/api/behaviors/overview.html", "api", "behaviors", "behaviors"},
	"/docs/api/behaviors/decorate": {"docs/api/behaviors/decorate.html", "api", "behaviors", "decorate"},
	"/docs/api/behaviors/wait":     {"docs/api/behaviors/wait.html", "api", "behaviors", "wait"},
	"/docs/api/behaviors/repeat":   {"docs/api/behaviors/repeat.html", "api", "behaviors", "repeat"},
	"/docs/api/behaviors/copy":     {"docs/api/behaviors/copy.html", "api", "behaviors", "copy"},
	"/docs/api/behaviors/lookup":   {"docs/api/behaviors/lookup.html", "api", "behaviors", "lookup"},
	"/docs/api/behaviors/shellTransform": {"docs/api/behaviors/shellTransform.html", "api", "behaviors", "shellTransform"},

	// Predicates
	"/docs/api/predicates/equals":       {"docs/api/predicates/equals.html", "api", "predicates", "equals"},
	"/docs/api/predicates/deepEquals":   {"docs/api/predicates/deepEquals.html", "api", "predicates", "deepEquals"},
	"/docs/api/predicates/contains":     {"docs/api/predicates/contains.html", "api", "predicates", "contains"},
	"/docs/api/predicates/startsWith":   {"docs/api/predicates/startsWith.html", "api", "predicates", "startsWith"},
	"/docs/api/predicates/endsWith":     {"docs/api/predicates/endsWith.html", "api", "predicates", "endsWith"},
	"/docs/api/predicates/matches":      {"docs/api/predicates/matches.html", "api", "predicates", "matches"},
	"/docs/api/predicates/exists":       {"docs/api/predicates/exists.html", "api", "predicates", "exists"},
	"/docs/api/predicates/not":          {"docs/api/predicates/not.html", "api", "predicates", "not"},
	"/docs/api/predicates/or":           {"docs/api/predicates/or.html", "api", "predicates", "or"},
	"/docs/api/predicates/and":          {"docs/api/predicates/and.html", "api", "predicates", "and"},
	"/docs/api/predicates/inject":       {"docs/api/predicates/inject.html", "api", "predicates", "inject"},
	"/docs/api/predicates/xpath":        {"docs/api/predicates/xpath.html", "api", "predicates", "xpath"},
	"/docs/api/predicates/jsonpath":     {"docs/api/predicates/jsonpath.html", "api", "predicates", "jsonpath"},

	// Proxy
	"/docs/api/proxies":                {"docs/api/proxy/overview.html", "api", "proxies", "proxies"},
	"/docs/api/proxy/proxyOnce":        {"docs/api/proxy/proxyOnce.html", "api", "proxies", "proxyOnce"},
	"/docs/api/proxy/proxyAlways":      {"docs/api/proxy/proxyAlways.html", "api", "proxies", "proxyAlways"},
	"/docs/api/proxy/proxyTransparent": {"docs/api/proxy/proxyTransparent.html", "api", "proxies", "proxyTransparent"},
	"/docs/api/proxy/predicateGenerators": {"docs/api/proxy/predicateGenerators.html", "api", "proxies", "predicateGenerators"},

	// Faults
	"/docs/api/faults": {"docs/api/faults.html", "api", "faults", "faults"},

	// Protocols
	"/docs/protocols/http":  {"docs/protocols/http.html", "protocols", "http", "http"},
	"/docs/protocols/https": {"docs/protocols/https.html", "protocols", "https", "https"},
	"/docs/protocols/tcp":   {"docs/protocols/tcp.html", "protocols", "tcp", "tcp"},
	"/docs/protocols/smtp":  {"docs/protocols/smtp.html", "protocols", "smtp", "smtp"},
}

// ServeDoc handles GET /docs/* requests
func (d *Docs) ServeDoc(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Normalize path
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/docs"
	}

	route, ok := docRoutes[path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := web.DocPageData{
		PageData: web.PageData{
			Title:       route.title,
			Description: "Documentation for " + route.title,
		},
		Section:    route.section,
		Subsection: route.subsection,
	}

	if err := web.Render(w, route.template, data); err != nil {
		http.Error(w, "Template not found: "+route.template, http.StatusInternalServerError)
	}
}
