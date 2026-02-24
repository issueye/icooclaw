package channel

import (
	"context"
	"log/slog"
	"sync"

	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
)

// Manager 通道管理器
type Manager struct {
	channels map[string]Channel
	bus      *bus.MessageBus
	config   config.ChannelsConfig
	db       interface{} // *gorm.DB
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewManager 创建通道管理器
func NewManager(bus *bus.MessageBus, cfg config.ChannelsConfig, db interface{}, logger *slog.Logger) *Manager {
	m := &Manager{
		channels: make(map[string]Channel),
		bus:      bus,
		config:   cfg,
		db:       db,
		logger:   logger,
	}

	// 根据配置自动注册通道
	m.registerFromConfig()

	return m
}

// registerFromConfig 根据配置自动注册通道
func (m *Manager) registerFromConfig() {
	// WebSocket 通道
	if m.config.WebSocket.Enabled {
		wsChannel := NewWebSocketChannel(m.config.WebSocket, m.bus, m.logger)
		m.Register("websocket", wsChannel)
	}

	// Webhook 通道
	if m.config.Webhook.Enabled {
		webhookChannel := NewWebhookChannel(m.config.Webhook, m.bus, m.logger)
		m.Register("webhook", webhookChannel)
	}
}

// Register 注册通道
func (m *Manager) Register(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
	m.logger.Info("Channel registered", "name", name)
}

// Get 获取通道
func (m *Manager) Get(name string) (Channel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channel, ok := m.channels[name]
	if !ok {
		return nil, ErrChannelNotFound
	}
	return channel, nil
}

// StartAll 启动所有启用的通道
func (m *Manager) StartAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx := context.Background()
	for name, channel := range m.channels {
		if err := channel.Start(ctx); err != nil {
			m.logger.Error("Failed to start channel", "name", name, "error", err)
			continue
		}
		m.logger.Info("Channel started", "name", name)
	}
	return nil
}

// StopAll 停止所有通道
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []error
	for name, channel := range m.channels {
		if err := channel.Stop(); err != nil {
			m.logger.Error("Failed to stop channel", "name", name, "error", err)
			errs = append(errs, err)
			continue
		}
		m.logger.Info("Channel stopped", "name", name)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// List 列出所有通道
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

// Count 获取通道数量
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// StartChannel 启动指定通道
func (m *Manager) StartChannel(ctx context.Context, name string) error {
	m.mu.RLock()
	channel, ok := m.channels[name]
	m.mu.RUnlock()

	if !ok {
		return ErrChannelNotFound
	}

	return channel.Start(ctx)
}

// StopChannel 停止指定通道
func (m *Manager) StopChannel(name string) error {
	m.mu.RLock()
	channel, ok := m.channels[name]
	m.mu.RUnlock()

	if !ok {
		return ErrChannelNotFound
	}

	return channel.Stop()
}

// 错误定义
var (
	ErrChannelNotFound   = &ChannelError{"channel not found"}
	ErrChannelNotRunning = &ChannelError{"channel not running"}
)

type ChannelError struct {
	msg string
}

func (e *ChannelError) Error() string {
	return e.msg
}
