package public

import (
	"net/http"
	"testing"

	"deadcomments/internal/domain"
)

func TestCreateMessageExplainsModerationOutcome(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.CommentStatus
		reason  string
		code    int
		message string
	}{
		{
			name:    "approved",
			status:  domain.CommentApproved,
			reason:  "auto moderation",
			code:    http.StatusCreated,
			message: "Comment posted.",
		},
		{
			name:    "pending",
			status:  domain.CommentPending,
			reason:  "manual moderation",
			code:    http.StatusAccepted,
			message: "Comment submitted and waiting for moderation.",
		},
		{
			name:    "rejected rate limit",
			status:  domain.CommentRejected,
			reason:  "rate limit",
			code:    http.StatusForbidden,
			message: "Comment rejected: too many comments were submitted recently. Please try again later.",
		},
		{
			name:    "spam links",
			status:  domain.CommentSpam,
			reason:  "too many links",
			code:    http.StatusForbidden,
			message: "Comment rejected by spam protection: too many links.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusForCreatedComment(tt.status); got != tt.code {
				t.Fatalf("expected status code %d, got %d", tt.code, got)
			}
			if got := createMessage(tt.status, tt.reason); got != tt.message {
				t.Fatalf("expected message %q, got %q", tt.message, got)
			}
		})
	}
}
