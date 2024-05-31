package models

type RegisterUser struct {
	Login string `json:"login" valid:"required"`
	Password string `json:"password" valid:"required"`
}
