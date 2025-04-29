package service

import (
	"errors"
)

const (
	ErrCodeInternalError = 100
	ErrCodeBadRequest    = 101
	ErrCodeConflict      = 102
	ErrCodeEmptyUpdate   = 103

	ErrCodeUserNotFound = 200

	ErrCodeUserAlreadyExists = 300
)

var (
	ErrInternal           = &Error{Code: ErrCodeInternalError, Message: "service internal error"}
	ErrUserAlreadyExists  = &Error{Code: ErrCodeUserAlreadyExists, Message: "user already exists"}
	ErrUserNotFound       = &Error{Code: ErrCodeUserNotFound, Message: "user not found"}
	ErrDuplicateKeyError  = &Error{Code: ErrCodeConflict, Message: "duplicate key error"}
	ErrEmptyUpdateRequest = &Error{Code: ErrCodeEmptyUpdate, Message: "empty request"}
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (res *Error) Error() string {
	return res.Message
}

func ToError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
