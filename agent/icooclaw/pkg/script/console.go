// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// Console provides console logging functions.
type Console struct {
	logger *slog.Logger
}

// NewConsole creates a new Console builtin.
func NewConsole(logger *slog.Logger) *Console {
	if logger == nil {
		logger = slog.Default()
	}
	return &Console{logger: logger}
}

// Name returns the builtin name.
func (c *Console) Name() string {
	return "console"
}

// Object returns the console object.
func (c *Console) Object() map[string]any {
	return map[string]any{
		"log":     c.Log,
		"info":    c.Info,
		"debug":   c.Debug,
		"warn":    c.Warn,
		"error":   c.Error,
		"table":   c.Table,
		"time":    c.Time,
		"timeEnd": c.TimeEnd,
	}
}

func (c *Console) Log(args ...any) {
	c.logger.Info(fmt.Sprint(args...))
}

func (c *Console) Info(args ...any) {
	c.logger.Info(fmt.Sprint(args...))
}

func (c *Console) Debug(args ...any) {
	c.logger.Debug(fmt.Sprint(args...))
}

func (c *Console) Warn(args ...any) {
	c.logger.Warn(fmt.Sprint(args...))
}

func (c *Console) Error(args ...any) {
	c.logger.Error(fmt.Sprint(args...))
}

func (c *Console) Table(data any) {
	b, _ := json.MarshalIndent(data, "", "  ")
	c.logger.Info(string(b))
}

func (c *Console) Time(label string) {
	c.logger.Debug("Timer started", "label", label)
}

func (c *Console) TimeEnd(label string) {
	c.logger.Debug("Timer ended", "label", label)
}