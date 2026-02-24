package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// SubAgentConfig 子代理配置
type SubAgentConfig struct {
	Name         string        `json:"name"`
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	SystemPrompt string        `json:"system_prompt"`
	Interval     time.Duration `json:"interval"` // 执行间隔
	Enabled      bool          `json:"enabled"`
}

// SubAgent 后台子Agent
type SubAgent struct {
	name    string
	agent   *Agent
	cfg     SubAgentConfig
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *slog.Logger
	running bool
	mu      sync.RWMutex

	// 执行状态
	lastRun    time.Time
	lastResult string
	execCount  int
}

// NewSubAgent 创建子Agent
func NewSubAgent(name string, agent *Agent, cfg SubAgentConfig, logger *slog.Logger) *SubAgent {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Interval == 0 {
		cfg.Interval = 60 * time.Second
	}

	return &SubAgent{
		name:    name,
		agent:   agent,
		cfg:     cfg,
		logger:  logger,
		lastRun: time.Now(),
	}
}

// Config 获取配置
func (s *SubAgent) Config() SubAgentConfig {
	return s.cfg
}

// Name 获取名称
func (s *SubAgent) Name() string {
	return s.name
}

// Start 启动子Agent
func (s *SubAgent) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()
	s.logger.Info("SubAgent started", "name", s.name, "interval", s.cfg.Interval)
}

// Stop 停止子Agent
func (s *SubAgent) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.running = false
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("SubAgent stopped", "name", s.name)
}

// IsRunning 检查是否运行
func (s *SubAgent) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// LastRun 获取最后运行时间
func (s *SubAgent) LastRun() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRun
}

// LastResult 获取最后结果
func (s *SubAgent) LastResult() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastResult
}

// ExecCount 获取执行次数
func (s *SubAgent) ExecCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.execCount
}

// run 运行子Agent
func (s *SubAgent) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	// 立即执行一次
	s.execute()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.execute()
		}
	}
}

// execute 执行任务
func (s *SubAgent) execute() {
	s.logger.Debug("SubAgent executing", "name", s.name)

	// 创建子代理上下文
	subCtx := context.Background()

	// 构建执行消息
	message := s.buildExecutionPrompt()

	// 执行处理
	response, err := s.agent.ProcessMessage(subCtx, message)

	s.mu.Lock()
	s.lastRun = time.Now()
	s.execCount++
	if err != nil {
		s.lastResult = fmt.Sprintf("error: %v", err)
		s.logger.Error("SubAgent execution failed", "name", s.name, "error", err)
	} else {
		s.lastResult = response
		s.logger.Debug("SubAgent executed", "name", s.name, "response_length", len(response))
	}
	s.mu.Unlock()

	// 保存到记忆（如果启用）
	if err == nil && s.agent.memory != nil {
		key := fmt.Sprintf("subagent_%s_%d", s.name, s.execCount)
		summary := fmt.Sprintf("Ran at %s: %s", s.lastRun.Format(time.RFC3339), truncate(response, 200))
		_ = s.agent.memory.RememberHistory(key, summary)
	}
}

// buildExecutionPrompt 构建执行提示
func (s *SubAgent) buildExecutionPrompt() string {
	// 可以根据需要自定义执行逻辑
	return s.cfg.SystemPrompt
}

// Trigger 触发立即执行
func (s *SubAgent) Trigger(ctx context.Context) error {
	if !s.IsRunning() {
		return fmt.Errorf("subagent not running")
	}

	s.logger.Info("SubAgent triggered", "name", s.name)
	go s.execute()
	return nil
}

// UpdateConfig 更新配置
func (s *SubAgent) UpdateConfig(cfg SubAgentConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果间隔改变，需要重启
	if cfg.Interval != s.cfg.Interval && s.running {
		s.cancel()
		s.running = false
		s.wg.Wait()

		s.cfg = cfg
		s.ctx, s.cancel = context.WithCancel(context.Background())
		s.running = true
		s.wg.Add(1)
		go s.run()
	} else {
		s.cfg = cfg
	}

	s.logger.Info("SubAgent config updated", "name", s.name)
	return nil
}

// SubAgentManager 子Agent管理器
type SubAgentManager struct {
	agents map[string]*SubAgent
	logger *slog.Logger
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSubAgentManager 创建子Agent管理器
func NewSubAgentManager(ctx context.Context, logger *slog.Logger) *SubAgentManager {
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(ctx)
	return &SubAgentManager{
		agents: make(map[string]*SubAgent),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Register 注册子Agent
func (m *SubAgentManager) Register(name string, agent *Agent, cfg SubAgentConfig) *SubAgent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.agents[name]; ok {
		m.logger.Warn("SubAgent already registered, stopping old one", "name", name)
		existing.Stop()
	}

	subAgent := NewSubAgent(name, agent, cfg, m.logger)
	m.agents[name] = subAgent
	m.logger.Info("SubAgent registered", "name", name)
	return subAgent
}

// Unregister 注销子Agent
func (m *SubAgentManager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[name]
	if !ok {
		return fmt.Errorf("subagent not found: %s", name)
	}

	agent.Stop()
	delete(m.agents, name)
	m.logger.Info("SubAgent unregistered", "name", name)
	return nil
}

// Start 启动所有子Agent
func (m *SubAgentManager) Start() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, agent := range m.agents {
		if agent.cfg.Enabled {
			agent.Start(m.ctx)
		}
	}
}

// Stop 停止所有子Agent
func (m *SubAgentManager) Stop() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, agent := range m.agents {
		agent.Stop()
	}
}

// StartOne 启动指定子Agent
func (m *SubAgentManager) StartOne(name string) error {
	m.mu.RLock()
	agent, ok := m.agents[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("subagent not found: %s", name)
	}

	agent.Start(m.ctx)
	return nil
}

// StopOne 停止指定子Agent
func (m *SubAgentManager) StopOne(name string) error {
	m.mu.RLock()
	agent, ok := m.agents[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("subagent not found: %s", name)
	}

	agent.Stop()
	return nil
}

// Get 获取子Agent
func (m *SubAgentManager) Get(name string) (*SubAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agent, ok := m.agents[name]
	return agent, ok
}

// List 列出所有子Agent
func (m *SubAgentManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}
	return names
}

// ListRunning 列出运行中的子Agent
func (m *SubAgentManager) ListRunning() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.agents))
	for name, agent := range m.agents {
		if agent.IsRunning() {
			names = append(names, name)
		}
	}
	return names
}

// GetStatus 获取所有子Agent状态
func (m *SubAgentManager) GetStatus() []SubAgentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make([]SubAgentStatus, 0, len(m.agents))
	for name, agent := range m.agents {
		status = append(status, SubAgentStatus{
			Name:       name,
			Running:    agent.IsRunning(),
			LastRun:    agent.LastRun(),
			LastResult: agent.LastResult(),
			ExecCount:  agent.ExecCount(),
			Interval:   agent.Config().Interval,
			Enabled:    agent.Config().Enabled,
		})
	}
	return status
}

// Trigger 触发子Agent执行
func (m *SubAgentManager) Trigger(name string) error {
	m.mu.RLock()
	agent, ok := m.agents[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("subagent not found: %s", name)
	}

	return agent.Trigger(m.ctx)
}

// TriggerAll 触发所有子Agent执行
func (m *SubAgentManager) TriggerAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, agent := range m.agents {
		if agent.IsRunning() {
			go agent.Trigger(m.ctx)
		}
	}
}

// UpdateConfig 更新子Agent配置
func (m *SubAgentManager) UpdateConfig(name string, cfg SubAgentConfig) error {
	m.mu.RLock()
	agent, ok := m.agents[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("subagent not found: %s", name)
	}

	return agent.UpdateConfig(cfg)
}

// Count 子Agent数量
func (m *SubAgentManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}

// Close 关闭管理器
func (m *SubAgentManager) Close() error {
	m.cancel()
	m.Stop()
	return nil
}

// SubAgentStatus 子Agent状态
type SubAgentStatus struct {
	Name       string        `json:"name"`
	Running    bool          `json:"running"`
	LastRun    time.Time     `json:"last_run"`
	LastResult string        `json:"last_result"`
	ExecCount  int           `json:"exec_count"`
	Interval   time.Duration `json:"interval"`
	Enabled    bool          `json:"enabled"`
}

// SubAgentExecutor 子代理执行器接口
type SubAgentExecutor interface {
	Execute(ctx context.Context) (string, error)
}

// TaskSubAgent 基于任务的子代理
type TaskSubAgent struct {
	*SubAgent
	executor SubAgentExecutor
}

// NewTaskSubAgent 创建任务子代理
func NewTaskSubAgent(name string, agent *Agent, cfg SubAgentConfig, executor SubAgentExecutor, logger *slog.Logger) *TaskSubAgent {
	subAgent := NewSubAgent(name, agent, cfg, logger)
	return &TaskSubAgent{
		SubAgent: subAgent,
		executor: executor,
	}
}

// execute 执行任务
func (s *TaskSubAgent) execute() {
	s.logger.Debug("TaskSubAgent executing", "name", s.name)

	response, err := s.executor.Execute(s.ctx)

	s.mu.Lock()
	s.lastRun = time.Now()
	s.execCount++
	if err != nil {
		s.lastResult = fmt.Sprintf("error: %v", err)
		s.logger.Error("TaskSubAgent execution failed", "name", s.name, "error", err)
	} else {
		s.lastResult = response
		s.logger.Debug("TaskSubAgent executed", "name", s.name)
	}
	s.mu.Unlock()
}

// EventSubAgent 事件驱动的子代理
type EventSubAgent struct {
	*SubAgent
	eventType string
	handler   func(ctx context.Context, eventData interface{}) (string, error)
}

// NewEventSubAgent 创建事件子代理
func NewEventSubAgent(name string, agent *Agent, cfg SubAgentConfig, eventType string, handler func(ctx context.Context, eventData interface{}) (string, error), logger *slog.Logger) *EventSubAgent {
	subAgent := NewSubAgent(name, agent, cfg, logger)
	return &EventSubAgent{
		SubAgent:  subAgent,
		eventType: eventType,
		handler:   handler,
	}
}

// OnEvent 处理事件
func (s *EventSubAgent) OnEvent(ctx context.Context, eventData interface{}) (string, error) {
	if s.handler == nil {
		return "", fmt.Errorf("no handler registered")
	}
	return s.handler(ctx, eventData)
}

// GetEventType 获取事件类型
func (s *EventSubAgent) GetEventType() string {
	return s.eventType
}
