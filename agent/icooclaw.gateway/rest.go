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
	"icooclaw.core/config"
	"icooclaw.core/consts"
	"icooclaw.core/storage"
)

// RESTGateway REST API 网关实现
type RESTGateway struct {
	workspace   string
	config      *config.Config
	logger      Logger
	dataStorage *storage.Storage
	server      *http.Server
	router      *chi.Mux
	running     bool
	mu          sync.RWMutex

	handlers *Handlers
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
	logger.Info("数据库初始化成功", "path", cfg.Database.Path)

	dataStorage := storage.NewStorage(db)
	g := &RESTGateway{
		mu:          sync.RWMutex{},
		workspace:   wsConfig.Path,
		config:      cfg,
		logger:      logger,
		dataStorage: dataStorage,
		handlers:    NewHandlers(logger, dataStorage),
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
		g.logger.Error("REST 网关关闭失败", "error", err)
		return err
	}

	g.mu.Lock()
	g.running = false
	g.mu.Unlock()

	g.logger.Info("REST 网关关闭成功")
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
