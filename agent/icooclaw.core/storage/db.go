package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库连接实例
var DB *gorm.DB

// SqlLogger 自定义 SQL Logger，使用 slog 输出 Debug 级别
type SqlLogger struct {
	logger.Interface
	level logger.LogLevel
}

// LogMode 实现 logger.Interface
func (l *SqlLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &SqlLogger{
		Interface: l.Interface.LogMode(level),
		level:     level,
	}
}

// Debug 实现 Debug 方法 - SQL 日志以 debug 级别输出
func (l *SqlLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	slog.Debug("SQL", "query", fmt.Sprintf(msg, args...))
}

// Info 实现 Info 方法
func (l *SqlLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	slog.Info("SQL", "query", fmt.Sprintf(msg, args...))
}

// Warn 实现 Warn 方法
func (l *SqlLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	slog.Warn("SQL", "query", fmt.Sprintf(msg, args...))
}

// Error 实现 Error 方法
func (l *SqlLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	slog.Error("SQL", "query", fmt.Sprintf(msg, args...))
}

// InitDB 初始化数据库
func InitDB(dsn string) (*gorm.DB, error) {
	// 确保数据库目录存在
	dbPath := dsn
	if !filepath.IsAbs(dsn) {
		absPath, err := filepath.Abs(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		dbPath = absPath
	}

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: &SqlLogger{
			Interface: logger.Default.LogMode(logger.Info),
			level:     logger.Info,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移
	err = DB.AutoMigrate(
		&Session{},
		&Message{},
		&Task{},
		&Skill{},
		&Memory{},
		&ChannelConfig{},
		&ProviderConfig{},
		&MCPConfig{},
		&ParamConfig{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return DB, nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GenerateSessionKey 生成会话 Key
func GenerateSessionKey(channel, chatID string) string {
	return fmt.Sprintf("%s:%s", channel, chatID)
}
