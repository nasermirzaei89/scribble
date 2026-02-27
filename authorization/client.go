package authorization

import (
	"context"
	"fmt"

	authcontext "github.com/nasermirzaei89/scribble/authentication/context"
)

type Client struct {
	authzSvc *Service
}

func NewClient(authzSvc *Service) *Client {
	// TODO: validate arguments
	return &Client{
		authzSvc: authzSvc,
	}
}

// CheckAccess checks if the current user in the context has permission to perform the action on the object within the
// domain.
func (c *Client) CheckAccess(ctx context.Context, domain, object, action string) error {
	subject := authcontext.GetSubject(ctx)

	res, err := c.authzSvc.CheckAccess(ctx, CheckAccessRequest{
		Subject: subject,
		Domain:  domain,
		Object:  object,
		Action:  action,
	})
	if err != nil {
		return fmt.Errorf("error on check permission: %w", err)
	}

	if !res.Allowed {
		return &AccessDeniedError{
			Subject: subject,
			Domain:  domain,
			Object:  object,
			Action:  action,
		}
	}

	return nil
}

func (c *Client) CanI(ctx context.Context, domain, object, action string) bool {
	return c.Can(ctx, authcontext.GetSubject(ctx), domain, object, action)
}

func (c *Client) Can(ctx context.Context, subject, domain, object, action string) bool {
	res, err := c.authzSvc.CheckAccess(ctx, CheckAccessRequest{
		Subject: subject,
		Domain:  domain,
		Object:  object,
		Action:  action,
	})

	return err == nil && res.Allowed
}

func (c *Client) AddPolicyForSubject(ctx context.Context, subject, domain, object string, action ...string) error {
	reqs := make([]AddPolicyRequest, 0, len(action))

	for i := range action {
		reqs = append(reqs, AddPolicyRequest{
			Subject: subject,
			Domain:  domain,
			Object:  object,
			Action:  action[i],
		})
	}

	err := c.authzSvc.AddPolicy(ctx, reqs...)
	if err != nil {
		return fmt.Errorf("error on add policy: %w", err)
	}

	return nil
}

func (c *Client) RemovePolicyForSubject(ctx context.Context, subject, domain, object string, action ...string) error {
	reqs := make([]RemovePolicyRequest, 0, len(action))

	for i := range action {
		reqs = append(reqs, RemovePolicyRequest{
			Subject: subject,
			Domain:  domain,
			Object:  object,
			Action:  action[i],
		})
	}

	err := c.authzSvc.RemovePolicy(ctx, reqs...)
	if err != nil {
		return fmt.Errorf("error on remove policy: %w", err)
	}

	return nil
}

func (c *Client) AddToGroup(ctx context.Context, sub string, group ...string) error {
	err := c.authzSvc.AddToGroup(ctx, sub, group...)
	if err != nil {
		return fmt.Errorf("error on add to group: %w", err)
	}

	return nil
}

func (c *Client) RemoveFromGroup(ctx context.Context, sub string, group ...string) error {
	err := c.authzSvc.RemoveFromGroup(ctx, sub, group...)
	if err != nil {
		return fmt.Errorf("error on remove from group: %w", err)
	}

	return nil
}
