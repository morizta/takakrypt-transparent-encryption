package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/takakrypt/transparent-encryption/internal/agent"
	"github.com/takakrypt/transparent-encryption/internal/config"
)

var (
	configDir = flag.String("config", "./", "Configuration directory path")
	logLevel  = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	daemon    = flag.Bool("daemon", false, "Run as daemon")
)

func main() {
	flag.Parse()

	cfg, err := config.Load(*configDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	agentService, err := agent.New(cfg, *configDir)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down agent...")
		cancel()
	}()

	if err := agentService.Start(ctx); err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	fmt.Println("Agent stopped gracefully")
}