package http

import (
	"errors"
	"net/http"

	"github.com/BorisRostovskiy/ESL/internal/log"
	"github.com/BorisRostovskiy/ESL/internal/service"
)

var (
	apiErrorCodeStatus = map[int]int{
		service.ErrCodeBadRequest:        http.StatusBadRequest,
		service.ErrCodeInternalError:     http.StatusInternalServerError,
		service.ErrCodeUserAlreadyExists: http.StatusConflict,
		service.ErrCodeUserNotFound:      http.StatusNotFound,
		service.ErrCodeConflict:          http.StatusConflict,
		service.ErrCodeEmptyUpdate:       http.StatusBadRequest,
	}
	ErrInternal = &Error{
		Status:  http.StatusInternalServerError,
		Code:    service.ErrCodeInternalError,
		Message: "internal handlers error",
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
		e.Code = service.ErrCodeInternalError
		e.Status = http.StatusInternalServerError
	}
	return responseObject(w, e.Status, e)
}

func ToError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// make response from error
func errResponse(status, code int, err error) response {
	if rErr := ToError(err); rErr != nil {
		return rErr
	}
	return &Error{Status: status, Code: code, Message: err.Error()}
}

// parse errors from parsing requests
func errRequest(r *http.Request, err error) response {
	log.WithError(r, err)
	return errResponse(http.StatusBadRequest, service.ErrCodeBadRequest, err)
}
func errRequestf(r *http.Request, format string, args ...interface{}) response {
	err := log.WithErrorf(r, format, args...)
	return errResponse(http.StatusBadRequest, service.ErrCodeBadRequest, err)
}

// parse errors from service requests
func errApi(r *http.Request, format string, args ...interface{}) response {
	err := log.WithErrorf(r, format, args...)
	if e := service.ToError(errors.Unwrap(err)); e != nil {
		if status, ok := apiErrorCodeStatus[e.Code]; ok {
			return errResponse(status, e.Code, e)
		}
	}
	return ErrInternal
}
