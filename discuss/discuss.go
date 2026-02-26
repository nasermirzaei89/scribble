package discuss

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nasermirzaei89/scribble/authorization"
)

const ServiceName = "github.com/nasermirzaei89/scribble/discuss"

type Service interface {
	CreateComment(ctx context.Context, req CreateCommentRequest) (*Comment, error)
	ListComments(ctx context.Context, postID string) ([]*Comment, error)
	CountComments(ctx context.Context, postID string) (int, error)
}

type BaseService struct {
	commentRepo CommentRepository
}

var _ Service = (*BaseService)(nil)

func NewService(commentRepo CommentRepository, authzClient *authorization.Client) Service { //nolint:ireturn
	return NewAuthorizationMiddleware(authzClient, NewBaseService(commentRepo))
}

func NewBaseService(commentRepo CommentRepository) *BaseService {
	return &BaseService{
		commentRepo: commentRepo,
	}
}

type CreateCommentRequest struct {
	PostID   string
	AuthorID string
	Content  string
	ReplyTo  string
}

func (svc *BaseService) CreateComment(ctx context.Context, req CreateCommentRequest) (*Comment, error) {
	var replyTo *string
	if req.ReplyTo != "" {
		replyTo = &req.ReplyTo
	}

	comment := &Comment{
		ID:        uuid.NewString(),
		PostID:    req.PostID,
		AuthorID:  req.AuthorID,
		ReplyTo:   replyTo,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	err := svc.commentRepo.Insert(ctx, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	return comment, nil
}

func (svc *BaseService) ListComments(ctx context.Context, postID string) ([]*Comment, error) {
	comments, err := svc.commentRepo.List(ctx, &ListCommentsParams{PostID: postID})
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	return comments, nil
}

func (svc *BaseService) CountComments(ctx context.Context, postID string) (int, error) {
	count, err := svc.commentRepo.Count(ctx, &CountCommentsParams{PostID: postID})
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return count, nil
}
