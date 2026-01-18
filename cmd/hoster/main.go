package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// Version information (set by build)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("hoster %s (built %s)\n", Version, BuildTime)
		return ExitSuccess
	}

	// Load configuration
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		return ExitConfigError
	}

	// Setup logger
	logger := SetupLogger(cfg)
	logger.Info("starting hoster",
		"version", Version,
		"config", *configPath,
	)

	// Create server
	server, err := NewServer(cfg, logger)
	if err != nil {
		if sErr, ok := err.(*ServerError); ok {
			logger.Error("failed to create server",
				"error", sErr.Err,
				"operation", sErr.Op,
			)
			return sErr.ExitCode
		}
		logger.Error("failed to create server", "error", err)
		return ExitConfigError
	}

	// Start server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		if sErr, ok := err.(*ServerError); ok {
			logger.Error("server error",
				"error", sErr.Err,
				"operation", sErr.Op,
			)
			return sErr.ExitCode
		}
		logger.Error("server error", "error", err)
		return ExitConfigError
	}

	return ExitSuccess
}
