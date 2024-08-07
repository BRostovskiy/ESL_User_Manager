package api

import (
	"errors"

	"github.com/BorisRostovskiy/ESL/internal/common"
)

var (
	ErrInternal          = &Error{Code: common.ErrCodeInternalError, Message: "api internal error"}
	ErrUserAlreadyExists = &Error{Code: common.ErrCodeUserAlreadyExists, Message: "user already exists"}
	ErrUserNotFound      = &Error{Code: common.ErrCodeUserNotFound, Message: "user not found"}
	ErrDuplicateKeyError = &Error{Code: common.ErrCodeConflict, Message: "duplicate key error"}
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
