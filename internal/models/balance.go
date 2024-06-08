package models

type Balance struct {
	//may be use float
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
