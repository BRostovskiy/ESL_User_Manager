package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/repository"
	"github.com/BorisRostovskiy/ESL/internal/service"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	//"github.com/pkg/errors"
)

const (
	errPgCheckViolation      = "23514"
	errPgNotNullViolation    = "23502"
	errPgForeignKeyViolation = "23503"
	errPgUniqueKeyViolation  = "23505"

	couldNotRetrieveAffected = "could not retrieve affected rows: %w"
	schema                   = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		nickname TEXT NOT NULL,
		password TEXT NOT NULL,
		email TEXT NOT NULL,
		country TEXT NOT NULL,
		created_at TIMESTAMP,
		updated_at TIMESTAMP DEFAULT NOW(),
		CONSTRAINT id_uq UNIQUE (id),
	    CONSTRAINT nickname_uq UNIQUE(nickname),
	    CONSTRAINT email_uq UNIQUE (email)
	);
	CREATE INDEX IF NOT EXISTS country_idx ON users USING btree(country);
`
)

type (
	// Repo implements service.UserRepo interface
	Repo struct {
		conn *sqlx.DB
		log  *logrus.Logger
	}
)

// New sets up a new Postgres repository.
func New(cfg *Config, log *logrus.Logger) (*Repo, error) {
	repo := new(Repo)

	conn, err := setupConnectionPool(cfg)
	if err != nil {
		return nil, fmt.Errorf("an error occurred during setup connection pool: %w", err)
	}

	//if err = applySchema(conn); err != nil {
	//	return nil, fmt.Errorf("an error occurred during applying schema: %w", err)
	//}

	repo.conn = conn
	repo.log = log
	return repo, nil
}

// CreateUser creates new user with generated ID
func (r *Repo) CreateUser(ctx context.Context, newUser *service.User) (*service.User, error) {
	newUser.ID = uuid.New().String()
	newUser.CreatedAt = time.Now()
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), 8)
	if err != nil {
		r.log.Errorf("generate pwd error: %v", err)
		return nil, repository.GeneratePwdError
	}

	_, err = r.conn.ExecContext(ctx,
		`INSERT INTO users (id, first_name, last_name, nickname, password, email, country, created_at) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		newUser.ID, newUser.FirstName, newUser.LastName, newUser.NickName,
		hashedPwd, newUser.Email, newUser.Country, newUser.CreatedAt)

	if err != nil {
		if isPgViolation(err, errPgForeignKeyViolation) {
			return nil, repository.DuplicateKeyError
		}
		return nil, fmt.Errorf("could not create new user: %w", err)
	}

	return newUser, nil
}

const ()

// ListUsers get (filtered) list of users, sorted DESC
func (r *Repo) ListUsers(ctx context.Context, limit, offset int, filter *service.Filter) ([]service.User, error) {
	users := make([]User, 0)

	query := `SELECT id, first_name, last_name, nickname, email, country, created_at, updated_at
					FROM users %s
					ORDER BY created_at
					DESC %s`

	var queryArgs []interface{}
	where := ""
	limitOffset := ""

	{
		i := 1
		if filter != nil && filter.IsValid() {
			where = fmt.Sprintf("WHERE %s=$%d", filter.By.String(), i)
			queryArgs = append(queryArgs, filter.Query)
			i++
		}
		if limit > 0 && offset >= 0 {
			limitOffset = fmt.Sprintf("OFFSET $%d LIMIT $%d", i, i+1)
			queryArgs = append(queryArgs, offset)
			queryArgs = append(queryArgs, limit)
		}
	}

	query = fmt.Sprintf(query, where, limitOffset)
	err := r.conn.SelectContext(ctx, &users, query, queryArgs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not perform select all from users: %w", err)
	}

	result := make([]service.User, len(users))
	for i, u := range users {
		result[i] = service.User{
			ID:        u.Id,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			NickName:  u.NickName,
			Email:     u.Email,
			Country:   u.Country,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		}
	}
	return result, nil
}

// UpdateUser update all user fields except the password(need to be separate function)
func (r *Repo) UpdateUser(ctx context.Context, user *service.User) error {
	query := `UPDATE users SET first_name=$1, last_name=$2, nickname=$3, email=$4, country=$5, updated_at=$6`
	args := []interface{}{
		user.FirstName,
		user.LastName,
		user.NickName,
		user.Email,
		user.Country,
		time.Now(),
	}
	if user.Password != "" {
		query += "password=$7 WHERE id=$8"
		args = append(args, user.Password)
	} else {
		query += "WHERE id=$7"
	}

	args = append(args, user.ID)
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, query, args...)

	if err != nil {
		_ = tx.Rollback()
		if isPgViolation(err, errPgForeignKeyViolation) {
			return repository.DuplicateKeyError
		}
		return fmt.Errorf("could not update user: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(couldNotRetrieveAffected, err)
	}
	if affected == 0 {
		return repository.NoUsersFoundError
	}

	return tx.Commit()
}

// GetUser retrieve user by ID
func (r *Repo) GetUser(ctx context.Context, userID string) (*service.User, error) {
	var user User
	query := `SELECT
    	id,
		first_name, 
		last_name, 
		nickname, 
		email, 
		country, 
		created_at, 
		updated_at
 	FROM users WHERE id=$1`

	err := r.conn.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.NoUsersFoundError
		}
		return nil, fmt.Errorf("could not perform select all from users: %v", err)
	}
	return &service.User{
		ID:        user.Id,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		NickName:  user.NickName,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil

}

// DeleteUser delete user by ID
func (r *Repo) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id=$1`
	_, err := r.conn.ExecContext(ctx, query, userID)

	if err != nil {
		return fmt.Errorf("could not delete user: %w", err)
	}
	return nil
}

// TestConnection tests that the Store can properly connect to the Postgres Server.
func (r *Repo) TestConnection(_ context.Context) error {
	err := r.conn.Ping()
	return err
}

func setupConnectionPool(cfg *Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", cfg.User, cfg.Pwd, cfg.Server, cfg.DBName)
	conn, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// applySchema applies the schema to a connection
func applySchema(conn *sqlx.DB) error {
	tx, err := conn.Begin()

	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec(schema); err != nil {
		return err
	}

	return tx.Commit()
}

func isPgViolation(err error, filter ...pq.ErrorCode) bool {
	var pgErr *pq.Error
	ok := errors.As(err, &pgErr)
	if !ok {
		return false
	}
	if filter == nil {
		filter = []pq.ErrorCode{
			errPgCheckViolation,
			errPgNotNullViolation,
			errPgUniqueKeyViolation,
			errPgForeignKeyViolation,
		}
	}
	for _, typ := range filter {
		if pgErr.Code == typ {
			return true
		}
	}
	return false
}
