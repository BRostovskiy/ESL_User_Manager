package http

import (
	"errors"
	"net/http"

	"github.com/BorisRostovskiy/ESL/internal/api"
	"github.com/BorisRostovskiy/ESL/internal/common"
	"github.com/BorisRostovskiy/ESL/internal/log"
)

var apiErrorCodeStatus = map[int]int{
	common.ErrCodeBadRequest:        http.StatusBadRequest,
	common.ErrCodeInternalError:     http.StatusInternalServerError,
	common.ErrCodeUserAlreadyExists: http.StatusConflict,
	common.ErrCodeUserNotFound:      http.StatusNotFound,
	common.ErrCodeConflict:          http.StatusConflict,
}

var (
	ErrInternal = &Error{
		Status:  http.StatusInternalServerError,
		Code:    common.ErrCodeInternalError,
		Message: "internal servers error",
	}
)

type Error struct {
	Status  int    `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) WriteTo(w http.ResponseWriter) error {
	if e.Status <= 0 {
		e.Code = common.ErrCodeInternalError
		e.Status = http.StatusInternalServerError
	}
	return common.RespondObject(w, e.Status, e)
}

func ToError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// make response from error
func errResponse(status, code int, err error) common.Response {
	if rErr := ToError(err); rErr != nil {
		return rErr
	}
	return &Error{Status: status, Code: code, Message: err.Error()}
}

// parse errors from parsing requests
func errRequest(r *http.Request, err error) common.Response {
	log.WithError(r, err)
	return errResponse(http.StatusBadRequest, common.ErrCodeBadRequest, err)
}
func errRequestf(r *http.Request, format string, args ...interface{}) common.Response {
	err := log.WithErrorf(r, format, args...)
	return errResponse(http.StatusBadRequest, common.ErrCodeBadRequest, err)
}

// parse errors from service requests
func errApi(r *http.Request, format string, args ...interface{}) common.Response {
	err := log.WithErrorf(r, format, args...)
	if e := api.ToError(errors.Unwrap(err)); e != nil {
		if status, ok := apiErrorCodeStatus[e.Code]; ok {
			return errResponse(status, e.Code, e)
		}
	}
	return ErrInternal
}
