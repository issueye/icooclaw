// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// Utils provides utility functions.
type Utils struct{}

// NewUtils creates a new Utils builtin.
func NewUtils() *Utils {
	return &Utils{}
}

// Name returns the builtin name.
func (u *Utils) Name() string {
	return "utils"
}

// Object returns the utils object.
func (u *Utils) Object() map[string]any {
	return map[string]any{
		"sleep":      u.Sleep,
		"now":        u.Now,
		"timestamp":  u.Timestamp,
		"formatTime": u.FormatTime,
		"parseTime":  u.ParseTime,
		"env":        u.Env,
		"envOr":      u.EnvOr,
		"cwd":        u.Cwd,
		"hostname":   u.Hostname,
		"uuid":       u.UUID,
	}
}

// Sleep pauses execution for the specified milliseconds.
func (u *Utils) Sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// Now returns the current time in RFC3339 format.
func (u *Utils) Now() string {
	return time.Now().Format(time.RFC3339)
}

// Timestamp returns the current Unix timestamp.
func (u *Utils) Timestamp() int64 {
	return time.Now().Unix()
}

// FormatTime formats a timestamp.
func (u *Utils) FormatTime(timestamp int64, layout string) string {
	if layout == "" {
		layout = time.RFC3339
	}
	return time.Unix(timestamp, 0).Format(layout)
}

// ParseTime parses a time string.
func (u *Utils) ParseTime(timeStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// Env returns an environment variable.
func (u *Utils) Env(key string) string {
	return os.Getenv(key)
}

// EnvOr returns an environment variable or default value.
func (u *Utils) EnvOr(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// Cwd returns the current working directory.
func (u *Utils) Cwd() string {
	cwd, _ := os.Getwd()
	return cwd
}

// Hostname returns the hostname.
func (u *Utils) Hostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// UUID generates a new UUID.
func (u *Utils) UUID() string {
	return uuid.New().String()
}

// UUIDv4 generates a new UUID v4.
func (u *Utils) UUIDv4() string {
	return uuid.New().String()
}

// ShortID generates a short ID.
func (u *Utils) ShortID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())[:12]
}