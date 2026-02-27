package builtons

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// === Console 对象 ===

type Console struct {
	logger *slog.Logger
}

func NewConsole(logger *slog.Logger) *Console {
	if logger == nil {
		logger = slog.Default()
	}
	return &Console{logger: logger}
}

func (c *Console) Log(args ...interface{}) {
	c.logger.Info(fmt.Sprint(args...))
}

func (c *Console) Info(args ...interface{}) {
	c.logger.Info(fmt.Sprint(args...))
}

func (c *Console) Debug(args ...interface{}) {
	c.logger.Debug(fmt.Sprint(args...))
}

func (c *Console) Warn(args ...interface{}) {
	c.logger.Warn(fmt.Sprint(args...))
}

func (c *Console) Error(args ...interface{}) {
	c.logger.Error(fmt.Sprint(args...))
}

func (c *Console) Table(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	c.logger.Info(string(b))
}

func (c *Console) Time(label string) {
	c.logger.Debug("Timer started", "label", label)
}

func (c *Console) TimeEnd(label string) {
	c.logger.Debug("Timer ended", "label", label)
}
