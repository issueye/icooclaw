package scheduler

import (
	"context"
	"testing"
	"time"
)

// TestRetryConfig 测试重试配置
func TestRetryConfig(t *testing.T) {
	config := RetryConfig{
		MaxRetries:  3,
		Interval:    time.Second * 5,
		Backoff:     2.0,
		MaxInterval: time.Minute,
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.Interval != time.Second*5 {
		t.Errorf("Expected Interval to be 5s, got %v", config.Interval)
	}

	if config.Backoff != 2.0 {
		t.Errorf("Expected Backoff to be 2.0, got %f", config.Backoff)
	}

	if config.MaxInterval != time.Minute {
		t.Errorf("Expected MaxInterval to be 1m, got %v", config.MaxInterval)
	}
}

// TestTaskMetrics 测试任务指标
func TestTaskMetrics(t *testing.T) {
	metrics := TaskMetrics{
		TotalRuns:     10,
		SuccessRuns:   8,
		FailedRuns:    2,
		RetryCount:    3,
		AvgDuration:   time.Second * 30,
		MaxDuration:   time.Minute,
		LastRunAt:     time.Now(),
		LastSuccessAt: time.Now(),
		LastError:     "test error",
	}

	if metrics.TotalRuns != 10 {
		t.Errorf("Expected TotalRuns to be 10, got %d", metrics.TotalRuns)
	}

	if metrics.SuccessRuns != 8 {
		t.Errorf("Expected SuccessRuns to be 8, got %d", metrics.SuccessRuns)
	}

	if metrics.FailedRuns != 2 {
		t.Errorf("Expected FailedRuns to be 2, got %d", metrics.FailedRuns)
	}

	if metrics.RetryCount != 3 {
		t.Errorf("Expected RetryCount to be 3, got %d", metrics.RetryCount)
	}
}

// TestTaskRunStatus 测试任务运行状态
func TestTaskRunStatus(t *testing.T) {
	tests := []struct {
		status    TaskRunStatus
		expectStr string
	}{
		{TaskPending, "待运行"},
		{TaskRunning, "运行中"},
		{TaskCompleted, "已完成"},
		{TaskTerminated, "已终止"},
		{TaskFailed, "运行失败"},
		{TaskRetrying, "重试中"},
	}

	statusNames := map[TaskRunStatus]string{
		TaskPending:    "TaskPending",
		TaskRunning:    "TaskRunning",
		TaskCompleted:  "TaskCompleted",
		TaskTerminated: "TaskTerminated",
		TaskFailed:     "TaskFailed",
		TaskRetrying:   "TaskRetrying",
	}

	for _, tt := range tests {
		name := statusNames[tt.status]
		if name == "" {
			t.Errorf("Unknown status: %d", tt.status)
		}
		t.Logf("Status %d: %s", tt.status, name)
	}
}

// TestTaskType 测试任务类型
func TestTaskType(t *testing.T) {
	if TaskTypeCron != 0 {
		t.Errorf("Expected TaskTypeCron to be 0, got %d", TaskTypeCron)
	}

	if TaskTypeInterval != 1 {
		t.Errorf("Expected TaskTypeInterval to be 1, got %d", TaskTypeInterval)
	}

	if TaskTypeOnce != 2 {
		t.Errorf("Expected TaskTypeOnce to be 2, got %d", TaskTypeOnce)
	}
}

// TestTaskCallbackMock 测试任务回调模拟
type MockTaskCallback struct {
	onStartCalls           int
	onCompleteCalls       int
	onNextRunCalculatedCalls int
	lastStartTask         *TaskInfo
	lastCompleteTask      *TaskInfo
	lastCompleteErr       error
	lastNextRun           time.Time
}

func (m *MockTaskCallback) OnStart(ctx context.Context, task *TaskInfo) {
	m.onStartCalls++
	m.lastStartTask = task
}

func (m *MockTaskCallback) OnComplete(ctx context.Context, task *TaskInfo, err error) {
	m.onCompleteCalls++
	m.lastCompleteTask = task
	m.lastCompleteErr = err
}

func (m *MockTaskCallback) OnNextRunCalculated(task *TaskInfo, nextRun time.Time) {
	m.onNextRunCalculatedCalls++
	m.lastNextRun = nextRun
}

// TestMockTaskCallback 测试模拟回调
func TestMockTaskCallback(t *testing.T) {
	callback := &MockTaskCallback{}
	ctx := context.Background()
	task := &TaskInfo{
		ID:   1,
		Name: "test-task",
	}

	callback.OnStart(ctx, task)
	if callback.onStartCalls != 1 {
		t.Errorf("Expected onStartCalls to be 1, got %d", callback.onStartCalls)
	}
	if callback.lastStartTask != task {
		t.Error("Expected lastStartTask to match")
	}

	callback.OnComplete(ctx, task, nil)
	if callback.onCompleteCalls != 1 {
		t.Errorf("Expected onCompleteCalls to be 1, got %d", callback.onCompleteCalls)
	}

	nextRun := time.Now()
	callback.OnNextRunCalculated(task, nextRun)
	if callback.onNextRunCalculatedCalls != 1 {
		t.Errorf("Expected onNextRunCalculatedCalls to be 1, got %d", callback.onNextRunCalculatedCalls)
	}
	if callback.lastNextRun != nextRun {
		t.Error("Expected lastNextRun to match")
	}
}

// TestMockTaskExecutor 测试模拟任务执行器
type MockTaskExecutor struct {
	executeCalls  int
	executeErr    error
	executeDelay  time.Duration
}

func (m *MockTaskExecutor) Execute(ctx context.Context) error {
	m.executeCalls++
	if m.executeDelay > 0 {
		time.Sleep(m.executeDelay)
	}
	return m.executeErr
}

// TestMockTaskExecutorExecute 测试模拟执行器执行
func TestMockTaskExecutorExecute(t *testing.T) {
	executor := &MockTaskExecutor{}
	ctx := context.Background()

	err := executor.Execute(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if executor.executeCalls != 1 {
		t.Errorf("Expected executeCalls to be 1, got %d", executor.executeCalls)
	}

	executor.executeErr = ErrTaskAlreadyRunning
	err = executor.Execute(ctx)
	if err != ErrTaskAlreadyRunning {
		t.Errorf("Expected error ErrTaskAlreadyRunning, got %v", err)
	}
	if executor.executeCalls != 2 {
		t.Errorf("Expected executeCalls to be 2, got %d", executor.executeCalls)
	}
}
