package models

type Accrual struct {
	Order   string  `json:"order"`
	Status  string `json:"status"`
	Accrual *int64 `json:"accrual,omitempty"`
}
