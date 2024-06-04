package models

type Balance struct {
	//may be use float
	Current   int `json:"current"`
	Withdrawn int `json:"withdrawn"`
}
