package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/clients"
	"github.com/BorisRostovskiy/ESL/internal/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// UserRepo define repository interface
type UserRepo interface {
	TestConnection(ctx context.Context) error
	GetUser(ctx context.Context, userId string) (*User, error)
	CreateUser(ctx context.Context, in *User) (*User, error)
	ListUsers(ctx context.Context, limit, offset int, filter *Filter) ([]User, error)
	UpdateUser(ctx context.Context, in *User) error
	DeleteUser(ctx context.Context, userId string) error
}

type Users struct {
	repo   UserRepo
	log    *logrus.Logger
	notify clients.ChannelNotificator
}

func New(repo UserRepo, log *logrus.Logger, n clients.ChannelNotificator) *Users {
	return &Users{
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

func (s Users) CreateUser(ctx context.Context, in *User) (*User, error) {
	user, err := s.repo.CreateUser(ctx, in)

	if err != nil {
		s.log.WithField("component", "service").Debug(err)
		if errors.Is(err, repository.DuplicateKeyError) {
			return nil, ErrUserAlreadyExists
		}
		return nil, ErrInternal
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	_ = s.notify.Notify(ctx, clients.ChannelCreate, fmt.Sprintf("user with ID=%s has been created", user.ID))
	in.ID = user.ID
	in.CreatedAt = user.CreatedAt
	return in, nil
}

func (s Users) ListUsers(ctx context.Context, limit, offset int, filter *Filter) ([]User, error) {
	users, err := s.repo.ListUsers(ctx, limit, offset, filter)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s Users) UpdateUser(ctx context.Context, updatedUser *User) error {
	existedUser, err := s.repo.GetUser(ctx, updatedUser.ID)
	if err != nil {
		if errors.Is(err, repository.NoUsersFoundError) {
			return ErrUserNotFound
		}
		return err
	}

	updated := false
	if fn := updatedUser.FirstName; fn != "" && fn != existedUser.FirstName {
		existedUser.FirstName = fn
		updated = true
	}
	if ln := updatedUser.LastName; ln != "" && ln != existedUser.LastName {
		existedUser.LastName = ln
		updated = true
	}
	if nn := updatedUser.LastName; nn != "" && nn != existedUser.NickName {
		existedUser.NickName = nn
		updated = true
	}
	if cntr := updatedUser.Country; cntr != "" && cntr != existedUser.Country {
		existedUser.Country = cntr
		updated = true
	}

	if email := updatedUser.Email; email != "" && email != existedUser.Email {
		existedUser.Email = email
		updated = true
	}
	if pwd := updatedUser.Password; pwd != "" {
		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), 8)
		if err != nil {
			return fmt.Errorf("could not generate new hashed password for user: %w", err)
		}
		existedUser.Password = string(hashedPwd)
		updated = true
	}

	if !updated {
		return ErrEmptyUpdateRequest
	}

	if err = existedUser.Validate(true); err != nil {
		return fmt.Errorf("updated user not valid: %v", err)
	}

	if err = s.repo.UpdateUser(ctx, existedUser); err != nil {
		if errors.Is(err, repository.DuplicateKeyError) {
			return ErrDuplicateKeyError
		}
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	_ = s.notify.Notify(ctx, clients.ChannelUpdate, fmt.Sprintf("user with ID=%s has been updated", existedUser.ID))
	return nil
}

func (s Users) DeleteUser(ctx context.Context, id string) error {
	err := s.repo.DeleteUser(ctx, id)
	if err != nil {
		if errors.Is(err, repository.NoUsersFoundError) {
			return ErrUserNotFound
		}
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	_ = s.notify.Notify(ctx, clients.ChannelDelete, fmt.Sprintf("user with ID=%s has been deleted", id))

	return nil
}
