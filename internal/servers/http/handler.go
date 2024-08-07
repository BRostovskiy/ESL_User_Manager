package http

import (
	"net/http"

	usersApi "github.com/BorisRostovskiy/ESL/internal/api"
	"github.com/BorisRostovskiy/ESL/internal/common"

	"github.com/hellofresh/health-go/v5"
	"github.com/sirupsen/logrus"
)

type handler struct {
	log *logrus.Logger
	api usersApi.UsersAPI
}

func New(log *logrus.Logger, api usersApi.UsersAPI, h *health.Health) http.Handler {
	return router(&handler{
		log: log,
		api: api,
	}, log, h)
}

func (h handler) handle(hf func(r *http.Request) common.Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.respond(w, hf(r))
	}
}

func (h handler) respond(w http.ResponseWriter, r common.Response) {
	w.Header().Set(common.HeaderContentType, "application/json; charset=utf-8")
	if err := r.WriteTo(w); err != nil {
		h.log.Errorf("failed to write response: %s", err)
	}
}

// Create user
func (h handler) createUser(r *http.Request) common.Response {
	cu := &common.CreateUser{}
	if err := cu.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("create user decode error: %v", err)
		return errRequest(r, err)
	}
	user, err := h.api.CreateUser(r.Context(), &cu.User)
	if err != nil {
		return errApi(r, "failed to perform user creation: %w", err)
	}
	cu.User.Id = user.Id
	cu.User.CreatedAt = user.CreatedAt
	cu.User.UpdatedAt = user.UpdatedAt
	cu.User.Password = ""
	return cu
}

// List users
func (h handler) listUsers(r *http.Request) common.Response {
	lu := &common.ListUsers{}
	if err := lu.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("list users decode error: %v", err)
		return errRequest(r, err)
	}

	filters := map[string]string{"filter": lu.Filter, "filterBy": lu.FilterBy}

	users, err := h.api.ListUsers(r.Context(), lu.Limit, lu.Offset, filters)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform list users: %v", err)
		return errApi(r, "could not list users: %w", err)
	}
	lu.Users = users
	if len(users) == 0 {
		return lu
	}

	numUsers, err := h.api.CountUsers(r.Context(), filters)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform count users: %v", err)
		return errApi(r, "could not get users counted: %w", err)
	}
	np, err := common.GenerateNextPage(lu.Limit, lu.Offset, numUsers, lu.Filter, lu.FilterBy)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("could not marshal next page: %v", err)
		return errRequestf(r, "could not marshal next page structure: %w", err)
	}
	lu.NextPage = np

	return lu
}

// Update user
func (h handler) updateUser(r *http.Request) common.Response {
	uu := &common.UpdateUser{}
	if err := uu.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("update user decode error: %v", err)
		return errRequestf(r, "failed to parse request: %w", err)
	}

	err := h.api.UpdateUser(r.Context(), uu.User.Id, uu.UpdatedFields)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform update user: %v", err)
		return errApi(r, "could not list users: %w", err)
	}

	return uu
}

// Delete user
func (h handler) deleteUser(r *http.Request) common.Response {
	du := &common.DeleteUser{}
	if err := du.Decode(r); err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("update user decode error: %v", err)
		return errRequestf(r, "failed to parse request: %w", err)
	}

	err := h.api.DeleteUser(r.Context(), du.Id)
	if err != nil {
		h.log.WithField("component", "http_handler").
			Debugf("failed to perform delete user: %v", err)
		return errApi(r, "could not perform delete user: %w", err)
	}

	return du
}
