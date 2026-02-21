package discuss

import (
	"context"
	"time"
)

type Comment struct {
	ID        string
	PostID    string
	AuthorID  string
	ReplyTo   *string
	Content   string
	CreatedAt time.Time
}

type CommentRepository interface {
	Insert(ctx context.Context, comment *Comment) (err error)
	List(ctx context.Context, params *ListCommentsParams) (comments []*Comment, err error)
}

type ListCommentsParams struct {
	PostID string
}
