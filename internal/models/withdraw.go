package models

type Withdraw struct {
	Order int `json:"order"`
	Sum   int `json:"sum"`
}
