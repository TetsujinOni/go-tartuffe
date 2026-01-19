package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TetsujinOni/go-tartuffe/internal/api"
	"github.com/TetsujinOni/go-tartuffe/internal/config"
	"github.com/TetsujinOni/go-tartuffe/pkg/version"
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "save":
			runSave()
			return
		case "replay":
			runReplay()
			return
		case "stop":
			runStop()
			return
		}
	}

	// Default: start command
	runStart()
}

func runStart() {
	// Define command line flags
	port := flag.Int("port", 2525, "the port to run the mountebank server on")
	host := flag.String("host", "", "the hostname to bind the mountebank server to")
	allowInjection := flag.Bool("allowInjection", false, "set to allow JavaScript injection")
	localOnly := flag.Bool("localOnly", false, "only accept requests from localhost")
	showVersion := flag.Bool("version", false, "show version information")

	// Config file options
	configFile := flag.String("configfile", "", "file to load imposters from, can be an EJS template")
	noParse := flag.Bool("noParse", false, "prevent EJS template rendering, treat config as raw JSON")

	// Logging options
	logLevel := flag.String("loglevel", "info", "level for logging (debug, info, warn, error)")
	logFile := flag.String("logfile", "mb.log", "path to use for logging")
	noLogFile := flag.Bool("nologfile", false, "prevent logging to the filesystem")

	// Other options
	pidFile := flag.String("pidfile", "mb.pid", "where the pid is stored for the stop command")
	debug := flag.Bool("debug", false, "include stub match information in imposter retrievals")
	ipWhitelist := flag.String("ipWhitelist", "*", "pipe-delimited list of allowed IP addresses")
	origin := flag.String("origin", "", "safe origin for CORS requests")
	apiKey := flag.String("apikey", "", "API key for authentication")

	// Persistence options
	dataDir := flag.String("datadir", "", "directory to persist imposters to")

	// Plugin options
	protoFile := flag.String("protofile", "", "path to protocols.json for custom protocols")
	pluginsDir := flag.String("plugins", "", "directory containing Go plugin .so files")
	impostersRepository := flag.String("impostersRepository", "", "repository connection string (e.g., redis://localhost:6379)")

	// Formatter options
	formatter := flag.String("formatter", "", "path to custom formatter module (Go plugin)")

	flag.Parse()

	// Formatter is a future enhancement - for now just log if specified
	if *formatter != "" {
		log.Printf("custom formatter specified: %s (not yet implemented)", *formatter)
	}

	// Handle version flag
	if *showVersion {
		fmt.Printf("go-tartuffe version %s (compatible with mountebank %s)\n",
			version.Version, version.MountebankVersion)
		os.Exit(0)
	}

	// Set up logging
	setupLogging(*logLevel, *logFile, *noLogFile)

	// Write PID file
	if *pidFile != "" {
		if err := os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			log.Printf("warning: failed to write pid file: %v", err)
		}
	}

	// Create server
	srv := api.NewServer(api.ServerConfig{
		Port:                *port,
		Host:                *host,
		AllowInjection:      *allowInjection,
		LocalOnly:           *localOnly,
		Debug:               *debug,
		IPWhitelist:         *ipWhitelist,
		Origin:              *origin,
		APIKey:              *apiKey,
		DataDir:             *dataDir,
		ProtoFile:           *protoFile,
		PluginsDir:          *pluginsDir,
		ImpostersRepository: *impostersRepository,
	})

	// Handle graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Load persisted imposters from datadir (if using filesystem repository)
	if *dataDir != "" {
		if err := srv.LoadPersistedImposters(); err != nil {
			log.Printf("warning: failed to load persisted imposters: %v", err)
		}
	}

	// Load config file if specified
	if *configFile != "" {
		log.Printf("loading config from %s", *configFile)
		cfg, err := config.LoadFile(*configFile, *noParse)
		if err != nil {
			log.Fatalf("failed to load config file: %v", err)
		}

		// Load imposters into server
		if err := srv.LoadImposters(cfg.Imposters); err != nil {
			log.Fatalf("failed to load imposters: %v", err)
		}
		log.Printf("loaded %d imposters from config file", len(cfg.Imposters))
	}

	// Wait for shutdown signal
	<-done
	log.Println("shutting down...")

	// Remove PID file
	if *pidFile != "" {
		os.Remove(*pidFile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	log.Println("server stopped")
}

func runSave() {
	saveFlags := flag.NewFlagSet("save", flag.ExitOnError)
	port := saveFlags.Int("port", 2525, "the port mountebank is running on")
	host := saveFlags.String("host", "localhost", "the hostname mountebank is running on")
	saveFile := saveFlags.String("savefile", "mb.json", "file to save imposters to")
	removeProxies := saveFlags.Bool("removeProxies", false, "removes proxies from the configuration")
	formatterPath := saveFlags.String("formatter", "", "path to custom formatter (not implemented)")
	apiKey := saveFlags.String("apikey", "", "API key for authentication")

	saveFlags.Parse(os.Args[2:])

	// Formatter is a future enhancement
	if *formatterPath != "" {
		log.Printf("custom formatter specified: %s (not yet implemented)", *formatterPath)
	}

	// Get imposters from running server
	url := fmt.Sprintf("http://%s:%d/imposters?replayable=true", *host, *port)
	if *removeProxies {
		url += "&removeProxies=true"
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}
	if *apiKey != "" {
		req.Header.Set("X-Api-Key", *apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to connect to mountebank: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("failed to get imposters: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}

	// Pretty print the JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatalf("failed to parse response: %v", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("failed to format JSON: %v", err)
	}

	// Write to file
	if err := os.WriteFile(*saveFile, prettyJSON, 0644); err != nil {
		log.Fatalf("failed to write save file: %v", err)
	}

	fmt.Printf("saved imposters to %s\n", *saveFile)
}

func runReplay() {
	replayFlags := flag.NewFlagSet("replay", flag.ExitOnError)
	port := replayFlags.Int("port", 2525, "the port mountebank is running on")
	host := replayFlags.String("host", "localhost", "the hostname mountebank is running on")
	apiKey := replayFlags.String("apikey", "", "API key for authentication")

	replayFlags.Parse(os.Args[2:])

	// Get imposters with removeProxies
	getURL := fmt.Sprintf("http://%s:%d/imposters?replayable=true&removeProxies=true", *host, *port)
	client := &http.Client{}
	getReq, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}
	if *apiKey != "" {
		getReq.Header.Set("X-Api-Key", *apiKey)
	}

	resp, err := client.Do(getReq)
	if err != nil {
		log.Fatalf("failed to connect to mountebank: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("failed to get imposters: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}

	// PUT the imposters back (without proxies)
	putURL := fmt.Sprintf("http://%s:%d/imposters", *host, *port)
	putReq, err := http.NewRequest("PUT", putURL, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	if *apiKey != "" {
		putReq.Header.Set("X-Api-Key", *apiKey)
	}

	putResp, err := client.Do(putReq)
	if err != nil {
		log.Fatalf("failed to PUT imposters: %v", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != 200 {
		respBody, _ := io.ReadAll(putResp.Body)
		log.Fatalf("failed to replay imposters: %s", string(respBody))
	}

	fmt.Println("switched to replay mode (proxies removed)")
}

func runStop() {
	stopFlags := flag.NewFlagSet("stop", flag.ExitOnError)
	pidFile := stopFlags.String("pidfile", "mb.pid", "where the pid is stored")

	stopFlags.Parse(os.Args[2:])

	// Read PID from file
	data, err := os.ReadFile(*pidFile)
	if err != nil {
		// If pidfile doesn't exist, there's nothing to stop - exit successfully
		// This matches mountebank's behavior for compatibility with test harness
		if os.IsNotExist(err) {
			fmt.Println("no pidfile found, nothing to stop")
			os.Exit(0)
		}
		log.Fatalf("failed to read pid file: %v", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		log.Fatalf("invalid pid in file: %v", err)
	}

	// Send SIGTERM to process
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("failed to find process: %v", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead, which is fine - exit successfully
		if err == os.ErrProcessDone {
			fmt.Printf("process %d already stopped\n", pid)
			os.Remove(*pidFile)
			os.Exit(0)
		}
		log.Fatalf("failed to stop process: %v", err)
	}

	// Remove pidfile after successful stop
	os.Remove(*pidFile)
	fmt.Printf("stopped mountebank process %d\n", pid)
}

func setupLogging(level, file string, noFile bool) {
	// For now, just set basic logging
	// Could be enhanced with proper log levels and file output
	if !noFile && file != "" {
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stdout, f))
		}
	}

	// Log level filtering would require a custom logger
	_ = level
}
