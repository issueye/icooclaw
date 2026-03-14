package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/gateway/sse"
	"icooclaw/pkg/gateway/websocket"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server represents the gateway HTTP server.
type Server struct {
	router   chi.Router
	server   *http.Server
	storage  *storage.Storage
	logger   *slog.Logger
	handlers *Handlers
	schedule *scheduler.Scheduler

	// New components
	wsManager    *websocket.Manager
	sseBroker    *sse.Broker
	bus          *bus.MessageBus
	agentManager *agent.AgentManager
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxConcurrentWS int
}

// DefaultServerConfig returns the default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Addr:            ":8080",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxConcurrentWS: 100,
	}
}

// NewServer creates a new gateway server.
func NewServer(
	cfg *ServerConfig,
	wsCfg *websocket.ManagerConfig,
	logger *slog.Logger,
	store *storage.Storage,
	schedule *scheduler.Scheduler,
	bus *bus.MessageBus,
	agentManager *agent.AgentManager,
) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	s := &Server{
		router:       r,
		storage:      store,
		logger:       logger,
		schedule:     schedule,
		bus:          bus,
		agentManager: agentManager,
	}

	// Create WebSocket manager
	s.WithWebSocket(wsCfg)

	// Create handlers
	s.handlers = NewHandlers(
		logger,
		store,
		schedule,
		s.agentManager,
		s.bus,
		s.wsManager,
	)

	// Setup middleware
	s.setupMiddleware()

	// Register routes
	RegisterRoutes(r, s.handlers)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// WithWebSocket enables WebSocket support.
func (s *Server) WithWebSocket(cfg *websocket.ManagerConfig) *Server {
	if cfg == nil {
		cfg = websocket.DefaultManagerConfig()
	}
	s.wsManager = websocket.NewManager(cfg, s.logger)
	return s
}

// WithSSE enables Server-Sent Events support.
func (s *Server) WithSSE() *Server {
	s.sseBroker = sse.NewBroker(s.logger)
	return s
}

// WithBus sets the message bus.
func (s *Server) WithBus(b *bus.MessageBus) *Server {
	s.bus = b
	if s.wsManager != nil {
		s.wsManager.WithBus(b)
	}
	return s
}

func (s *Server) WithAgentManager(m *agent.AgentManager) *Server {
	s.agentManager = m
	if s.wsManager != nil {
		s.wsManager.WithAgentManager(m)
	}
	return s
}

// Setup initializes all components.
func (s *Server) Setup() *Server {
	// Update chat handler with components
	if s.handlers.Chat != nil {
		s.handlers.Chat = s.handlers.Chat.
			WithWebSocketManager(s.wsManager).
			WithBus(s.bus)
	}

	// Re-register routes with updated handlers
	s.router = chi.NewRouter()
	s.setupMiddleware()
	RegisterRoutes(s.router, s.handlers)

	// Add WebSocket routes
	if s.wsManager != nil {
		s.router.Get("/ws", s.handlers.Chat.HandleWebSocket)
		s.router.Get("/ws/{session_id}", s.handlers.Chat.HandleWebSocketWithSessionID)
	}

	// Add SSE routes
	if s.sseBroker != nil {
		s.router.Get("/events", s.sseBroker.Handler())
	}

	s.server.Handler = s.router

	return s
}

// setupMiddleware sets up the middleware chain.
func (s *Server) setupMiddleware() {
	// Request ID
	s.router.Use(middleware.RequestID)

	// Real IP
	s.router.Use(middleware.RealIP)

	// Logger
	s.router.Use(middleware.Logger)

	// Recoverer
	s.router.Use(middleware.Recoverer)

	// Timeout - increased for long LLM responses
	s.router.Use(middleware.Timeout(300 * time.Second))

	// CORS
	s.router.Use(corsMiddleware)
}

// corsMiddleware handles CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.With("name", "【网关服务】").Info("已启动", "addr", s.server.Addr)

	// Start WebSocket manager if configured
	if s.wsManager != nil {
		go func() {
			ctx := context.Background()
			if err := s.wsManager.Run(ctx); err != nil {
				s.logger.With("name", "【网关服务】").Error("WebSocket管理器错误：", "error", err)
			}
		}()
	}

	// Start SSE broker if configured
	if s.sseBroker != nil {
		go func() {
			ctx := context.Background()
			s.sseBroker.Run(ctx)
		}()
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.With("name", "【网关服务】").Error("【网关服务】已启动失败", "error", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.With("name", "【网关服务】").Info("正在关闭")

	// Stop WebSocket manager
	if s.wsManager != nil {
		s.wsManager.Stop()
	}

	return s.server.Shutdown(ctx)
}

// Router returns the chi router.
func (s *Server) Router() chi.Router {
	return s.router
}

// WebSocketManager returns the WebSocket manager.
func (s *Server) WebSocketManager() *websocket.Manager {
	return s.wsManager
}

// SSEBroker returns the SSE broker.
func (s *Server) SSEBroker() *sse.Broker {
	return s.sseBroker
}

// Bus returns the message bus.
func (s *Server) Bus() *bus.MessageBus {
	return s.bus
}
