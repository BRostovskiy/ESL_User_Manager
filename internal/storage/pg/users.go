package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BorisRostovskiy/ESL/internal/storage"
	"github.com/BorisRostovskiy/ESL/internal/storage/models"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	//"github.com/pkg/errors"
)

const (
	couldNotRetrieveAffected = "could not retrieve affected rows: %w"
	duplicateKeyViolation    = "duplicate key value violates unique constraint"
	schema                   = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		nickname TEXT NOT NULL,
		password TEXT NOT NULL,
		email TEXT NOT NULL,
		country TEXT NOT NULL,
		created_at TIMESTAMP ,
		updated_at TIMESTAMP DEFAULT NOW(),
		CONSTRAINT id_uq UNIQUE (id),
	    CONSTRAINT email_uq UNIQUE (email)
	);
	CREATE INDEX IF NOT EXISTS country_idx ON users USING btree(country);
`
)

type (
	// Repo implements api.UserRepo interface
	Repo struct {
		conn *sqlx.DB
	}
)

// New sets up a new Postgres storage.
func New(cfg *Config) (*Repo, error) {
	repo := new(Repo)

	conn, err := setupConnectionPool(cfg)
	if err != nil {
		return nil, fmt.Errorf("an error occurred during setup connection pool: %w", err)
	}

	if err = applySchema(conn); err != nil {
		return nil, fmt.Errorf("an error occurred during applying schema: %w", err)
	}

	repo.conn = conn
	return repo, nil
}

// CreateUser creates new user with generated ID
func (r *Repo) CreateUser(ctx context.Context, newUser *models.User) (*models.User, error) {
	newUser.Id = uuid.New().String()
	newUser.CreatedAt = time.Now()

	_, err := r.conn.ExecContext(ctx,
		`INSERT INTO users (id, first_name, last_name, nickname, password, email, country, created_at) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		newUser.Id, newUser.FirstName, newUser.LastName, newUser.NickName,
		newUser.Password, newUser.Email, newUser.Country, newUser.CreatedAt)

	if err != nil {
		if pqErr, ok := err.(*pgconn.PgError); ok && strings.Contains(pqErr.Message, duplicateKeyViolation) {
			return nil, storage.DuplicateKeyError
		}
		return nil, fmt.Errorf("could not create new user: %w", err)
	}

	return newUser, nil
}

// ListUsers get (filtered) list of users, sorted DESC
func (r *Repo) ListUsers(ctx context.Context, limit, offset int, filterParams map[string]string) ([]models.User, error) {
	users := make([]models.User, 0)

	query := `SELECT id, first_name, last_name, nickname, email, country, created_at, updated_at
					FROM users %s
					ORDER BY created_at
					DESC %s`

	var queryArgs []interface{}
	i := 1
	where := ""
	if filterBy, filter, ok := extractQueryFilter(filterParams); ok {
		where = fmt.Sprintf("WHERE %s=$%d", filterBy, i)
		queryArgs = append(queryArgs, filter)
		i++
	}

	limitOffset := ""
	if limit > 0 && offset >= 0 {
		limitOffset = fmt.Sprintf("OFFSET $%d LIMIT $%d", i, i+1)
		queryArgs = append(queryArgs, offset)
		queryArgs = append(queryArgs, limit)
	}

	query = fmt.Sprintf(query, where, limitOffset)
	err := r.conn.SelectContext(ctx, &users, query, queryArgs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not perform select all from users: %w", err)
	}

	return users, nil
}

// CountUsers helper function
func (r *Repo) CountUsers(ctx context.Context, filterParams map[string]string) (int, error) {
	var result int
	query := `SELECT COUNT(*) FROM users %s`
	var queryArgs []interface{}
	i := 1
	where := ""
	if filterBy, filter, ok := extractQueryFilter(filterParams); ok {
		where = fmt.Sprintf("WHERE %s=$%d", filterBy, i)
		queryArgs = append(queryArgs, filter)
		i++
	}
	query = fmt.Sprintf(query, where)
	err := r.conn.GetContext(ctx, &result, query, queryArgs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, nil
		}
		return result, fmt.Errorf("could not perform select all from users: %w", err)
	}
	return result, nil
}

// UpdateUser update all user fields except the password(need to be separate function)
func (r *Repo) UpdateUser(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET first_name=$1, last_name=$2, nickname=$3, email=$4, country=$5, updated_at=$6 
				WHERE id=$7`
	result, err := r.conn.ExecContext(ctx, query, user.FirstName, user.LastName, user.NickName,
		user.Email, user.Country, time.Now(), user.Id)

	if err != nil {
		if pqErr, ok := err.(*pgconn.PgError); ok && strings.Contains(pqErr.Message, duplicateKeyViolation) {
			return storage.DuplicateKeyError
		}
		return fmt.Errorf("could not update user: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(couldNotRetrieveAffected, err)
	}
	if affected == 0 {
		return storage.NoUsersFoundError
	}

	return nil
}

// GetUser retrieve user by ID
func (r *Repo) GetUser(ctx context.Context, id string) (*models.User, error) {
	var user models.User
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

	err := r.conn.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.NoUsersFoundError
		}
		return nil, fmt.Errorf("could not perform select all from users: %v", err)
	}
	return &user, nil

}

// DeleteUser delete user by ID
func (r *Repo) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id=$1`
	_, err := r.conn.ExecContext(ctx, query, id)

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

func extractQueryFilter(q map[string]string) (string, string, bool) {
	filterBy, okFilterBy := q["filterBy"]
	filter, okFilter := q["filter"]
	ok := false
	if okFilterBy && okFilter && filterBy != "" && filter != "" {
		ok = true
	}
	return filterBy, filter, ok
}
