package subagent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubAgentConfig_Defaults(t *testing.T) {
	cfg := SubAgentConfig{
		Name:     "test",
		Provider: "openai",
		Model:    "gpt-4",
	}

	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "gpt-4", cfg.Model)
	assert.Equal(t, time.Duration(0), cfg.Interval)
	assert.False(t, cfg.Enabled)
}

func TestNewSubAgent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name:     "test-agent",
		Interval: 30 * time.Second,
	}

	subAgent := NewSubAgent("test-agent", nil, cfg, logger)

	assert.NotNil(t, subAgent)
	assert.Equal(t, "test-agent", subAgent.Name())
	assert.Equal(t, cfg, subAgent.Config())
	assert.False(t, subAgent.IsRunning())
	assert.Equal(t, 0, subAgent.ExecCount())
}

func TestNewSubAgent_DefaultLogger(t *testing.T) {
	cfg := SubAgentConfig{
		Name:     "test",
		Interval: time.Second,
	}

	subAgent := NewSubAgent("test", nil, cfg, nil)
	assert.NotNil(t, subAgent)
}

func TestNewSubAgent_DefaultInterval(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name: "test",
		// Interval not set
	}

	subAgent := NewSubAgent("test", nil, cfg, logger)
	assert.Equal(t, 60*time.Second, subAgent.Config().Interval)
}

func TestSubAgent_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name:         "test",
		Interval:     time.Hour, // 使用较长间隔避免测试期间执行
		Enabled:      true,
		SystemPrompt: "test prompt",
	}

	subAgent := NewSubAgent("test", nil, cfg, logger)

	// Start 会立即执行一次 execute()，但由于 agent 为 nil 会 panic
	// 所以我们只测试状态切换，不实际启动
	ctx := context.Background()

	// 直接测试状态管理
	subAgent.mu.Lock()
	subAgent.running = true
	subAgent.ctx, subAgent.cancel = context.WithCancel(ctx)
	subAgent.mu.Unlock()

	assert.True(t, subAgent.IsRunning())

	// 测试停止
	subAgent.Stop()
	assert.False(t, subAgent.IsRunning())
}

func TestSubAgent_Trigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name:         "test",
		Interval:     time.Hour,
		Enabled:      true,
		SystemPrompt: "test prompt",
	}

	subAgent := NewSubAgent("test", nil, cfg, logger)

	// 未启动时触发应该失败
	err := subAgent.Trigger(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// 设置运行状态（不实际启动 goroutine）
	subAgent.mu.Lock()
	subAgent.running = true
	subAgent.mu.Unlock()

	// Trigger 会调用 execute，由于 agent 为 nil 会 panic
	// 这里只测试 Trigger 的前置检查
	assert.True(t, subAgent.IsRunning())
}

func TestSubAgent_LastRun(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name:     "test",
		Interval: time.Minute,
	}

	subAgent := NewSubAgent("test", nil, cfg, logger)
	lastRun := subAgent.LastRun()
	assert.False(t, lastRun.IsZero())
}

func TestSubAgentManager_New(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	manager := NewSubAgentManager(ctx, logger)
	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.Count())
}

func TestSubAgentManager_New_DefaultLogger(t *testing.T) {
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, nil)
	assert.NotNil(t, manager)
}

func TestSubAgentManager_Register(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{Name: "test", Interval: time.Minute}
	subAgent := manager.Register("test", nil, cfg)

	assert.NotNil(t, subAgent)
	assert.Equal(t, 1, manager.Count())

	// 再次注册应该替换旧的
	cfg2 := SubAgentConfig{Name: "test2", Interval: time.Second}
	subAgent2 := manager.Register("test", nil, cfg2)

	assert.NotNil(t, subAgent2)
	assert.Equal(t, 1, manager.Count())
	assert.Equal(t, time.Second, subAgent2.Config().Interval)
}

func TestSubAgentManager_Unregister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{Name: "test", Interval: time.Minute}
	manager.Register("test", nil, cfg)
	assert.Equal(t, 1, manager.Count())

	// 注销
	err := manager.Unregister("test")
	assert.NoError(t, err)
	assert.Equal(t, 0, manager.Count())

	// 注销不存在的
	err = manager.Unregister("nonexistent")
	assert.Error(t, err)
}

func TestSubAgentManager_Get(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{Name: "test", Interval: time.Minute}
	manager.Register("test", nil, cfg)

	subAgent, ok := manager.Get("test")
	assert.True(t, ok)
	assert.NotNil(t, subAgent)

	_, ok = manager.Get("nonexistent")
	assert.False(t, ok)
}

func TestSubAgentManager_List(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{Name: "test", Interval: time.Minute}
	manager.Register("agent1", nil, cfg)
	manager.Register("agent2", nil, cfg)
	manager.Register("agent3", nil, cfg)

	names := manager.List()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "agent1")
	assert.Contains(t, names, "agent2")
	assert.Contains(t, names, "agent3")
}

func TestSubAgentManager_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{
		Name:     "test",
		Interval: time.Hour, // 长间隔避免执行
		Enabled:  true,
	}
	manager.Register("agent1", nil, cfg)
	manager.Register("agent2", nil, cfg)

	// 由于 agent 为 nil，Start 会导致 panic
	// 这里只测试 manager 的状态管理
	assert.Equal(t, 2, manager.Count())

	// 测试 ListRunning
	running := manager.ListRunning()
	assert.Len(t, running, 0) // 未启动
}

func TestSubAgentManager_StartStopOne(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{
		Name:     "test",
		Interval: time.Hour,
		Enabled:  true,
	}
	manager.Register("agent1", nil, cfg)

	// 操作不存在的
	err := manager.StartOne("nonexistent")
	assert.Error(t, err)

	err = manager.StopOne("nonexistent")
	assert.Error(t, err)
}

func TestSubAgentManager_Trigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{
		Name:         "test",
		Interval:     time.Hour,
		Enabled:      true,
		SystemPrompt: "test",
	}
	manager.Register("agent1", nil, cfg)

	// 触发不存在的
	err := manager.Trigger("nonexistent")
	assert.Error(t, err)
}

func TestSubAgentManager_GetStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{
		Name:     "test",
		Interval: time.Minute,
		Enabled:  true,
	}
	manager.Register("agent1", nil, cfg)
	manager.Register("agent2", nil, cfg)

	status := manager.GetStatus()
	assert.Len(t, status, 2)

	for _, s := range status {
		assert.Contains(t, []string{"agent1", "agent2"}, s.Name)
		assert.Equal(t, time.Minute, s.Interval)
		assert.True(t, s.Enabled)
	}
}

func TestSubAgentManager_Close(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()
	manager := NewSubAgentManager(ctx, logger)

	cfg := SubAgentConfig{
		Name:     "test",
		Interval: time.Hour, // 长间隔避免执行
		Enabled:  true,
	}
	manager.Register("agent1", nil, cfg)

	// 不调用 Start，只测试 Close
	err := manager.Close()
	assert.NoError(t, err)
}

func TestSubAgentStatus_Structure(t *testing.T) {
	status := SubAgentStatus{
		Name:       "test",
		Running:    true,
		LastRun:    time.Now(),
		LastResult: "result",
		ExecCount:  5,
		Interval:   time.Minute,
		Enabled:    true,
	}

	assert.Equal(t, "test", status.Name)
	assert.True(t, status.Running)
	assert.Equal(t, "result", status.LastResult)
	assert.Equal(t, 5, status.ExecCount)
	assert.Equal(t, time.Minute, status.Interval)
	assert.True(t, status.Enabled)
}

func TestSubAgentExecutor_Interface(t *testing.T) {
	// 确保 SubAgentExecutor 接口定义正确
	var _ SubAgentExecutor = &mockExecutor{}
}

type mockExecutor struct {
	result string
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context) (string, error) {
	return m.result, m.err
}

func TestTaskSubAgent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name:     "task-agent",
		Interval: time.Minute,
		Enabled:  true,
	}

	executor := &mockExecutor{result: "task result"}
	taskAgent := NewTaskSubAgent("task-agent", nil, cfg, executor, logger)

	require.NotNil(t, taskAgent)
	assert.Equal(t, "task-agent", taskAgent.Name())
}

func TestEventSubAgent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{
		Name:     "event-agent",
		Interval: time.Minute,
		Enabled:  true,
	}

	handler := func(ctx context.Context, eventData interface{}) (string, error) {
		return "event handled", nil
	}

	eventAgent := NewEventSubAgent("event-agent", nil, cfg, "test-event", handler, logger)

	require.NotNil(t, eventAgent)
	assert.Equal(t, "test-event", eventAgent.eventType)

	// 测试事件处理
	result, err := eventAgent.OnEvent(context.Background(), map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Equal(t, "event handled", result)
}

func TestEventSubAgent_NoHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := SubAgentConfig{Name: "test", Interval: time.Minute}

	eventAgent := NewEventSubAgent("test", nil, cfg, "test-event", nil, logger)

	_, err := eventAgent.OnEvent(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler")
}