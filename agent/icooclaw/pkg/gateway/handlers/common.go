package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
)

type CommonHandler struct {
	logger *slog.Logger
}

func NewCommonHandler(logger *slog.Logger) *CommonHandler {
	return &CommonHandler{logger: logger}
}

// HealthCheck 健康检查
func (h *CommonHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	models.WriteData(w, models.BaseResponse[map[string]string]{
		Code:    http.StatusOK,
		Message: "OK",
		Data: map[string]string{
			"status": "healthy",
		},
	})
}
