// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestScheduler_AddTask(t *testing.T) {
	s := NewScheduler(nil)

	task := &Task{
		ID:          "test-1",
		Name:        "Test Task",
		Schedule:    EveryMinute,
		Description: "A test task",
		Handler:     func(ctx context.Context) error { return nil },
		Enabled:     true,
	}

	err := s.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	if len(s.tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(s.tasks))
	}
}

func TestScheduler_AddDuplicateTask(t *testing.T) {
	s := NewScheduler(nil)

	task := &Task{
		ID:       "test-1",
		Name:     "Test Task",
		Schedule: EveryMinute,
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	}

	_ = s.AddTask(task)

	err := s.AddTask(task)
	if err == nil {
		t.Error("Expected error for duplicate task")
	}
}

func TestScheduler_RemoveTask(t *testing.T) {
	s := NewScheduler(nil)

	task := &Task{
		ID:       "test-1",
		Name:     "Test Task",
		Schedule: EveryMinute,
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	}

	_ = s.AddTask(task)

	err := s.RemoveTask("test-1")
	if err != nil {
		t.Fatalf("Failed to remove task: %v", err)
	}

	if len(s.tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(s.tasks))
	}
}

func TestScheduler_EnableDisableTask(t *testing.T) {
	s := NewScheduler(nil)

	task := &Task{
		ID:       "test-1",
		Name:     "Test Task",
		Schedule: EveryMinute,
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	}

	_ = s.AddTask(task)

	err := s.DisableTask("test-1")
	if err != nil {
		t.Fatalf("Failed to disable task: %v", err)
	}

	if s.tasks["test-1"].Enabled {
		t.Error("Task should be disabled")
	}

	err = s.EnableTask("test-1")
	if err != nil {
		t.Fatalf("Failed to enable task: %v", err)
	}

	if !s.tasks["test-1"].Enabled {
		t.Error("Task should be enabled")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	s := NewScheduler(nil)

	task := &Task{
		ID:       "test-1",
		Name:     "Test Task",
		Schedule: EveryMinute,
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	}

	_ = s.AddTask(task)

	s.Start()
	if !s.IsRunning() {
		t.Error("Scheduler should be running")
	}

	s.Stop()
	if s.IsRunning() {
		t.Error("Scheduler should be stopped")
	}
}

func TestScheduler_RunTask(t *testing.T) {
	s := NewScheduler(nil)

	executed := false
	task := &Task{
		ID:       "test-1",
		Name:     "Test Task",
		Schedule: EveryMinute,
		Handler: func(ctx context.Context) error {
			executed = true
			return nil
		},
		Enabled: true,
	}

	_ = s.AddTask(task)

	err := s.RunTask("test-1")
	if err != nil {
		t.Fatalf("Failed to run task: %v", err)
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	if !executed {
		t.Error("Task should have been executed")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		duration  string
		want      string
		wantError bool
	}{
		{"1m", EveryMinute, false},
		{"5m", Every5Minutes, false},
		{"15m", Every15Minutes, false},
		{"30m", Every30Minutes, false},
		{"1h", EveryHour, false},
		{"2h", Every2Hours, false},
		{"30s", "", true}, // Too short
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.duration, func(t *testing.T) {
			got, err := ParseDuration(tt.duration)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("Expected %s, got %s", tt.want, got)
				}
			}
		})
	}
}

func TestSchedulerTool(t *testing.T) {
	s := NewScheduler(nil)

	task := &Task{
		ID:       "test-1",
		Name:     "Test Task",
		Schedule: EveryMinute,
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	}
	_ = s.AddTask(task)

	tool := NewTool(s, nil)

	if tool.Name() != "scheduler" {
		t.Errorf("Expected name 'scheduler', got %s", tool.Name())
	}

	// Test list action
	result := tool.Execute(nil, map[string]any{"action": "list"})
	if !result.Success {
		t.Errorf("List action failed: %v", result.Error)
	}

	// Test run action
	result = tool.Execute(nil, map[string]any{"action": "run", "task_id": "test-1"})
	if !result.Success {
		t.Errorf("Run action failed: %v", result.Error)
	}

	// Test disable action
	result = tool.Execute(nil, map[string]any{"action": "disable", "task_id": "test-1"})
	if !result.Success {
		t.Errorf("Disable action failed: %v", result.Error)
	}

	// Test enable action
	result = tool.Execute(nil, map[string]any{"action": "enable", "task_id": "test-1"})
	if !result.Success {
		t.Errorf("Enable action failed: %v", result.Error)
	}
}
