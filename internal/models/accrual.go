package models

type Accrual struct {
	Order   int64  `json:"order"`
	Status  string `json:"status"`
	Accrual *int64 `json:"accrual,omitempty"`
}
