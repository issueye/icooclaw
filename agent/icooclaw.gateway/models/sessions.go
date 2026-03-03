package models

type QuerySession struct {
	KeyWord string         `json:"key_word"`
	Params  map[string]any `json:"params"`
}
