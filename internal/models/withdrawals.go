package models

type Withdrawals struct {
	Order       int64   `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
