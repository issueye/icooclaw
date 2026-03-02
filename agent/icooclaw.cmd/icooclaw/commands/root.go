package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"icooclaw.ai/agent"
	"icooclaw.ai/config"
	"icooclaw.ai/provider"
	"icooclaw.ai/storage"
	"icooclaw.ai/tools"
	bus "icooclaw.bus"
	channel "icooclaw.channel"
	scheduler "icooclaw.scheduler"
	utils "icooclaw.utils"
)

// 全局组件
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

// 错误定义
var ErrNotInitialized = errors.New("components not initialized")

// RootCmd 代表根命令
var rootCmd = &cobra.Command{
	Use:   "icooclaw",
	Short: "icooclaw - AI Agent CLI Tool",
	Long: `icooclaw is an AI Agent CLI tool.

Examples:
  icooclaw serve        # Start web server (WebSocket/Webhook)
  icooclaw chat         # Start interactive chat (REPL)
  icooclaw run "hello"  # Run single message
  icooclaw cron list    # List scheduled tasks
  icooclaw config get   # Get configuration`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 跳过不需要初始化的命令
		switch cmd.Name() {
		case "help", "version", "completion":
			return nil
		}
		return initComponents()
	},
}

// initComponents 初始化所有必需的组件
func initComponents() error {
	var err error

	// Step 1: 加载配置
	if cfg, err = config.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Step 2: 初始化日志
	logger = config.InitLogger(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	slog.SetDefault(logger)

	// Step 3: 初始化工作空间
	workspace, err := initWorkspace()
	if err != nil {
		return fmt.Errorf("init workspace: %w", err)
	}

	// Step 4: 初始化数据库
	if db, err = storage.InitDB(cfg.Database.Path); err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	logger.Info("Database initialized", "path", cfg.Database.Path)

	// Step 5: 初始化消息总线
	messageBus = bus.NewMessageBus()
	messageBus.SetLogger(logger)

	// Step 6: 初始化 Provider 注册表
	providerReg = provider.NewRegistry()
	if err = providerReg.RegisterFromConfig(cfg.Providers); err != nil {
		return fmt.Errorf("register providers: %w", err)
	}
	logger.Info("Providers registered", "count", providerReg.Count())

	// Step 7: 初始化通道管理器
	channelManager = channel.NewManager(messageBus, cfg.Channels, db, logger)
	logger.Info("Channels registered", "count", channelManager.Count())

	// Step 8: 创建默认 Agent
	if err = initDefaultAgent(workspace); err != nil {
		return fmt.Errorf("init agent: %w", err)
	}

	// Step 9: 初始化工具
	toolRegistry = tools.InitTools(cfg, logger, channelManager)
	agentInstance.SetTools(toolRegistry)

	// Step 10: 初始化调度器
	schedulerInst = scheduler.NewScheduler(messageBus, storage.NewStorage(db), cfg, logger)
	logger.Info("All components initialized")

	return nil
}

// initWorkspace 初始化工作空间
func initWorkspace() (string, error) {
	workspace, err := utils.ExpandPath(cfg.Workspace)
	if err != nil {
		return "", fmt.Errorf("expand workspace path: %w", err)
	}

	if workspace == "" {
		workspace = cfg.Tools.Workspace
	}

	if workspace != "" {
		if err := config.InitWorkspace(workspace); err != nil {
			return "", fmt.Errorf("init workspace directory: %w", err)
		}
		logger.Info("Workspace initialized", "path", workspace)
	}

	return workspace, nil
}

// initDefaultAgent 初始化默认 Agent
func initDefaultAgent(workspace string) error {
	agentConfig := cfg.Agents.Defaults
	defaultProviderName := cfg.Agents.DefaultProvider

	if defaultProviderName == "" {
		providers := providerReg.List()
		if len(providers) == 0 {
			return errors.New("no providers available")
		}
		defaultProviderName = providers[0]
	}

	defaultProvider, err := providerReg.Get(defaultProviderName)
	if err != nil {
		return fmt.Errorf("get default provider '%s': %w", defaultProviderName, err)
	}

	agentInstance = agent.NewAgent(
		agentConfig.Name,
		defaultProvider,
		storage.NewStorage(db),
		agentConfig,
		logger,
		workspace,
	)
	logger.Info("Agent initialized", "name", agentConfig.Name, "provider", defaultProviderName)

	return nil
}

// getContext 创建一个可取消的上下文
func getContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// handleSignals 设置信号处理
func handleSignals(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		if logger != nil {
			logger.Info("Received signal, shutting down", "signal", sig.String())
		}
		cancel()
	}()
}

// cleanup 执行清理操作
func cleanup() {
	if logger == nil {
		return
	}
	logger.Info("Cleaning up...")

	// 停止通道
	if channelManager != nil {
		if wsCh, err := channelManager.Get("websocket"); err == nil {
			wsCh.Stop()
		}
	}

	// 停止调度器
	if schedulerInst != nil && schedulerInst.IsRunning() {
		schedulerInst.Stop()
	}

	logger.Info("Cleanup completed")
}

// checkInitialized 检查组件是否已初始化
func checkInitialized() error {
	if agentInstance == nil {
		return ErrNotInitialized
	}
	return nil
}

// printProviders 打印可用的提供商
func printProviders() {
	if providerReg == nil {
		fmt.Println("Providers not initialized")
		return
	}

	providers := providerReg.List()
	fmt.Println("Available providers:")
	for _, name := range providers {
		p, err := providerReg.Get(name)
		if err != nil {
			continue
		}
		fmt.Printf("  - %s: %s\n", name, p.GetDefaultModel())
	}
}

func Execute() error {
	// 设置模板目录路径
	if err := findTemplatesDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to find templates directory: %v\n", err)
	}

	// 添加持久化标志
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// 绑定标志到 viper
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	return rootCmd.Execute()
}

// findTemplatesDir 查找 templates 目录并设置到 config.TemplatesDir
func findTemplatesDir() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	searchPaths := []string{
		"./templates",
		filepath.Join(filepath.Dir(execPath), "templates"),
		filepath.Join(filepath.Dir(execPath), "..", "templates"),
		filepath.Join(filepath.Dir(execPath), "..", "..", "templates"),
	}

	for _, path := range searchPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			config.TemplatesDir = absPath
			return nil
		}
	}

	return fmt.Errorf("templates directory not found in search paths")
}
