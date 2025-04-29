package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/handlers"

	"github.com/BorisRostovskiy/ESL/internal/service"

	"github.com/go-chi/chi/v5"
)

type User struct {
	ID        string    `json:"id,omitempty"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	NickName  string    `json:"nickname"`
	Password  string    `json:"password,omitempty"`
	Email     string    `json:"email"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) marshal(su *service.User) {
	u.ID = su.ID
	u.FirstName = su.FirstName
	u.LastName = su.LastName
	u.NickName = su.NickName
	u.Email = su.Email
	u.Country = su.Country
	u.CreatedAt = su.CreatedAt
	u.UpdatedAt = su.UpdatedAt
}

type CreateUser struct {
	User *service.User
}

func (cu *CreateUser) Decode(r *http.Request) error {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		return fmt.Errorf("malformed data: %w", err)
	}

	cu.User = &service.User{
		FirstName: u.FirstName,
		LastName:  u.LastName,
		NickName:  u.NickName,
		Password:  u.Password,
		Email:     strings.ToLower(u.Email),
		Country:   u.Country,
	}
	if err := cu.User.Validate(false); err != nil {
		return fmt.Errorf("invalid user: %w", err)
	}

	return nil
}
func (cu *CreateUser) WriteTo(w http.ResponseWriter) error {
	var u User
	u.marshal(cu.User)
	return responseObject(w, http.StatusCreated, u)
}

// ListUsers
type listUsers struct {
	Users    []User          `json:"users"`
	Limit    int             `json:"-"`
	Offset   int             `json:"-"`
	Filter   *service.Filter `json:"-"`
	NextPage string          `json:"next_page,omitempty"`
}

func (lu *listUsers) Decode(r *http.Request) error {
	np, err := nextPage(r)
	if err != nil {
		return fmt.Errorf("could not load nextPage: %v", err)
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
func (lu *listUsers) WriteTo(w http.ResponseWriter) error {
	return responseObject(w, http.StatusOK, lu)
}

// UpdateUser
type updateUser struct {
	service.User
}

func (uu *updateUser) Decode(r *http.Request) error {
	tmp := struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Nickname  *string `json:"nickname"`
		Password  *string `json:"password"`
		Email     *string `json:"email"`
		Country   *string `json:"country"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&tmp); err != nil {
		return fmt.Errorf("malformed update user data: %w", err)
	}

	if tmp.FirstName != nil {
		uu.User.WithFirstName(*tmp.FirstName)
	}
	if tmp.LastName != nil {
		uu.User.WithLastName(*tmp.LastName)
	}
	if tmp.Nickname != nil {
		uu.User.WithNickName(*tmp.Nickname)
	}
	if tmp.Country != nil {
		uu.User.WithCountry(*tmp.Country)
	}

	if tmp.Password != nil {
		uu.User.WithPassword(*tmp.Password)
	}
	if tmp.Email != nil {
		uu.User.WithEmail(*tmp.Email)
	}

	uid := chi.URLParam(r, "uid")
	if uid == "" {
		return fmt.Errorf("user id is mandatory")
	}
	uu.User.ID = uid
	return nil
}
func (uu *updateUser) WriteTo(w http.ResponseWriter) error {
	return responseObject(w, http.StatusOK, nil)
}

// DeleteUser
type deleteUser struct {
	service.User
}

func (du *deleteUser) Decode(r *http.Request) error {
	du.User.ID = chi.URLParam(r, "uid")
	if du.User.ID == "" {
		return fmt.Errorf("id is mandatory")
	}
	return nil
}
func (du *deleteUser) WriteTo(w http.ResponseWriter) error {
	return responseObject(w, http.StatusOK, nil)
}

func nextPage(r *http.Request) (*handlers.NextPage, error) {
	return handlers.LoadNextPage(r.URL.Query().Get("next_page"),
		r.URL.Query().Get("filter"),
		r.URL.Query().Get("filterBy"),
		func() (int, error) {
			if r.URL.Query().Get("pagination") != "" {
				p, err := strconv.ParseInt(r.URL.Query().Get("pagination"), 10, 64)
				if err != nil {
					return -1, fmt.Errorf("malformed pagination")
				}
				return int(p), nil
			}
			return -1, nil
		},
	)
}
