package models

import (
	"testing"
	"time"

	//userManager "github.com/RostovskiyBoris/ESL/internal/gen/go/proto/user-manager/v1"
	pb "github.com/BorisRostovskiy/ESL/internal/servers/grpc/gen/user-manager"
	"github.com/BorisRostovskiy/ESL/internal/storage"

	//userManager "github.com/BorisRostovskiy/ESL/internal/servers/grpc/proto/user-manager/v1"
	"github.com/stretchr/testify/assert"
)

const (
	id1 = "67cfa917-1cec-48ff-913c-243fe5749e92"
	id2 = "07cfa089-2rer-67yl-063h-273um7849e926"
)

var (
	createdAt, _ = time.Parse(time.RFC3339, "2022-07-20T12:45:44Z")
	user         = &storage.User{
		FirstName: "User",
		LastName:  "One",
		NickName:  "userOne11",
		Email:     "abc@gmail.com",
		Country:   "NL",
	}
)

func TestNewUserFromRequest(t *testing.T) {
	type args struct {
		in *pb.CreateUserRequest
	}
	tests := []struct {
		name string
		args args
		want *storage.User
		err  error
	}{
		{
			name: "OK",
			args: args{in: &pb.CreateUserRequest{
				User: storage.User2Proto(user.WithPassword("qwerty")),
			}},
			want: user.WithPassword("qwerty"),
		},
		{
			name: "empty first name error",
			args: args{in: &pb.CreateUserRequest{
				User: storage.User2Proto(&storage.User{}),
			}},
			err: storage.EmptyFirstNameError,
		},
		{
			name: "empty last name error",
			args: args{in: &pb.CreateUserRequest{
				User: storage.User2Proto(&storage.User{FirstName: "Name"}),
			}},
			err: storage.EmptyLastNameError,
		},
		{
			name: "empty password error",
			args: args{in: &pb.CreateUserRequest{
				User: storage.User2Proto(&storage.User{FirstName: "Name", LastName: "LastName"}),
			}},
			err: storage.EmptyPasswordError,
		},
		{
			name: "empty email error",
			args: args{in: &pb.CreateUserRequest{
				User: storage.User2Proto(storage.User{FirstName: "Name", LastName: "LastName"}.WithPassword("qwerty")),
			}},
			err: storage.EmptyEmailError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := storage.NewUserFromRequest(tc.args.in)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, got, tc.want)
			} else {
				assert.Equal(t, err, tc.err)
			}
		})
	}
}

func TestUser2UpdateFromRequest(t *testing.T) {
	type args struct {
		in *pb.UpdateUserRequest
	}
	tests := []struct {
		name string
		args args
		want *storage.User
		err  error
	}{
		{
			name: "OK",
			args: args{in: &pb.UpdateUserRequest{
				Id:   id1,
				User: storage.User2Proto(user.WithId(id1)),
			}},
			want: user.WithId(id1),
		},
		{
			name: "empty id error",
			args: args{in: &pb.UpdateUserRequest{
				Id:   "",
				User: storage.User2Proto(user.WithId(id1)),
			}},
			err: storage.EmptyIdError,
		},
		{
			name: "cheater error",
			args: args{in: &pb.UpdateUserRequest{
				Id:   id1,
				User: storage.User2Proto(user.WithId(id2)),
			}},
			err: storage.CheaterError,
		},
		{
			name: "empty first name error",
			args: args{in: &pb.UpdateUserRequest{
				Id:   id1,
				User: storage.User2Proto(&storage.User{Id: id1}),
			}},
			err: storage.EmptyFirstNameError,
		},
		{
			name: "empty last name error",
			args: args{in: &pb.UpdateUserRequest{
				Id:   id1,
				User: storage.User2Proto(&storage.User{Id: id1, FirstName: "Name"}),
			}},
			err: storage.EmptyLastNameError,
		},
		{
			name: "empty email error",
			args: args{in: &pb.UpdateUserRequest{
				Id:   id1,
				User: storage.User2Proto(&storage.User{Id: id1, FirstName: "Name", LastName: "LastName"}),
			}},
			err: storage.EmptyEmailError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := storage.Proto2User(tc.args.in)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, got, tc.want)
			} else {
				assert.Equal(t, err, tc.err)
			}
		})
	}
}

func TestUser_WithCreateAt(t *testing.T) {
	type args struct {
		c time.Time
	}
	tests := []struct {
		name string
		args args
		want *storage.User
	}{
		{
			name: "OK",
			args: args{createdAt},
			want: &storage.User{CreatedAt: createdAt},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := storage.User{}
			got := u.WithCreateAt(tt.args.c)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUser_WithEmail(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name string
		args args
		want *storage.User
	}{
		{
			name: "OK",
			args: args{email: "abc@gmail.com"},
			want: &storage.User{Email: "abc@gmail.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := storage.User{}
			got := u.WithEmail(tt.args.email)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUser_WithId(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want *storage.User
	}{
		{
			name: "OK",
			args: args{id: id1},
			want: &storage.User{Id: id1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := storage.User{}
			got := u.WithId(tt.args.id)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestUser_WithPassword(t *testing.T) {
	type args struct {
		pwd string
	}
	tests := []struct {
		name string
		args args
		want *storage.User
	}{
		{
			name: "OK",
			args: args{pwd: "qwerty"},
			want: &storage.User{Password: "qwerty"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := storage.User{}
			got := u.WithPassword(tt.args.pwd)
			assert.Equal(t, got, tt.want)
		})
	}
}
