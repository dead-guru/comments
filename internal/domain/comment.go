package domain

import "time"

type CommentStatus string
type CommentSort string

const (
	CommentPending  CommentStatus = "pending"
	CommentApproved CommentStatus = "approved"
	CommentRejected CommentStatus = "rejected"
	CommentSpam     CommentStatus = "spam"
	CommentDeleted  CommentStatus = "deleted"
)

const (
	CommentSortOldest CommentSort = "oldest"
	CommentSortNewest CommentSort = "newest"
	CommentSortBest   CommentSort = "best"
)

type Comment struct {
	ID                string
	SiteID            int64
	PageID            int64
	ParentID          *string
	RootID            *string
	Depth             int
	Path              string
	AuthorName        string
	AuthorDisplayName string
	AuthorEmailHash   *string
	AuthorAvatarHash  *string
	AuthorWebsite     *string
	IdentityID        *int64
	TripcodePublic    *string
	TripcodeKind      TripcodeKind
	BadgeType         *BadgeType
	BadgeLabel        *string
	BodyMarkdown      string
	BodyHTML          string
	Status            CommentStatus
	IPHash            *string
	UserAgentHash     *string
	MetadataJSON      *string
	ModerationReason  *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	EditedAt          *time.Time
	Children          []*Comment
	ReplyingToAuthor  *string
}

type PublicComment struct {
	ID                string           `json:"id"`
	ParentID          *string          `json:"parent_id"`
	AuthorName        string           `json:"author_name"`
	AuthorDisplayName string           `json:"author_display_name"`
	AuthorWebsite     *string          `json:"author_website"`
	AuthorAvatarHash  *string          `json:"author_avatar_hash"`
	TripcodePublic    *string          `json:"tripcode_public"`
	TripcodeKind      TripcodeKind     `json:"tripcode_kind"`
	BadgeType         *BadgeType       `json:"badge_type"`
	BadgeLabel        *string          `json:"badge_label"`
	BodyHTML          string           `json:"body_html"`
	Status            CommentStatus    `json:"status"`
	CreatedAt         time.Time        `json:"created_at"`
	EditedAt          *time.Time       `json:"edited_at"`
	ReplyingToAuthor  *string          `json:"replying_to_author"`
	Children          []*PublicComment `json:"children"`
}

type CommentCreateInput struct {
	SiteKey       string
	PageKey       string
	PageTitle     string
	PageURL       string
	AuthorName    string
	AuthorEmail   string
	AuthorWebsite string
	BodyMarkdown  string
	ParentID      *string
	Honeypot      string
	Origin        string
	Referer       string
	IP            string
	UserAgent     string
}
