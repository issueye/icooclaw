package storage

import (
	"gorm.io/gorm"
)

// Storage 存储封装
type Storage struct {
	db             *gorm.DB
	skill          *SkillStorage          // 技能
	memory         *MemoryStorage         // 记忆
	task           *TaskStorage           // 任务
	session        *SessionStorage        // 会话
	message        *MessageStorage        // 消息
	channelConfig  *ChannelConfigStorage  // 渠道配置
	mcpConfig      *MCPConfigStorage      // MCP 配置
	providerConfig *ProviderConfigStorage // 提供程序配置
	paramConfig    *ParamConfigStorage    // 参数配置
}

// NewStorage 创建存储实例
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{
		db:             db,
		skill:          NewSkillStorage(db),
		memory:         NewMemoryStorage(db),
		task:           NewTaskStorage(db),
		session:        NewSessionStorage(db),
		message:        NewMessageStorage(db),
		mcpConfig:      NewMCPConfigStorage(db),
		channelConfig:  NewChannelConfigStorage(db),
		providerConfig: NewProviderConfigStorage(db),
		paramConfig:    NewParamConfigStorage(db),
	}
}

// DB 返回原生数据库实例
func (s *Storage) DB() *gorm.DB {
	return s.db
}

func (s *Storage) GetProviderConfig() (*ProviderConfig, error) {
	var config ProviderConfig
	err := s.db.First(&config).Error
	return &config, err
}

func (s *Storage) ChannelConfig() *ChannelConfigStorage {
	return s.channelConfig
}

func (s *Storage) MCPConfig() *MCPConfigStorage {
	return s.mcpConfig
}

func (s *Storage) Skill() *SkillStorage {
	return s.skill
}

func (s *Storage) Memory() *MemoryStorage {
	return s.memory
}

func (s *Storage) Task() *TaskStorage {
	return s.task
}

func (s *Storage) Session() *SessionStorage {
	return s.session
}

func (s *Storage) Message() *MessageStorage {
	return s.message
}

func (s *Storage) ProviderConfig() *ProviderConfigStorage {
	return s.providerConfig
}

func (s *Storage) ParamConfig() *ParamConfigStorage {
	return s.paramConfig
}
