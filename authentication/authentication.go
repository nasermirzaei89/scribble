package authentication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authcontext "github.com/nasermirzaei89/scribble/authentication/context"
	"github.com/nasermirzaei89/scribble/authorization"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	authzClient *authorization.Client
	bloomFilter *BloomFilter
}

func NewService(userRepo UserRepository, sessionRepo SessionRepository, authzClient *authorization.Client) *Service {
	return &Service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		authzClient: authzClient,
	}
}

func (svc *Service) LoadBloomFilter(ctx context.Context, minCapacity uint, falsePositiveRate float64) error {
	usernames, err := svc.userRepo.ListUsernames(ctx)
	if err != nil {
		return fmt.Errorf("failed to list usernames for bloom filter: %w", err)
	}

	capacity := uint(len(usernames))
	if capacity < minCapacity {
		capacity = minCapacity
	}

	bf := NewBloomFilter(capacity, falsePositiveRate)
	for _, u := range usernames {
		bf.Add(u)
	}

	svc.bloomFilter = bf
	return nil
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

	if svc.bloomFilter != nil && svc.bloomFilter.Test(username) {
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
		var alreadyExistsErr *UserAlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			if svc.bloomFilter != nil {
				svc.bloomFilter.Add(username)
			}

			return alreadyExistsErr
		}

		return fmt.Errorf("failed to register user: %w", err)
	}

	if svc.bloomFilter != nil {
		svc.bloomFilter.Add(username)
	}

	err = svc.authzClient.AddToGroup(ctx, user.ID, authcontext.Authenticated)
	if err != nil {
		return fmt.Errorf("failed to add user to authenticated group: %w", err)
	}

	return nil
}

var ErrInvalidCredentials = errors.New("invalid credentials")

const defaultSessionDuration = 30 * 24 * time.Hour

func (svc *Service) Login(ctx context.Context, username, password string) (*Session, error) {
	// TODO: validate username and password
	user, err := svc.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if _, ok := errors.AsType[*UserByUsernameNotFoundError](err); ok {
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
