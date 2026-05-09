package domain

import "time"

type CommentRating struct {
	ID        int64
	CommentID string
	ActorHash *string
	Value     int
	CreatedAt time.Time
}

type PageRating struct {
	ID        int64
	PageID    int64
	ActorHash *string
	Value     int
	CreatedAt time.Time
}
