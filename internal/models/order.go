package models

type Order struct {
	Number     int     `json:"number"`
	Status     *string `json:"status,omitempty"`
	Accrual    *int    `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}
