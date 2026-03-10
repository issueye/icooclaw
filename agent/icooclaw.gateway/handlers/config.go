package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/spf13/viper"
	"icooclaw.core/config"
	"icooclaw.gateway/models"
)

type ConfigHandler struct {
	logger *slog.Logger
}

func NewConfigHandler(logger *slog.Logger) *ConfigHandler {
	return &ConfigHandler{logger: logger}
}

// GetConfig 获取当前配置
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		h.logger.Error("加载配置失败", "error", err)
		http.Error(w, "加载配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*config.Config]{
		Code:    http.StatusOK,
		Message: "获取配置成功",
		Data:    cfg,
	})
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Config map[string]interface{} `json:"config"`
}

// UpdateConfig 更新配置（覆盖配置文件）
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*UpdateConfigRequest](r)
	if err != nil {
		h.logger.Error("绑定更新配置请求失败", "error", err)
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	if req.Config == nil {
		http.Error(w, "配置内容不能为空", http.StatusBadRequest)
		return
	}

	// 遍历配置项并设置
	for key, value := range req.Config {
		viper.Set(key, value)
	}

	// 写入配置文件
	if err := writeConfigFile(); err != nil {
		h.logger.Error("保存配置文件失败", "error", err)
		http.Error(w, "保存配置文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("配置已更新")

	// 重新加载配置返回
	cfg, err := config.Load()
	if err != nil {
		h.logger.Error("重新加载配置失败", "error", err)
		http.Error(w, "重新加载配置失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*config.Config]{
		Code:    http.StatusOK,
		Message: "配置更新成功",
		Data:    cfg,
	})
}

// OverwriteConfigRequest 覆盖配置文件请求
type OverwriteConfigRequest struct {
	Content string `json:"content"` // TOML 配置文件内容
}

// OverwriteConfig 覆盖配置文件（完整替换配置文件内容）
func (h *ConfigHandler) OverwriteConfig(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*OverwriteConfigRequest](r)
	if err != nil {
		h.logger.Error("绑定覆盖配置请求失败", "error", err)
		http.Error(w, "无效的请求参数", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "配置内容不能为空", http.StatusBadRequest)
		return
	}

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// 如果没有配置文件，使用默认路径
		configFile = "config.toml"
	}

	// 备份原配置文件
	backupFile := configFile + ".bak"
	originalContent, err := os.ReadFile(configFile)
	if err == nil {
		// 原文件存在，创建备份
		if err := os.WriteFile(backupFile, originalContent, 0644); err != nil {
			h.logger.Warn("备份配置文件失败", "error", err)
		}
	}

	// 写入新配置
	if err := os.WriteFile(configFile, []byte(req.Content), 0644); err != nil {
		h.logger.Error("写入配置文件失败", "error", err)
		http.Error(w, "写入配置文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 重新读取配置验证是否有效
	viper.Reset()
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		// 配置无效，恢复备份
		h.logger.Error("新配置无效，恢复备份", "error", err)
		if backupData, readErr := os.ReadFile(backupFile); readErr == nil {
			_ = os.WriteFile(configFile, backupData, 0644)
		}
		http.Error(w, "配置文件格式无效: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 重新加载配置
	cfg, err := config.Load()
	if err != nil {
		h.logger.Error("重新加载配置失败", "error", err)
		http.Error(w, "重新加载配置失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("配置文件已覆盖", "file", configFile)

	models.WriteData(w, models.BaseResponse[*config.Config]{
		Code:    http.StatusOK,
		Message: "配置文件覆盖成功",
		Data:    cfg,
	})
}

// GetConfigFileContent 获取配置文件原始内容
func (h *ConfigHandler) GetConfigFileContent(w http.ResponseWriter, r *http.Request) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		models.WriteData(w, models.BaseResponse[map[string]string]{
			Code:    http.StatusNotFound,
			Message: "未找到配置文件",
			Data: map[string]string{
				"content": "",
				"path":    "",
			},
		})
		return
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		h.logger.Error("读取配置文件失败", "error", err)
		http.Error(w, "读取配置文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]string]{
		Code:    http.StatusOK,
		Message: "获取配置文件成功",
		Data: map[string]string{
			"content": string(content),
			"path":    configFile,
		},
	})
}

// writeConfigFile 写入配置文件
func writeConfigFile() error {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// 如果没有配置文件，创建默认的
		configFile = "config.toml"
		viper.SetConfigFile(configFile)
	}

	// 检查文件是否存在，不存在则创建
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 创建目录
		// 文件不存在，使用 SafeWriteConfig 创建
		return viper.SafeWriteConfigAs(configFile)
	}

	// 文件存在，使用 WriteConfig 覆盖
	return viper.WriteConfigAs(configFile)
}

// GetConfigJSON 获取配置的 JSON 格式（用于前端展示）
func (h *ConfigHandler) GetConfigJSON(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		h.logger.Error("加载配置失败", "error", err)
		http.Error(w, "加载配置失败", http.StatusInternalServerError)
		return
	}

	// 将配置转换为 JSON
	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		h.logger.Error("序列化配置失败", "error", err)
		http.Error(w, "序列化配置失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}