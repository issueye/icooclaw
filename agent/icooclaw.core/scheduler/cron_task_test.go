package scheduler

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// TestCronParser_IsValid 测试 Cron 表达式验证
func TestCronParser_IsValid(t *testing.T) {
	parser := NewCronParser()

	tests := []struct {
		expr    string
		isValid bool
	}{
		{"* * * * *", true},
		{"*/5 * * * *", true},
		{"0 * * * *", true},
		{"0 0 * * *", true},
		{"0 0 1 * *", true},
		{"*/10 * * * * *", true},
		{"invalid", false},
		{"", false},
		{"* * *", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := parser.IsValid(tt.expr)
			if result != tt.isValid {
				t.Errorf("Expected IsValid(%s) to be %v, got %v", tt.expr, tt.isValid, result)
			}
		})
	}
}

// TestCronParser_NextRun 测试计算下次执行时间
func TestCronParser_NextRun(t *testing.T) {
	parser := NewCronParser()

	now := time.Now()

	next, ok := parser.NextRun("* * * * *", now)
	if !ok {
		t.Error("Expected valid cron expression to return next run time")
	}

	if next.Before(now) {
		t.Error("Expected next run time to be after now")
	}

	next, ok = parser.NextRun("0 0 1 * *", now)
	if !ok {
		t.Error("Expected valid cron expression to return next run time")
	}

	_, ok = parser.NextRun("invalid", now)
	if ok {
		t.Error("Expected invalid cron expression to return false")
	}
}

// TestCronParser_ShouldRun 测试判断是否应该执行
func TestCronParser_ShouldRun(t *testing.T) {
	parser := NewCronParser()

	now := time.Now()

	result := parser.ShouldRun("* * * * *", now)
	if !result {
		t.Error("Expected should run to be true for every minute cron")
	}

	result = parser.ShouldRun("0 0 1 * *", now)
	if result {
		t.Error("Expected should run to be false for once a month cron")
	}

	result = parser.ShouldRun("invalid", now)
	if result {
		t.Error("Expected should run to be false for invalid cron")
	}
}

// TestNewCronTaskRunner 创建 Cron 任务运行器测试
func TestNewCronTaskRunner(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-cron-task",
		Type:     TaskTypeCron,
		CronExpr: "*/5 * * * *",
		Enabled:  true,
	}

	logger := slog.Default()

	runner, err := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if runner == nil {
		t.Fatal("Expected runner to be not nil")
	}

	if runner.GetName() != task.Name {
		t.Errorf("Expected name to be %s, got %s", task.Name, runner.GetName())
	}

	if runner.GetType() != TaskTypeCron {
		t.Errorf("Expected type to be TaskTypeCron, got %v", runner.GetType())
	}

	if runner.GetStatus() != TaskPending {
		t.Errorf("Expected status to be TaskPending, got %v", runner.GetStatus())
	}
}

// TestNewCronTaskRunnerInvalidCron 测试无效 Cron 表达式
func TestNewCronTaskRunnerInvalidCron(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-invalid",
		Type:     TaskTypeCron,
		CronExpr: "invalid-cron",
		Enabled:  true,
	}

	logger := slog.Default()

	runner, err := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)
	if err == nil {
		t.Error("Expected error for invalid cron expression")
	}

	if runner != nil {
		t.Error("Expected runner to be nil for invalid cron")
	}
}

// TestCronTaskRunner_ShouldRun 测试任务是否应该运行
func TestCronTaskRunner_ShouldRun(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-should-run",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	now := time.Now()
	result := runner.ShouldRun(now)
	if !result {
		t.Error("Expected should run to be true for every minute cron")
	}

	task.CronExpr = "0 0 1 * *"
	runner2, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)
	result = runner2.ShouldRun(now)
	if result {
		t.Error("Expected should run to be false for once a month cron")
	}
}

// TestCronTaskRunner_Run 测试任务执行
func TestCronTaskRunner_Run(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-run",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	ctx := context.Background()
	err := runner.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if runner.GetStatus() != TaskCompleted {
		t.Errorf("Expected status to be TaskCompleted, got %v", runner.GetStatus())
	}
}

// TestCronTaskRunner_RunWhileRunning 测试任务运行中时再次运行
func TestCronTaskRunner_RunWhileRunning(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-running",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	ctx := context.Background()

	runner.status = TaskRunning

	err := runner.Run(ctx)
	if err != nil {
		t.Errorf("Expected no error when running while running, got %v", err)
	}
}

// TestCronTaskRunner_Stop 测试任务停止
func TestCronTaskRunner_Stop(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-stop",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	ctx := context.Background()
	err := runner.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if runner.GetStatus() != TaskTerminated {
		t.Errorf("Expected status to be TaskTerminated, got %v", runner.GetStatus())
	}
}

// TestCronTaskRunner_GetInfo 测试获取任务信息
func TestCronTaskRunner_GetInfo(t *testing.T) {
	task := &TaskInfo{
		ID:          1,
		Name:        "test-info",
		Type:        TaskTypeCron,
		CronExpr:    "* * * * *",
		Enabled:     true,
		Description: "Test task description",
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

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
}

// TestCronTaskRunner_SetCallback 测试设置回调
func TestCronTaskRunner_SetCallback(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-callback",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	callback := &MockTaskCallback{}
	runner.SetCallback(callback)

	if runner.callback != callback {
		t.Error("Expected callback to be set")
	}
}

// TestCronTaskRunner_GetMetrics 测试获取执行指标
func TestCronTaskRunner_GetMetrics(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-metrics",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	metrics := runner.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be not nil")
	}

	if metrics.TotalRuns != 0 {
		t.Errorf("Expected TotalRuns to be 0, got %d", metrics.TotalRuns)
	}
}

// TestCronTaskRunner_SetRetryConfig 测试设置重试配置
func TestCronTaskRunner_SetRetryConfig(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-retry",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

	config := RetryConfig{
		MaxRetries:  5,
		Interval:    time.Second * 10,
		Backoff:     1.5,
		MaxInterval: time.Minute * 5,
	}

	runner.SetRetryConfig(config)
	metrics := runner.GetMetrics()
	_ = metrics
}

// TestCronTaskRunner_CallbackExecution 测试回调执行
func TestCronTaskRunner_CallbackExecution(t *testing.T) {
	task := &TaskInfo{
		ID:       1,
		Name:     "test-callback-exec",
		Type:     TaskTypeCron,
		CronExpr: "* * * * *",
		Enabled:  true,
	}

	logger := slog.Default()
	runner, _ := NewCronTaskRunner(task.Name, task, task.CronExpr, logger)

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

	if callback.onNextRunCalculatedCalls != 1 {
		t.Errorf("Expected onNextRunCalculatedCalls to be 1, got %d", callback.onNextRunCalculatedCalls)
	}
}
