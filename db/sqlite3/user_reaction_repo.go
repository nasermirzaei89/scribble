package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/nasermirzaei89/scribble/reactions"
)

const tableReactions = "reactions"

type UserReactionRepository struct {
	db *sql.DB
}

var _ reactions.UserReactionRepository = (*UserReactionRepository)(nil)

func NewUserReactionRepository(db *sql.DB) *UserReactionRepository {
	return &UserReactionRepository{db: db}
}

const (
	userReactionFieldTargetType = "target_type"
	userReactionFieldTargetID   = "target_id"
	userReactionFieldUserID     = "user_id"
	userReactionFieldEmoji      = "emoji"
	userReactionFieldCreatedAt  = "created_at"
)

func reactionColumns() []string {
	return []string{
		userReactionFieldTargetType,
		userReactionFieldTargetID,
		userReactionFieldUserID,
		userReactionFieldEmoji,
		userReactionFieldCreatedAt,
	}
}

func scanUserReaction(row sq.RowScanner) (*reactions.UserReaction, error) {
	var reaction reactions.UserReaction

	err := row.Scan(
		&reaction.TargetType,
		&reaction.TargetID,
		&reaction.UserID,
		&reaction.Emoji,
		&reaction.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan reaction row: %w", err)
	}

	return &reaction, nil
}

func (repo *UserReactionRepository) FindByUserTarget(
	ctx context.Context,
	targetType reactions.TargetType,
	targetID string,
	userID string,
) (*reactions.UserReaction, error) {
	q := sq.Select(reactionColumns()...).
		From(tableReactions).
		Where(sq.Eq{
			userReactionFieldTargetType: targetType,
			userReactionFieldTargetID:   targetID,
			userReactionFieldUserID:     userID,
		})

	q = q.RunWith(repo.db)

	reaction, err := scanUserReaction(q.QueryRowContext(ctx))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &reactions.UserReactionNotFoundError{
				TargetType: targetType,
				TargetID:   targetID,
				UserID:     userID,
			}
		}

		return nil, fmt.Errorf("failed to find reaction by user target: %w", err)
	}

	return reaction, nil
}

func (repo *UserReactionRepository) Upsert(ctx context.Context, reaction *reactions.UserReaction) error {
	query := fmt.Sprintf(`
INSERT INTO %s (target_type, target_id, user_id, emoji, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(target_type, target_id, user_id)
DO UPDATE SET
    emoji = excluded.emoji,
    created_at = excluded.created_at
`, tableReactions)

	_, err := repo.db.ExecContext(
		ctx,
		query,
		reaction.TargetType,
		reaction.TargetID,
		reaction.UserID,
		reaction.Emoji,
		reaction.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert reaction: %w", err)
	}

	return nil
}

func (repo *UserReactionRepository) DeleteByUserTarget(
	ctx context.Context,
	targetType reactions.TargetType,
	targetID string,
	userID string,
) error {
	q := sq.Delete(tableReactions).
		Where(sq.Eq{
			userReactionFieldTargetType: targetType,
			userReactionFieldTargetID:   targetID,
			userReactionFieldUserID:     userID,
		}).
		RunWith(repo.db)

	_, err := q.ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	return nil
}

func (repo *UserReactionRepository) CountByTarget(
	ctx context.Context,
	targetType reactions.TargetType,
	targetID string,
) (map[string]int, error) {
	q := sq.Select(userReactionFieldEmoji, "COUNT(*)").
		From(tableReactions).
		Where(sq.Eq{
			userReactionFieldTargetType: targetType,
			userReactionFieldTargetID:   targetID,
		}).
		GroupBy(userReactionFieldEmoji).
		RunWith(repo.db)

	rows, err := q.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query reaction counts: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			slog.ErrorContext(ctx, "failed to close reaction rows", "error", err)
		}
	}()

	counts := make(map[string]int)

	for rows.Next() {
		var emoji string

		var count int

		err := rows.Scan(&emoji, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reaction count row: %w", err)
		}

		counts[emoji] = count
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate reaction count rows: %w", err)
	}

	return counts, nil
}
