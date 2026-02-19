package auth

import (
	"context"
	"fmt"
	"time"
)

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionRepository interface {
	Insert(ctx context.Context, session *Session) (err error)
	Find(ctx context.Context, id string) (session *Session, err error)
	Delete(ctx context.Context, id string) (err error)
}

type SessionNotFoundError struct {
	ID string
}

func (err SessionNotFoundError) Error() string {
	return fmt.Sprintf("session with id %q not found", err.ID)
}

type SessionExpiredError struct {
	ID string
}

func (err SessionExpiredError) Error() string {
	return fmt.Sprintf("session with id %q has expired", err.ID)
}
