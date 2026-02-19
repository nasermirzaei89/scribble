package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/nasermirzaei89/scribble/auth"
)

const tableUsers = "users"

type UserRepository struct {
	db *sql.DB
}

var _ auth.UserRepository = (*UserRepository)(nil)

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const (
	userFieldID           = "id"
	userFieldUsername     = "username"
	userFieldPasswordHash = "password_hash"
	userFieldRegisteredAt = "registered_at"
)

func userColumns() []string {
	return []string{
		userFieldID,
		userFieldUsername,
		userFieldPasswordHash,
		userFieldRegisteredAt,
	}
}

func scanUser(row sq.RowScanner) (*auth.User, error) {
	var user auth.User

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RegisteredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return &user, nil
}

func (repo *UserRepository) Insert(ctx context.Context, user *auth.User) error {
	q := sq.Insert(tableUsers).
		Columns(userColumns()...).
		Values(user.ID, user.Username, user.PasswordHash, user.RegisteredAt)

	q = q.RunWith(repo.db)

	_, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to exec insert: %w", err)
	}

	return nil
}

func (repo *UserRepository) FindByUsername(ctx context.Context, username string) (*auth.User, error) {
	q := sq.Select(userColumns()...).
		From(tableUsers).
		Where(sq.Eq{userFieldUsername: username})

	q = q.RunWith(repo.db)

	row := q.QueryRowContext(ctx)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &auth.UserByUsernameNotFoundError{Username: username}
		}

		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return user, nil
}
