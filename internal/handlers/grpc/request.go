package grpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/handlers"
	pb "github.com/BorisRostovskiy/ESL/internal/handlers/grpc/gen/user-manager"
	"github.com/BorisRostovskiy/ESL/internal/service"
)

type createUser struct {
	service.User
}

func (cu *createUser) Decode(from *pb.CreateUserRequest) error {
	cu.User.FirstName = from.GetFirstName()
	cu.User.LastName = from.GetLastName()
	cu.User.NickName = from.GetNickname()
	cu.User.Email = strings.ToLower(from.GetEmail())
	cu.User.Password = from.GetPassword()
	cu.User.Country = from.GetCountry()
	return cu.User.Validate(false)
}
func (cu *createUser) Encode() *pb.User {
	return user2PB(&cu.User)
}

// ListUsers
type listUsers struct {
	NextPage string
	Limit    int
	Offset   int
	Filter   *service.Filter
}

func (lu *listUsers) Decode(r *pb.ListUsersRequest) error {
	np, err := nextPage(r)
	if err != nil {
		return err
	}
	return lu.decode(np)
}
func (lu *listUsers) decode(np *handlers.NextPage) error {
	if np.FilterBy != "" {
		filter, err := service.NewFilter(np.FilterBy, np.Filter)
		if err != nil {
			return err
		}
		lu.Filter = filter
	}
	lu.Limit = np.Limit
	lu.Offset = np.Offset
	return nil
}
func (lu *listUsers) Encode(users []service.User) *pb.ListUsersResponse {
	r := &pb.ListUsersResponse{
		Users: make([]*pb.User, len(users)),
	}
	// if no users were found
	if len(users) == 0 {
		return r
	}
	for i := range users {
		r.Users[i] = user2PB(&users[i])
	}
	return r
}

type updateUser struct {
	service.User
}

func (uu *updateUser) Decode(r *pb.UpdateUserRequest) error {
	if r.FirstName != nil {
		uu.User.WithFirstName(r.GetFirstName())
	}
	if r.LastName != nil {
		uu.User.WithLastName(r.GetLastName())
	}
	if r.Nickname != nil {
		uu.User.WithNickName(r.GetNickname())
	}
	if r.Country != nil {
		uu.User.WithCountry(r.GetCountry())
	}

	if r.Password != nil {
		uu.User.WithPassword(r.GetPassword())
	}
	if r.Email != nil {
		uu.User.WithEmail(r.GetEmail())
	}

	if r.Id == "" {
		return fmt.Errorf("user id is mandatory")
	}
	uu.User.ID = r.Id
	return nil
}

func nextPage(r *pb.ListUsersRequest) (*handlers.NextPage, error) {
	return handlers.LoadNextPage(r.GetNextPage(), r.GetFilter(), r.GetFilterBy(), func() (int, error) {
		return int(r.GetPagination()), nil
	})
}

func user2PB(u *service.User) *pb.User {
	return &pb.User{
		Id:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Nickname:  u.NickName,
		Email:     u.Email,
		Country:   u.Country,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}
