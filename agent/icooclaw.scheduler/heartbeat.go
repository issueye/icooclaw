package scheduler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HeartbeatConfig 心跳服务配置（独立结构）
type HeartbeatOptions struct {
	Enabled          bool
	Interval         time.Duration
	Workspace        string
	NotifyOnWake     bool
	CheckHEARTBEATMD bool
}

// Heartbeat 心跳服务
type Heartbeat struct {
	bus        MessageBus
	logger     Logger
	slogLogger *slog.Logger
	config     *HeartbeatOptions
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	mu         sync.RWMutex
}

// NewHeartbeat 创建心跳服务（使用接口）
func NewHeartbeat(bus MessageBus, config SchedulerConfig, logger Logger) *Heartbeat {
	if logger == nil {
		logger = slog.Default()
	}

	slogLogger := slog.Default()
	if l, ok := logger.(*slog.Logger); ok {
		slogLogger = l
	}

	heartbeatCfg := &HeartbeatOptions{
		Enabled:          config.IsHeartbeatEnabled(),
		Interval:         config.GetHeartbeatInterval(),
		Workspace:        config.GetWorkspace(),
		NotifyOnWake:     true,
		CheckHEARTBEATMD: true,
	}

	if heartbeatCfg.Interval <= 0 {
		heartbeatCfg.Interval = 30 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Heartbeat{
		bus:        bus,
		logger:     logger,
		slogLogger: slogLogger,
		config:     heartbeatCfg,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// NewHeartbeatWithOptions 使用选项创建心跳服务
func NewHeartbeatWithOptions(bus MessageBus, opts *HeartbeatOptions, logger Logger) *Heartbeat {
	if logger == nil {
		logger = slog.Default()
	}

	slogLogger := slog.Default()
	if l, ok := logger.(*slog.Logger); ok {
		slogLogger = l
	}

	if opts == nil {
		opts = &HeartbeatOptions{
			Enabled:          true,
			Interval:         30 * time.Minute,
			NotifyOnWake:     true,
			CheckHEARTBEATMD: true,
		}
	}

	if opts.Interval <= 0 {
		opts.Interval = 30 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Heartbeat{
		bus:        bus,
		logger:     logger,
		slogLogger: slogLogger,
		config:     opts,
		ctx:        ctx,
		cancel:     cancel,
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
		h.slogLogger.Info("Heartbeat is disabled")
		return nil
	}

	h.wg.Add(1)
	go h.run()

	h.running = true
	h.slogLogger.Info("Heartbeat started", "interval", h.config.Interval)
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
	h.slogLogger.Info("Heartbeat stopped")
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
	h.slogLogger.Debug("Heartbeat triggered")

	// 发布心跳事件
	if h.bus != nil {
		h.bus.Publish(HeartbeatEvent{
			EventType: "heartbeat",
			Timestamp: time.Now().Unix(),
			Interval:  h.config.Interval.Seconds(),
		})
	}

	// 检查 HEARTBEAT.md 文件
	if h.config.CheckHEARTBEATMD {
		h.checkHeartbeatFile()
	}

	h.slogLogger.Debug("Heartbeat event processed", "interval", h.config.Interval)
}

// HeartbeatEvent 心跳事件
type HeartbeatEvent struct {
	EventType string
	Channel   string
	Timestamp int64
	Interval  float64
	Tasks     []map[string]interface{}
	Source    string
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
			h.slogLogger.Debug("HEARTBEAT.md not found")
			return
		}
	}

	// 读取文件内容
	content, err := os.ReadFile(heartbeatFile)
	if err != nil {
		h.slogLogger.Warn("Failed to read HEARTBEAT.md", "error", err)
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
		h.slogLogger.Info("Found periodic tasks in HEARTBEAT.md", "count", len(tasks))

		// 发布事件通知
		if h.config.NotifyOnWake && h.bus != nil {
			h.bus.Publish(HeartbeatEvent{
				EventType: "task_start",
				Channel:   "heartbeat",
				Tasks:     tasks,
				Source:    "HEARTBEAT.md",
				Timestamp: time.Now().Unix(),
			})
		}
	}
}

// SetInterval 设置间隔
func (h *Heartbeat) SetInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.config.Interval = interval
	h.slogLogger.Info("Heartbeat interval updated", "interval", interval)
}

// SetConfig 设置配置
func (h *Heartbeat) SetConfig(opts *HeartbeatOptions) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.config = opts
	h.slogLogger.Info("Heartbeat config updated", "enabled", opts.Enabled, "interval", opts.Interval)
}

// GetConfig 获取配置
func (h *Heartbeat) GetConfig() *HeartbeatOptions {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 返回副本
	return &HeartbeatOptions{
		Enabled:          h.config.Enabled,
		Interval:         h.config.Interval,
		Workspace:        h.config.Workspace,
		NotifyOnWake:     h.config.NotifyOnWake,
		CheckHEARTBEATMD: h.config.CheckHEARTBEATMD,
	}
}
