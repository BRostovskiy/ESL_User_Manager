package grpc

import (
	"context"
	"errors"
	"net/http"

	"github.com/BorisRostovskiy/ESL/internal/api"
	"github.com/BorisRostovskiy/ESL/internal/common"
	"github.com/BorisRostovskiy/ESL/internal/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var apiErrorCodeStatus = map[int]codes.Code{
	common.ErrCodeBadRequest:        codes.InvalidArgument,
	common.ErrCodeInternalError:     codes.Internal,
	common.ErrCodeUserAlreadyExists: codes.AlreadyExists,
	common.ErrCodeUserNotFound:      codes.NotFound,
	common.ErrCodeConflict:          codes.AlreadyExists,
}

var (
	ErrInternal = status.Error(codes.Internal, "internal servers error")
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
	if e := api.ToError(err); e != nil {
		if code, ok := apiErrorCodeStatus[e.Code]; ok {
			return status.Error(code, err.Error())
		}
	}
	return ErrInternal
}

func errApif(ctx context.Context, format string, args ...interface{}) error {
	r := &http.Request{}
	err := log.WithErrorf(r.WithContext(ctx), format, args...)
	if e := api.ToError(errors.Unwrap(err)); e != nil {
		if code, ok := apiErrorCodeStatus[e.Code]; ok {
			return status.Error(code, err.Error())
		}
	}
	return ErrInternal
}
