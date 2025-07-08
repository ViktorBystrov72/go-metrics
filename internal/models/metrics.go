package models

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // gauge или counter
	Delta *int64   `json:"delta,omitempty"` // для counter
	Value *float64 `json:"value,omitempty"` // для gauge
	Hash  string   `json:"hash,omitempty"`  // хеш для проверки целостности
}
