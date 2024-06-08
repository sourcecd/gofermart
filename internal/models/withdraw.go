package models

type Withdraw struct {
	Order int64   `json:"order"`
	Sum   float64 `json:"sum"`
}
