package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/nasermirzaei89/scribble/auth"
)

const tableSessions = "sessions"

type SessionRepository struct {
	db *sql.DB
}

var _ auth.SessionRepository = (*SessionRepository)(nil)

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

const (
	sessionFieldID        = "id"
	sessionFieldUserID    = "user_id"
	sessionFieldCreatedAt = "created_at"
	sessionFieldExpiresAt = "expires_at"
)

func sessionColumns() []string {
	return []string{
		sessionFieldID,
		sessionFieldUserID,
		sessionFieldCreatedAt,
		sessionFieldExpiresAt,
	}
}

func scanSession(row sq.RowScanner) (*auth.Session, error) {
	var session auth.Session

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return &session, nil
}

func (repo *SessionRepository) Insert(ctx context.Context, session *auth.Session) error {
	q := sq.Insert(tableSessions).
		Columns(sessionColumns()...).
		Values(session.ID, session.UserID, session.CreatedAt, session.ExpiresAt)

	q = q.RunWith(repo.db)

	_, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to exec insert: %w", err)
	}

	return nil
}

func (repo *SessionRepository) Find(ctx context.Context, id string) (*auth.Session, error) {
	q := sq.Select(sessionColumns()...).
		From(tableSessions).
		Where(sq.Eq{sessionFieldID: id})

	q = q.RunWith(repo.db)

	row := q.QueryRowContext(ctx)

	session, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &auth.SessionNotFoundError{ID: id}
		}

		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	return session, nil
}

func (repo *SessionRepository) Delete(ctx context.Context, id string) error {
	q := sq.Delete(tableSessions).
		Where(sq.Eq{sessionFieldID: id})

	q = q.RunWith(repo.db)

	result, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to exec delete: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &auth.SessionNotFoundError{ID: id}
	}

	return nil
}
