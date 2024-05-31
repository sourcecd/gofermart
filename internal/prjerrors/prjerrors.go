package prjerrors

import "errors"

var (
	ErrAlreadyExists = errors.New("login already exists")
)