package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleHealth 健康检查
func (g *RESTGateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetSessions 获取会话列表
func (g *RESTGateway) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	if g.storage == nil {
		http.Error(w, "Storage not configured", http.StatusInternalServerError)
		return
	}

	userID := r.URL.Query().Get("user_id")
	channel := r.URL.Query().Get("channel")

	sessions, err := g.storage.GetSessions(userID, channel)
	if err != nil {
		g.logger.Error("Failed to get sessions", "error", err)
		http.Error(w, "Failed to get sessions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleGetSessionMessages 获取会话消息
func (g *RESTGateway) handleGetSessionMessages(w http.ResponseWriter, r *http.Request) {
	if g.storage == nil {
		http.Error(w, "Storage not configured", http.StatusInternalServerError)
		return
	}

	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := g.storage.GetSessionMessages(uint(sessionID), limit)
	if err != nil {
		g.logger.Error("Failed to get session messages", "error", err)
		http.Error(w, "Failed to get session messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleDeleteSession 删除会话
func (g *RESTGateway) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if g.storage == nil {
		http.Error(w, "Storage not configured", http.StatusInternalServerError)
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	if err := g.storage.DeleteSession(sessionID); err != nil {
		g.logger.Error("Failed to delete session", "error", err)
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleGetProviders 获取 Provider 信息
func (g *RESTGateway) handleGetProviders(w http.ResponseWriter, r *http.Request) {
	if g.agent == nil {
		http.Error(w, "Agent not configured", http.StatusInternalServerError)
		return
	}

	provider := g.agent.GetProvider()
	if provider == nil {
		http.Error(w, "Provider not configured", http.StatusInternalServerError)
		return
	}

	info := map[string]interface{}{
		"provider":      provider.GetName(),
		"default_model": provider.GetDefaultModel(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleGetSkills 获取技能列表
func (g *RESTGateway) handleGetSkills(w http.ResponseWriter, r *http.Request) {
	if g.skills == nil {
		http.Error(w, "Skills not configured", http.StatusInternalServerError)
		return
	}

	skills, err := g.skills.GetAllSkills()
	if err != nil {
		g.logger.Error("Failed to get skills", "error", err)
		http.Error(w, "Failed to get skills", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skills)
}

// handleGetSkill 获取单个技能
func (g *RESTGateway) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	if g.skills == nil {
		http.Error(w, "Skills not configured", http.StatusInternalServerError)
		return
	}

	name := chi.URLParam(r, "id")
	if name == "" {
		http.Error(w, "Skill name required", http.StatusBadRequest)
		return
	}

	skill, err := g.skills.GetSkillByName(name)
	if err != nil {
		g.logger.Error("Failed to get skill", "error", err)
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skill)
}

func (c *RESTGateway) handleRestChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
		ChatID  string `json:"chat_id"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if c.agent == nil {
		http.Error(w, "Agent not configured", http.StatusInternalServerError)
		return
	}

	resp, err := c.agent.ProcessMessage(context.Background(), req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": resp})
}

func (c *RESTGateway) handleRestChatStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var req struct {
		Content string `json:"content"`
		ChatID  string `json:"chat_id"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	_ = flusher
}
