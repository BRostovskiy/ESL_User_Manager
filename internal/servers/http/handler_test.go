package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/api"
	"github.com/BorisRostovskiy/ESL/internal/clients"
	"github.com/BorisRostovskiy/ESL/internal/common"
	"github.com/BorisRostovskiy/ESL/internal/storage"
	"github.com/BorisRostovskiy/ESL/internal/storage/models"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

func addChiURLParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for k, v := range params {
		ctx.URLParams.Add(k, v)
	}

	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
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

func TestServer_CreateUser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	logger := logrus.New()
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	httpSvc := handler{log: logger, api: api.New(repo, logger, notificationSvc)}

	type expectation struct {
		responseCode    int
		responsePayload string
		errResponse     string
	}
	type input struct {
		reqPayload io.Reader
		urlVars    map[string]string
	}

	tests := map[string]struct {
		in     input
		want   expectation
		repo   func(r *storage.MockUserRepo)
		notify func(n *clients.MockChannelNotificator)
	}{
		"CreateUser Ok": {
			in: input{
				reqPayload: strings.NewReader(`{"first_name": "User5", "last_name": "Lastname5", "nickname": "user5_lastname", "email": "user5@gmail.com", "password": "qwerty123", "country": "NL"}`),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(&models.User{Id: id1}, nil).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {
				n.EXPECT().Notify(gomock.Any(), clients.ChannelCreate,
					fmt.Sprintf("user with ID=%s has been created", id1))
			},
			want: expectation{
				responseCode:    http.StatusCreated,
				responsePayload: `{"id":"67cfa917-1cec-48ff-913c-243fe5749e92","first_name":"User5","last_name":"Lastname5","nickname":"user5_lastname","email":"user5@gmail.com","country":"NL","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}`,
			},
		},
		"No Email Error": {
			in: input{
				reqPayload: strings.NewReader(`{"first_name": "User5", "password": "qwerty123", "country": "NL"}`),
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"invalid user: empty email"}`,
			},
		},
		"No Password Error": {
			in: input{
				reqPayload: strings.NewReader(`{"first_name": "User5", "email": "user5@gmail.com", "country": "NL"}`),
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"invalid user: empty password"}`,
			},
		},
		"Duplicate key Error": {
			in: input{
				reqPayload: strings.NewReader(`{"first_name": "User5", "last_name": "Lastname5", "nickname": "user5_lastname", "email": "user5@gmail.com", "password": "qwerty123", "country": "NL"}`),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil, storage.DuplicateKeyError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusConflict,
				errResponse:  `{"code":300,"message":"user already exists"}`,
			},
		},
		"Other repo problem Error": {
			in: input{
				reqPayload: strings.NewReader(`{"first_name": "User5", "last_name": "Lastname5", "nickname": "user5_lastname", "email": "user5@gmail.com", "password": "qwerty123", "country": "NL"}`),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf(somethingHappens)).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusInternalServerError,
				errResponse:  `{"code":100,"message":"api internal error"}`,
			},
		},
	}
	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			tt.repo(repo)
			tt.notify(notificationSvc)

			w := httptest.NewRecorder()
			r := mux.SetURLVars(
				httptest.NewRequest(http.MethodPost, "/api/v1/users", tt.in.reqPayload),
				tt.in.urlVars)

			err := httpSvc.createUser(r).WriteTo(w)
			assert.NoError(t, err)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()
			data, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.responseCode, res.StatusCode)
			if tt.want.errResponse == "" {
				assert.Equal(t, tt.want.responsePayload, string(data))
			} else {
				assert.Equal(t, tt.want.errResponse, string(data))
			}
		})
	}
}

func TestServer_ListUsers(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	common.AllowedFilters.WithValues([]string{"country"})
	logger := logrus.New()
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	httpSvc := handler{log: logger, api: api.New(repo, logger, notificationSvc)}

	type expectation struct {
		responseCode    int
		responsePayload string
		paginated       bool
		errResponse     string
	}
	type input struct {
		reqPayload    io.Reader
		requestParams map[string]string
		urlVars       map[string]string
	}

	tests := map[string]struct {
		in   input
		want expectation
		repo func(r *storage.MockUserRepo)
	}{
		"ListUsers no pagination Ok": {
			in: input{},
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
				responseCode:    http.StatusOK,
				responsePayload: `{"users":[{"id":"49afc235-3lry-23rf-343h-223jq6849e926","first_name":"User","last_name":"One","nickname":"userOne11","email":"user_three@gmail.com","country":"NL","created_at":"2022-07-20T12:45:44Z","updated_at":"0001-01-01T00:00:00Z"},{"id":"49afc235-3lry-23rf-343h-223jq6849e926","first_name":"User","last_name":"One","nickname":"userOne11","email":"user_three@gmail.com","country":"NL","created_at":"2022-07-20T12:45:44Z","updated_at":"0001-01-01T00:00:00Z"},{"id":"49afc235-3lry-23rf-343h-223jq6849e926","first_name":"User","last_name":"One","nickname":"userOne11","email":"user_three@gmail.com","country":"NL","created_at":"2022-07-20T12:45:44Z","updated_at":"0001-01-01T00:00:00Z"}]}`,
			},
		},
		"ListUsers nextPage no filter Ok": {
			in: input{
				requestParams: map[string]string{"next_page": nextPageNoFilter},
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
				responseCode:    http.StatusOK,
				responsePayload: `{"users":[{"id":"49afc235-3lry-23rf-343h-223jq6849e926","first_name":"User","last_name":"One","nickname":"userOne11","email":"user_three@gmail.com","country":"NL","created_at":"2022-07-20T12:45:44Z","updated_at":"0001-01-01T00:00:00Z"}]}`,
			},
		},
		"ListUsers pagination and filtering OK": {
			in: input{
				requestParams: map[string]string{
					"pagination": "1",
					"filterBy":   "country",
					"filter":     "NL",
				},
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
				paginated:       true,
				responseCode:    http.StatusOK,
				responsePayload: `{"users":[{"id":"49afc235-3lry-23rf-343h-223jq6849e926","first_name":"User","last_name":"One","nickname":"userOne11","email":"user_three@gmail.com","country":"NL","created_at":"2022-07-20T12:45:44Z","updated_at":"0001-01-01T00:00:00Z"}],"next_page":"eyJsaW1pdCI6MSwib2Zmc2V0IjoxLCJmaWx0ZXJfYnkiOiJjb3VudHJ5IiwiZmlsdGVyIjoiTkwiLCJ0aW1lIjoiMjAyNC0wOC0wN1QxMTozMDoyOS4yOTA4NDgrMDI6MDAifQ=="}`,
			},
		},
		"ListUsers nextPage base64 decode Error": {
			in: input{
				requestParams: map[string]string{
					"next_page": "@",
				},
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"could not load nextPage: could not decode next_page argument: illegal base64 data at input byte 0"}`,
			},
		},
		"ListUsers nextPage json unmarshal Error": {
			in: input{
				requestParams: map[string]string{
					"next_page": "QAo=",
				},
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"could not load nextPage: could not unmarshal limit offset: invalid character '@' looking for beginning of value"}`,
			},
		},
		"ListUsers filterBy and filter Error": {
			in: input{
				requestParams: map[string]string{
					"filterBy": "FirstName",
				},
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"could not load nextPage: parameters filter and filterBy should be used together"}`,
			},
		},
		"ListUsers invalid filterBy Error": {
			in: input{
				requestParams: map[string]string{
					"filter":   "ololo",
					"filterBy": "FirstName",
				},
			},
			repo: func(r *storage.MockUserRepo) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"filterBy parameter 'FirstName' not supported"}`,
			},
		},
		"ListUsers repo Error": {
			in: input{},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().ListUsers(gomock.Any(), -1, 0, map[string]string{"filter": "", "filterBy": ""}).
					Return(nil, fmt.Errorf(somethingHappens)).Times(1)
			},
			want: expectation{
				responseCode: http.StatusInternalServerError,
				errResponse:  `{"code":100,"message":"internal servers error"}`,
			},
		},
		"ListUsers count users Error": {
			in: input{
				requestParams: map[string]string{
					"pagination": "1",
				},
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
				responseCode: http.StatusInternalServerError,
				errResponse:  `{"code":100,"message":"internal servers error"}`,
			},
		},
		"ListUsers empty response": {
			in: input{},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().ListUsers(gomock.Any(), -1, 0, map[string]string{"filter": "", "filterBy": ""}).
					Return([]models.User{}, nil).Times(1)
			},
			want: expectation{
				responseCode:    http.StatusOK,
				responsePayload: `{"users":[]}`,
			},
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			tt.repo(repo)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/users", tt.in.reqPayload)
			r = mux.SetURLVars(r, tt.in.urlVars)

			values := r.URL.Query()
			for k, v := range tt.in.requestParams {
				values.Add(k, v)
			}
			r.URL.RawQuery = values.Encode()

			err := httpSvc.listUsers(r).WriteTo(w)
			assert.NoError(t, err)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()
			data, err := io.ReadAll(res.Body)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.responseCode, res.StatusCode)
			if tt.want.errResponse == "" {
				if tt.want.paginated {
					var got common.ListUsers
					err = json.Unmarshal(data, &got)
					assert.NoError(t, err)

					var pl common.ListUsers
					err = json.Unmarshal(data, &pl)
					assert.NoError(t, err)

					assert.True(t, got.NextPage == pl.NextPage)
					assert.True(t, reflect.DeepEqual(got, pl))
				} else {
					assert.Equal(t, tt.want.responsePayload, string(data))
				}
			} else {
				assert.Equal(t, tt.want.errResponse, string(data))
			}
		})
	}
}
func TestServer_UpdateUser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	logger := logrus.New()
	//logger.Level = logrus.DebugLevel
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	httpSvc := handler{log: logger, api: api.New(repo, logger, notificationSvc)}

	type expectation struct {
		responseCode    int
		responsePayload string
		errResponse     string
	}
	type input struct {
		reqUrl     string
		reqPayload io.Reader
		urlVars    map[string]string
	}

	tests := map[string]struct {
		in     input
		want   expectation
		repo   func(r *storage.MockUserRepo)
		notify func(n *clients.MockChannelNotificator)
	}{
		"Update user OK": {
			in: input{
				reqUrl:     fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars:    map[string]string{"uid": id1},
				reqPayload: strings.NewReader(fmt.Sprintf(`{"email": "%s"}`, email2)),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(user.WithId(id1).WithEmail(email1), nil).Times(1)
				r.EXPECT().UpdateUser(gomock.Any(), user.WithId(id1).WithEmail(email2)).Return(nil).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {
				n.EXPECT().Notify(gomock.Any(), clients.ChannelUpdate, fmt.Sprintf("user with ID=%s has been updated", id1))
			},
			want: expectation{
				responseCode:    http.StatusOK,
				responsePayload: `null`,
			},
		},
		"Update user structure is empty error": {
			in: input{
				reqUrl:     fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars:    map[string]string{"uid": id1},
				reqPayload: strings.NewReader(`{}`),
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"failed to parse request: empty request"}`,
			},
		},

		"Update user id empty error": {
			in: input{
				reqUrl:     fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars:    map[string]string{},
				reqPayload: strings.NewReader(fmt.Sprintf(`{"email": "%s"}`, email2)),
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"failed to parse request: user id is mandatory"}`,
			},
		},
		"Update no users found error": {
			in: input{
				reqUrl:     fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars:    map[string]string{"uid": id1},
				reqPayload: strings.NewReader(fmt.Sprintf(`{"email": "%s"}`, email2)),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(nil, storage.NoUsersFoundError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusNotFound,
				errResponse:  `{"code":200,"message":"user not found"}`,
			},
		},
		"Update duplicate key error": {
			in: input{
				reqUrl:     fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars:    map[string]string{"uid": id1},
				reqPayload: strings.NewReader(fmt.Sprintf(`{"email": "%s"}`, email2)),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(user.WithId(id1).WithEmail(email1), nil).Times(1)
				r.EXPECT().UpdateUser(gomock.Any(), user.WithId(id1).WithEmail(email2)).Return(storage.DuplicateKeyError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusConflict,
				errResponse:  `{"code":102,"message":"duplicate key error"}`,
			},
		},
		"Update repo something happens error": {
			in: input{
				reqUrl:     fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars:    map[string]string{"uid": id1},
				reqPayload: strings.NewReader(fmt.Sprintf(`{"email": "%s"}`, email2)),
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().GetUser(gomock.Any(), id1).Return(user.WithId(id1).WithEmail(email1), nil).Times(1)
				r.EXPECT().UpdateUser(gomock.Any(), user.WithId(id1).WithEmail(email2)).Return(fmt.Errorf(somethingHappens)).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusInternalServerError,
				errResponse:  `{"code":100,"message":"internal servers error"}`,
			},
		},
	}
	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			tt.repo(repo)
			tt.notify(notificationSvc)
			r := addChiURLParams(httptest.NewRequest(http.MethodPut, tt.in.reqUrl, tt.in.reqPayload), tt.in.urlVars)
			w := httptest.NewRecorder()

			err := httpSvc.updateUser(r).WriteTo(w)
			assert.NoError(t, err)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()
			data, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.responseCode, res.StatusCode)
			if tt.want.errResponse == "" {
				assert.Equal(t, tt.want.responsePayload, string(data))
			} else {
				assert.Equal(t, tt.want.errResponse, string(data))
			}
		})
	}
}

func TestServer_DeleteUser(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	repo := storage.NewMockUserRepo(ctrl)
	logger := logrus.New()
	//logger.Level = logrus.DebugLevel
	notificationSvc := clients.NewMockChannelNotificator(ctrl)
	httpSvc := handler{log: logger, api: api.New(repo, logger, notificationSvc)}

	type expectation struct {
		responseCode    int
		responsePayload string
		errResponse     string
	}
	type input struct {
		reqUrl     string
		reqPayload io.Reader
		urlVars    map[string]string
	}

	tests := map[string]struct {
		in     input
		want   expectation
		repo   func(r *storage.MockUserRepo)
		notify func(n *clients.MockChannelNotificator)
	}{
		"DeleteUser Ok": {
			in: input{
				reqUrl:  fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars: map[string]string{"uid": id1},
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().DeleteUser(gomock.Any(), id1).Return(nil).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {
				n.EXPECT().Notify(gomock.Any(), clients.ChannelDelete, fmt.Sprintf("user with ID=%s has been deleted", id1))
			},
			want: expectation{
				responseCode:    http.StatusOK,
				responsePayload: `null`,
			},
		},
		"Delete user no id error": {
			in: input{
				reqUrl:  fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars: map[string]string{},
			},
			repo:   func(r *storage.MockUserRepo) {},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusBadRequest,
				errResponse:  `{"code":101,"message":"failed to parse request: id is mandatory"}`,
			},
		},
		"Delete user not found error": {
			in: input{
				reqUrl:  fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars: map[string]string{"uid": id1},
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().DeleteUser(gomock.Any(), id1).Return(storage.NoUsersFoundError).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusNotFound,
				errResponse:  `{"code":200,"message":"user not found"}`,
			},
		},
		"Delete user repo error": {
			in: input{
				reqUrl:  fmt.Sprintf("/api/v1/users/%s", id1),
				urlVars: map[string]string{"uid": id1},
			},
			repo: func(r *storage.MockUserRepo) {
				r.EXPECT().DeleteUser(gomock.Any(), id1).Return(fmt.Errorf(somethingHappens)).Times(1)
			},
			notify: func(n *clients.MockChannelNotificator) {},
			want: expectation{
				responseCode: http.StatusInternalServerError,
				errResponse:  `{"code":100,"message":"internal servers error"}`,
			},
		},
	}
	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			tt.repo(repo)
			tt.notify(notificationSvc)
			r := addChiURLParams(httptest.NewRequest(http.MethodDelete, tt.in.reqUrl, tt.in.reqPayload), tt.in.urlVars)
			w := httptest.NewRecorder()

			err := httpSvc.deleteUser(r).WriteTo(w)
			assert.NoError(t, err)
			res := w.Result()
			defer func() { _ = res.Body.Close() }()
			data, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.responseCode, res.StatusCode)
			if tt.want.errResponse == "" {
				assert.Equal(t, tt.want.responsePayload, string(data))
			} else {
				assert.Equal(t, tt.want.errResponse, string(data))
			}
		})
	}
}
