package prjerrors

import "errors"

var (
	ErrAlreadyExists           = errors.New("user already exists")
	ErrNotExists               = errors.New("user does not exists or wrong password")
	ErrOrderAlreadyExists      = errors.New("order already exists")
	ErrOtherOrderAlreadyExists = errors.New("order already exists another user")
)
