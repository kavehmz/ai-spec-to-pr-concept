// Package main is the entry point for the hub service.
// It initializes the hub and registers endpoints.
package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"trading/internal/date"
	"trading/internal/hub"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to listen on")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Create hub configuration
	config := hub.Config{
		Port:     *port,
		LogLevel: *logLevel,
	}

	// Create a new hub
	p := hub.New(config)

	// Register endpoints
	dateEndpoint := date.New(date.Config{})
	p.RegisterEndpoint("date", dateEndpoint)

	// Set up logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevel(*logLevel),
	}))
	slog.SetDefault(logger)

	// Log startup
	slog.Info("Starting hub service", "port", *port, "log_level", *logLevel)

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start the hub in a goroutine
	errChan := make(chan error)
	go func() {
		errChan <- p.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		if err != nil {
			slog.Error("Hub error", "error", err)
			os.Exit(1)
		}
	case <-shutdown:
		slog.Info("Shutting down hub service")
	}
}

// getLogLevel converts a string log level to a slog.Level
func getLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
