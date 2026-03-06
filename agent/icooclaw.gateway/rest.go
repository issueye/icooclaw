package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/docgen"
	"icooclaw.ai/agent"
	"icooclaw.ai/provider"
	"icooclaw.core/bus"
	"icooclaw.core/config"
	"icooclaw.core/consts"
	"icooclaw.core/storage"
	"icooclaw.core/ws"
)

// RESTGateway REST API 网关实现
type RESTGateway struct {
	workspace    string
	config       *config.Config
	logger       *slog.Logger
	dataStorage  *storage.Storage
	server       *http.Server
	router       *chi.Mux
	running      bool
	mu           sync.RWMutex
	agentManager *agent.AgentManager

	handlers  *Handlers
	wsManager *ws.Manager
}

// NewRESTGateway 创建 REST 网关
func NewRESTGateway() (*RESTGateway, error) {
	// Step 1: 加载配置
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// Step 2: 初始化日志
	logger := config.InitLogger(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	slog.SetDefault(logger)

	// Step 3: 初始化工作空间（检查并创建关键文件）
	wsConfig, err := config.InitWorkspaceWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化工作空间失败: %w", err)
	}

	// Step 4: 初始化数据库
	db, err := storage.InitDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	logger.WithGroup("[Gateway]").Info("数据库初始化成功", "path", cfg.Database.Path)

	dataStorage := storage.NewStorage(db)

	// 初始化 WebSocket 管理器
	wsManager := ws.NewManager(dataStorage, logger)

	g := &RESTGateway{
		mu:          sync.RWMutex{},
		workspace:   wsConfig.Path,
		config:      cfg,
		logger:      logger,
		dataStorage: dataStorage,
		wsManager:   wsManager,
		handlers:    NewHandlers(logger, dataStorage, wsManager, nil), // agentManager 会在 startAgent 中创建
	}

	g.setupRouter()
	return g, nil
}

// setupRouter 设置路由
func (g *RESTGateway) setupRouter() {
	r := chi.NewRouter()

	// 中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(g.corsMiddleware)

	// 注册路由
	RegisterRoutes(r, g.handlers)
	// 打印路由
	docgen.PrintRoutes(r)
	g.router = r
}

// corsMiddleware CORS 中间件
func (g *RESTGateway) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start 启动网关
func (g *RESTGateway) Start(ctx context.Context) error {
	if g.config == nil || !g.config.Gateway.Enabled {
		g.logger.WithGroup("[Gateway]").Info("REST 网关已禁用")
		return nil
	}

	host := g.config.Gateway.Host
	if host == "" {
		host = consts.DEF_GATEWAY_HOST
	}
	port := g.config.Gateway.Port
	if port == 0 {
		port = consts.DEF_GATEWAY_PORT
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	g.server = &http.Server{
		Addr:         addr,
		Handler:      g.router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	g.mu.Lock()
	g.running = true
	g.mu.Unlock()

	go func() {
		g.logger.WithGroup("[Gateway]").Info("REST 网关启动", slog.String("addr", addr))
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			g.logger.WithGroup("[Gateway]").Error("REST 网关启动失败", "error", err)
		}
	}()

	// 创建并启动 AI Agent
	go g.startAgent(ctx)

	return nil
}

// startAgent 创建并启动 Agent Manager
func (g *RESTGateway) startAgent(_ context.Context) {
	// 获取消息总线（使用 WebSocket Manager 的消息总线）
	messageBus := g.wsManager.Bus()

	// 加载 Agent 配置
	agentConfig := g.config.Agents.Defaults
	if agentConfig.Name == "" {
		agentConfig.Name = "gateway-agent"
	}

	// 获取默认 Provider 配置
	providerName := g.config.Agents.DefaultProvider
	if providerName == "" {
		providerName = "openai" // 默认使用 OpenAI
	}

	// 根据 Provider 名称获取配置
	var providerSettings config.ProviderSettings
	switch providerName {
	case "openai":
		providerSettings = g.config.Providers.OpenAI
	case "openrouter":
		providerSettings = g.config.Providers.OpenRouter
	case "deepseek":
		providerSettings = g.config.Providers.DeepSeek
	case "anthropic":
		providerSettings = g.config.Providers.Anthropic
	default:
		providerSettings = g.config.Providers.OpenAI
	}

	// 创建 Provider
	prov, err := createProvider(providerSettings, agentConfig.Model, g.logger)
	if err != nil {
		g.logger.WithGroup("[Gateway]").Error("创建 Provider 失败", "error", err)
		return
	}

	// 创建 Agent Manager 配置
	managerConfig := agent.DefaultAgentManagerConfig()
	// 可以从配置中读取这些值
	// managerConfig.MaxAgents = g.config.Agents.MaxAgents // 如果有这个配置项

	// 创建 Agent Manager
	g.agentManager = agent.NewAgentManager(
		managerConfig,
		g.dataStorage,
		prov,
		&agentConfig,
		g.workspace,
		g.logger,
	)

	g.logger.WithGroup("[Gateway]").Info("创建 Agent Manager 成功",
		"max_agents", managerConfig.MaxAgents,
		"idle_timeout", managerConfig.IdleTimeout.String(),
		"pre_start_count", managerConfig.PreStartCount,
		"provider", providerName,
		"model", agentConfig.Model,
	)

	// 设置消息总线，开始监听消息
	g.agentManager.SetMessageBus(messageBus)
}

// createProvider 创建 LLM Provider
func createProvider(providerSettings config.ProviderSettings, model string, logger *slog.Logger) (provider.Provider, error) {
	// 这里可以根据配置创建不同的 Provider
	// 目前使用 OpenAI Provider
	apiKey := providerSettings.APIKey

	if apiKey == "" {
		return nil, fmt.Errorf("API key 不能为空")
	}

	// 如果未指定模型，使用 Provider 的默认模型
	useModel := model
	if useModel == "" {
		useModel = providerSettings.Model
	}

	prov := provider.NewOpenAIProvider(apiKey, useModel)
	logger.WithGroup("[Gateway]").Info("创建 Provider 成功",
		"model", prov.GetDefaultModel(),
	)
	return prov, nil
}

// Stop 停止网关
func (g *RESTGateway) Stop() error {
	g.logger.WithGroup("[Gateway]").Info("开始关闭网关")

	// 关闭 Agent Manager
	if g.agentManager != nil {
		g.agentManager.Close()
	}

	// 关闭 WebSocket 管理器
	if g.wsManager != nil {
		g.wsManager.Close()
	}

	if g.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := g.server.Shutdown(ctx); err != nil {
		g.logger.WithGroup("[Gateway]").Error("REST 网关关闭失败", "error", err)
		return err
	}

	g.mu.Lock()
	g.running = false
	g.mu.Unlock()

	g.logger.WithGroup("[Gateway]").Info("REST 网关关闭成功")
	return nil
}

// IsRunning 检查是否运行
func (g *RESTGateway) IsRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.running
}

// Router 获取路由器
func (g *RESTGateway) Router() http.Handler {
	return g.router
}

// Mount 挂载到外部路由器
func (g *RESTGateway) Mount(r chi.Router, pattern string) {
	r.Mount(pattern, g.router)
	g.logger.WithGroup("[Gateway]").Info("REST 网关挂载到路由", "pattern", pattern)
}

// WSManager 获取 WebSocket 管理器
func (g *RESTGateway) WSManager() *ws.Manager {
	return g.wsManager
}

// Bus 获取消息总线
func (g *RESTGateway) Bus() *bus.MessageBus {
	return g.wsManager.Bus()
}
