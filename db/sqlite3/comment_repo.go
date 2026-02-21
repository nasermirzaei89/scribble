package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/nasermirzaei89/scribble/discuss"
)

const tableComments = "comments"

type CommentRepository struct {
	db *sql.DB
}

var _ discuss.CommentRepository = (*CommentRepository)(nil)

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

const (
	commentFieldID        = "id"
	commentFieldPostID    = "post_id"
	commentFieldAuthorID  = "author_id"
	commentFieldReplyTo   = "reply_to"
	commentFieldContent   = "content"
	commentFieldCreatedAt = "created_at"
)

func commentColumns() []string {
	return []string{
		commentFieldID,
		commentFieldPostID,
		commentFieldAuthorID,
		commentFieldReplyTo,
		commentFieldContent,
		commentFieldCreatedAt,
	}
}

func scanComment(row sq.RowScanner) (*discuss.Comment, error) {
	var comment discuss.Comment

	err := row.Scan(
		&comment.ID,
		&comment.PostID,
		&comment.AuthorID,
		&comment.ReplyTo,
		&comment.Content,
		&comment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return &comment, nil
}

func (repo *CommentRepository) Insert(ctx context.Context, comment *discuss.Comment) error {
	q := sq.Insert(tableComments).
		Columns(commentColumns()...).
		Values(
			comment.ID,
			comment.PostID,
			comment.AuthorID,
			comment.ReplyTo,
			comment.Content,
			comment.CreatedAt,
		)

	q = q.RunWith(repo.db)

	_, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to exec insert: %w", err)
	}

	return nil
}

func (repo *CommentRepository) List(
	ctx context.Context,
	params *discuss.ListCommentsParams,
) ([]*discuss.Comment, error) {
	query := sq.Select(commentColumns()...).
		From(tableComments).
		OrderBy(commentFieldCreatedAt + " ASC")

	if params.PostID != "" {
		query = query.Where(sq.Eq{commentFieldPostID: params.PostID})
	}

	query = query.RunWith(repo.db)

	rows, err := query.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			slog.ErrorContext(ctx, "failed to close rows", "error", err)
		}
	}()

	comments := make([]*discuss.Comment, 0)

	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan comment failed: %w", err)
		}

		comments = append(comments, comment)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return comments, nil
}
