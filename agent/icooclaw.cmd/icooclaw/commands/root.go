package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"icooclaw.ai/agent"
	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
	bus "icooclaw.core/bus"
	channel "icooclaw.core/channel"
	"icooclaw.core/config"
	scheduler "icooclaw.core/scheduler"
	"icooclaw.core/storage"
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
  icooclaw gateway        # 启动网关
  icooclaw chat         # Start interactive chat (REPL)
  icooclaw run "hello"  # Run single message
  icooclaw cron list    # List scheduled tasks
  icooclaw config get   # Get configuration`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 跳过不需要初始化的命令
		switch cmd.Name() {
		case "help", "version", "completion", "gateway":
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

	// Step 3: 初始化工作空间（检查并创建关键文件）
	// wsConfig, err := config.InitWorkspaceWithConfig(cfg)
	// if err != nil {
	// 	return fmt.Errorf("init workspace: %w", err)
	// }
	// workspace := wsConfig.Path

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
	// channelManager = channel.NewManager(
	// 	&messageBusAdapter{bus: messageBus},
	// 	&channelConfigAdapter{cfg: cfg.Channels},
	// 	&storageReaderAdapter{storage: storage.NewStorage(db)},
	// 	logger,
	// )
	logger.Info("Channels registered", "count", channelManager.Count())

	// Step 8: 创建默认 Agent
	// if err = initDefaultAgent(workspace); err != nil {
	// 	return fmt.Errorf("init agent: %w", err)
	// }

	// Step 9: 初始化工具
	toolRegistry = tools.InitTools(cfg, logger, channelManager)
	agentInstance.SetTools(toolRegistry)

	logger.Info("All components initialized")

	return nil
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
