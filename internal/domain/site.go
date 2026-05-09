package domain

import "time"

type ModerationMode string

const (
	ModerationManual ModerationMode = "manual"
	ModerationAuto   ModerationMode = "auto"
)

type Theme string

const (
	ThemeAuto    Theme = "auto"
	ThemeLight   Theme = "light"
	ThemeDark    Theme = "dark"
	ThemeMinimal Theme = "minimal"
)

type Site struct {
	ID                    int64
	Key                   string
	Name                  string
	AllowedOrigins        []string
	DefaultModerationMode ModerationMode
	DefaultPageState      PageState
	DefaultTheme          Theme
	MaxCommentLength      int
	AllowReplies          bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
