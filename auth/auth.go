package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authcontext "github.com/nasermirzaei89/scribble/auth/context"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
}

func NewService(userRepo UserRepository, sessionRepo SessionRepository) *Service {
	return &Service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func HashPassword(password string) (string, error) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(bcryptHash), nil
}

func (svc *Service) Register(ctx context.Context, username, password string) error {
	// TODO: validate username and password
	_, err := svc.userRepo.FindByUsername(ctx, username)
	if err != nil {
		var userByUsernameNotFoundErr *UserByUsernameNotFoundError
		if !errors.As(err, &userByUsernameNotFoundErr) {
			return fmt.Errorf("failed to check if username already exists: %w", err)
		}
	} else {
		return &UserAlreadyExistsError{Username: username}
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: passwordHash,
		RegisteredAt: time.Now(),
	}

	err = svc.userRepo.Insert(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	return nil
}

var ErrInvalidCredentials = errors.New("invalid credentials")

const defaultSessionDuration = 30 * 24 * time.Hour

func (svc *Service) Login(ctx context.Context, username, password string) (*Session, error) {
	// TODO: validate username and password
	user, err := svc.userRepo.FindByUsername(ctx, username)
	if err != nil {
		var userByUsernameNotFoundErr *UserByUsernameNotFoundError
		if errors.As(err, &userByUsernameNotFoundErr) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("failed to compare password hash: %w", err)
	}

	timeNow := time.Now()

	session := &Session{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		CreatedAt: timeNow,
		ExpiresAt: timeNow.Add(defaultSessionDuration),
	}

	err = svc.sessionRepo.Insert(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (svc *Service) Logout(ctx context.Context, sessionID string) error {
	err := svc.sessionRepo.Delete(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (svc *Service) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := svc.sessionRepo.Find(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if session.ExpiresAt.Before(time.Now()) {
		// TODO: delete expired session
		return nil, &SessionExpiredError{ID: sessionID}
	}

	return session, nil
}

func (svc *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	user, err := svc.userRepo.Find(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	user.PasswordHash = "" // clear password hash before returning user

	return user, nil
}

func (svc *Service) GetCurrentUser(ctx context.Context) (*User, error) {
	sub := authcontext.GetSubject(ctx)
	if sub == authcontext.Anonymous {
		return nil, ErrCurrentUserNotFound
	}

	user, err := svc.GetUser(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	user.PasswordHash = "" // clear password hash before returning user

	return user, nil
}
