// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"time"
)

// TaskExecutionRecord represents a task execution record.
type TaskExecutionRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"index;not null" json:"task_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Success   bool      `json:"success"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name.
func (TaskExecutionRecord) TableName() string {
	return "task_executions"
}
