package scheduler

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// TestNewOnceTaskRunner 创建一次性任务运行器测试
func TestNewOnceTaskRunner(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-task",
		Type:    TaskTypeOnce,
		Enabled: true,
		Message: "Test message",
		Channel: "test-channel",
		ChatID:  "123456",
	}

	logger := slog.Default()

	runner, err := NewOnceTaskRunner(task.Name, task, logger)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if runner == nil {
		t.Fatal("Expected runner to be not nil")
	}

	if runner.GetName() != task.Name {
		t.Errorf("Expected name to be %s, got %s", task.Name, runner.GetName())
	}

	if runner.GetType() != TaskTypeOnce {
		t.Errorf("Expected type to be TaskTypeOnce, got %v", runner.GetType())
	}

	if runner.GetStatus() != TaskPending {
		t.Errorf("Expected status to be TaskPending, got %v", runner.GetStatus())
	}
}

// TestOnceTaskRunner_Run 测试一次性任务执行
func TestOnceTaskRunner_Run(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-run",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	ctx := context.Background()
	err := runner.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if runner.GetStatus() != TaskCompleted {
		t.Errorf("Expected status to be TaskCompleted, got %v", runner.GetStatus())
	}
}

// TestOnceTaskRunner_RunWhileRunning 测试任务运行中时再次运行
func TestOnceTaskRunner_RunWhileRunning(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-running",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	ctx := context.Background()

	runner.status = TaskRunning

	err := runner.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error when running while running, got %v", err)
	}

	if runner.GetStatus() != TaskRunning {
		t.Errorf("Expected status to remain TaskRunning, got %v", runner.GetStatus())
	}
}

// TestOnceTaskRunner_Stop 测试任务停止
func TestOnceTaskRunner_Stop(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-stop",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	ctx := context.Background()
	err := runner.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if runner.GetStatus() != TaskTerminated {
		t.Errorf("Expected status to be TaskTerminated, got %v", runner.GetStatus())
	}
}

// TestOnceTaskRunner_GetStatus 测试获取任务状态
func TestOnceTaskRunner_GetStatus(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-status",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	status := runner.GetStatus()
	if status != TaskPending {
		t.Errorf("Expected status to be TaskPending, got %v", status)
	}

	runner.status = TaskRunning
	status = runner.GetStatus()
	if status != TaskRunning {
		t.Errorf("Expected status to be TaskRunning, got %v", status)
	}
}

// TestOnceTaskRunner_GetType 测试获取任务类型
func TestOnceTaskRunner_GetType(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-type",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	taskType := runner.GetType()
	if taskType != TaskTypeOnce {
		t.Errorf("Expected type to be TaskTypeOnce, got %v", taskType)
	}
}

// TestOnceTaskRunner_GetName 测试获取任务名称
func TestOnceTaskRunner_GetName(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-name",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	name := runner.GetName()
	if name != task.Name {
		t.Errorf("Expected name to be %s, got %s", task.Name, name)
	}
}

// TestOnceTaskRunner_GetInfo 测试获取任务信息
func TestOnceTaskRunner_GetInfo(t *testing.T) {
	task := &TaskInfo{
		ID:          1,
		Name:        "test-once-info",
		Type:        TaskTypeOnce,
		Enabled:     true,
		Description: "Test once task description",
		Message:     "Test message",
		Channel:     "test-channel",
		ChatID:      "123456",
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	info := runner.GetInfo()
	if info == nil {
		t.Fatal("Expected info to be not nil")
	}

	if info.ID != task.ID {
		t.Errorf("Expected ID to be %d, got %d", task.ID, info.ID)
	}

	if info.Name != task.Name {
		t.Errorf("Expected Name to be %s, got %s", task.Name, info.Name)
	}

	if info.Description != task.Description {
		t.Errorf("Expected Description to be %s, got %s", task.Description, info.Description)
	}
}

// TestOnceTaskRunner_ShouldRun 测试任务是否应该运行
func TestOnceTaskRunner_ShouldRun(t *testing.T) {
	task := &TaskInfo{
		ID:        1,
		Name:      "test-once-should-run",
		Type:      TaskTypeOnce,
		Enabled:   true,
		LastRunAt: time.Time{},
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	now := time.Now()
	result := runner.ShouldRun(now)
	if !result {
		t.Error("Expected should run to be true for never run task")
	}

	runner.status = TaskCompleted
	result = runner.ShouldRun(now)
	if result {
		t.Error("Expected should run to be false for completed task")
	}
}

// TestOnceTaskRunner_ShouldRunAfterRun 测试任务运行后是否应该再次运行
func TestOnceTaskRunner_ShouldRunAfterRun(t *testing.T) {
	task := &TaskInfo{
		ID:        1,
		Name:      "test-once-after-run",
		Type:      TaskTypeOnce,
		Enabled:   true,
		LastRunAt: time.Now(),
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	now := time.Now()
	result := runner.ShouldRun(now)
	if result {
		t.Error("Expected should run to be false for already run task")
	}
}

// TestOnceTaskRunner_SetCallback 测试设置回调
func TestOnceTaskRunner_SetCallback(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-callback",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	callback := &MockTaskCallback{}
	runner.SetCallback(callback)

	if runner.callback != callback {
		t.Error("Expected callback to be set")
	}
}

// TestOnceTaskRunner_GetMetrics 测试获取执行指标
func TestOnceTaskRunner_GetMetrics(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-metrics",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	metrics := runner.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be not nil")
	}

	if metrics.TotalRuns != 0 {
		t.Errorf("Expected TotalRuns to be 0, got %d", metrics.TotalRuns)
	}
}

// TestOnceTaskRunner_SetRetryConfig 测试设置重试配置
func TestOnceTaskRunner_SetRetryConfig(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-retry",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	config := RetryConfig{
		MaxRetries:  3,
		Interval:    time.Second * 5,
		Backoff:     2.0,
		MaxInterval: time.Minute,
	}

	runner.SetRetryConfig(config)
}

// TestOnceTaskRunner_CallbackExecution 测试回调执行
func TestOnceTaskRunner_CallbackExecution(t *testing.T) {
	task := &TaskInfo{
		ID:      1,
		Name:    "test-once-callback-exec",
		Type:    TaskTypeOnce,
		Enabled: true,
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	callback := &MockTaskCallback{}
	runner.SetCallback(callback)

	ctx := context.Background()
	runner.Run(ctx)

	if callback.onStartCalls != 1 {
		t.Errorf("Expected onStartCalls to be 1, got %d", callback.onStartCalls)
	}

	if callback.onCompleteCalls != 1 {
		t.Errorf("Expected onCompleteCalls to be 1, got %d", callback.onCompleteCalls)
	}

	if callback.onNextRunCalculatedCalls != 0 {
		t.Errorf("Expected onNextRunCalculatedCalls to be 0 for once task, got %d", callback.onNextRunCalculatedCalls)
	}
}

// TestOnceTaskRunner_LastRunAtUpdated 测试最后运行时间更新
func TestOnceTaskRunner_LastRunAtUpdated(t *testing.T) {
	task := &TaskInfo{
		ID:        1,
		Name:      "test-once-last-run",
		Type:      TaskTypeOnce,
		Enabled:   true,
		LastRunAt: time.Time{},
	}

	logger := slog.Default()
	runner, _ := NewOnceTaskRunner(task.Name, task, logger)

	beforeRun := time.Now()
	ctx := context.Background()
	runner.Run(ctx)
	afterRun := time.Now()

	info := runner.GetInfo()
	if info.LastRunAt.Before(beforeRun) || info.LastRunAt.After(afterRun) {
		t.Error("Expected LastRunAt to be updated after run")
	}
}
