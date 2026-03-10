package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"icooclaw.core/storage"
	"icooclaw.gateway/models"
)

// SkillHandler 技能处理器
type SkillHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

// NewSkillHandler 创建技能处理器
func NewSkillHandler(logger *slog.Logger, storage *storage.Storage) *SkillHandler {
	return &SkillHandler{logger: logger, storage: storage}
}

// Page 分页获取技能列表
func (h *SkillHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QuerySkill](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	skills, err := h.storage.Skill().Page(req)
	if err != nil {
		h.logger.Error("获取技能列表失败", "error", err)
		http.Error(w, "获取技能列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQuerySkill]{
		Code:    http.StatusOK,
		Message: "技能列表获取成功",
		Data:    skills,
	})
}

// Save 保存技能（创建或更新）
func (h *SkillHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定保存技能请求失败", "error", err)
		http.Error(w, "绑定保存技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().CreateOrUpdate(req)
	if err != nil {
		h.logger.Error("保存技能失败", "error", err)
		http.Error(w, "保存技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能保存成功",
		Data:    req,
	})
}

// Create 创建技能
func (h *SkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定创建技能请求失败", "error", err)
		http.Error(w, "绑定创建技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().Create(req)
	if err != nil {
		h.logger.Error("创建技能失败", "error", err)
		http.Error(w, "创建技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能创建成功",
		Data:    req,
	})
}

// Update 更新技能
func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定更新技能请求失败", "error", err)
		http.Error(w, "绑定更新技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().Update(req)
	if err != nil {
		h.logger.Error("更新技能失败", "error", err)
		http.Error(w, "更新技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能更新成功",
		Data:    req,
	})
}

// Delete 删除技能
func (h *SkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除技能请求失败", "error", err)
		http.Error(w, "绑定删除技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().Delete(id)
	if err != nil {
		h.logger.Error("删除技能失败", "error", err)
		http.Error(w, "删除技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "技能删除成功",
	})
}

// GetByID 根据ID获取技能
func (h *SkillHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取技能请求失败", "error", err)
		http.Error(w, "绑定获取技能请求失败", http.StatusBadRequest)
		return
	}

	skill, err := h.storage.Skill().GetByID(id)
	if err != nil {
		h.logger.Error("获取技能失败", "error", err)
		http.Error(w, "获取技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能获取成功",
		Data:    skill,
	})
}

// GetByName 根据名称获取技能
func (h *SkillHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Name string `json:"name"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取技能请求失败", "error", err)
		http.Error(w, "绑定获取技能请求失败", http.StatusBadRequest)
		return
	}

	skill, err := h.storage.Skill().GetByName(req.Name)
	if err != nil {
		h.logger.Error("获取技能失败", "error", err)
		http.Error(w, "获取技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能获取成功",
		Data:    skill,
	})
}

// GetAll 获取所有技能
func (h *SkillHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().GetAll()
	if err != nil {
		h.logger.Error("获取所有技能失败", "error", err)
		http.Error(w, "获取所有技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能列表获取成功",
		Data:    skills,
	})
}

// GetEnabled 获取启用的技能
func (h *SkillHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().GetEnabled()
	if err != nil {
		h.logger.Error("获取启用技能失败", "error", err)
		http.Error(w, "获取启用技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Skill]{
		Code:    http.StatusOK,
		Message: "启用技能列表获取成功",
		Data:    skills,
	})
}

// Upsert 创建或更新技能
func (h *SkillHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定创建或更新技能请求失败", "error", err)
		http.Error(w, "绑定创建或更新技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().Upsert(req)
	if err != nil {
		h.logger.Error("创建或更新技能失败", "error", err)
		http.Error(w, "创建或更新技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能创建或更新成功",
		Data:    req,
	})
}

// ==================== 批量操作接口 ====================

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []string `json:"ids"`
}

// BatchDelete 批量删除技能
// 根据ID列表批量删除技能
func (h *SkillHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[BatchDeleteRequest](r)
	if err != nil {
		h.logger.Error("绑定批量删除请求失败", "error", err)
		http.Error(w, "绑定批量删除请求失败", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "ID列表不能为空", http.StatusBadRequest)
		return
	}

	count, err := h.storage.Skill().DeleteByIDs(req.IDs)
	if err != nil {
		h.logger.Error("批量删除技能失败", "error", err)
		http.Error(w, "批量删除技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]int64]{
		Code:    http.StatusOK,
		Message: "批量删除成功",
		Data:    map[string]int64{"deleted": count},
	})
}

// BatchUpdateEnabledRequest 批量更新启用状态请求
type BatchUpdateEnabledRequest struct {
	IDs     []string `json:"ids"`
	Enabled bool     `json:"enabled"`
}

// BatchUpdateEnabled 批量更新启用状态
// 批量启用或禁用技能
func (h *SkillHandler) BatchUpdateEnabled(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[BatchUpdateEnabledRequest](r)
	if err != nil {
		h.logger.Error("绑定批量更新请求失败", "error", err)
		http.Error(w, "绑定批量更新请求失败", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "ID列表不能为空", http.StatusBadRequest)
		return
	}

	count, err := h.storage.Skill().BatchUpdateEnabled(req.IDs, req.Enabled)
	if err != nil {
		h.logger.Error("批量更新启用状态失败", "error", err)
		http.Error(w, "批量更新启用状态失败", http.StatusInternalServerError)
		return
	}

	action := "启用"
	if !req.Enabled {
		action = "禁用"
	}

	models.WriteData(w, models.BaseResponse[map[string]int64]{
		Code:    http.StatusOK,
		Message: "批量" + action + "成功",
		Data:    map[string]int64{"updated": count},
	})
}

// BatchUpdateAlwaysLoadRequest 批量更新始终加载状态请求
type BatchUpdateAlwaysLoadRequest struct {
	IDs        []string `json:"ids"`
	AlwaysLoad bool     `json:"always_load"`
}

// BatchUpdateAlwaysLoad 批量更新始终加载状态
// 批量设置技能是否始终加载
func (h *SkillHandler) BatchUpdateAlwaysLoad(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[BatchUpdateAlwaysLoadRequest](r)
	if err != nil {
		h.logger.Error("绑定批量更新请求失败", "error", err)
		http.Error(w, "绑定批量更新请求失败", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "ID列表不能为空", http.StatusBadRequest)
		return
	}

	count, err := h.storage.Skill().BatchUpdateAlwaysLoad(req.IDs, req.AlwaysLoad)
	if err != nil {
		h.logger.Error("批量更新始终加载状态失败", "error", err)
		http.Error(w, "批量更新始终加载状态失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]int64]{
		Code:    http.StatusOK,
		Message: "批量更新成功",
		Data:    map[string]int64{"updated": count},
	})
}

// ExportSkills 导出所有技能
// 将技能数据导出为JSON格式
func (h *SkillHandler) ExportSkills(w http.ResponseWriter, r *http.Request) {
	data, err := h.storage.Skill().ExportSkills()
	if err != nil {
		h.logger.Error("导出技能失败", "error", err)
		http.Error(w, "导出技能失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=skills_export.json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ImportSkillsRequest 导入技能请求
type ImportSkillsRequest struct {
	Data      string `json:"data"`       // Base64编码的JSON数据
	Overwrite bool   `json:"overwrite"`  // 是否覆盖已存在的技能
}

// ImportSkillsResponse 导入技能响应
type ImportSkillsResponse struct {
	Success int `json:"success"`
	Skip    int `json:"skip"`
}

// ImportSkills 导入技能
// 从JSON数据导入技能
func (h *SkillHandler) ImportSkills(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[ImportSkillsRequest](r)
	if err != nil {
		h.logger.Error("绑定导入请求失败", "error", err)
		http.Error(w, "绑定导入请求失败", http.StatusBadRequest)
		return
	}

	// 解析Base64数据
	data := []byte(req.Data)
	
	success, skip, err := h.storage.Skill().ImportSkills(data, req.Overwrite)
	if err != nil {
		h.logger.Error("导入技能失败", "error", err)
		http.Error(w, "导入技能失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[ImportSkillsResponse]{
		Code:    http.StatusOK,
		Message: "技能导入成功",
		Data:    ImportSkillsResponse{Success: success, Skip: skip},
	})
}

// GetByTag 根据标签获取技能
func (h *SkillHandler) GetByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	if tag == "" {
		http.Error(w, "标签不能为空", http.StatusBadRequest)
		return
	}

	skills, err := h.storage.Skill().GetByTag(tag)
	if err != nil {
		h.logger.Error("根据标签获取技能失败", "error", err, "tag", tag)
		http.Error(w, "获取技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能列表获取成功",
		Data:    skills,
	})
}

// GetTags 获取所有标签
// 返回系统中所有技能的标签列表
func (h *SkillHandler) GetTags(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().GetAll()
	if err != nil {
		h.logger.Error("获取技能列表失败", "error", err)
		http.Error(w, "获取标签失败", http.StatusInternalServerError)
		return
	}

	// 提取所有标签
	tagMap := make(map[string]bool)
	for _, skill := range skills {
		if skill.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(skill.Metadata), &metadata); err == nil {
				if tags, ok := metadata["tags"].([]interface{}); ok {
					for _, t := range tags {
						if tagStr, ok := t.(string); ok && tagStr != "" {
							tagMap[tagStr] = true
						}
					}
				}
			}
		}
	}

	// 转换为列表
	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	models.WriteData(w, models.BaseResponse[[]string]{
		Code:    http.StatusOK,
		Message: "标签列表获取成功",
		Data:    tags,
	})
}
