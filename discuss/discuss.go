package discuss

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	commentRepo CommentRepository
}

func NewService(commentRepo CommentRepository) *Service {
	return &Service{
		commentRepo: commentRepo,
	}
}

type CreateCommentRequest struct {
	PostID   string
	AuthorID string
	Content  string
	ReplyTo  string
}

func (svc *Service) CreateComment(ctx context.Context, req CreateCommentRequest) (*Comment, error) {
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

func (svc *Service) ListComments(ctx context.Context, postID string) ([]*Comment, error) {
	comments, err := svc.commentRepo.List(ctx, &ListCommentsParams{PostID: postID})
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	return comments, nil
}
