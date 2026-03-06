package storage

import (
	"database/sql/driver"
	"strings"
	"time"
)

const tableNamePrefix = "icooclaw_"

type Model struct {
	ID        string    `gorm:"primaryKey" json:"id"`         // 主键 uuid
	CreatedAt time.Time `gorm:"created_at" json:"created_at"` // 创建时间
	UpdatedAt time.Time `gorm:"updated_at" json:"updated_at"` // 更新时间
}

type StringArray []string

func (s *StringArray) Scan(value interface{}) error {
	return nil
}

func (s StringArray) Value() (driver.Value, error) {
	return strings.Join(s, ","), nil
}

func (s StringArray) String() string {
	return strings.Join(s, ",")
}

type Page struct {
	Size  int `json:"size"`
	Page  int `json:"page"`
	Total int `json:"total"`
}
