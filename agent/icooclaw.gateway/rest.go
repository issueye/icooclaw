package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RESTGateway REST API 网关实现
type RESTGateway struct {
	config  Config
	storage StorageReader
	agent   AgentReader
	skills  SkillReader
	logger  Logger
	server  *http.Server
	router  *chi.Mux
	running bool
	mu      sync.RWMutex
}

// NewRESTGateway 创建 REST 网关
func NewRESTGateway(cfg Config, storage StorageReader, agent AgentReader, skills SkillReader, logger Logger) *RESTGateway {
	g := &RESTGateway{
		config:  cfg,
		storage: storage,
		agent:   agent,
		skills:  skills,
		logger:  logger,
	}

	g.setupRouter()
	return g
}

// SetAgent 设置 Agent 引用
func (g *RESTGateway) SetAgent(agent AgentReader) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.agent = agent
}

// SetSkills 设置技能读取器
func (g *RESTGateway) SetSkills(skills SkillReader) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.skills = skills
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

	// 健康检查
	r.Get("/api/v1/health", g.handleHealth)

	// 会话相关
	r.Get("/api/v1/sessions", g.handleGetSessions)
	r.Get("/api/v1/sessions/{id}/messages", g.handleGetSessionMessages)
	r.Delete("/api/v1/sessions/{id}", g.handleDeleteSession)

	// Provider 信息
	r.Get("/api/v1/providers", g.handleGetProviders)

	// 技能相关
	r.Get("/api/v1/skills", g.handleGetSkills)
	r.Get("/api/v1/skills/{id}", g.handleGetSkill)

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
	if g.config == nil || !g.config.Enabled() {
		g.logger.Info("REST Gateway is disabled")
		return nil
	}

	host := g.config.Host()
	if host == "" {
		host = "0.0.0.0"
	}
	port := g.config.Port()
	if port == 0 {
		port = 8080
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
		g.logger.Info("REST Gateway starting", "host", host, "port", port)
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			g.logger.Error("REST Gateway error", "error", err)
		}
	}()

	g.logger.Info("REST Gateway started", "addr", addr)
	return nil
}

// Stop 停止网关
func (g *RESTGateway) Stop() error {
	if g.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := g.server.Shutdown(ctx); err != nil {
		g.logger.Error("REST Gateway shutdown error", "error", err)
		return err
	}

	g.mu.Lock()
	g.running = false
	g.mu.Unlock()

	g.logger.Info("REST Gateway stopped")
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
}