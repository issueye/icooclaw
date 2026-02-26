package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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

// 初始化组件的全局变量
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

// RootCmd 代表根命令
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
		// 跳过 help、version 和 completion 命令的初始化
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		return initComponents()
	},
}

// initComponents 初始化所有必需的组件
func initComponents() error {
	var err error

	// 1. 初始化配置文件
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. 初始化日志
	logger = config.InitLogger(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	slog.SetDefault(logger)

	// 3. 初始化工作空间目录
	workspace := cfg.Workspace
	if workspace == "" {
		workspace = cfg.Tools.Workspace
	}
	if workspace != "" {
		if err := config.InitWorkspace(workspace); err != nil {
			return fmt.Errorf("failed to initialize workspace: %w", err)
		}
		logger.Info("Workspace initialized", "path", workspace)
	}

	// 4. 初始化数据库
	db, err = storage.InitDB(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Info("Database initialized", "path", cfg.Database.Path)

	// 4. 初始化消息总线
	messageBus = bus.NewMessageBus()
	messageBus.SetLogger(logger)
	logger.Info("Message bus initialized")

	// 5. 初始化提供者注册表
	providerReg = provider.NewRegistry()
	if err := providerReg.RegisterFromConfig(cfg.Providers); err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}
	logger.Info("Providers registered", "count", providerReg.Count())

	// 6. 初始化通道管理器
	channelManager = channel.NewManager(messageBus, cfg.Channels, db, logger)
	logger.Info("Channels registered", "count", channelManager.Count())

	// 7. 创建默认代理
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
		workspace,
	)

	// 8. 初始化工具
	toolRegistry = tools.InitTools(cfg, logger, channelManager)
	agentInstance.SetTools(toolRegistry)

	// 9. 初始化调度器
	schedulerInst = scheduler.NewScheduler(messageBus, storage.NewStorage(db), cfg, logger)

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
		<-sigCh
		fmt.Println("\n收到中断信号。正在关闭...")
		cancel()
	}()
}

// cleanup 执行清理操作
func cleanup() {
	logger.Info("正在关闭...")

	// 停止 WebSocket 通道
	if channelManager != nil {
		if wsCh, err := channelManager.Get("websocket"); err == nil {
			wsCh.Stop()
		}
	}

	// 停止调度器
	if schedulerInst != nil && schedulerInst.IsRunning() {
		schedulerInst.Stop()
	}

	logger.Info("关闭完成")
}

// printProviders 打印可用的提供商
func printProviders() {
	providers := providerReg.List()
	fmt.Println("可用的提供商:")
	for _, name := range providers {
		p, _ := providerReg.Get(name)
		fmt.Printf("  - %s: %s\n", name, p.GetDefaultModel())
	}
}

func Execute() error {
	// 设置模板目录路径
	if err := findTemplatesDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to find templates directory: %v\n", err)
	}

	// 添加持久化标志
	rootCmd.PersistentFlags().StringP("config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "日志级别 (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "详细输出")

	// 绑定标志到 viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	return rootCmd.Execute()
}

// findTemplatesDir 查找 templates 目录并设置到 config.TemplatesDir
func findTemplatesDir() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
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

	return fmt.Errorf("templates directory not found")
}
