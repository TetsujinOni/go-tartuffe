package repository

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseConnectionString parses a repository connection string into a Config.
// Supported formats:
//   - memory://                              - In-memory repository
//   - file:///path/to/dir                    - Filesystem repository
//   - redis://host:port/db?prefix=mb:        - Redis repository
//   - postgres://user:pass@host:port/db      - PostgreSQL repository
//   - mongodb://user:pass@host:port/db       - MongoDB repository
func ParseConnectionString(connStr string) (*Config, error) {
	// Handle special case: empty string means in-memory
	if connStr == "" {
		return &Config{
			Scheme: "memory",
		}, nil
	}

	// Parse as URL
	u, err := url.Parse(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	cfg := &Config{
		Scheme:  u.Scheme,
		Options: make(map[string]interface{}),
	}

	switch u.Scheme {
	case "memory":
		// No additional config needed

	case "file":
		// file:///path/to/dir
		cfg.ConnectionString = u.Path
		if cfg.ConnectionString == "" {
			return nil, fmt.Errorf("file scheme requires a path")
		}

	case "redis":
		// redis://host:port/db?prefix=mb:
		cfg.ConnectionString = connStr

		// Parse host and port
		host := u.Hostname()
		if host == "" {
			host = "localhost"
		}
		port := u.Port()
		if port == "" {
			port = "6379"
		}
		cfg.Options["host"] = host
		cfg.Options["port"] = port

		// Parse database number from path
		if u.Path != "" && u.Path != "/" {
			dbStr := strings.TrimPrefix(u.Path, "/")
			if db, err := strconv.Atoi(dbStr); err == nil {
				cfg.Options["db"] = db
			}
		}

		// Parse password
		if u.User != nil {
			if pass, ok := u.User.Password(); ok {
				cfg.Options["password"] = pass
			}
		}

		// Parse query parameters
		for key, values := range u.Query() {
			if len(values) > 0 {
				cfg.Options[key] = values[0]
			}
		}

	case "postgres", "postgresql":
		// postgres://user:pass@host:port/db
		cfg.ConnectionString = connStr
		cfg.Options["host"] = u.Hostname()
		cfg.Options["port"] = u.Port()
		if u.User != nil {
			cfg.Options["user"] = u.User.Username()
			if pass, ok := u.User.Password(); ok {
				cfg.Options["password"] = pass
			}
		}
		if u.Path != "" && u.Path != "/" {
			cfg.Options["database"] = strings.TrimPrefix(u.Path, "/")
		}
		for key, values := range u.Query() {
			if len(values) > 0 {
				cfg.Options[key] = values[0]
			}
		}

	case "mongodb", "mongodb+srv":
		// mongodb://user:pass@host:port/db
		cfg.ConnectionString = connStr
		cfg.Options["host"] = u.Hostname()
		cfg.Options["port"] = u.Port()
		if u.User != nil {
			cfg.Options["user"] = u.User.Username()
			if pass, ok := u.User.Password(); ok {
				cfg.Options["password"] = pass
			}
		}
		if u.Path != "" && u.Path != "/" {
			cfg.Options["database"] = strings.TrimPrefix(u.Path, "/")
		}
		for key, values := range u.Query() {
			if len(values) > 0 {
				cfg.Options[key] = values[0]
			}
		}

	default:
		// For unknown schemes, just pass through the connection string
		cfg.ConnectionString = connStr
	}

	return cfg, nil
}

// GetScheme extracts just the scheme from a connection string
func GetScheme(connStr string) string {
	if connStr == "" {
		return "memory"
	}

	idx := strings.Index(connStr, "://")
	if idx == -1 {
		return ""
	}

	return connStr[:idx]
}
