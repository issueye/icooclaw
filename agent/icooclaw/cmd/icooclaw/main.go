// Package main provides the entry point for icooclaw.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/config"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

var (
	cfgFile string
	version = "dev"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "icooclaw",
	Short: "AI Agent Framework",
	Long:  `icooclaw is an AI agent framework with multi-channel support.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the agent",
	Run:   runStart,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("icooclaw version:", version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.toml)")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Ensure directories exist
	if err := cfg.EnsureWorkspace(); err != nil {
		slog.Error("failed to create workspace", "error", err)
		os.Exit(1)
	}
	if err := cfg.EnsureDatabasePath(); err != nil {
		slog.Error("failed to create database directory", "error", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg)

	slog.Info("starting icooclaw", "version", version)

	// Initialize storage
	dbPath, _ := cfg.GetDatabasePath()
	store, err := storage.New(dbPath)
	if err != nil {
		slog.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize message bus
	messageBus := bus.NewMessageBus(bus.DefaultConfig())

	// Initialize provider factory
	providerFactory := providers.NewFactory(store)

	// Initialize tools registry
	toolRegistry := tools.NewRegistry()

	// Initialize memory loader
	memLoader := memory.NewLoader(store, 100, slog.Default())

	// Initialize agent
	ag := agent.New("default",
		agent.WithBus(messageBus),
		agent.WithStorage(store),
		agent.WithTools(toolRegistry),
		agent.WithMemory(memLoader),
		agent.WithLogger(slog.Default()),
	)

	// Get default provider
	defaultProvider, err := providerFactory.Get(cfg.Agent.DefaultProvider.ToString())
	if err != nil {
		slog.Warn("default provider not found, will need to configure", "provider", cfg.Agent.DefaultProvider)
	} else {
		ag = agent.New("default",
			agent.WithBus(messageBus),
			agent.WithStorage(store),
			agent.WithTools(toolRegistry),
			agent.WithMemory(memLoader),
			agent.WithProvider(defaultProvider),
			agent.WithLogger(slog.Default()),
		)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("shutting down...")
		cancel()
	}()

	// Start agent
	slog.Info("agent started", "name", ag.Name())
	if err := ag.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("agent error", "error", err)
		os.Exit(1)
	}

	slog.Info("agent stopped")
}

func setupLogging(cfg *config.Config) {
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Logging.Level),
	}

	var handler slog.Handler
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func parseLogLevel(level string) slog.Level {
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
