package http

import (
	"net/http"

	"github.com/BorisRostovskiy/ESL/internal/handlers"
)

// Create user
func (h handler) createUser(r *http.Request) response {
	cu := &CreateUser{}
	if err := cu.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("create user decode error: %v", err)
		return errRequest(r, err)
	}
	user, err := h.api.CreateUser(r.Context(), cu.User)
	if err != nil {
		return errApi(r, "failed to perform user creation: %w", err)
	}

	cu.User = user
	return cu
}

// List users
func (h handler) listUsers(r *http.Request) response {
	lu := &listUsers{}
	if err := lu.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("list users decode error: %v", err)
		return errRequest(r, err)
	}

	users, err := h.api.ListUsers(r.Context(), lu.Limit, lu.Offset, lu.Filter)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform list users: %v", err)
		return errApi(r, "could not list users: %w", err)
	}

	lu.Users = make([]User, len(users))
	if len(users) == 0 {
		return lu
	}

	for i, su := range users {
		var u User
		u.marshal(&su)
		lu.Users[i] = u
	}

	np, err := handlers.GenerateNextPage(lu.Limit, lu.Offset, len(users), lu.Filter)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("could not marshal next page: %v", err)
		return errRequestf(r, "could not marshal next page structure: %w", err)
	}

	lu.NextPage = np

	return lu
}

// Update user
func (h handler) updateUser(r *http.Request) response {
	uu := &updateUser{}
	if err := uu.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("update user decode error: %v", err)
		return errRequestf(r, "failed to parse request: %w", err)
	}

	err := h.api.UpdateUser(r.Context(), &uu.User)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform update user: %v", err)
		return errApi(r, "could not list users: %w", err)
	}

	return uu
}

// Delete user
func (h handler) deleteUser(r *http.Request) response {
	du := &deleteUser{}
	if err := du.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("update user decode error: %v", err)
		return errRequestf(r, "failed to parse request: %w", err)
	}

	err := h.api.DeleteUser(r.Context(), du.User.ID)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform delete user: %v", err)
		return errApi(r, "could not perform delete user: %w", err)
	}

	return du
}
