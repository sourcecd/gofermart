package models

type Withdrawals struct {
	Order       int64  `json:"order"`
	Sum         int64  `json:"sum"`
	ProcessedAt string `json:"processed_at"`
}
