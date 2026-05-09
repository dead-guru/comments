package domain

import "time"

type CommentReaction struct {
	ID          int64
	CommentID   string
	ReactionKey string
	ActorHash   *string
	CreatedAt   time.Time
}
