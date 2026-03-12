package main

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
	"icooclaw/pkg/skill"
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
	Ctx             context.Context       // 上下文
	Cancel          context.CancelFunc    // 上下文取消函数
	Cfg             *config.Config        // 配置
	Storage         *storage.Storage      // 存储实例
	MessageBus      *bus.MessageBus       // 消息总线
	ProviderFactory *providers.Factory    // 提供商工厂
	DefaultProvider providers.Provider    // 默认提供商
	ToolRegistry    *tools.Registry       // 工具注册表
	MemoryLoader    *memory.DefaultLoader // 记忆加载器
	SkillLoader     *skill.DefaultLoader  // skill 加载加载器
	AgentLoop       *agent.Loop           // 代理循环
	AgentRegistry   *agent.AgentRegistry  // 代理注册表
	ChannelManager  *channels.Manager     // 渠道管理器
	Gw              *gateway.Server       // 网关服务器
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
	a.SkillLoader = skill.NewLoader(a.Storage, slog.Default())
}

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
func (a *App) InitConfig() {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	// 确保目录存在
	if err := cfg.EnsureWorkspace(); err != nil {
		slog.Error("创建工作目录失败", "error", err)
		os.Exit(1)
	}
	if err := cfg.EnsureDatabasePath(); err != nil {
		slog.Error("创建数据库目录失败", "error", err)
		os.Exit(1)
	}
}

func (a *App) InitLog() {
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(a.Cfg.Logging.Level),
	}

	var handler slog.Handler
	if a.Cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func (a *App) InitAgent() {
	// 初始化代理循环
	agentLoop := agent.NewLoop(
		agent.WithLoopBus(a.MessageBus),
		agent.WithLoopProvider(a.DefaultProvider),
		agent.WithLoopProviderFactory(a.ProviderFactory),
		agent.WithLoopTools(a.ToolRegistry),
		agent.WithLoopMemory(a.MemoryLoader),
		agent.WithLoopSkills(a.SkillLoader),
		agent.WithLoopStorage(a.Storage),
		agent.WithLoopLogger(slog.Default()),
	)

	// 初始化代理注册表
	a.AgentRegistry = agent.NewAgentRegistry(slog.Default())
	a.AgentLoop = agentLoop
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

	// 创建网关服务器
	a.Gw = gateway.NewServer(serverCfg, a.Storage, slog.Default()).
		WithWebSocket(websocket.DefaultManagerConfig()).
		WithSSE().
		WithBus(a.MessageBus).
		WithAgentLoop(a.AgentLoop).
		WithAgentRegistry(a.AgentRegistry).
		Setup()
}

func (a *App) Init() {
	// 初始化上下文
	a.Ctx, a.Cancel = context.WithCancel(context.Background())
	// 初始化配置
	a.InitConfig()
	// 初始化日志
	a.InitLog()
	// 初始化存储
	a.InitStorage()
	// 初始化消息总线
	a.InitBus()
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
	// 初始化代理
	a.InitAgent()
}

func (a *App) Run() {
	// 启动渠道管理器
	go func() {
		if err := a.ChannelManager.StartAll(a.Ctx); err != nil {
			slog.Error("渠道管理器错误", "error", err)
		}
	}()

	// 启动网关服务器
	if err := a.Gw.Start(); err != nil && err != http.ErrServerClosed {
		slog.Error("网关服务错误", "error", err)
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
		if err := a.ChannelManager.StopAll(shutdownCtx); err != nil {
			slog.Error("关闭渠道管理器失败", "error", err)
		}

		// 关闭网关服务器
		if err := a.Gw.Shutdown(shutdownCtx); err != nil {
			slog.Error("网关服务关闭失败", "error", err)
		}

		// 取消上下文
		a.Cancel()
	}()

	// 在后台启动代理循环
	go func() {
		if err := a.AgentLoop.Run(a.Ctx); err != nil && err != context.Canceled {
			slog.Error("代理循环错误", "error", err)
		}
	}()
}

func (a *App) Close() {
	// 取消上下文
	a.Cancel()

	// 关闭存储
	a.Storage.Close()
}
