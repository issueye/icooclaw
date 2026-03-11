// Package main provides the entry point for icooclaw.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/config"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway"
	"icooclaw/pkg/gateway/websocket"
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
	Short: "启动代理服务",
	Run:   runStart,
}

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "启动网关服务",
	Run:   runGateway,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("icooclaw version:", version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.toml)")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(gatewayCmd)
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

func runGateway(cmd *cobra.Command, args []string) {
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

	slog.Info("正在启动网关服务", "version", version)

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
	defaultModel, err := store.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || (defaultModel != nil && defaultModel.Value == "") {
		slog.Warn("未找到默认模型，需要配置", "key", consts.DEFAULT_MODEL_KEY)
	} else {
		arrs := strings.Split(defaultModel.Value, "/")
		if len(arrs) != 2 {
			slog.Warn("默认模型格式错误，需要配置", "model", defaultModel.Value)
			return
		}

		slog.Info("默认模型", "model", defaultModel.Value)
		defaultProvider, err = providerFactory.Get(arrs[0])
		if err != nil {
			slog.Warn("未找到默认提供商，需要配置", "provider", arrs[0])
		}
	}

	// Initialize agent loop
	agentLoop := agent.NewLoop(
		agent.WithLoopBus(messageBus),
		agent.WithLoopProvider(defaultProvider),
		agent.WithLoopTools(toolRegistry),
		agent.WithLoopMemory(memLoader),
		agent.WithLoopStorage(store),
		agent.WithLoopLogger(slog.Default()),
	)

	// Initialize agent registry
	agentRegistry := agent.NewAgentRegistry(slog.Default())

	// Create gateway server config
	serverCfg := gateway.DefaultServerConfig()
	if cfg.Gateway.Port > 0 {
		serverCfg.Addr = fmt.Sprintf(":%d", cfg.Gateway.Port)
	}

	// Create gateway server
	gw := gateway.NewServer(serverCfg, store, slog.Default()).
		WithWebSocket(websocket.DefaultManagerConfig()).
		WithSSE().
		WithBus(messageBus).
		WithAgentLoop(agentLoop).
		WithAgentRegistry(agentRegistry).
		Setup()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("正在关闭网关服务...")

		// Shutdown gateway server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := gw.Shutdown(shutdownCtx); err != nil {
			slog.Error("网关服务关闭失败", "error", err)
		}

		cancel()
	}()

	// Start agent loop in background
	go func() {
		if err := agentLoop.Run(ctx); err != nil && err != context.Canceled {
			slog.Error("代理循环错误", "error", err)
		}
	}()

	// Start gateway server
	if err := gw.Start(); err != nil && err != http.ErrServerClosed {
		slog.Error("网关服务错误", "error", err)
		os.Exit(1)
	}

	slog.Info("网关服务已停止")
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
