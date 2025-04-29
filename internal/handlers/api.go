package handlers

import (
	"context"

	"github.com/BorisRostovskiy/ESL/internal/service"
)

type UsersService interface {
	HealthCheck(ctx context.Context) error
	CreateUser(ctx context.Context, in *service.User) (*service.User, error)
	ListUsers(ctx context.Context, limit, offset int, filter *service.Filter) ([]service.User, error)
	UpdateUser(ctx context.Context, updated *service.User) error
	DeleteUser(ctx context.Context, id string) error
}
