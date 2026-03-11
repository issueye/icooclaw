package models

type Page struct {
	Size  int `json:"size"`
	Page  int `json:"page"`
	Total int `json:"total"`
}
