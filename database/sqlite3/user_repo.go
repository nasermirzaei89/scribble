package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/nasermirzaei89/scribble/authentication"
)

const tableUsers = "users"

type UserRepository struct {
	db *sql.DB
}

var _ authentication.UserRepository = (*UserRepository)(nil)

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

func scanUser(row sq.RowScanner) (*authentication.User, error) {
	var user authentication.User

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

func (repo *UserRepository) Insert(ctx context.Context, user *authentication.User) error {
	q := sq.Insert(tableUsers).
		Columns(userColumns()...).
		Values(user.ID, user.Username, user.PasswordHash, user.RegisteredAt)

	q = q.RunWith(repo.db)

	_, err := q.ExecContext(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
			return &authentication.UserAlreadyExistsError{Username: user.Username}
		}

		return fmt.Errorf("failed to exec insert: %w", err)
	}

	return nil
}

func (repo *UserRepository) Find(ctx context.Context, userID string) (*authentication.User, error) {
	q := sq.Select(userColumns()...).
		From(tableUsers).
		Where(sq.Eq{userFieldID: userID})

	q = q.RunWith(repo.db)

	row := q.QueryRowContext(ctx)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &authentication.UserNotFoundError{ID: userID}
		}

		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return user, nil
}

func (repo *UserRepository) ListUsernames(ctx context.Context) ([]string, error) {
	q := sq.Select(userFieldUsername).From(tableUsers).RunWith(repo.db)

	rows, err := q.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query usernames: %w", err)
	}
	defer rows.Close()

	var usernames []string

	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, fmt.Errorf("failed to scan username: %w", err)
		}

		usernames = append(usernames, username)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate usernames: %w", err)
	}

	return usernames, nil
}

func (repo *UserRepository) FindByUsername(ctx context.Context, username string) (*authentication.User, error) {
	q := sq.Select(userColumns()...).
		From(tableUsers).
		Where(sq.Eq{userFieldUsername: username})

	q = q.RunWith(repo.db)

	row := q.QueryRowContext(ctx)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &authentication.UserByUsernameNotFoundError{Username: username}
		}

		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return user, nil
}
