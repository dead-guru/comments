package domain

import "time"

type Annotation struct {
	ID              string
	SiteID          int64
	SiteKey         string
	PageID          int64
	PageKey         string
	CommentID       string
	Selector        string
	SelectedText    string
	SelectionPrefix string
	SelectionSuffix string
	TextStart       *int64
	TextEnd         *int64
	TextHash        string
	MetadataJSON    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Comment         *Comment
}

type PublicAnnotation struct {
	ID              string         `json:"id"`
	Selector        string         `json:"selector"`
	SelectedText    string         `json:"selected_text"`
	SelectionPrefix string         `json:"selection_prefix"`
	SelectionSuffix string         `json:"selection_suffix"`
	TextStart       *int64         `json:"text_start"`
	TextEnd         *int64         `json:"text_end"`
	TextHash        string         `json:"text_hash"`
	CreatedAt       time.Time      `json:"created_at"`
	Comment         *PublicComment `json:"comment"`
}

type AnnotationCreateInput struct {
	CommentCreateInput
	Selector        string
	SelectedText    string
	SelectionPrefix string
	SelectionSuffix string
	TextStart       *int64
	TextEnd         *int64
	MetadataJSON    *string
}
