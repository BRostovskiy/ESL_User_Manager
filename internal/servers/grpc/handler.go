package grpc

import (
	"context"
	"fmt"

	"github.com/BorisRostovskiy/ESL/internal/api"
	"github.com/BorisRostovskiy/ESL/internal/common"
	pb "github.com/BorisRostovskiy/ESL/internal/servers/grpc/gen/user-manager"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserManagerHandler struct {
	api api.UsersAPI
	log *logrus.Logger
	pb.UnimplementedUserManagerServer
}

func New(api api.UsersAPI, log *logrus.Logger) UserManagerHandler {
	return UserManagerHandler{api: api, log: log}
}

func (ums UserManagerHandler) CreateUser(ctx context.Context, r *pb.CreateUserRequest) (*pb.User, error) {
	cu := &common.CreateUser{}
	if err := cu.DecodePB(r); err != nil {
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
	cu.Id = user.Id
	cu.CreatedAt = user.CreatedAt
	cu.UpdatedAt = user.UpdatedAt
	return cu.Encode(), nil
}

func (ums UserManagerHandler) ListUsers(ctx context.Context, r *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	lu := &common.ListUsers{}
	if err := lu.DecodePB(r); err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("list users decode error: %v", err)
		return nil, errRequest(ctx, err)
	}

	filters := map[string]string{"filter": lu.Filter, "filterBy": lu.FilterBy}
	users, err := ums.api.ListUsers(ctx, lu.Limit, lu.Offset, filters)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform list users: %v", err)
		return nil, errApif(ctx, "could not list users: %w", err)
	}

	resp := lu.Encode(users)
	if len(resp.Users) == 0 {
		return resp, nil
	}

	numUsers, err := ums.api.CountUsers(ctx, filters)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform count users: %v", err)
		return nil, errApif(ctx, "could not get users counted: %w", err)
	}

	np, err := common.GenerateNextPage(lu.Limit, lu.Offset, numUsers, lu.Filter, lu.FilterBy)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("could not marshal next page: %v", err)
		return nil, errRequestf(ctx, "could not marshal next page structure: %w", err)
	}
	resp.NextPage = np

	return resp, nil
}

func (ums UserManagerHandler) UpdateUser(ctx context.Context, r *pb.UpdateUserRequest) (*emptypb.Empty, error) {
	uu := &common.UpdateUser{}
	if err := uu.DecodePB(r); err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("update user decode error: %v", err)
		return nil, errRequest(ctx, err)
	}

	err := ums.api.UpdateUser(ctx, uu.User.Id, uu.UpdatedFields)
	if err != nil {
		ums.log.WithField("component", "grpc_handler").
			Debugf("failed to perform update user: %v", err)
		return nil, errApi(ctx, err)
	}
	return &emptypb.Empty{}, nil
}

func (ums UserManagerHandler) DeleteUser(ctx context.Context, r *pb.DeleteUserRequest) (*emptypb.Empty, error) {
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
