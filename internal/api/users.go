package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/clients"
	"github.com/BorisRostovskiy/ESL/internal/storage"
	"github.com/BorisRostovskiy/ESL/internal/storage/models"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type UsersAPI interface {
	HealthCheck(ctx context.Context) error
	CountUsers(ctx context.Context, filterParams map[string]string) (int, error)
	CreateUser(ctx context.Context, in *models.User) (*models.User, error)
	ListUsers(ctx context.Context, limit, offset int, filterParams map[string]string) ([]models.User, error)
	UpdateUser(ctx context.Context, userId string, updatedFields map[string]string) error
	DeleteUser(ctx context.Context, id string) error
}

// UserRepo define storage storage interface
type UserRepo interface {
	TestConnection(ctx context.Context) error
	GetUser(ctx context.Context, userId string) (*models.User, error)
	CreateUser(ctx context.Context, in *models.User) (*models.User, error)
	ListUsers(ctx context.Context, limit, offset int, filterParams map[string]string) ([]models.User, error)
	CountUsers(ctx context.Context, filterParams map[string]string) (int, error)
	UpdateUser(ctx context.Context, in *models.User) error
	DeleteUser(ctx context.Context, userId string) error
}

type Users struct {
	repo   UserRepo
	log    *logrus.Logger
	notify clients.ChannelNotificator
}

func New(repo UserRepo, log *logrus.Logger, n clients.ChannelNotificator) Users {
	return Users{
		repo:   repo,
		log:    log,
		notify: n,
	}
}

// HealthCheck provide simple check of db status
func (s Users) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.repo.TestConnection(ctx)
}

func (s Users) CreateUser(ctx context.Context, in *models.User) (*models.User, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(in.Password), 8)
	if err != nil {
		return nil, fmt.Errorf("could not generate new hashed password for user: %w", err)
	}
	in.Password = string(hashedPwd)
	user, err := s.repo.CreateUser(ctx, in)
	if err != nil {
		s.log.WithField("component", "api").Debug(err)
		if errors.Is(err, storage.DuplicateKeyError) {
			return nil, ErrUserAlreadyExists
		}
		return nil, ErrInternal
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	_ = s.notify.Notify(ctx, clients.ChannelCreate, fmt.Sprintf("user with ID=%s has been created", user.Id))
	return user, nil
}

func (s Users) ListUsers(ctx context.Context, limit, offset int, filterParams map[string]string) ([]models.User, error) {
	users, err := s.repo.ListUsers(ctx, limit, offset, filterParams)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s Users) CountUsers(ctx context.Context, filterParams map[string]string) (int, error) {
	return s.repo.CountUsers(ctx, filterParams)
}

func (s Users) UpdateUser(ctx context.Context, userId string, updatedFields map[string]string) error {
	existedUser, err := s.repo.GetUser(ctx, userId)
	if err != nil {
		if errors.Is(err, storage.NoUsersFoundError) {
			return ErrUserNotFound
		}
		return err
	}
	// all done
	if len(updatedFields) == 0 {
		return nil
	}

	if fn, ok := updatedFields["first_name"]; ok && fn != existedUser.FirstName {
		existedUser.FirstName = fn
	}
	if ln, ok := updatedFields["last_name"]; ok && ln != existedUser.LastName {
		existedUser.LastName = ln
	}
	if nn, ok := updatedFields["nick_name"]; ok && nn != existedUser.NickName {
		existedUser.NickName = nn
	}
	if cntr, ok := updatedFields["country"]; ok && cntr != existedUser.Country {
		existedUser.Country = cntr
	}
	if e, ok := updatedFields["email"]; ok && e != existedUser.Email {
		existedUser.Email = e
	}
	if pwd, ok := updatedFields["password"]; ok && pwd != "" {
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), 8)
		if err != nil {
			return fmt.Errorf("could not generate new hashed password for user: %w", err)
		}
		existedUser.Password = string(hashedPwd)
	}

	if err = s.repo.UpdateUser(ctx, existedUser); err != nil {
		if errors.Is(err, storage.DuplicateKeyError) {
			return ErrDuplicateKeyError
		}
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	_ = s.notify.Notify(ctx, clients.ChannelUpdate, fmt.Sprintf("user with ID=%s has been updated", existedUser.Id))
	return nil
}

func (s Users) DeleteUser(ctx context.Context, id string) error {
	err := s.repo.DeleteUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.NoUsersFoundError) {
			return ErrUserNotFound
		}
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	_ = s.notify.Notify(ctx, clients.ChannelDelete, fmt.Sprintf("user with ID=%s has been deleted", id))

	return nil
}
