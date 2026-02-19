package contents

import (
	"context"
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
	List(ctx context.Context) (posts []*Post, err error)
}
