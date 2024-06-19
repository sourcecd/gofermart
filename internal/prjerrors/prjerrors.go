package prjerrors

import "errors"

var (
	ErrAlreadyExists           = errors.New("user already exists")
	ErrNotExists               = errors.New("user does not exists or wrong password")
	ErrOrderAlreadyExists      = errors.New("order already exists")
	ErrOtherOrderAlreadyExists = errors.New("order already exists another user")
	ErrEmptyData               = errors.New("no content")
	ErrNotEnough               = errors.New("not enought money")

	ErrAuthCredsNotFound = errors.New("auth creds not found")
	ErrReqJSONParse      = errors.New("request json parse failed")
	ErrValidateLogPass   = errors.New("validate login or password false (maybe empty)")
)
