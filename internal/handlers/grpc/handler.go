package grpc

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/BorisRostovskiy/ESL/internal/handlers"
	pb "github.com/BorisRostovskiy/ESL/internal/handlers/grpc/gen/user-manager"
)

type UserManagerServer struct {
	api handlers.UsersService
	log *logrus.Logger
	pb.UnimplementedUserManagerServer
}

func New(api handlers.UsersService, log *logrus.Logger) UserManagerServer {
	return UserManagerServer{api: api, log: log}
}

func (ums UserManagerServer) CreateUser(ctx context.Context, r *pb.CreateUserRequest) (*pb.User, error) {
	cu := &createUser{}
	if err := cu.Decode(r); err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("create user decode error: %v", err)
		return nil, errRequestf(ctx, "failed to parse request: %w", err)
	}
	user, err := ums.api.CreateUser(ctx, &cu.User)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform user creation: %v", err)
		return nil, errApi(ctx, err)
	}
	cu.User.ID = user.ID
	cu.User.CreatedAt = user.CreatedAt
	cu.User.UpdatedAt = user.UpdatedAt
	return cu.Encode(), nil
}

func (ums UserManagerServer) ListUsers(ctx context.Context, r *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	lu := &listUsers{}
	if err := lu.Decode(r); err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("list users decode error: %v", err)
		return nil, errRequest(ctx, err)
	}

	users, err := ums.api.ListUsers(ctx, lu.Limit, lu.Offset, lu.Filter)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform list users: %v", err)
		return nil, errApif(ctx, "could not list users: %w", err)
	}

	resp := lu.Encode(users)
	if len(resp.Users) == 0 {
		return resp, nil
	}

	np, err := handlers.GenerateNextPage(lu.Limit, lu.Offset, len(users), lu.Filter)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("could not marshal next page: %v", err)
		return nil, errRequestf(ctx, "could not marshal next page structure: %w", err)
	}
	resp.NextPage = &np

	return resp, nil
}

func (ums UserManagerServer) UpdateUser(ctx context.Context, r *pb.UpdateUserRequest) (*emptypb.Empty, error) {
	uu := &updateUser{}
	if err := uu.Decode(r); err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("update user decode error: %v", err)
		return nil, errRequest(ctx, err)
	}

	err := ums.api.UpdateUser(ctx, &uu.User)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform update user: %v", err)
		return nil, errApi(ctx, err)
	}
	return &emptypb.Empty{}, nil
}

func (ums UserManagerServer) DeleteUser(ctx context.Context, r *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	if r.GetId() == "" {
		return nil, errRequest(ctx, fmt.Errorf("id is mandatory"))
	}

	if err := ums.api.DeleteUser(ctx, r.GetId()); err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform delete user: %v", err)
		return nil, errApi(ctx, err)
	}
	return &emptypb.Empty{}, nil
}
