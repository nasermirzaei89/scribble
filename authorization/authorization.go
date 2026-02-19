package authorization

import (
	"context"
	"fmt"
)

// Provider is the interface that wraps the authorization policy enforcement.
type Provider interface {
	Enforce(sub, obj, res, act string) (bool, error)
	AddGroupingPolicy(sub, group string) error
}

// Service wraps a Provider.
type Service struct {
	provider Provider
}

// NewService creates a new authorization Service backed by the given Provider.
func NewService(provider Provider) (*Service, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider must not be nil")
	}

	return &Service{provider: provider}, nil
}

// Enforce delegates to the underlying Provider.
func (svc *Service) Enforce(sub, obj, res, act string) (bool, error) {
	return svc.provider.Enforce(sub, obj, res, act)
}

// AddGroupingPolicy adds a subject → group mapping to the policy.
func (svc *Service) AddGroupingPolicy(sub, group string) error {
	return svc.provider.AddGroupingPolicy(sub, group)
}

// AccessDeniedError is returned when a subject does not have permission to
// perform an action.
type AccessDeniedError struct {
	Sub    string
	Obj    string
	Res    string
	Action string
}

func (e *AccessDeniedError) Error() string {
	return fmt.Sprintf("access denied: sub=%q obj=%q res=%q action=%q", e.Sub, e.Obj, e.Res, e.Action)
}

// Client is a convenience wrapper around Service that reads the current
// subject from a context.Context.
type Client struct {
	svc *Service
}

// NewClient creates a new Client backed by the given Service.
func NewClient(svc *Service) *Client {
	return &Client{svc: svc}
}

// CheckAccess checks whether the current context subject may perform action on
// the given service (obj) and resource (res). It returns an *AccessDeniedError
// when access is not allowed.
func (c *Client) CheckAccess(ctx context.Context, obj, res, action string) error {
	sub := subjectFromContext(ctx)

	allowed, err := c.svc.Enforce(sub, obj, res, action)
	if err != nil {
		return fmt.Errorf("failed to enforce authorization policy: %w", err)
	}

	if !allowed {
		return &AccessDeniedError{Sub: sub, Obj: obj, Res: res, Action: action}
	}

	return nil
}

// AddToGroup adds the given subject to the named group in the policy store.
func (c *Client) AddToGroup(ctx context.Context, sub, group string) error {
	err := c.svc.AddGroupingPolicy(sub, group)
	if err != nil {
		return fmt.Errorf("failed to add subject %q to group %q: %w", sub, group, err)
	}

	return nil
}
