package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	pb "github.com/BorisRostovskiy/ESL/internal/servers/grpc/gen/user-manager"
	"github.com/BorisRostovskiy/ESL/internal/storage/models"

	"github.com/go-chi/chi/v5"
)

type CreateUser struct {
	models.User
}

func (cu *CreateUser) Decode(r *http.Request) error {
	if err := json.NewDecoder(r.Body).Decode(cu); err != nil {
		return fmt.Errorf("malformed data: %w", err)
	}

	if err := cu.Validate(); err != nil {
		return fmt.Errorf("invalid user: %w", err)
	}
	cu.Email = strings.ToLower(cu.Email)
	return nil
}
func (cu *CreateUser) DecodePB(r *pb.CreateUserRequest) error {
	cu.FirstName = r.GetFirstName()
	cu.LastName = r.GetLastName()
	cu.NickName = r.GetNickname()
	cu.Email = strings.ToLower(r.GetEmail())
	cu.Password = r.GetPassword()
	cu.Country = r.GetCountry()
	return cu.Validate()
}
func (cu *CreateUser) Encode() *pb.User {
	return user2PB(&cu.User)
}
func (cu *CreateUser) WriteTo(w http.ResponseWriter) error {
	return RespondObject(w, http.StatusCreated, cu)
}

// ListUsers
type ListUsers struct {
	Users    []models.User `json:"users"`
	NextPage string        `json:"next_page,omitempty"`
	Limit    int           `json:"-"`
	Offset   int           `json:"-"`
	FilterBy string        `json:"-"`
	Filter   string        `json:"-"`
}

func (lu *ListUsers) Decode(r *http.Request) error {
	np, err := NextPageFromHTTP(r)
	if err != nil {
		return fmt.Errorf("could not load nextPage: %v", err)
	}
	return lu.decode(np)
}
func (lu *ListUsers) DecodePB(r *pb.ListUsersRequest) error {
	np, err := NextPageFromPB(r)
	if err != nil {
		return err
	}
	return lu.decode(np)
}
func (lu *ListUsers) decode(np *NextPage) error {
	if np.FilterBy != "" {
		if _, ok := AllowedFilters[np.FilterBy]; !ok {
			return fmt.Errorf("filterBy parameter '%v' not supported", np.FilterBy)
		}

		if np.Filter == "" {
			return fmt.Errorf("filter parameter should not been empty")
		}
		lu.Filter = np.Filter
		lu.FilterBy = np.FilterBy
	}
	lu.Limit = np.Limit
	lu.Offset = np.Offset
	return nil
}
func (lu *ListUsers) Encode(users []models.User) *pb.ListUsersResponse {
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
func (lu *ListUsers) WriteTo(w http.ResponseWriter) error {
	return RespondObject(w, http.StatusOK, lu)
}

// UpdateUser
type UpdateUser struct {
	models.User
	UpdatedFields map[string]string
}

func (uu *UpdateUser) Decode(r *http.Request) error {
	uu.UpdatedFields = make(map[string]string)
	type au models.User
	tmp := struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Nickname  *string `json:"nickname"`
		Password  *string `json:"password"`
		Email     *string `json:"email"`
		Country   *string `json:"country"`
		*au
	}{}
	if err := json.NewDecoder(r.Body).Decode(&tmp); err != nil {
		return fmt.Errorf("malformed update user data: %w", err)
	}

	if tmp.FirstName != nil {
		uu.UpdatedFields["first_name"] = *tmp.FirstName
	}
	if tmp.LastName != nil {
		uu.UpdatedFields["last_name"] = *tmp.LastName
	}
	if tmp.Nickname != nil {
		uu.UpdatedFields["nick_name"] = *tmp.Nickname
	}
	if tmp.Country != nil {
		uu.UpdatedFields["country"] = *tmp.Country
	}

	if tmp.Password != nil {
		if *tmp.Password == "" {
			return fmt.Errorf("password could not been empty")
		}
		uu.UpdatedFields["password"] = *tmp.Password
	}
	if e := tmp.Email; e != nil {
		if err := models.ValidateEmail(*e); err != nil {
			return err
		}
		uu.UpdatedFields["email"] = strings.ToLower(*e)
	}
	if len(uu.UpdatedFields) == 0 {
		return fmt.Errorf("empty request")
	}
	uid := chi.URLParam(r, "uid")
	if uid == "" {
		return fmt.Errorf("user id is mandatory")
	}

	uu.Id = uid
	return nil
}
func (uu *UpdateUser) DecodePB(r *pb.UpdateUserRequest) error {
	uu.UpdatedFields = make(map[string]string)
	if r.FirstName != nil {
		uu.UpdatedFields["first_name"] = r.GetFirstName()
	}

	if r.LastName != nil {
		uu.UpdatedFields["last_name"] = r.GetLastName()
	}

	if r.Nickname != nil {
		uu.UpdatedFields["nick_name"] = r.GetNickname()
	}

	if r.Country != nil {
		uu.UpdatedFields["country"] = r.GetCountry()
	}

	if r.Password != nil {
		if r.GetPassword() == "" {
			return fmt.Errorf("password could not been empty")
		}
		uu.UpdatedFields["password"] = r.GetPassword()
	}
	if e := r.Email; e != nil {
		if err := models.ValidateEmail(*e); err != nil {
			return err
		}
		uu.UpdatedFields["email"] = strings.ToLower(*e)
	}
	if len(uu.UpdatedFields) == 0 {
		return fmt.Errorf("empty request")
	}
	if r.Id == "" {
		return fmt.Errorf("user id is mandatory")
	}
	uu.Id = r.Id
	return nil
}
func (uu *UpdateUser) WriteTo(w http.ResponseWriter) error {
	return RespondObject(w, http.StatusOK, nil)
}

// DeleteUser
type DeleteUser struct {
	models.User
}

func (du *DeleteUser) Decode(r *http.Request) error {
	du.Id = chi.URLParam(r, "uid")
	if du.Id == "" {
		return fmt.Errorf("id is mandatory")
	}
	return nil
}
func (du *DeleteUser) WriteTo(w http.ResponseWriter) error {
	return RespondObject(w, http.StatusOK, nil)
}

func user2PB(u *models.User) *pb.User {
	return &pb.User{
		Id:        u.Id,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Nickname:  u.NickName,
		Email:     u.Email,
		Country:   u.Country,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}
