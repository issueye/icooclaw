package models

import (
	"encoding/json"
	"net/http"
)

type BaseResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (r *BaseResponse[T]) Success(data T) {
	r.Code = http.StatusOK
	r.Message = "Success"
	r.Data = data
}

func (r *BaseResponse[T]) Error(code int, message string) {
	r.Code = code
	r.Message = message
}

func (r *BaseResponse[T]) BadRequest(message string) {
	r.Code = http.StatusBadRequest
	r.Message = message
}

func (r *BaseResponse[T]) NotFound(message string) {
	r.Code = http.StatusNotFound
	r.Message = message
}

func (r *BaseResponse[T]) InternalServerError(message string) {
	r.Code = http.StatusInternalServerError
	r.Message = message
}

func Bind[T any](r *http.Request) (T, error) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	return req, nil
}

// IDRequest 通用 ID 请求
type IDRequest struct {
	ID string `json:"id"`
}

// BindID 从请求体绑定 ID
func BindID(r *http.Request) (string, error) {
	var req IDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", err
	}
	return req.ID, nil
}
