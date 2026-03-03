package scheduler

import (
	"log/slog"
	"testing"
	"time"
)

// MockTaskStorage 模拟任务存储
type MockTaskStorage struct {
	tasks      []TaskInfo
	getErr     error
	updateErr  error
	updateCall int
}

func (m *MockTaskStorage) GetEnabledTasks() ([]TaskInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.tasks, nil
}

func (m *MockTaskStorage) GetTaskByName(name string) (*TaskInfo, error) {
	for _, task := range m.tasks {
		if task.Name == name {
			return &task, nil
		}
	}
	return nil, ErrTaskNotFound
}

func (m *MockTaskStorage) UpdateTask(task *TaskInfo) error {
	m.updateCall++
	return m.updateErr
}

// MockLogger 模拟日志
type MockLogger struct {
	debugCalls int
	infoCalls  int
	warnCalls  int
	errorCalls int
	messages   []string
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.debugCalls++
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.infoCalls++
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.warnCalls++
	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.errorCalls++
	m.messages = append(m.messages, "ERROR: "+msg)
}

// NewMockTaskStorage 创建模拟任务存储
func NewMockTaskStorage(tasks []TaskInfo) *MockTaskStorage {
	return &MockTaskStorage{
		tasks: tasks,
	}
}

// TestDefaultSchedulerConfig 测试默认调度器配置
func TestDefaultSchedulerConfig(t *testing.T) {
	config := NewDefaultSchedulerConfig()

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{
			name:     "TaskTimeout",
			getter:   func() interface{} { return config.GetTaskTimeout() },
			expected: 30 * time.Minute,
		},
		{
			name:     "QueueSize",
			getter:   func() interface{} { return config.GetQueueSize() },
			expected: 100,
		},
		{
			name:     "CheckInterval",
			getter:   func() interface{} { return config.GetCheckInterval() },
			expected: 60 * time.Second,
		},
		{
			name:     "IsEnabled",
			getter:   func() interface{} { return config.IsEnabled() },
			expected: true,
		},
		{
			name:     "HeartbeatInterval",
			getter:   func() interface{} { return config.GetHeartbeatInterval() },
			expected: 30 * time.Minute,
		},
		{
			name:     "IsHeartbeatEnabled",
			getter:   func() interface{} { return config.IsHeartbeatEnabled() },
			expected: true,
		},
		{
			name:     "GetWorkspace",
			getter:   func() interface{} { return config.GetWorkspace() },
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestDefaultSchedulerConfigWithValues 测试带值的配置
func TestDefaultSchedulerConfigWithValues(t *testing.T) {
	config := &DefaultSchedulerConfig{
		TaskTimeout:       10 * time.Minute,
		QueueSize:         50,
		CheckInterval:     30 * time.Second,
		Enabled:           true,
		HeartbeatInterval: 15 * time.Minute,
		HeartbeatEnabled:  true,
		Workspace:         "/test/workspace",
	}

	if config.GetTaskTimeout() != 10*time.Minute {
		t.Errorf("Expected TaskTimeout to be 10m, got %v", config.GetTaskTimeout())
	}

	if config.GetQueueSize() != 50 {
		t.Errorf("Expected QueueSize to be 50, got %d", config.GetQueueSize())
	}

	if config.GetCheckInterval() != 30*time.Second {
		t.Errorf("Expected CheckInterval to be 30s, got %v", config.GetCheckInterval())
	}

	if !config.IsEnabled() {
		t.Error("Expected IsEnabled to be true")
	}

	if config.GetHeartbeatInterval() != 15*time.Minute {
		t.Errorf("Expected HeartbeatInterval to be 15m, got %v", config.GetHeartbeatInterval())
	}

	if !config.IsHeartbeatEnabled() {
		t.Error("Expected IsHeartbeatEnabled to be true")
	}

	if config.GetWorkspace() != "/test/workspace" {
		t.Errorf("Expected Workspace to be /test/workspace, got %s", config.GetWorkspace())
	}
}

// TestDefaultSchedulerConfigZeroValues 测试零值配置
func TestDefaultSchedulerConfigZeroValues(t *testing.T) {
	config := &DefaultSchedulerConfig{}

	if config.GetTaskTimeout() != 30*time.Minute {
		t.Errorf("Expected default TaskTimeout, got %v", config.GetTaskTimeout())
	}

	if config.GetQueueSize() != 100 {
		t.Errorf("Expected default QueueSize, got %d", config.GetQueueSize())
	}

	if config.GetCheckInterval() != 60*time.Second {
		t.Errorf("Expected default CheckInterval, got %v", config.GetCheckInterval())
	}

	if config.GetHeartbeatInterval() != 30*time.Minute {
		t.Errorf("Expected default HeartbeatInterval, got %v", config.GetHeartbeatInterval())
	}
}

// TestSchedulerConfigSchedulerOptions 测试调度器选项
func TestSchedulerConfigSchedulerOptions(t *testing.T) {
	opts := SchedulerOptions{
		TaskTimeout:   5 * time.Minute,
		QueueSize:     200,
		CheckInterval: 10 * time.Second,
	}

	if opts.TaskTimeout != 5*time.Minute {
		t.Errorf("Expected TaskTimeout to be 5m, got %v", opts.TaskTimeout)
	}

	if opts.QueueSize != 200 {
		t.Errorf("Expected QueueSize to be 200, got %d", opts.QueueSize)
	}

	if opts.CheckInterval != 10*time.Second {
		t.Errorf("Expected CheckInterval to be 10s, got %v", opts.CheckInterval)
	}
}

// TestNewScheduler 创建调度器测试
func TestNewScheduler(t *testing.T) {
	tasks := []TaskInfo{
		{
			ID:       1,
			Name:     "test-cron",
			Type:     TaskTypeCron,
			CronExpr: "*/5 * * * *",
			Enabled:  true,
		},
	}

	storage := NewMockTaskStorage(tasks)
	config := NewDefaultSchedulerConfig()
	logger := slog.Default()

	scheduler := NewSchedulerWithSlog(storage, config, logger)

	if scheduler == nil {
		t.Fatal("Expected scheduler to be not nil")
	}

	if scheduler.storage != storage {
		t.Error("Expected storage to match")
	}

	if scheduler.config != config {
		t.Error("Expected config to match")
	}

	if scheduler.slogLogger != logger {
		t.Error("Expected logger to match")
	}

	if scheduler.TaskRunners == nil {
		t.Error("Expected TaskRunners to be initialized")
	}

	if scheduler.taskCh == nil {
		t.Error("Expected taskCh to be initialized")
	}
}

// TestNewSchedulerWithNilLogger 测试使用 nil logger 创建调度器
func TestNewSchedulerWithNilLogger(t *testing.T) {
	tasks := []TaskInfo{}
	storage := NewMockTaskStorage(tasks)
	config := NewDefaultSchedulerConfig()

	scheduler := NewScheduler(storage, config, nil)

	if scheduler == nil {
		t.Fatal("Expected scheduler to be not nil")
	}

	if scheduler.logger != nil {
		t.Error("Expected logger to be nil")
	}
}

// TestSchedulerStartStop 调度器启动停止测试
func TestSchedulerStartStop(t *testing.T) {
	tasks := []TaskInfo{}
	storage := NewMockTaskStorage(tasks)
	config := NewDefaultSchedulerConfig()
	logger := slog.Default()

	scheduler := NewSchedulerWithSlog(storage, config, logger)

	// 测试启动
	err := scheduler.Start()
	if err != nil {
		t.Errorf("Expected no error on start, got %v", err)
	}

	if !scheduler.IsRunning() {
		t.Error("Expected scheduler to be running")
	}

	// 测试重复启动
	err = scheduler.Start()
	if err != nil {
		t.Errorf("Expected no error on second start, got %v", err)
	}

	// 测试停止
	err = scheduler.Stop()
	if err != nil {
		t.Errorf("Expected no error on stop, got %v", err)
	}

	if scheduler.IsRunning() {
		t.Error("Expected scheduler to be stopped")
	}

	// 测试重复停止
	err = scheduler.Stop()
	if err != nil {
		t.Errorf("Expected no error on second stop, got %v", err)
	}
}

// TestSchedulerLoadTasks 调度器加载任务测试
func TestSchedulerLoadTasks(t *testing.T) {
	tasks := []TaskInfo{
		{
			ID:      1,
			Name:    "test-once",
			Type:    TaskTypeOnce,
			Enabled: true,
		},
	}

	storage := NewMockTaskStorage(tasks)
	config := NewDefaultSchedulerConfig()
	logger := slog.Default()

	scheduler := NewSchedulerWithSlog(storage, config, logger)

	err := scheduler.Start()
	if err != nil {
		t.Errorf("Expected no error on start, got %v", err)
	}

	runner, ok := scheduler.GetTaskRunner("test-once")
	if !ok {
		t.Error("Expected task runner to be loaded")
	}

	if runner.GetName() != "test-once" {
		t.Errorf("Expected task name to be test-once, got %s", runner.GetName())
	}

	scheduler.Stop()
}

// TestSchedulerLoadInvalidTasks 测试加载无效任务
func TestSchedulerLoadInvalidTasks(t *testing.T) {
	tasks := []TaskInfo{
		{
			ID:       1,
			Name:     "test-invalid",
			Type:     TaskTypeCron,
			CronExpr: "invalid-cron",
			Enabled:  true,
		},
	}

	storage := NewMockTaskStorage(tasks)
	config := NewDefaultSchedulerConfig()
	logger := slog.Default()

	scheduler := NewSchedulerWithSlog(storage, config, logger)

	err := scheduler.Start()
	if err != nil {
		t.Errorf("Expected no error on start, got %v", err)
	}

	// 无效的 cron 表达式应该不会被加载
	_, ok := scheduler.GetTaskRunner("test-invalid")
	if ok {
		t.Error("Expected invalid task runner not to be loaded")
	}

	scheduler.Stop()
}

// TestSchedulerGetTaskRunner 测试获取任务运行器
func TestSchedulerGetTaskRunner(t *testing.T) {
	tasks := []TaskInfo{}
	storage := NewMockTaskStorage(tasks)
	config := NewDefaultSchedulerConfig()
	logger := slog.Default()

	scheduler := NewSchedulerWithSlog(storage, config, logger)

	// 测试获取不存在的任务
	_, ok := scheduler.GetTaskRunner("non-existent")
	if ok {
		t.Error("Expected non-existent task runner not to be found")
	}

	scheduler.Start()

	// 测试获取不存在的任务（调度器运行中）
	_, ok = scheduler.GetTaskRunner("non-existent")
	if ok {
		t.Error("Expected non-existent task runner not to be found")
	}

	scheduler.Stop()
}

// TestSchedulerStorageError 测试存储错误
func TestSchedulerStorageError(t *testing.T) {
	tasks := []TaskInfo{}
	storage := &MockTaskStorage{
		tasks:  tasks,
		getErr: ErrTaskNotFound,
	}
	config := NewDefaultSchedulerConfig()
	logger := slog.Default()

	scheduler := NewSchedulerWithSlog(storage, config, logger)

	err := scheduler.Start()
	if err != nil {
		t.Errorf("Expected no error on start, got %v", err)
	}

	scheduler.Stop()
}
