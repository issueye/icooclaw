package app

import (
	"context"
	"fmt"
	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
	"icooclaw/pkg/config"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway"
	"icooclaw/pkg/gateway/websocket"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/scheduler"
	schedulerTool "icooclaw/pkg/scheduler/tool"
	"icooclaw/pkg/skill"
	skillTool "icooclaw/pkg/skill/tool"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/tools/builtin"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type App struct {
	Ctx             context.Context      // 上下文
	Cancel          context.CancelFunc   // 上下文取消函数
	Logger          *slog.Logger         // 日志记录器
	Cfg             *config.Config       // 配置
	Storage         *storage.Storage     // 存储实例
	MessageBus      *bus.MessageBus      // 消息总线
	ProviderFactory *providers.Factory   // 提供商工厂
	DefaultProvider providers.Provider   // 默认提供商
	ToolRegistry    *tools.Registry      // 工具注册表
	MemoryLoader    memory.Loader        // 记忆加载器
	SkillLoader     skill.Loader         // skill 加载加载器
	AgentManager    *agent.AgentManager  // 代理管理器
	AgentRegistry   *agent.AgentRegistry // 代理注册表
	ChannelManager  *channels.Manager    // 渠道管理器
	Gw              *gateway.Server      // 网关服务器
	Scheduler       *scheduler.Scheduler // 任务调度器
}

func NewApp() *App {
	return &App{}
}

// InitBus 初始化消息总线
func (a *App) InitBus() {
	a.MessageBus = bus.NewMessageBus(bus.DefaultConfig())
}

// InitTool 初始化工具，包括内置工具
func (a *App) InitTool() {
	// 初始化工具注册表
	a.ToolRegistry = tools.NewRegistry()

	// 注册内置工具
	builtin.RegisterBuiltinTools(a.ToolRegistry)

	// 注册定时任务
	schedulerTl := schedulerTool.NewTool(a.Storage.Task(), a.Scheduler, a.MessageBus, a.Logger)
	a.ToolRegistry.Register(schedulerTl)

	// 注册技能工具
	skilltl := skillTool.NewInstallTool(a.Cfg.Agent.Workspace, a.Storage.Skill())
	a.ToolRegistry.Register(skilltl)
}

// InitProvider 初始化提供商工厂
func (a *App) InitProvider() {
	factory := providers.NewFactory(a.Storage)

	// 获取默认提供商
	var defaultProvider providers.Provider
	defaultModel, err := a.Storage.Param().Get(consts.DEFAULT_MODEL_KEY)
	if err != nil || defaultModel == nil || defaultModel.Value == "" {
		slog.Warn("未找到默认模型，需要配置", "key", consts.DEFAULT_MODEL_KEY)
	} else {
		arrs := strings.Split(defaultModel.Value, "/")
		if len(arrs) != 2 {
			slog.Warn("默认模型格式错误，需要配置", "model", defaultModel.Value)
			return
		}

		slog.Info("默认模型", "model", defaultModel.Value)
		defaultProvider, err = factory.Get(arrs[0])
		if err != nil {
			slog.Warn("未找到默认提供商，需要配置", "provider", arrs[0])
		}
	}

	// 设置默认提供商
	a.DefaultProvider = defaultProvider
	a.ProviderFactory = factory
}

// InitMemory 初始化记忆加载器
func (a *App) InitMemory() {
	a.MemoryLoader = memory.NewLoader(a.Storage, 100, slog.Default())
}

// InitSkill 初始化 skill 加载器
func (a *App) InitSkill() {
	a.SkillLoader = skill.NewLoader(a.Cfg.Agent.Workspace, a.Storage, slog.Default())
}

// InitStorage 初始化存储
func (a *App) InitStorage() {
	dbPath, _ := a.Cfg.GetDatabasePath()
	store, err := storage.New(a.Cfg.Mode, dbPath)
	if err != nil {
		slog.Error("初始化存储失败", "error", err)
		os.Exit(1)
	}

	// 设置存储实例
	a.Storage = store
}

// InitConfig 初始化配置
func (a *App) InitConfig(cfgFile string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		return err
	}

	// 确保目录存在
	if err := cfg.EnsureWorkspace(); err != nil {
		slog.Error("创建工作目录失败", "error", err)
		return err
	}
	if err := cfg.EnsureDatabasePath(); err != nil {
		slog.Error("创建数据库目录失败", "error", err)
		return err
	}

	// 设置配置实例
	a.Cfg = cfg

	return nil
}

// InitLog 初始化日志记录器
func (a *App) InitLog() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(a.Cfg.Logging.Level),
	}

	var handler slog.Handler
	if a.Cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	slog.SetDefault(logger)
	return logger
}

// InitChannel 初始化渠道
func (a *App) InitChannel() {
	// 初始化渠道管理器
	channelManager := channels.NewManager(a.MessageBus, a.Storage, slog.Default())
	if err := channelManager.InitChannels(a.Ctx); err != nil {
		slog.Warn("初始化渠道失败", "error", err)
	}

	// 设置渠道管理器
	a.ChannelManager = channelManager
}

// InitGateway 初始化网关服务器
func (a *App) InitGateway() {
	// 创建网关服务器配置
	serverCfg := gateway.DefaultServerConfig()
	if a.Cfg.Gateway.Port > 0 {
		serverCfg.Addr = fmt.Sprintf(":%d", a.Cfg.Gateway.Port)
	}

	// 创建 WebSocket 管理器
	wsManager := websocket.NewManager(
		websocket.DefaultManagerConfig(),
		a.Logger,
	)
	wsManager.WithAgentManager(a.AgentManager)

	// 创建网关服务器
	a.Gw = gateway.NewServer(
		serverCfg,
		slog.Default(),
		a.Storage,
		a.Scheduler,
		a.MessageBus,
		wsManager,
		a.AgentManager,
	).WithSSE().Setup()
}

func (a *App) Init(path string) error {
	// 初始化上下文
	a.Ctx, a.Cancel = context.WithCancel(context.Background())
	// 初始化配置
	if err := a.InitConfig(path); err != nil {
		return err
	}
	// 初始化日志
	a.Logger = a.InitLog()
	// 初始化存储
	a.InitStorage()
	// 初始化消息总线
	a.InitBus()
	// 初始化任务调度器
	a.Scheduler = scheduler.NewScheduler(
		a.Storage.Task(),
		a.MessageBus,
		a.Logger,
	)
	// 初始化工具
	a.InitTool()
	// 初始化记忆加载器
	a.InitMemory()
	// 初始化 skill 加载器
	a.InitSkill()
	// 初始化提供商工厂
	a.InitProvider()
	// 初始化渠道
	a.InitChannel()
	// 初始化智能体管理器
	a.AgentManager = agent.NewAgentManager(a.Ctx, a.Logger).
		WithProviderFactory(a.ProviderFactory).
		WithBus(a.MessageBus).
		WithMemory(a.MemoryLoader).
		WithTools(a.ToolRegistry).
		WithSkills(a.SkillLoader).
		WithStorage(a.Storage)

	// 初始化网关服务器
	a.InitGateway()
	return nil
}

// RunGateway 运行网关服务
func (a *App) RunGateway() {
	// 启动渠道管理器
	go func() {
		err := a.ChannelManager.StartAll(a.Ctx)
		if err != nil {
			slog.Error("渠道管理器错误", "error", err)
		}
	}()

	// 启动任务调度器
	a.Scheduler.Start()

	// 启动网关服务器
	err := a.Gw.Start()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("网关服务错误", "error", err)
		os.Exit(1)
	}

	// 启动智能体管理器
	err = a.AgentManager.Start()
	if err != nil {
		slog.Error("智能体管理器启动失败", "error", err)
		os.Exit(1)
	}

	// 处理关闭信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("正在关闭网关服务...")

		// 关闭渠道管理器
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		err := a.ChannelManager.StopAll(shutdownCtx)
		if err != nil {
			slog.Error("关闭渠道管理器失败", "error", err)
		}

		// 关闭网关服务器
		err = a.Gw.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("网关服务关闭失败", "error", err)
		}

		// 取消上下文
		a.Cancel()
	}()
}

func (a *App) Close() {
	// 取消上下文
	if a.Cancel != nil {
		a.Cancel()
	}

	// 关闭存储
	if a.Storage != nil {
		a.Storage.Close()
	}

	// 关闭智能体管理器
	if a.AgentManager != nil {
		a.AgentManager.Stop()
	}
	a.AgentManager = nil
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
