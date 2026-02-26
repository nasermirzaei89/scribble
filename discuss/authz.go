package discuss

import (
	"context"
	"fmt"

	"github.com/nasermirzaei89/scribble/authorization"
)

const (
	ActionCreateComment = "createComment"
	ActionListComments  = "listComments"
	ActionCountComments = "countComments"
)

type AuthorizationMiddleware struct {
	authzClient *authorization.Client
	next        Service
}

var _ Service = (*AuthorizationMiddleware)(nil)

func NewAuthorizationMiddleware(authzClient *authorization.Client, next Service) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		authzClient: authzClient,
		next:        next,
	}
}

func (mw *AuthorizationMiddleware) CreateComment(ctx context.Context, req CreateCommentRequest) (*Comment, error) {
	err := mw.authzClient.CheckAccess(ctx, ServiceName, req.PostID, ActionCreateComment)
	if err != nil {
		return nil, fmt.Errorf("failed to check authorization: %w", err)
	}

	comment, err := mw.next.CreateComment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call next method: %w", err)
	}

	return comment, nil
}

func (mw *AuthorizationMiddleware) ListComments(ctx context.Context, postID string) ([]*Comment, error) {
	err := mw.authzClient.CheckAccess(ctx, ServiceName, postID, ActionListComments)
	if err != nil {
		return nil, fmt.Errorf("failed to check authorization: %w", err)
	}

	comments, err := mw.next.ListComments(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to call next method: %w", err)
	}

	return comments, nil
}

func (mw *AuthorizationMiddleware) CountComments(ctx context.Context, postID string) (int, error) {
	err := mw.authzClient.CheckAccess(ctx, ServiceName, postID, ActionCountComments)
	if err != nil {
		return 0, fmt.Errorf("failed to check authorization: %w", err)
	}

	count, err := mw.next.CountComments(ctx, postID)
	if err != nil {
		return 0, fmt.Errorf("failed to call next method: %w", err)
	}

	return count, nil
}
