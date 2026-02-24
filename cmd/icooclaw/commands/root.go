package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/icooclaw/icooclaw/internal/agent"
	"github.com/icooclaw/icooclaw/internal/agent/tools"
	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/channel"
	"github.com/icooclaw/icooclaw/internal/config"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/scheduler"
	"github.com/icooclaw/icooclaw/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Global variables for initialized components
var (
	cfg            *config.Config
	logger         *slog.Logger
	db             *gorm.DB
	messageBus     *bus.MessageBus
	providerReg    *provider.Registry
	channelManager *channel.Manager
	agentInstance  *agent.Agent
	schedulerInst  *scheduler.Scheduler
	toolRegistry   *tools.Registry
)

// RootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "icooclaw",
	Short: "icooclaw - AI 代理 CLI 工具",
	Long: `icooclaw 是一个 AI 代理 CLI 工具，提供多种命令用于与 AI 代理交互、管理定时任务等。

示例:
  icooclaw serve        # 启动 Web 服务器 (WebSocket/Webhook)
  icooclaw chat         # 启动交互式聊天 (REPL)
  icooclaw run "你好"   # 运行单条消息
  icooclaw cron list    # 列出定时任务
  icooclaw config get   # 获取配置`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for help, version, and completion commands
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		return initComponents()
	},
}

// initComponents initializes all required components
func initComponents() error {
	var err error

	// 1. Initialize config
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Initialize logger
	logger = config.InitLogger(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(logger)

	// 3. Initialize database
	db, err = storage.InitDB(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Info("Database initialized", "path", cfg.Database.Path)

	// 4. Initialize message bus
	messageBus = bus.NewMessageBus()
	messageBus.SetLogger(logger)
	logger.Info("Message bus initialized")

	// 5. Initialize provider registry
	providerReg = provider.NewRegistry()
	if err := providerReg.RegisterFromConfig(cfg.Providers); err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}
	logger.Info("Providers registered", "count", providerReg.Count())

	// 6. Initialize channel manager
	channelManager = channel.NewManager(messageBus, cfg.Channels, db, logger)
	logger.Info("Channels registered", "count", channelManager.Count())

	// 7. Create default agent
	agentConfig := cfg.Agents.Defaults
	defaultProviderName := cfg.Agents.DefaultProvider
	if defaultProviderName == "" {
		providers := providerReg.List()
		if len(providers) > 0 {
			defaultProviderName = providers[0]
		}
	}

	defaultProvider, err := providerReg.Get(defaultProviderName)
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	agentInstance = agent.NewAgent(
		agentConfig.Name,
		defaultProvider,
		storage.NewStorage(db),
		agentConfig,
		logger,
	)

	// 8. Initialize tools
	toolRegistry = tools.InitTools(cfg, logger, channelManager)
	agentInstance.SetTools(toolRegistry)

	// 9. Initialize scheduler
	schedulerInst = scheduler.NewScheduler(messageBus, storage.NewStorage(db), cfg, logger)

	return nil
}

// getContext creates a cancellable context
func getContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// handleSignals sets up signal handling
func handleSignals(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n收到中断信号。正在关闭...")
		cancel()
	}()
}

// cleanup performs cleanup operations
func cleanup() {
	logger.Info("正在关闭...")

	// Stop WebSocket channel
	if channelManager != nil {
		if wsCh, err := channelManager.Get("websocket"); err == nil {
			wsCh.Stop()
		}
	}

	// Stop scheduler
	if schedulerInst != nil && schedulerInst.IsRunning() {
		schedulerInst.Stop()
	}

	logger.Info("关闭完成")
}

// printProviders prints available providers
func printProviders() {
	providers := providerReg.List()
	fmt.Println("可用的提供商:")
	for _, name := range providers {
		p, _ := providerReg.Get(name)
		fmt.Printf("  - %s: %s\n", name, p.GetDefaultModel())
	}
}

func Execute() error {
	// Add persistent flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "日志级别 (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "详细输出")

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	return rootCmd.Execute()
}
