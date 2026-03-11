package models

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func WriteData[T any](w http.ResponseWriter, resp BaseResponse[T]) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Code)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(BaseResponse[T]{
			Code:    http.StatusInternalServerError,
			Message: "写入响应失败",
		})

		slog.Error("写入响应失败", "error", err)
	}
}
