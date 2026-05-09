package domain

import "time"

type IdentityType string

const (
	IdentityReserved IdentityType = "reserved"
	IdentityAdmin    IdentityType = "admin"
	IdentitySystem   IdentityType = "system"
)

type TripcodeKind string

const (
	TripcodeNone      TripcodeKind = "none"
	TripcodeAnonymous TripcodeKind = "anonymous"
	TripcodeReserved  TripcodeKind = "reserved"
)

type BadgeType string

const (
	BadgeVerified BadgeType = "verified"
	BadgeAdmin    BadgeType = "admin"
	BadgeAuthor   BadgeType = "author"
	BadgeCustom   BadgeType = "custom"
)

type Identity struct {
	ID               int64
	SiteID           *int64
	DisplayName      string
	NormalizedName   string
	Type             IdentityType
	SecretHash       string
	PublicTripcode   string
	BadgeType        BadgeType
	BadgeLabel       *string
	CreatedByAdminID *int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type IdentityCreateInput struct {
	SiteID           *int64
	DisplayName      string
	Secret           string
	PublicTripcode   string
	BadgeType        BadgeType
	BadgeLabel       string
	CreatedByAdminID *int64
}

type IdentityUpdateInput struct {
	ID             int64
	SiteID         *int64
	DisplayName    string
	PublicTripcode string
	BadgeType      BadgeType
	BadgeLabel     string
}

type IdentityResolution struct {
	DisplayName    string
	IdentityID     *int64
	TripcodePublic *string
	TripcodeKind   TripcodeKind
	BadgeType      *BadgeType
	BadgeLabel     *string
}
