package reactions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

type Service struct {
	userReactionRepo UserReactionRepository
}

func NewService(userReactionRepo UserReactionRepository) *Service {
	return &Service{userReactionRepo: userReactionRepo}
}

type ReactionOption struct {
	Emoji     string
	Count     int
	Selected  bool
	Available bool
}

type TargetReactions struct {
	TargetType TargetType
	TargetID   string
	Options    []ReactionOption
}

func (svc *Service) AllowedEmojis(
	_ context.Context,
	targetType TargetType,
	_ string,
) ([]string, error) {
	if !targetType.IsValid() {
		return nil, InvalidTargetTypeError{TargetType: targetType}
	}

	return []string{"👍", "👎", "😂"}, nil
}

func (svc *Service) ToggleReaction(
	ctx context.Context,
	targetType TargetType,
	targetID string,
	userID string,
	emoji string,
) error {
	if !targetType.IsValid() {
		return InvalidTargetTypeError{TargetType: targetType}
	}

	allowedEmojis, err := svc.AllowedEmojis(ctx, targetType, targetID)
	if err != nil {
		return fmt.Errorf("failed to get allowed emojis: %w", err)
	}

	if !slices.Contains(allowedEmojis, emoji) {
		return InvalidEmojiError{
			TargetType: targetType,
			TargetID:   targetID,
			Emoji:      emoji,
			Allowed:    allowedEmojis,
		}
	}

	existingReaction, err := svc.userReactionRepo.FindByUserTarget(ctx, targetType, targetID, userID)
	if err != nil {
		var notFoundErr *UserReactionNotFoundError
		if !errors.As(err, &notFoundErr) {
			return fmt.Errorf("failed to get existing reaction: %w", err)
		}
	}

	if existingReaction != nil && existingReaction.Emoji == emoji {
		err = svc.userReactionRepo.DeleteByUserTarget(ctx, targetType, targetID, userID)
		if err != nil {
			return fmt.Errorf("failed to remove reaction: %w", err)
		}

		return nil
	}

	userReaction := &UserReaction{
		TargetType: targetType,
		TargetID:   targetID,
		UserID:     userID,
		Emoji:      emoji,
		CreatedAt:  time.Now(),
	}

	err = svc.userReactionRepo.Upsert(ctx, userReaction)
	if err != nil {
		return fmt.Errorf("failed to set reaction: %w", err)
	}

	return nil
}

func (svc *Service) GetTargetReactions(
	ctx context.Context,
	targetType TargetType,
	targetID string,
	currentUserID *string,
) (*TargetReactions, error) {
	allowedEmojis, err := svc.AllowedEmojis(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed emojis: %w", err)
	}

	counts, err := svc.userReactionRepo.CountByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts by target: %w", err)
	}

	selectedEmoji := ""

	if currentUserID != nil && *currentUserID != "" {
		userReaction, err := svc.userReactionRepo.FindByUserTarget(ctx, targetType, targetID, *currentUserID)
		if err != nil {
			var notFoundErr *UserReactionNotFoundError
			if !errors.As(err, &notFoundErr) {
				return nil, fmt.Errorf("failed to get user reaction: %w", err)
			}
		}

		if userReaction != nil {
			selectedEmoji = userReaction.Emoji
		}
	}

	options := make([]ReactionOption, 0, len(allowedEmojis))
	availableEmojiSet := make(map[string]struct{}, len(allowedEmojis))

	for _, emoji := range allowedEmojis {
		availableEmojiSet[emoji] = struct{}{}

		options = append(options, ReactionOption{
			Emoji:     emoji,
			Count:     counts[emoji],
			Selected:  emoji == selectedEmoji,
			Available: true,
		})
	}

	additionalEmojis := make([]string, 0)

	for emoji, count := range counts {
		if count <= 0 {
			continue
		}

		if _, ok := availableEmojiSet[emoji]; ok {
			continue
		}

		additionalEmojis = append(additionalEmojis, emoji)
	}

	slices.Sort(additionalEmojis)

	for _, emoji := range additionalEmojis {
		options = append(options, ReactionOption{
			Emoji:     emoji,
			Count:     counts[emoji],
			Selected:  emoji == selectedEmoji,
			Available: false,
		})
	}

	return &TargetReactions{
		TargetType: targetType,
		TargetID:   targetID,
		Options:    options,
	}, nil
}
