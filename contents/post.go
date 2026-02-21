package contents

import (
	"context"
	"fmt"
	"time"
)

type Post struct {
	ID        string
	AuthorID  string
	Content   string
	CreatedAt time.Time
}

type PostRepository interface {
	Insert(ctx context.Context, post *Post) (err error)
	Find(ctx context.Context, postID string) (post *Post, err error)
	List(ctx context.Context) (posts []*Post, err error)
}

type PostNotFoundError struct {
	ID string
}

func (err PostNotFoundError) Error() string {
	return fmt.Sprintf("post with id %q not found", err.ID)
}
