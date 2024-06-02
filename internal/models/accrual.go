package models

type Accrual struct {
	Order   int    `json:"order"`
	Status  string `json:"status"`
	Accrual *int   `json:"accrual,omitempty"`
}
