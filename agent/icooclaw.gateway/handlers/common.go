package handlers

import (
	"log/slog"
	"net/http"
)

type CommonHandler struct {
	logger *slog.Logger
}

func NewCommonHandler(logger *slog.Logger) *CommonHandler {
	return &CommonHandler{logger: logger}
}

func (h *CommonHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
