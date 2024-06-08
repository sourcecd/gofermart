package models

type Withdrawals struct {
	Order       string   `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
