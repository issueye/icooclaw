package scheduler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
)

// HeartbeatConfig 心跳服务配置
type HeartbeatConfig struct {
	Enabled          bool
	Interval         time.Duration
	Workspace        string
	NotifyOnWake     bool
	CheckHEARTBEATMD bool
}

// Heartbeat 心跳服务
type Heartbeat struct {
	bus     *bus.MessageBus
	storage interface{}
	logger  *slog.Logger
	config  *HeartbeatConfig
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
}

// NewHeartbeat 创建心跳服务
func NewHeartbeat(bus *bus.MessageBus, storage interface{}, cfg *config.Config, logger *slog.Logger) *Heartbeat {
	if logger == nil {
		logger = slog.Default()
	}

	heartbeatCfg := &HeartbeatConfig{
		Enabled:          true,
		Interval:         30 * time.Minute,
		Workspace:        cfg.Workspace,
		NotifyOnWake:     true,
		CheckHEARTBEATMD: true,
	}

	// 从配置中读取
	if cfg.Scheduler.HeartbeatInterval > 0 {
		heartbeatCfg.Interval = time.Duration(cfg.Scheduler.HeartbeatInterval) * time.Minute
	}
	heartbeatCfg.Enabled = cfg.Scheduler.Enabled

	if heartbeatCfg.Interval <= 0 {
		heartbeatCfg.Interval = 30 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Heartbeat{
		bus:     bus,
		storage: storage,
		logger:  logger,
		config:  heartbeatCfg,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start 启动心跳服务
func (h *Heartbeat) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil
	}

	if !h.config.Enabled {
		h.logger.Info("Heartbeat is disabled")
		return nil
	}

	h.wg.Add(1)
	go h.run()

	h.running = true
	h.logger.Info("Heartbeat started", "interval", h.config.Interval)
	return nil
}

// Stop 停止心跳服务
func (h *Heartbeat) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return nil
	}

	h.cancel()
	h.wg.Wait()

	h.running = false
	h.logger.Info("Heartbeat stopped")
	return nil
}

// IsRunning 检查是否运行
func (h *Heartbeat) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// run 运行心跳
func (h *Heartbeat) run() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	// 首次执行
	h.beat()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.beat()
		}
	}
}

// beat 发送心跳
func (h *Heartbeat) beat() {
	h.logger.Debug("Heartbeat triggered")

	// 发布心跳事件
	_ = bus.NewEvent(
		bus.EventHeartbeat,
		"", // channel
		"", // chatID
		0,  // sessionID
		map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"interval":  h.config.Interval.Seconds(),
		},
	)

	// 检查 HEARTBEAT.md 文件
	if h.config.CheckHEARTBEATMD {
		h.checkHeartbeatFile()
	}

	h.logger.Debug("Heartbeat event processed", "interval", h.config.Interval)
}

// checkHeartbeatFile 检查 HEARTBEAT.md 文件
func (h *Heartbeat) checkHeartbeatFile() {
	if h.config.Workspace == "" {
		return
	}

	// 查找 HEARTBEAT.md 文件
	heartbeatFile := filepath.Join(h.config.Workspace, "HEARTBEAT.md")

	// 也检查当前目录
	if _, err := os.Stat(heartbeatFile); os.IsNotExist(err) {
		heartbeatFile = "HEARTBEAT.md"
		if _, err := os.Stat(heartbeatFile); os.IsNotExist(err) {
			h.logger.Debug("HEARTBEAT.md not found")
			return
		}
	}

	// 读取文件内容
	content, err := os.ReadFile(heartbeatFile)
	if err != nil {
		h.logger.Warn("Failed to read HEARTBEAT.md", "error", err)
		return
	}

	// 解析文件内容
	lines := strings.Split(string(content), "\n")
	var tasks []map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析任务
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			task := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			tasks = append(tasks, map[string]interface{}{
				"description": task,
				"due":         "periodic",
			})
		}
	}

	if len(tasks) > 0 {
		h.logger.Info("Found periodic tasks in HEARTBEAT.md", "count", len(tasks))

		// 发布事件通知
		if h.config.NotifyOnWake {
			_ = bus.NewEvent(
				bus.EventTaskStart,
				"heartbeat",
				"",
				0,
				map[string]interface{}{
					"tasks":     tasks,
					"source":    "HEARTBEAT.md",
					"timestamp": time.Now(),
				},
			)
		}
	}
}

// SetInterval 设置间隔
func (h *Heartbeat) SetInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.config.Interval = interval
	h.logger.Info("Heartbeat interval updated", "interval", interval)
}

// SetConfig 设置配置
func (h *Heartbeat) SetConfig(cfg *HeartbeatConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.config = cfg
	h.logger.Info("Heartbeat config updated", "enabled", cfg.Enabled, "interval", cfg.Interval)
}

// GetConfig 获取配置
func (h *Heartbeat) GetConfig() *HeartbeatConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return &HeartbeatConfig{
		Enabled:          h.config.Enabled,
		Interval:         h.config.Interval,
		Workspace:        h.config.Workspace,
		NotifyOnWake:     h.config.NotifyOnWake,
		CheckHEARTBEATMD: h.config.CheckHEARTBEATMD,
	}
}

// HeartbeatManager 心跳服务管理器
type HeartbeatManager struct {
	heartbeats map[string]*Heartbeat
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewHeartbeatManager 创建心跳服务管理器
func NewHeartbeatManager(logger *slog.Logger) *HeartbeatManager {
	return &HeartbeatManager{
		heartbeats: make(map[string]*Heartbeat),
		logger:     logger,
	}
}

// Register 注册心跳服务
func (m *HeartbeatManager) Register(name string, heartbeat *Heartbeat) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.heartbeats[name] = heartbeat
}

// StartAll 启动所有心跳服务
func (m *HeartbeatManager) StartAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, heartbeat := range m.heartbeats {
		if err := heartbeat.Start(); err != nil {
			m.logger.Error("Failed to start heartbeat", "name", name, "error", err)
			continue
		}
	}
	return nil
}

// StopAll 停止所有心跳服务
func (m *HeartbeatManager) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, heartbeat := range m.heartbeats {
		if err := heartbeat.Stop(); err != nil {
			m.logger.Error("Failed to stop heartbeat", "name", name, "error", err)
			continue
		}
	}
	return nil
}

// Get 获取心跳服务
func (m *HeartbeatManager) Get(name string) *Heartbeat {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.heartbeats[name]
}

// List 列出所有心跳服务
func (m *HeartbeatManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.heartbeats))
	for name := range m.heartbeats {
		names = append(names, name)
	}
	return names
}
