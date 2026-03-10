package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"icooclaw/pkg/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server represents the gateway HTTP server.
type Server struct {
	router  chi.Router
	server  *http.Server
	storage *storage.Storage
	logger  *slog.Logger
	handlers *Handlers
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultServerConfig returns the default server configuration.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServer creates a new gateway server.
func NewServer(cfg *ServerConfig, store *storage.Storage, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	s := &Server{
		router:   r,
		storage:  store,
		logger:   logger,
		handlers: NewHandlers(logger, store),
	}

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

	// Timeout
	s.router.Use(middleware.Timeout(60 * time.Second))

	// CORS
	s.router.Use(corsMiddleware)
}

// corsMiddleware handles CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
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

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting gateway server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down gateway server")
	return s.server.Shutdown(ctx)
}

// Router returns the chi router.
func (s *Server) Router() chi.Router {
	return s.router
}