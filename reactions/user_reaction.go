package reactions

import (
	"context"
	"fmt"
	"time"
)

type TargetType string

const (
	TargetTypePost    TargetType = "post"
	TargetTypeComment TargetType = "comment"
)

func (targetType TargetType) IsValid() bool {
	switch targetType {
	case TargetTypePost, TargetTypeComment:
		return true
	default:
		return false
	}
}

type UserReaction struct {
	TargetType TargetType
	TargetID   string
	UserID     string
	Emoji      string
	CreatedAt  time.Time
}

type UserReactionRepository interface {
	FindByUserTarget(
		ctx context.Context,
		targetType TargetType,
		targetID string,
		userID string,
	) (reaction *UserReaction, err error)
	Upsert(ctx context.Context, reaction *UserReaction) (err error)
	DeleteByUserTarget(ctx context.Context, targetType TargetType, targetID string, userID string) (err error)
	CountByTarget(ctx context.Context, targetType TargetType, targetID string) (counts map[string]int, err error)
}

type UserReactionNotFoundError struct {
	TargetType TargetType
	TargetID   string
	UserID     string
}

func (err UserReactionNotFoundError) Error() string {
	return fmt.Sprintf(
		"reaction for user %q on %s:%q not found",
		err.UserID,
		err.TargetType,
		err.TargetID,
	)
}

type InvalidTargetTypeError struct {
	TargetType TargetType
}

func (err InvalidTargetTypeError) Error() string {
	return fmt.Sprintf("invalid target type: %q", err.TargetType)
}

type InvalidEmojiError struct {
	TargetType TargetType
	TargetID   string
	Emoji      string
	Allowed    []string
}

func (err InvalidEmojiError) Error() string {
	return fmt.Sprintf(
		"emoji %q is not allowed for %s:%q; allowed: %v",
		err.Emoji,
		err.TargetType,
		err.TargetID,
		err.Allowed,
	)
}
