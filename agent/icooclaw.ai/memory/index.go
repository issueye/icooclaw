package memory

// Message 消息结构体
type Message struct {
	Role             string `json:"role"`              // 角色
	ToolCallID       string `json:"tool_call_id"`      // 调用ID
	ToolCallName     string `json:"tool_call_name"`    // 调用名称
	ToolArguments    string `json:"tool_arguments"`    // 调用参数
	ToolCallResult   string `json:"tool_call_result"`  // 调用结果
	Content          string `json:"content"`           // 内容
	Thinking         string `json:"thinking"`          // 思考内容
	ReasoningContent string `json:"reasoning_content"` // 推理内容
}

// Loader 记忆加载器接口 以会话ID为单位
type Loader interface {
	Load(id string, maxCount int) ([]*Message, error)      // Load 加载记忆 最多加载maxCount条 如果是0则加载所有
	Save(id string, msg *Message) error                    // Save 保存记忆
	BatchSave(id string, messages []*Message) error        // BatchSave 批量保存记忆
	Delete(id string) error                                // Delete 删除记忆
	Update(id string, msg *Message) error                  // Update 更新记忆
	Search(id string, query string) ([]*Message, error)    // Search 搜索记忆
	SummarizeMessages(messages []*Message) (string, error) // SummarizeMessages 对消息进行摘要 默认摘要100条
}

// Config 加载器配置结构体
type Config struct {
	ConsolidationThreshold int  `mapstructure:"consolidation_threshold"` // 合并阈值
	SummaryEnabled         bool `mapstructure:"summary_enabled"`         // 是否启用摘要 默认摘要
	AutoSave               bool `mapstructure:"auto_save"`               // 是否自动保存 默认自动保存
	MaxMemoryAge           int  `mapstructure:"max_memory_age"`          // 最大记忆年龄（天） 默认30天
	MaxSessionMemories     int  `mapstructure:"max_session_memories"`    // 最大会话记忆数 默认100条
	MaxUserMemories        int  `mapstructure:"max_user_memories"`       // 最大用户记忆数 默认0表示不限制
}

// DefMemCfg 默认默认配置
func DefMemCfg() Config {
	return Config{
		ConsolidationThreshold: 100,
		SummaryEnabled:         true,
		AutoSave:               true,
		MaxMemoryAge:           30,
		MaxSessionMemories:     100,
		MaxUserMemories:        0,
	}
}
