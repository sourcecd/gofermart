package models

type Balance struct {
	//may be use float
	Current   int64 `json:"current"`
	Withdrawn int64 `json:"withdrawn"`
}
