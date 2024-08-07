package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/api"
	"github.com/BorisRostovskiy/ESL/internal/clients"
	"github.com/BorisRostovskiy/ESL/internal/common"
	pb "github.com/BorisRostovskiy/ESL/internal/servers/grpc/gen/user-manager"
	"github.com/BorisRostovskiy/ESL/internal/storage"
	"github.com/BorisRostovskiy/ESL/internal/storage/models"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const (
	pwd              = "qwerty"
	id1              = "67cfa917-1cec-48ff-913c-243fe5749e92"
	id2              = "07cfa089-2rer-67yl-063h-273um7849e926"
	id3              = "49afc235-3lry-23rf-343h-223jq6849e926"
	email1           = "user_one@gmail.com"
	email2           = "user_two@gmail.com"
	email3           = "user_three@gmail.com"
	somethingHappens = "something happens"
	nextPageNoFilter = "eyJsaW1pdCI6Miwib2Zmc2V0IjoyLCJmaWx0ZXJfYnkiOiIiLCJmaWx0ZXIiOiIiLCJ0aW1lIjoiMjAzNS0wNy0yMVQxODo1ODoyOS42ODc0MzUrMDI6MDAifQo="
)

func stringPrt(s string) *string {
	return &s
}

var (
	createdAt, _ = time.Parse(time.RFC3339, "2022-07-20T12:45:44Z")
	user         = &models.User{
		FirstName: "User",
		LastName:  "One",
		NickName:  "userOne11",
		Email:     email1,
		Country:   "NL",
	}
)

func setupClient(repo *storage.MockUserRepo, notification *clients.MockChannelNotificator) (pb.UserManagerClient, func()) {
	lis := bufconn.Listen(1024 * 1024)

	logger := logrus.New()
	grpcSvc := New(api.New(repo, logger, notification), logger)

	baseServer := grpc.NewServer()
	pb.RegisterUserManagerServer(baseServer, grpcSvc)
	go func() {
		if err := baseServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	resolver.SetDefaultScheme("passthrough")
	conn, err := grpc.NewClient("", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Fatalf("dial: %v", err)
	}

	closer := func() {
		err = lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.GracefulStop()
	}

	client := pb.NewUserManagerClient(conn)

	return client, closer
}
func TestServer_CreateUser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	client, closer := setupClient(repo, notificationSvc)

	defer closer()
	type expectation struct {
		out *pb.User
		err error
	}

	tests := map[string]struct {
		in     *pb.CreateUserRequest
		want   expectation
		repo   func(r *storage.MockUserRepo)
		notify func(n *clients.MockChannelNotificator)
	}{
		"CreateUser Ok": {
			in: &pb.CreateUserRequest{
				Email:    email1,
				Password: pwd,
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(&models.User{Id: id1, Email: email1}, nil).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {
				n.EXPECT().Notify(gomock.Any(), clients.ChannelCreate,
					fmt.Sprintf("user with ID=%s has been created", id1))
			},
			want: expectation{
				out: &pb.User{
					Id:    id1,
					Email: email1,
				},
				err: nil,
			},
		},
		"No Email Error": {
			in: &pb.CreateUserRequest{
				Password: pwd,
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				out: nil,
				err: status.Error(codes.InvalidArgument, "failed to parse request: empty email"),
			},
		},
		"No Password Error": {
			in: &pb.CreateUserRequest{
				Email: email1,
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				out: nil,
				err: status.Error(codes.InvalidArgument, "failed to parse request: empty password"),
			},
		},
		"Duplicate key Error": {
			in: &pb.CreateUserRequest{
				Password: pwd,
				Email:    email1,
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil, storage.DuplicateKeyError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				out: nil,
				err: status.Error(codes.AlreadyExists, api.ErrUserAlreadyExists.Message),
			},
		},
		"Other repo problem Error": {
			in: &pb.CreateUserRequest{
				Password: pwd,
				Email:    email1,
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf(somethingHappens)).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				out: nil,
				err: status.Error(codes.Internal, api.ErrInternal.Message),
			},
		},
	}
	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			tt.repo(repo)
			tt.notify(notificationSvc)
			out, err := client.CreateUser(ctx, tt.in)

			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, out.Id, tt.want.out.Id)
			} else {
				assert.ErrorIs(t, err, tt.want.err)
			}
		})
	}
}

func TestServer_ListUsers(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	common.AllowedFilters.WithValues([]string{"country"})
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	client, closer := setupClient(repo, notificationSvc)

	defer closer()
	type expectation struct {
		out       *pb.ListUsersResponse
		paginated bool
		err       error
	}

	tests := map[string]struct {
		in   *pb.ListUsersRequest
		want expectation
		repo func(r *storage.MockUserRepo)
	}{
		"ListUsers no pagination Ok": {
			in: &pb.ListUsersRequest{},
			repo: func(r *storage.MockUserRepo) {
				f := map[string]string{"filter": "", "filterBy": ""}
				r.EXPECT().
					ListUsers(gomock.Any(), -1, 0, f).
					Return([]models.User{
						*user.WithId(id1).WithCreateAt(createdAt),
						*user.WithId(id2).WithEmail(email2).WithCreateAt(createdAt),
						*user.WithId(id3).WithEmail(email3).WithCreateAt(createdAt),
					}, nil).Times(1)
				r.EXPECT().CountUsers(gomock.Any(), f).Return(3, nil).Times(1)
			},
			want: expectation{
				err: nil,
				out: (&common.ListUsers{}).Encode([]models.User{
					*user.WithId(id1).WithCreateAt(createdAt),
					*user.WithId(id2).WithEmail(email2).WithCreateAt(createdAt),
					*user.WithId(id3).WithEmail(email3).WithCreateAt(createdAt),
				}),
			},
		},
		"ListUsers nextPage no filter Ok": {
			in: &pb.ListUsersRequest{
				NextPage: nextPageNoFilter,
			},
			repo: func(r *storage.MockUserRepo) {
				f := map[string]string{"filter": "", "filterBy": ""}
				r.EXPECT().
					ListUsers(gomock.Any(), 2, 2, f).
					Return([]models.User{
						*user.WithId(id3).WithEmail(email3).WithCreateAt(createdAt),
					}, nil).Times(1)
				r.EXPECT().CountUsers(gomock.Any(), f).Return(3, nil).Times(1)
			},
			want: expectation{
				err: nil,
				out: (&common.ListUsers{}).Encode([]models.User{
					*user.WithId(id3).WithEmail(email3).WithCreateAt(createdAt),
				}),
			},
		},
		"ListUsers pagination and filtering OK": {
			in: &pb.ListUsersRequest{
				Pagination: 1,
				FilterBy:   "country",
				Filter:     "NL",
			},
			repo: func(r *storage.MockUserRepo) {
				f := map[string]string{"filter": "NL", "filterBy": "country"}
				r.EXPECT().
					ListUsers(gomock.Any(), 1, 0, f).
					Return([]models.User{
						*user.WithId(id3).WithEmail(email3).WithCreateAt(createdAt),
					}, nil).Times(1)
				r.EXPECT().CountUsers(gomock.Any(), f).Return(2, nil).Times(1)
			},
			want: expectation{
				err:       nil,
				paginated: true,
				out: (&common.ListUsers{}).Encode([]models.User{
					*((&models.User{}).WithId(id3).WithEmail(email3).WithCreateAt(createdAt)),
				}),
			},
		},
		"ListUsers nextPage base64 decode Error": {
			in: &pb.ListUsersRequest{
				NextPage: "@",
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "could not decode next_page argument: illegal base64 data at input byte 0"),
				out: nil,
			},
		},
		"ListUsers nextPage json unmarshal Error": {
			in: &pb.ListUsersRequest{
				NextPage: "QAo=",
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "could not unmarshal limit offset: invalid character '@' looking for beginning of value"),
				out: nil,
			},
		},
		"ListUsers filterBy and filter Error": {
			in: &pb.ListUsersRequest{
				FilterBy: "FirstName",
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "parameters filter and filterBy should be used together"),
				out: nil,
			},
		},
		"ListUsers invalid filterBy Error": {
			in: &pb.ListUsersRequest{
				Filter:   "ololo",
				FilterBy: "FirstName",
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "filterBy parameter 'FirstName' not supported"),
				out: nil,
			},
		},
		"ListUsers repo Error": {
			in: &pb.ListUsersRequest{},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().ListUsers(gomock.Any(), -1, 0, map[string]string{"filter": "", "filterBy": ""}).
					Return(nil, fmt.Errorf(somethingHappens)).Times(1)
			},
			want: expectation{
				err: status.Error(codes.Internal, "internal servers error"),
				out: nil,
			},
		},
		"ListUsers count users Error": {
			in: &pb.ListUsersRequest{
				Pagination: 1,
			},
			repo: func(r *storage.MockUserRepo) {
				f := map[string]string{"filter": "", "filterBy": ""}
				r.EXPECT().
					ListUsers(gomock.Any(), 1, 0, f).
					Return([]models.User{
						*user.WithId(id3).WithEmail(email3).WithCreateAt(createdAt),
					}, nil).Times(1)
				r.EXPECT().CountUsers(gomock.Any(), f).Return(3, fmt.Errorf(somethingHappens)).Times(1)
			},
			want: expectation{
				err: status.Error(codes.Internal, "internal servers error"),
				out: nil,
			},
		},
		"ListUsers empty response": {
			in: &pb.ListUsersRequest{},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().ListUsers(gomock.Any(), -1, 0, map[string]string{"filter": "", "filterBy": ""}).
					Return([]models.User{}, nil).Times(1)
			},
			want: expectation{
				err: nil,
				out: &pb.ListUsersResponse{Users: []*pb.User{}},
			},
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			tt.repo(repo)
			out, err := client.ListUsers(ctx, tt.in)

			if tt.want.paginated {
				assert.NotEqual(t, out.NextPage, "")
			}
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Len(t, out.Users, len(tt.want.out.Users))
				for i, u := range out.Users {
					assert.Equal(t, u.Id, tt.want.out.Users[i].Id)
				}
			} else {
				assert.ErrorIs(t, err, tt.want.err)
			}
		})
	}
}

func TestServer_UpdateUser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	common.AllowedFilters.WithValues([]string{"country"})
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	client, closer := setupClient(repo, notificationSvc)

	defer closer()
	type expectation struct {
		err error
	}

	tests := map[string]struct {
		in     *pb.UpdateUserRequest
		want   expectation
		repo   func(r *storage.MockUserRepo)
		notify func(n *clients.MockChannelNotificator)
	}{
		"Update user OK": {
			in: &pb.UpdateUserRequest{
				Id:    id1,
				Email: stringPrt(email2),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(user.WithId(id1).WithEmail(email1), nil).Times(1)
				r.EXPECT().UpdateUser(gomock.Any(), user.WithId(id1).WithEmail(email2)).Return(nil).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {
				n.EXPECT().Notify(gomock.Any(), clients.ChannelUpdate, fmt.Sprintf("user with ID=%s has been updated", id1))
			},
			want: expectation{
				err: nil,
			},
		},
		"Update user structure is empty error": {
			in:     &pb.UpdateUserRequest{},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "empty request"),
			},
		},
		"Update user id empty error": {
			in: &pb.UpdateUserRequest{
				Email: stringPrt(email1),
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "user id is mandatory"),
			},
		},
		"Update no users found error": {
			in: &pb.UpdateUserRequest{
				Id:    id1,
				Email: stringPrt(email2),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(nil, storage.NoUsersFoundError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.NotFound, api.ErrUserNotFound.Error()),
			},
		},
		"Update duplicate key error": {
			in: &pb.UpdateUserRequest{
				Id:    id1,
				Email: stringPrt(email2),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(user.WithId(id1).WithEmail(email1), nil).Times(1)
				r.EXPECT().UpdateUser(gomock.Any(), user.WithId(id1).WithEmail(email2)).Return(storage.DuplicateKeyError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.AlreadyExists, api.ErrDuplicateKeyError.Error()),
			},
		},
		"Update repo something happens error": {
			in: &pb.UpdateUserRequest{
				Id:    id1,
				Email: stringPrt(email2),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(user.WithId(id1).WithEmail(email1), nil).Times(1)
				r.EXPECT().UpdateUser(gomock.Any(), user.WithId(id1).WithEmail(email2)).Return(fmt.Errorf(somethingHappens)).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.Internal, "internal servers error"),
			},
		},
	}
	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			tt.repo(repo)
			tt.notify(notificationSvc)
			_, err := client.UpdateUser(ctx, tt.in)
			if tt.want.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.want.err)
			}
		})
	}
}

func TestServer_DeleteUser(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	common.AllowedFilters.WithValues([]string{"country"})
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	client, closer := setupClient(repo, notificationSvc)

	defer closer()
	type expectation struct {
		err error
	}

	tests := map[string]struct {
		in     *pb.DeleteUserRequest
		want   expectation
		repo   func(r *storage.MockUserRepo)
		notify func(n *clients.MockChannelNotificator)
	}{
		"DeleteUser Ok": {
			in: &pb.DeleteUserRequest{
				Id: id1,
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().DeleteUser(gomock.Any(), id1).Return(nil).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {
				n.EXPECT().Notify(gomock.Any(), clients.ChannelDelete, fmt.Sprintf("user with ID=%s has been deleted", id1))
			},
			want: expectation{
				err: nil,
			},
		},
		"Delete user no id error": {
			in: &pb.DeleteUserRequest{
				Id: "",
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.InvalidArgument, "id is mandatory"),
			},
		},
		"Delete user not found error": {
			in: &pb.DeleteUserRequest{
				Id: id1,
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().DeleteUser(gomock.Any(), id1).Return(storage.NoUsersFoundError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.NotFound, api.ErrUserNotFound.Message),
			},
		},
		"Delete user repo error": {
			in: &pb.DeleteUserRequest{
				Id: id1,
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().DeleteUser(gomock.Any(), id1).Return(fmt.Errorf(somethingHappens)).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				err: status.Error(codes.Internal, "internal servers error"),
			},
		},
	}
	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			tt.repo(repo)
			tt.notify(notificationSvc)
			_, err := client.DeleteUser(ctx, tt.in)
			if tt.want.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.want.err)
			}
		})
	}
}
