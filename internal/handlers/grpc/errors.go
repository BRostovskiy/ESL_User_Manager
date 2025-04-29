package grpc

import (
	"context"
	"errors"
	"net/http"

	"github.com/BorisRostovskiy/ESL/internal/log"
	"github.com/BorisRostovskiy/ESL/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var apiErrorCodeStatus = map[int]codes.Code{
	service.ErrCodeBadRequest:        codes.InvalidArgument,
	service.ErrCodeInternalError:     codes.Internal,
	service.ErrCodeUserAlreadyExists: codes.AlreadyExists,
	service.ErrCodeUserNotFound:      codes.NotFound,
	service.ErrCodeConflict:          codes.AlreadyExists,
	service.ErrCodeEmptyUpdate:       codes.InvalidArgument,
}

var (
	ErrInternal = status.Error(codes.Internal, "internal handlers error")
)

// parse errors from parsing requests
func errRequest(ctx context.Context, err error) error {
	r := &http.Request{}
	log.WithError(r.WithContext(ctx), err)
	return status.Error(codes.InvalidArgument, err.Error())
}
func errRequestf(ctx context.Context, format string, args ...interface{}) error {
	r := &http.Request{}
	err := log.WithErrorf(r.WithContext(ctx), format, args...)
	return status.Error(codes.InvalidArgument, err.Error())
}

// parse errors from service requests
func errApi(ctx context.Context, err error) error {
	r := &http.Request{}
	log.WithError(r.WithContext(ctx), err)
	if e := service.ToError(err); e != nil {
		if code, ok := apiErrorCodeStatus[e.Code]; ok {
			return status.Error(code, err.Error())
		}
	}
	return ErrInternal
}

func errApif(ctx context.Context, format string, args ...interface{}) error {
	r := &http.Request{}
	err := log.WithErrorf(r.WithContext(ctx), format, args...)
	if e := service.ToError(errors.Unwrap(err)); e != nil {
		if code, ok := apiErrorCodeStatus[e.Code]; ok {
			return status.Error(code, err.Error())
		}
	}
	return ErrInternal
}
