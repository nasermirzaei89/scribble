package contents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	postRepo PostRepository
}

func NewService(postRepo PostRepository) *Service {
	return &Service{
		postRepo: postRepo,
	}
}

type CreatePostRequest struct {
	AuthorID string
	Content  string
}

func (svc *Service) CreatePost(ctx context.Context, req CreatePostRequest) (*Post, error) {
	post := &Post{
		ID:        uuid.NewString(),
		AuthorID:  req.AuthorID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	err := svc.postRepo.Insert(ctx, post)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return post, nil
}

func (svc *Service) ListPosts(ctx context.Context) ([]*Post, error) {
	posts, err := svc.postRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	return posts, nil
}
