package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/storage"
)

type SkillHandler struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewSkillHandler(logger *slog.Logger, storage *storage.Storage) *SkillHandler {
	return &SkillHandler{logger: logger, storage: storage}
}

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

func (h *SkillHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定保存技能请求失败", "error", err)
		http.Error(w, "绑定保存技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
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

func (h *SkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定创建技能请求失败", "error", err)
		http.Error(w, "绑定创建技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
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

func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定更新技能请求失败", "error", err)
		http.Error(w, "绑定更新技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
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

func (h *SkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除技能请求失败", "error", err)
		http.Error(w, "绑定删除技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().DeleteSkill(id)
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

func (h *SkillHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取技能请求失败", "error", err)
		http.Error(w, "绑定获取技能请求失败", http.StatusBadRequest)
		return
	}

	skill, err := h.storage.Skill().GetSkill(id)
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

func (h *SkillHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Name string `json:"name"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取技能请求失败", "error", err)
		http.Error(w, "绑定获取技能请求失败", http.StatusBadRequest)
		return
	}

	skill, err := h.storage.Skill().GetSkill(req.Name)
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

func (h *SkillHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().ListSkills()
	if err != nil {
		h.logger.Error("获取所有技能失败", "error", err)
		http.Error(w, "获取所有技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能列表获取成功",
		Data:    skills,
	})
}

func (h *SkillHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().ListEnabledSkills()
	if err != nil {
		h.logger.Error("获取启用技能失败", "error", err)
		http.Error(w, "获取启用技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Skill]{
		Code:    http.StatusOK,
		Message: "启用技能列表获取成功",
		Data:    skills,
	})
}

func (h *SkillHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定创建或更新技能请求失败", "error", err)
		http.Error(w, "绑定创建或更新技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
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