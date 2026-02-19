package auth

import (
	"context"
	"fmt"
	"time"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	RegisteredAt time.Time
}

type UserRepository interface {
	Insert(ctx context.Context, user *User) (err error)
	FindByUsername(ctx context.Context, username string) (user *User, err error)
}

type UserByUsernameNotFoundError struct {
	Username string
}

func (err UserByUsernameNotFoundError) Error() string {
	return fmt.Sprintf("user with username %q not found", err.Username)
}

type UserAlreadyExistsError struct {
	Username string
}

func (err UserAlreadyExistsError) Error() string {
	return fmt.Sprintf("user with username %q already exists", err.Username)
}
