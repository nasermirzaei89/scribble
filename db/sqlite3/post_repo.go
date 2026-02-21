package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/nasermirzaei89/scribble/contents"
)

const tablePosts = "posts"

type PostRepository struct {
	db *sql.DB
}

var _ contents.PostRepository = (*PostRepository)(nil)

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

const (
	postFieldID        = "id"
	postFieldAuthorID  = "author_id"
	postFieldContent   = "content"
	postFieldCreatedAt = "created_at"
)

func postColumns() []string {
	return []string{
		postFieldID,
		postFieldAuthorID,
		postFieldContent,
		postFieldCreatedAt,
	}
}

func scanPost(row sq.RowScanner) (*contents.Post, error) {
	var post contents.Post

	err := row.Scan(
		&post.ID,
		&post.AuthorID,
		&post.Content,
		&post.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return &post, nil
}

func (repo *PostRepository) Insert(ctx context.Context, post *contents.Post) error {
	q := sq.Insert(tablePosts).
		Columns(postColumns()...).
		Values(post.ID, post.AuthorID, post.Content, post.CreatedAt)

	q = q.RunWith(repo.db)

	_, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to exec insert: %w", err)
	}

	return nil
}

func (repo *PostRepository) Find(ctx context.Context, postID string) (*contents.Post, error) {
	q := sq.Select(postColumns()...).
		From(tablePosts).
		Where(sq.Eq{postFieldID: postID})

	q = q.RunWith(repo.db)

	row := q.QueryRowContext(ctx)

	post, err := scanPost(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contents.PostNotFoundError{ID: postID}
		}

		return nil, fmt.Errorf("failed to scan post: %w", err)
	}

	return post, nil
}

func (repo *PostRepository) List(ctx context.Context) ([]*contents.Post, error) {
	q := sq.Select(postColumns()...).
		From(tablePosts).
		OrderBy(postFieldID)

	q = q.RunWith(repo.db)

	rows, err := q.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			slog.ErrorContext(ctx, "failed to close rows", "error", err)
		}
	}()

	posts := make([]*contents.Post, 0)

	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		posts = append(posts, post)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return posts, nil
}
