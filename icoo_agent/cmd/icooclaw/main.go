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
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// Ensure directories exist
	if err := cfg.EnsureWorkspace(); err != nil {
		slog.Error("创建工作目录失败", "error", err)
		os.Exit(1)
	}
	if err := cfg.EnsureDatabasePath(); err != nil {
		slog.Error("创建数据库目录失败", "error", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg)

	slog.Info("正在启动 icooclaw", "version", version)

	// Initialize storage
	dbPath, _ := cfg.GetDatabasePath()
	store, err := storage.New(dbPath)
	if err != nil {
		slog.Error("初始化存储失败", "error", err)
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

	// Get default provider
	var defaultProvider providers.Provider
	defaultProvider, err = providerFactory.Get(cfg.Agent.DefaultProvider.ToString())
	if err != nil {
		slog.Warn("未找到默认提供商，需要配置", "provider", cfg.Agent.DefaultProvider)
	}

	// Initialize agent instance
	agentInstance := agent.NewAgentInstance(agent.AgentConfig{
		Name:              "default",
		Model:             cfg.Agent.DefaultModel,
		MaxToolIterations: 20,
	},
		agent.WithAgentBus(messageBus),
		agent.WithAgentStorage(store),
		agent.WithAgentTools(toolRegistry),
		agent.WithAgentMemory(memLoader),
		agent.WithAgentLogger(slog.Default()),
	)

	if defaultProvider != nil {
		agentInstance = agent.NewAgentInstance(agent.AgentConfig{
			Name:              "default",
			Model:             cfg.Agent.DefaultModel,
			MaxToolIterations: 20,
		},
			agent.WithAgentBus(messageBus),
			agent.WithAgentStorage(store),
			agent.WithAgentTools(toolRegistry),
			agent.WithAgentMemory(memLoader),
			agent.WithAgentProvider(defaultProvider),
			agent.WithAgentLogger(slog.Default()),
		)
	}

	// Initialize agent loop
	loop := agent.NewLoop(
		agent.WithLoopBus(messageBus),
		agent.WithLoopProvider(defaultProvider),
		agent.WithLoopTools(toolRegistry),
		agent.WithLoopMemory(memLoader),
		agent.WithLoopStorage(store),
		agent.WithLoopLogger(slog.Default()),
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("正在关闭...")
		cancel()
	}()

	// Start agent loop
	slog.Info("代理已启动", "name", agentInstance.Name())
	if err := loop.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("代理运行错误", "error", err)
		os.Exit(1)
	}

	slog.Info("代理已停止")
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
