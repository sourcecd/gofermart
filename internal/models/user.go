package models

type User struct {
	Login    string `json:"login" valid:"required"`
	Password string `json:"password" valid:"required"`
}
