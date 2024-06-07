package models

type Withdraw struct {
	Order int64 `json:"order"`
	Sum   int64 `json:"sum"`
}
