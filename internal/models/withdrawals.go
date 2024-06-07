package models

import "time"

type Withdrawals struct {
	Order       int64     `json:"order"`
	Sum         int64     `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
