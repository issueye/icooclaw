package scheduler

import (
	"testing"
	"time"
)

func TestCronParser_IsValid(t *testing.T) {
	parser := NewCronParser()

	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		{"Valid: every minute", "* * * * *", true},
		{"Valid: every hour", "0 * * * *", true},
		{"Valid: every day at midnight", "0 0 * * *", true},
		{"Valid: every day at noon", "0 12 * * *", true},
		{"Valid: specific time", "30 14 * * *", true},
		{"Valid: multiple times", "0,30 * * * *", true},
		{"Valid: range", "0 0 1-7 * *", true},
		{"Invalid: empty string", "", false},
		{"Invalid: malformed", "* * *", false},
		{"Invalid: letters", "abc def", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.IsValid(tt.expr)
			if result != tt.expected {
				t.Errorf("IsValid(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestCronParser_NextRun(t *testing.T) {
	parser := NewCronParser()
	now := time.Now()

	tests := []struct {
		name       string
		expr       string
		wantValid  bool
		wantInPast bool // 是否希望返回过去的时间
	}{
		{
			name:      "Every minute",
			expr:      "* * * * *",
			wantValid: true,
		},
		{
			name:      "Every hour",
			expr:      "0 * * * *",
			wantValid: true,
		},
		{
			name:      "Invalid expression",
			expr:      "invalid",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, valid := parser.NextRun(tt.expr, now)
			if valid != tt.wantValid {
				t.Errorf("NextRun(%q) valid = %v, want %v", tt.expr, valid, tt.wantValid)
			}
			if tt.wantValid && valid {
				if next.Before(now) && !tt.wantInPast {
					t.Errorf("NextRun(%q) = %v, should be in the future", tt.expr, next)
				}
			}
		})
	}
}

func TestCronParser_ShouldRun(t *testing.T) {
	parser := NewCronParser()

	tests := []struct {
		name string
		expr string
		now  time.Time
		want bool
	}{
		{
			name: "Should run - within 60 seconds",
			expr: "* * * * *",
			now:  time.Now(),
			want: true,
		},
		{
			name: "Should not run - invalid expression",
			expr: "invalid",
			now:  time.Now(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ShouldRun(tt.expr, tt.now)
			if result != tt.want {
				t.Errorf("ShouldRun(%q) = %v, want %v", tt.expr, result, tt.want)
			}
		})
	}
}

func TestSchedulerError_Error(t *testing.T) {
	tests := []struct {
		err    *SchedulerError
		expect string
	}{
		{ErrTaskNotFound, "task not found"},
		{ErrTaskAlreadyExists, "task already exists"},
		{ErrInvalidCronExpression, "invalid cron expression"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expect {
				t.Errorf("SchedulerError.Error() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestScheduler_NewScheduler(t *testing.T) {
	// 仅测试构造函数不 panic
	parser := NewCronParser()
	if parser == nil {
		t.Error("NewCronParser() returned nil")
	}

	if parser.gronx == nil {
		t.Error("parser.gronx is nil")
	}
}
