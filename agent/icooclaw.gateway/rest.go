package gateway

import (
	"context"
	"encoding/json"
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
		return nil, fmt.Errorf("加载配置失败：%w", err)
	}

	// Step 2: 初始化日志
	logger := config.InitLogger(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	slog.SetDefault(logger)

	// Step 3: 初始化工作空间（检查并创建关键文件）
	wsConfig, err := config.InitWorkspaceWithConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化工作空间失败：%w", err)
	}

	// Step 4: 初始化数据库
	db, err := storage.InitDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败：%w", err)
	}
	logger.Info("数据库初始化成功", "path", cfg.Database.Path)

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
		g.logger.Info("REST 网关已禁用")
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
		g.logger.Info("REST 网关启动", slog.String("addr", addr))
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			g.logger.Error("REST 网关启动失败", "error", err)
		}
	}()

	// 创建并启动 Agent Manager
	go g.startAgent(ctx)

	return nil
}

// startAgent 创建并启动 Agent Manager
func (g *RESTGateway) startAgent(_ context.Context) {
	// 获取消息总线（使用 WebSocket Manager 的消息总线）
	messageBus := g.wsManager.Bus()

	// 从 storage 读取启用的 Provider 配置
	enabledProviders, err := g.dataStorage.ProviderConfig().GetEnabled()
	if err != nil {
		g.logger.WithGroup("[Gateway]").Error("获取启用的 Provider 配置失败", "error", err)
		return
	}

	if len(enabledProviders) == 0 {
		g.logger.WithGroup("[Gateway]").Warn("没有启用的 Provider 配置，请先到设置页面配置 Provider")
		// 继续执行，允许运行时配置
		// 不创建 AgentManager，只监听消息总线
		g.agentManager = agent.NewAgentManager(
			agent.DefaultAgentManagerConfig(),
			g.dataStorage,
			nil, // provider 为空，等待运行时配置
			&config.AgentSettings{Name: "gateway-agent"},
			g.workspace,
			g.logger,
		)
		g.agentManager.SetMessageBus(messageBus)
		return
	}

	// 获取默认 Provider（第一个启用的）
	defaultProvider := enabledProviders[0]
	g.logger.WithGroup("[Gateway]").Info("使用默认 Provider",
		"name", defaultProvider.Name,
		"llms", len(defaultProvider.LLMs),
		"default_model", defaultProvider.DefaultModel,
	)

	// 确定最终使用的模型（优先级：param_configs > Provider 默认模型 > 配置文件）
	finalModel := g.determineModel(defaultProvider)
	g.logger.WithGroup("[Gateway]").Info("确定使用模型",
		"model", finalModel,
		"source", g.getModelSource(finalModel, defaultProvider),
	)

	// 加载 Agent 配置
	agentConfig := g.config.Agents.Defaults
	if agentConfig.Name == "" {
		agentConfig.Name = "gateway-agent"
	}
	// 使用确定的模型
	agentConfig.Model = finalModel

	// 创建 Provider
	prov, err := createProviderFromStorage(&defaultProvider, agentConfig.Model, g.logger)
	if err != nil {
		g.logger.WithGroup("[Gateway]").Error("创建 Provider 失败", "error", err)
		return
	}

	// 创建 Agent Manager 配置
	managerConfig := agent.DefaultAgentManagerConfig()

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
		"provider", defaultProvider.Name,
		"model", agentConfig.Model,
	)

	// 设置消息总线，开始监听消息
	g.agentManager.SetMessageBus(messageBus)
}

// determineModel 确定最终使用的模型
// 优先级：
// 1. param_configs 中配置的默认模型 (agent.default_model)
// 2. Provider 的默认模型 (default_model 字段)
// 3. Provider 支持的第一个模型 (llms[0].model)
// 4. 配置文件中的模型 (agents.defaults.model)
func (g *RESTGateway) determineModel(provider storage.ProviderConfig) string {
	// 1. 优先从 param_configs 读取
	paramModel := g.dataStorage.ParamConfig().GetStringValue("agent.default_model", "")
	if paramModel != "" {
		g.logger.WithGroup("[Gateway]").Debug("使用 param_configs 中的默认模型",
			"model", paramModel,
		)
		return paramModel
	}

	// 2. 使用 Provider 的默认模型
	if provider.DefaultModel != "" {
		g.logger.WithGroup("[Gateway]").Debug("使用 Provider 的默认模型",
			"model", provider.DefaultModel,
		)
		return provider.DefaultModel
	}

	// 3. 使用 Provider 支持的第一个模型
	if len(provider.LLMs) > 0 {
		firstModel := provider.LLMs[0].Model
		g.logger.WithGroup("[Gateway]").Debug("使用 Provider 支持的第一个模型",
			"model", firstModel,
		)
		return firstModel
	}

	// 4. 使用配置文件中的模型
	configModel := g.config.Agents.Defaults.Model
	if configModel != "" {
		g.logger.WithGroup("[Gateway]").Debug("使用配置文件中的模型",
			"model", configModel,
		)
		return configModel
	}

	// 都没有则返回空，由 Provider 使用其内部默认值
	g.logger.WithGroup("[Gateway]").Debug("未指定模型，将使用 Provider 内部默认值")
	return ""
}

// getModelSource 获取模型来源描述（用于日志）
func (g *RESTGateway) getModelSource(model string, provider storage.ProviderConfig) string {
	if model == "" {
		return "provider_default"
	}

	// 检查是否来自 param_configs
	paramModel := g.dataStorage.ParamConfig().GetStringValue("agent.default_model", "")
	if paramModel == model {
		return "param_configs"
	}

	// 检查是否来自 Provider 默认模型
	if provider.DefaultModel == model {
		return "provider_default_model"
	}

	// 检查是否来自 Provider 支持的模型
	for _, llm := range provider.LLMs {
		if llm.Model == model {
			return "provider_llms"
		}
	}

	// 来自配置文件
	return "config_file"
}

// createProviderFromStorage 从 storage 创建 LLM Provider
func createProviderFromStorage(providerConfig *storage.ProviderConfig, model string, logger *slog.Logger) (provider.Provider, error) {
	// 解析 Config 字段中的 JSON 配置
	var configMap map[string]interface{}
	if providerConfig.Config != "" {
		if err := json.Unmarshal([]byte(providerConfig.Config), &configMap); err != nil {
			return nil, fmt.Errorf("解析 Provider 配置失败：%w", err)
		}
	}

	// 获取 API Key
	apiKey := providerConfig.ApiKey
	if apiKey == "" {
		return nil, fmt.Errorf("API key 不能为空")
	}

	// 确定使用的模型
	useModel := model
	if useModel == "" && providerConfig.DefaultModel != "" {
		useModel = providerConfig.DefaultModel
	}
	if useModel == "" && len(providerConfig.LLMs) > 0 {
		useModel = providerConfig.LLMs[0].Model
	}

	// 根据 Provider 名称创建对应的 Provider
	switch providerConfig.Name {
	case "openai", "openrouter", "deepseek", "anthropic":
		// 对于 OpenAI 兼容的 Provider，使用 OpenAIProvider
		prov := provider.NewOpenAIProvider(apiKey, useModel)
		logger.WithGroup("[Gateway]").Info("创建 Provider 成功",
			"name", providerConfig.Name,
			"model", prov.GetDefaultModel(),
		)
		return prov, nil
	default:
		// 默认使用 OpenAIProvider
		prov := provider.NewOpenAIProvider(apiKey, useModel)
		logger.WithGroup("[Gateway]").Info("创建 Provider 成功",
			"name", providerConfig.Name,
			"model", prov.GetDefaultModel(),
		)
		return prov, nil
	}
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
	g.logger.Info("REST 网关挂载到路由", "pattern", pattern)
}

// WSManager 获取 WebSocket 管理器
func (g *RESTGateway) WSManager() *ws.Manager {
	return g.wsManager
}

// Bus 获取消息总线
func (g *RESTGateway) Bus() *bus.MessageBus {
	return g.wsManager.Bus()
}

// AgentManager 获取 Agent Manager
func (g *RESTGateway) AgentManager() *agent.AgentManager {
	return g.agentManager
}
