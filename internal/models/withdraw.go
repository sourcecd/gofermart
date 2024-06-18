package models

type Withdraw struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type Withdrawals struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}
