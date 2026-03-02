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
	"time"

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

	// Step 3: 初始化工作空间（检查并创建关键文件）
	wsConfig, err := config.InitWorkspaceWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("init workspace: %w", err)
	}
	workspace := wsConfig.Path

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
	channelManager = channel.NewManager(
		&messageBusAdapter{bus: messageBus},
		&channelConfigAdapter{cfg: cfg.Channels},
		db,
		logger,
	)
	logger.Info("Channels registered", "count", channelManager.Count())

	// Step 8: 创建默认 Agent
	if err = initDefaultAgent(workspace); err != nil {
		return fmt.Errorf("init agent: %w", err)
	}

	// Step 9: 初始化工具
	toolRegistry = tools.InitTools(cfg, logger, channelManager)
	agentInstance.SetTools(toolRegistry)

	// Step 10: 初始化调度器
	schedulerInst = scheduler.NewScheduler(
		&taskStorageAdapter{storage: storage.NewStorage(db)},
		&schedulerConfigAdapter{cfg: cfg.Scheduler},
		logger,
	)
	logger.Info("All components initialized")

	return nil
}

// ============ 适配器定义 ============

// messageBusAdapter 消息总线适配器
type messageBusAdapter struct {
	bus *bus.MessageBus
}

func (a *messageBusAdapter) PublishInbound(ctx context.Context, msg channel.InboundMessage) error {
	return a.bus.PublishInbound(ctx, bus.InboundMessage{
		ID:        msg.ID,
		Channel:   msg.Channel,
		ChatID:    msg.ChatID,
		UserID:    msg.UserID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		Metadata:  msg.Metadata,
	})
}

func (a *messageBusAdapter) Publish(event interface{}) error {
	if msg, ok := event.(bus.OutboundMessage); ok {
		return a.bus.PublishOutbound(context.Background(), msg)
	}
	return nil
}

func (a *messageBusAdapter) Subscribe(handler interface{}) error {
	return nil
}

// channelConfigAdapter 通道配置适配器
type channelConfigAdapter struct {
	cfg config.ChannelsConfig
}

func (a *channelConfigAdapter) WebSocketConfig() channel.WebSocketConfig {
	if a.cfg.WebSocket.Enabled {
		return &webSocketConfigAdapter{cfg: a.cfg.WebSocket}
	}
	return nil
}

func (a *channelConfigAdapter) WebhookConfig() channel.WebhookConfig {
	if a.cfg.Webhook.Enabled {
		return &webhookConfigAdapter{cfg: a.cfg.Webhook}
	}
	return nil
}

// webSocketConfigAdapter WebSocket 配置适配器
type webSocketConfigAdapter struct {
	cfg config.ChannelSettings
}

func (a *webSocketConfigAdapter) Enabled() bool { return a.cfg.Enabled }
func (a *webSocketConfigAdapter) Host() string  { return a.cfg.Host }
func (a *webSocketConfigAdapter) Port() int     { return a.cfg.Port }

// webhookConfigAdapter Webhook 配置适配器
type webhookConfigAdapter struct {
	cfg config.ChannelSettings
}

func (a *webhookConfigAdapter) Enabled() bool              { return a.cfg.Enabled }
func (a *webhookConfigAdapter) Host() string               { return a.cfg.Host }
func (a *webhookConfigAdapter) Port() int                  { return a.cfg.Port }
func (a *webhookConfigAdapter) Path() string               { return "" }
func (a *webhookConfigAdapter) Secret() string             { return a.cfg.Token }
func (a *webhookConfigAdapter) Extra() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range a.cfg.Extra {
		result[k] = v
	}
	return result
}

// taskStorageAdapter 任务存储适配器
type taskStorageAdapter struct {
	storage *storage.Storage
}

func (a *taskStorageAdapter) GetEnabledTasks() ([]scheduler.TaskInfo, error) {
	tasks, err := a.storage.GetEnabledTasks()
	if err != nil {
		return nil, err
	}

	result := make([]scheduler.TaskInfo, len(tasks))
	for i, t := range tasks {
		result[i] = scheduler.TaskInfo{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Type:        scheduler.TaskTypeCron,
			CronExpr:    t.CronExpr,
			Interval:    t.Interval,
			Message:     t.Message,
			Channel:     t.Channel,
			ChatID:      t.ChatID,
			Enabled:     t.Enabled,
			NextRunAt:   t.NextRunAt,
			LastRunAt:   t.LastRunAt,
		}
	}
	return result, nil
}

func (a *taskStorageAdapter) GetTaskByName(name string) (*scheduler.TaskInfo, error) {
	task, err := a.storage.GetTaskByName(name)
	if err != nil {
		return nil, err
	}

	return &scheduler.TaskInfo{
		ID:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		Type:        scheduler.TaskTypeCron,
		CronExpr:    task.CronExpr,
		Interval:    task.Interval,
		Message:     task.Message,
		Channel:     task.Channel,
		ChatID:      task.ChatID,
		Enabled:     task.Enabled,
		NextRunAt:   task.NextRunAt,
		LastRunAt:   task.LastRunAt,
	}, nil
}

func (a *taskStorageAdapter) UpdateTask(task *scheduler.TaskInfo) error {
	return a.storage.UpdateTask(&storage.Task{
		ID:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		CronExpr:    task.CronExpr,
		Interval:    task.Interval,
		Message:     task.Message,
		Channel:     task.Channel,
		ChatID:      task.ChatID,
		Enabled:     task.Enabled,
		NextRunAt:   task.NextRunAt,
		LastRunAt:   task.LastRunAt,
	})
}

// schedulerConfigAdapter 调度器配置适配器
type schedulerConfigAdapter struct {
	cfg config.SchedulerConfig
}

func (a *schedulerConfigAdapter) GetTaskTimeout() time.Duration {
	return 30 * time.Minute
}

func (a *schedulerConfigAdapter) GetQueueSize() int {
	return 100
}

func (a *schedulerConfigAdapter) GetCheckInterval() time.Duration {
	return time.Duration(a.cfg.HeartbeatInterval) * time.Minute
}

func (a *schedulerConfigAdapter) IsEnabled() bool {
	return a.cfg.Enabled
}

func (a *schedulerConfigAdapter) GetHeartbeatInterval() time.Duration {
	return time.Duration(a.cfg.HeartbeatInterval) * time.Minute
}

func (a *schedulerConfigAdapter) IsHeartbeatEnabled() bool {
	return a.cfg.Enabled
}

func (a *schedulerConfigAdapter) GetWorkspace() string {
	return ""
}

// ============ 初始化函数 ============

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
