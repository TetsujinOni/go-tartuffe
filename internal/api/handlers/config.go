package handlers

import (
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/TetsujinOni/go-tartuffe/internal/response"
	"github.com/TetsujinOni/go-tartuffe/pkg/version"
)

// ConfigResponse is the response for GET /config
type ConfigResponse struct {
	Version string        `json:"version"`
	Options ConfigOptions `json:"options"`
	Process ProcessInfo   `json:"process"`
}

// ConfigOptions contains server configuration
type ConfigOptions struct {
	Port           int      `json:"port"`
	Host           string   `json:"host,omitempty"`
	AllowInjection bool     `json:"allowInjection"`
	LocalOnly      bool     `json:"localOnly"`
	IPWhitelist    []string `json:"ipWhitelist,omitempty"`
	Debug          bool     `json:"debug"`
	Origin         string   `json:"origin,omitempty"`
}

// ProcessInfo contains process information
type ProcessInfo struct {
	GoVersion    string `json:"goVersion"`
	Architecture string `json:"architecture"`
	Platform     string `json:"platform"`
	RSS          uint64 `json:"rss"`
	HeapAlloc    uint64 `json:"heapAlloc"`
	HeapTotal    uint64 `json:"heapTotal"`
	Uptime       int64  `json:"uptime"`
	Cwd          string `json:"cwd"`
}

// Config is a handler for GET /config
type Config struct {
	port           int
	host           string
	allowInjection bool
	localOnly      bool
	debug          bool
	ipWhitelist    string
	origin         string
	startTime      int64
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(port int, host string, allowInjection, localOnly, debug bool, ipWhitelist, origin string, startTime int64) *Config {
	return &Config{
		port:           port,
		host:           host,
		allowInjection: allowInjection,
		localOnly:      localOnly,
		debug:          debug,
		ipWhitelist:    ipWhitelist,
		origin:         origin,
		startTime:      startTime,
	}
}

// GetConfig handles GET /config
func (c *Config) GetConfig(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cwd, _ := os.Getwd()

	// Parse IP whitelist into array
	var ipWhitelist []string
	if c.ipWhitelist != "" {
		for _, ip := range strings.Split(c.ipWhitelist, "|") {
			if ip != "" {
				ipWhitelist = append(ipWhitelist, ip)
			}
		}
	}

	resp := ConfigResponse{
		Version: version.Version,
		Options: ConfigOptions{
			Port:           c.port,
			Host:           c.host,
			AllowInjection: c.allowInjection,
			LocalOnly:      c.localOnly,
			Debug:          c.debug,
			IPWhitelist:    ipWhitelist,
			Origin:         c.origin,
		},
		Process: ProcessInfo{
			GoVersion:    runtime.Version(),
			Architecture: runtime.GOARCH,
			Platform:     runtime.GOOS,
			RSS:          m.Sys,
			HeapAlloc:    m.HeapAlloc,
			HeapTotal:    m.HeapSys,
			Cwd:          cwd,
		},
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
