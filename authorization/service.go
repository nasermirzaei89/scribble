package authorization

import (
	"context"
	"fmt"
)

type Service struct {
	authzProvider AuthorizationProvider
}

type AuthorizationProvider interface {
	CheckAccess(ctx context.Context, req CheckAccessRequest) (res *CheckAccessResponse, err error)
	AddPolicy(ctx context.Context, reqs ...AddPolicyRequest) (err error)
	AddToGroup(ctx context.Context, sub string, groups ...string) (err error)
	RemovePolicy(ctx context.Context, reqs ...RemovePolicyRequest) (err error)
	RemoveFromGroup(ctx context.Context, sub string, groups ...string) (err error)
}

func NewService(authzProvider AuthorizationProvider) (*Service, error) {
	// TODO: validate arguments
	return &Service{
		authzProvider: authzProvider,
	}, nil
}

type CheckAccessRequest struct {
	Subject string
	Domain  string
	Object  string
	Action  string
}

type CheckAccessResponse struct {
	// Allowed is required. True if the action would be allowed, false otherwise.
	Allowed bool
	// Denied is optional. True if the action would be denied, otherwise false.
	// If both allowed is false and denied is false, then the authorizer has no opinion on whether to authorize the
	// action.
	// Denied may not be true if Allowed is true.
	Denied bool
	// Reason is optional. It indicates why a request was allowed or denied.
	Reason string
}

type AccessDeniedError struct {
	Subject string
	Domain  string
	Object  string
	Action  string
}

func (err AccessDeniedError) Error() string {
	if err.Object != "" {
		return fmt.Sprintf(
			"access denied for subject '%s' and domain '%s' and object '%s' and action '%s'",
			err.Subject,
			err.Domain,
			err.Object,
			err.Action,
		)
	}

	return fmt.Sprintf(
		"access denied for subject '%s' and domain '%s' and action '%s'",
		err.Subject,
		err.Domain,
		err.Action,
	)
}

func (svc *Service) CheckAccess(ctx context.Context, req CheckAccessRequest) (*CheckAccessResponse, error) {
	res, err := svc.authzProvider.CheckAccess(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	return res, nil
}

type AddPolicyRequest struct {
	Subject string
	Domain  string
	Object  string
	Action  string
}

func (svc *Service) AddPolicy(ctx context.Context, reqs ...AddPolicyRequest) error {
	err := svc.authzProvider.AddPolicy(ctx, reqs...)
	if err != nil {
		return fmt.Errorf("failed to add policies: %w", err)
	}

	return nil
}

func (svc *Service) AddToGroup(ctx context.Context, sub string, groups ...string) error {
	err := svc.authzProvider.AddToGroup(ctx, sub, groups...)
	if err != nil {
		return fmt.Errorf("failed to add grouping policies: %w", err)
	}

	return nil
}

type RemovePolicyRequest struct {
	Subject string
	Domain  string
	Object  string
	Action  string
}

func (svc *Service) RemovePolicy(ctx context.Context, reqs ...RemovePolicyRequest) error {
	err := svc.authzProvider.RemovePolicy(ctx, reqs...)
	if err != nil {
		return fmt.Errorf("failed to remove policies: %w", err)
	}

	return nil
}

func (svc *Service) RemoveFromGroup(ctx context.Context, sub string, groups ...string) error {
	err := svc.authzProvider.RemoveFromGroup(ctx, sub, groups...)
	if err != nil {
		return fmt.Errorf("failed to remove grouping policies: %w", err)
	}

	return nil
}
