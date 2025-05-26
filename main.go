package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hypr-technologies/hyprknot/internal/api"
	"github.com/hypr-technologies/hyprknot/internal/config"
	"github.com/hypr-technologies/hyprknot/internal/knot"
	"github.com/hypr-technologies/hyprknot/internal/logger"
)

const (
	appName = "hyprknot"
)

var (
	appVersion = "dev" // Will be overridden by linker flags during build
)

func main() {
	// Parse command line flags
	var (
		configPath  = flag.String("config", "", "Path to configuration file")
		showHelp    = flag.Bool("help", false, "Show help message")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.NewLogger(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.Infof("Starting %s version %s", appName, appVersion)
	log.Infof("Configuration loaded from: %s", *configPath)

	// Initialize KnotDNS client
	knotClient := knot.NewClient(
		cfg.Knot.KnotcPath,
		cfg.Knot.SocketPath,
		cfg.Knot.AllowedZones,
		log,
	)

	// Test KnotDNS connection
	if err := knotClient.CheckHealth(); err != nil {
		log.Fatalf("KnotDNS health check failed: %v", err)
	}
	log.Info("KnotDNS connection established")

	// Setup routes
	router := api.SetupRoutes(cfg, knotClient, log)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.GetAddress(),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Starting HTTP server on %s", cfg.GetAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	log.Info("Server shutdown complete")
}

func printHelp() {
	fmt.Printf(`%s - Lightweight HTTP API wrapper for KnotDNS

USAGE:
    %s [OPTIONS]

OPTIONS:
    -config string    Path to configuration file (optional)
    -help            Show this help message
    -version         Show version information

EXAMPLES:
    # Run with default configuration
    %s

    # Run with custom configuration file
    %s -config /etc/hyprknot/config.yaml

CONFIGURATION:
    If no configuration file is specified, the application will use default settings.
    You can generate a sample configuration file by running the application once
    and then modifying the generated config.yaml file.

API ENDPOINTS:
    GET  /health                                    - Health check
    GET  /api/v1/zones                             - List zones
    GET  /api/v1/zones/{zone}/records              - List records in zone
    GET  /api/v1/zones/{zone}/records/{name}/{type} - Get specific record
    POST /api/v1/zones/{zone}/records              - Create record
    PUT  /api/v1/zones/{zone}/records/{name}/{type} - Update record
    DELETE /api/v1/zones/{zone}/records/{name}/{type} - Delete record
    POST /api/v1/zones/{zone}/reload               - Reload zone

AUTHENTICATION:
    API endpoints (except /health) require authentication via API key.
    Include the API key in the X-API-Key header or Authorization: Bearer header.

SUPPORTED RECORD TYPES:
    A, AAAA, PTR, CNAME, MX, TXT, NS

For more information, visit: https://github.com/hyprknot/hyprknot
`, appName, appName, appName, appName)
}
