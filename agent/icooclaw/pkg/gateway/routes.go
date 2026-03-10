package gateway

import (
	"log/slog"

	"icooclaw/pkg/gateway/handlers"
	"icooclaw/pkg/storage"

	"github.com/go-chi/chi/v5"
)

// Handlers 封装所有处理器
type Handlers struct {
	Common   *handlers.CommonHandler
	Session  *handlers.SessionHandler
	Message  *handlers.MessageHandler
	MCP      *handlers.MCPHandler
	Memory   *handlers.MemoryHandler
	Task     *handlers.TaskHandler
	Provider *handlers.ProviderHandler
	Skill    *handlers.SkillHandler
	Channel  *handlers.ChannelHandler
	Param    *handlers.ParamHandler
	Tool     *handlers.ToolHandler
	Binding  *handlers.BindingHandler
}

// NewHandlers 创建所有处理器
func NewHandlers(logger *slog.Logger, storage *storage.Storage) *Handlers {
	return &Handlers{
		Common:   handlers.NewCommonHandler(logger),
		Session:  handlers.NewSessionHandler(logger, storage),
		Message:  handlers.NewMessageHandler(logger, storage),
		MCP:      handlers.NewMCPHandler(logger, storage),
		Memory:   handlers.NewMemoryHandler(logger, storage),
		Task:     handlers.NewTaskHandler(logger, storage),
		Provider: handlers.NewProviderHandler(logger, storage),
		Skill:    handlers.NewSkillHandler(logger, storage),
		Channel:  handlers.NewChannelHandler(logger, storage),
		Param:    handlers.NewParamHandler(logger, storage),
		Tool:     handlers.NewToolHandler(logger, storage),
		Binding:  handlers.NewBindingHandler(logger, storage),
	}
}

// RegisterRoutes 注册所有 CRUD 路由
func RegisterRoutes(r chi.Router, h *Handlers) {
	// 健康检查
	r.Get("/api/v1/health", h.Common.HealthCheck)

	// Session 路由
	r.Route("/api/v1/sessions", func(r chi.Router) {
		r.Post("/page", h.Session.Page)     // 分页查询
		r.Post("/save", h.Session.Save)     // 保存
		r.Post("/create", h.Session.Create) // 创建新会话
		r.Post("/delete", h.Session.Delete) // 删除
		r.Post("/get", h.Session.GetByID)   // 获取单个
	})

	// Message 路由
	r.Route("/api/v1/messages", func(r chi.Router) {
		r.Post("/page", h.Message.Page)
		r.Post("/create", h.Message.Create)
		r.Post("/update", h.Message.Update)
		r.Post("/delete", h.Message.Delete)
		r.Post("/get", h.Message.GetByID)
	})

	// MCP 路由
	r.Route("/api/v1/mcp", func(r chi.Router) {
		r.Post("/page", h.MCP.Page)
		r.Post("/create", h.MCP.Create)
		r.Post("/update", h.MCP.Update)
		r.Post("/delete", h.MCP.Delete)
		r.Post("/get", h.MCP.GetByID)
		r.Get("/all", h.MCP.GetAll)
	})

	// Memory 路由
	r.Route("/api/v1/memories", func(r chi.Router) {
		r.Post("/page", h.Memory.Page)
		r.Post("/create", h.Memory.Create)
		r.Post("/update", h.Memory.Update)
		r.Post("/delete", h.Memory.Delete)
		r.Post("/get", h.Memory.GetByID)
		r.Post("/search", h.Memory.Search)
	})

	// Task 路由
	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Post("/page", h.Task.Page)
		r.Post("/create", h.Task.Create)
		r.Post("/update", h.Task.Update)
		r.Post("/delete", h.Task.Delete)
		r.Post("/get", h.Task.GetByID)
		r.Post("/toggle", h.Task.ToggleEnabled)
		r.Get("/all", h.Task.GetAll)
		r.Get("/enabled", h.Task.GetEnabled)
	})

	// Provider 路由
	r.Route("/api/v1/providers", func(r chi.Router) {
		r.Post("/page", h.Provider.Page)
		r.Post("/create", h.Provider.Create)
		r.Post("/update", h.Provider.Update)
		r.Post("/delete", h.Provider.Delete)
		r.Post("/get", h.Provider.GetByID)
		r.Get("/all", h.Provider.GetAll)
		r.Get("/enabled", h.Provider.GetEnabled)
	})

	// Skill 路由
	r.Route("/api/v1/skills", func(r chi.Router) {
		r.Post("/page", h.Skill.Page)
		r.Post("/create", h.Skill.Create)
		r.Post("/update", h.Skill.Update)
		r.Post("/delete", h.Skill.Delete)
		r.Post("/get", h.Skill.GetByID)
		r.Post("/get-by-name", h.Skill.GetByName)
		r.Post("/upsert", h.Skill.Upsert)
		r.Get("/all", h.Skill.GetAll)
		r.Get("/enabled", h.Skill.GetEnabled)
	})

	// Channel 路由
	r.Route("/api/v1/channels", func(r chi.Router) {
		r.Post("/page", h.Channel.Page)
		r.Post("/create", h.Channel.Create)
		r.Post("/update", h.Channel.Update)
		r.Post("/delete", h.Channel.Delete)
		r.Post("/get", h.Channel.GetByID)
		r.Get("/all", h.Channel.GetAll)
		r.Get("/enabled", h.Channel.GetEnabled)
	})

	// 参数配置路由
	r.Route("/api/v1/params", func(r chi.Router) {
		r.Post("/page", h.Param.Page)           // 分页查询
		r.Post("/create", h.Param.Create)       // 创建
		r.Post("/update", h.Param.Update)       // 更新
		r.Post("/delete", h.Param.Delete)       // 删除
		r.Post("/get", h.Param.GetByID)         // 通过 ID 获取
		r.Post("/get-by-key", h.Param.GetByKey) // 通过键获取
		r.Get("/all", h.Param.GetAll)           // 获取所有
		r.Post("/by-group", h.Param.GetByGroup) // 按分组获取

		// 便捷接口
		r.Post("/default-model/set", h.Param.SetDefaultModel) // 设置默认模型
		r.Get("/default-model/get", h.Param.GetDefaultModel)  // 获取默认模型
	})

	// Tool 路由
	r.Route("/api/v1/tools", func(r chi.Router) {
		r.Post("/page", h.Tool.Page)
		r.Post("/create", h.Tool.Create)
		r.Post("/update", h.Tool.Update)
		r.Post("/delete", h.Tool.Delete)
		r.Post("/get", h.Tool.GetByID)
		r.Get("/all", h.Tool.GetAll)
		r.Get("/enabled", h.Tool.GetEnabled)
	})

	// Binding 路由
	r.Route("/api/v1/bindings", func(r chi.Router) {
		r.Post("/page", h.Binding.Page)
		r.Post("/create", h.Binding.Create)
		r.Post("/update", h.Binding.Update)
		r.Post("/delete", h.Binding.Delete)
		r.Post("/get", h.Binding.GetByID)
		r.Get("/all", h.Binding.GetAll)
	})
}

// HandleCRUD 通用 CRUD 处理函数，用于 AI Skill 调用
type CRUDRequest struct {
	Resource string         `json:"resource"` // sessions, messages, mcp, memories, tasks, providers, skills, channels
	Action   string         `json:"action"`   // page, create, update, delete, get, get-all, etc.
	Data     map[string]any `json:"data"`     // 请求数据
}

// CRUDResponse 通用 CRUD 响应
type CRUDResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
