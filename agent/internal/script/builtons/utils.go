package builtons

import (
	"fmt"
	"os"
	"time"
)

// === utils 工具函数 ===

type Utils struct{}

func NewUtils() *Utils {
	return &Utils{}
}

// Sleep 休眠
func (u *Utils) Sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// Now 获取当前时间
func (u *Utils) Now() string {
	return time.Now().Format(time.RFC3339)
}

// Timestamp 获取时间戳
func (u *Utils) Timestamp() int64 {
	return time.Now().Unix()
}

// FormatTime 格式化时间
func (u *Utils) FormatTime(timestamp int64, layout string) string {
	if layout == "" {
		layout = time.RFC3339
	}
	return time.Unix(timestamp, 0).Format(layout)
}

// ParseTime 解析时间
func (u *Utils) ParseTime(timeStr string) (int64, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// Env 获取环境变量
func (u *Utils) Env(key string) string {
	return os.Getenv(key)
}

// EnvOr 获取环境变量或默认值
func (u *Utils) EnvOr(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// Cwd 获取当前工作目录
func (u *Utils) Cwd() string {
	cwd, _ := os.Getwd()
	return cwd
}

// Hostname 获取主机名
func (u *Utils) Hostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// UUID 生成 UUID
func (u *Utils) UUID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
