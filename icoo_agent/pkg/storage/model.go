package storage

import (
	"database/sql/driver"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const tableNamePrefix = "icooclaw_"

type Model struct {
	ID        string    `gorm:"column:id;type:char(36);primaryKey;comment:主键UUID" json:"id"`    // 主键 uuid
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;comment:创建时间" json:"created_at"` // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;comment:更新时间" json:"updated_at"` // 更新时间
}

func (c *Model) BeforeCreate(tx *gorm.DB) error {
	// 自动生成主键 uuid
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type StringArray []string

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return nil
	}

	if str == "" {
		*s = nil
		return nil
	}

	*s = strings.Split(str, ",")
	return nil
}

func (s StringArray) Value() (driver.Value, error) {
	return strings.Join(s, ","), nil
}

func (s StringArray) String() string {
	return strings.Join(s, ",")
}

type Page struct {
	Size  int   `json:"size"`
	Page  int   `json:"page"`
	Total int64 `json:"total"`
}
