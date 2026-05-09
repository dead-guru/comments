package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"deadcomments/internal/db"
	"deadcomments/internal/domain"
	dcevent "deadcomments/internal/event"
	dcmarkdown "deadcomments/internal/markdown"
	"deadcomments/internal/repository"
)

type testDeps struct {
	db         *sql.DB
	sites      *repository.SiteRepository
	pages      *repository.PageRepository
	comments   *repository.CommentRepository
	identities *repository.IdentityRepository
	moderation *repository.ModerationRepository
	events     *repository.EventRepository
	commentSvc *CommentService
	siteSvc    *SiteService
	modSvc     *ModerationService
}

func newTestDeps(t *testing.T) testDeps {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	sites := repository.NewSiteRepository(database)
	pages := repository.NewPageRepository(database)
	comments := repository.NewCommentRepository(database)
	identities := repository.NewIdentityRepository(database)
	moderation := repository.NewModerationRepository(database)
	events := repository.NewEventRepository(database)
	bus := dcevent.NewBus(events)
	bus.Subscribe(dcevent.NewAuditHandler(moderation))
	md := NewMarkdownService(dcmarkdown.NewRenderer())
	identitySvc := NewIdentityService(identities, "tripcode-secret", bus)
	modSvc := NewModerationService(moderation, comments, bus)
	return testDeps{
		db:         database,
		sites:      sites,
		pages:      pages,
		comments:   comments,
		identities: identities,
		moderation: moderation,
		events:     events,
		commentSvc: NewCommentService(sites, pages, comments, identitySvc, modSvc, md, "secret", bus),
		siteSvc:    NewSiteService(sites, bus),
		modSvc:     modSvc,
	}
}

func createSite(t *testing.T, deps testDeps, mode domain.ModerationMode) *domain.Site {
	t.Helper()
	site := &domain.Site{
		Key:                   "test-site",
		Name:                  "Test Site",
		AllowedOrigins:        []string{"https://blog.example", "http://localhost:3000"},
		DefaultModerationMode: mode,
		DefaultPageState:      domain.PageOpen,
		DefaultTheme:          domain.ThemeAuto,
		MaxCommentLength:      5000,
		AllowReplies:          true,
	}
	if err := deps.siteSvc.Create(context.Background(), site); err != nil {
		t.Fatal(err)
	}
	return site
}

func TestOriginValidation(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	if !deps.siteSvc.OriginAllowed(site, "https://blog.example/posts/one") {
		t.Fatal("expected allowed referer origin")
	}
	if deps.siteSvc.OriginAllowed(site, "https://evil.example") {
		t.Fatal("expected disallowed origin")
	}
}

func TestOriginValidationAllowsAllWhenNoOriginsConfigured(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	site.AllowedOrigins = nil

	if !deps.siteSvc.OriginAllowed(site, "https://anywhere.example/posts/one") {
		t.Fatal("expected empty origin allowlist to allow any origin")
	}
}

func TestCommentCreationAutoCreatesPage(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	comment, _, err := deps.commentSvc.Create(context.Background(), domain.CommentCreateInput{
		SiteKey:      "test-site",
		PageKey:      "/posts/hello",
		PageTitle:    "Hello",
		PageURL:      "https://blog.example/posts/hello",
		AuthorName:   "Oleksii",
		BodyMarkdown: "Nice post",
		Origin:       "https://blog.example",
		IP:           "203.0.113.1",
		UserAgent:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if comment.Status != domain.CommentApproved {
		t.Fatalf("expected approved, got %s", comment.Status)
	}
	page, err := deps.pages.BySiteAndKey(context.Background(), comment.SiteID, "/posts/hello")
	if err != nil || page == nil {
		t.Fatalf("expected page auto-created, page=%v err=%v", page, err)
	}
	if page.ApprovedCount != 1 || page.CommentsCount != 1 {
		t.Fatalf("expected counters updated, got approved=%d total=%d", page.ApprovedCount, page.CommentsCount)
	}
}

func TestReplyCreationStoresTreeFields(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	root, _, err := deps.commentSvc.Create(context.Background(), validInput(nil))
	if err != nil {
		t.Fatal(err)
	}
	reply, _, err := deps.commentSvc.Create(context.Background(), validInput(&root.ID))
	if err != nil {
		t.Fatal(err)
	}
	if reply.ParentID == nil || *reply.ParentID != root.ID {
		t.Fatalf("reply parent mismatch")
	}
	if reply.RootID == nil || *reply.RootID != root.ID {
		t.Fatalf("reply root mismatch: %#v", reply.RootID)
	}
	if reply.Depth != 1 {
		t.Fatalf("expected depth 1, got %d", reply.Depth)
	}
}

func TestModerationStatusDecision(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationManual)
	comment, reason, err := deps.commentSvc.Create(context.Background(), validInput(nil))
	if err != nil {
		t.Fatal(err)
	}
	if comment.Status != domain.CommentPending || reason != "manual moderation" {
		t.Fatalf("expected pending manual moderation, got %s %q", comment.Status, reason)
	}
	spamInput := validInput(nil)
	spamInput.PageKey = "/posts/spam"
	spamInput.Honeypot = "filled"
	spam, reason, err := deps.commentSvc.Create(context.Background(), spamInput)
	if err != nil {
		t.Fatal(err)
	}
	if spam.Status != domain.CommentSpam || reason != "honeypot" {
		t.Fatalf("expected honeypot spam, got %s %q", spam.Status, reason)
	}
}

func TestBuildTreeLimitsVisualNesting(t *testing.T) {
	rootID := "root"
	replyID := "reply"
	grandID := "grand"
	comments := []*domain.Comment{
		{ID: rootID, AuthorName: "Root"},
		{ID: replyID, ParentID: &rootID, RootID: &rootID, AuthorName: "Reply"},
		{ID: grandID, ParentID: &replyID, RootID: &rootID, AuthorName: "Grand"},
	}
	tree := BuildTree(comments)
	if len(tree) != 1 || len(tree[0].Children) != 2 {
		t.Fatalf("expected one root with two visually flat replies, got %#v", tree)
	}
	if tree[0].Children[1].ReplyingToAuthor == nil || *tree[0].Children[1].ReplyingToAuthor != "Reply" {
		t.Fatalf("expected deep reply to carry replying-to author")
	}
}

func TestApprovedPublicTreeGroupsRepliesUnderRootCreationOrder(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	page, _, err := deps.pages.FindOrCreate(context.Background(), site, "/posts/thread-order", "Thread Order", "https://blog.example/posts/thread-order")
	if err != nil {
		t.Fatal(err)
	}

	rootOld := "z-root-old"
	rootNew := "a-root-new"
	replyOld := "reply-old"
	replyOldTwo := "reply-old-two"
	replyNew := "reply-new"
	for _, comment := range []*domain.Comment{
		{ID: rootOld, SiteID: site.ID, PageID: page.ID, AuthorName: "Root Old", AuthorDisplayName: "Root Old", BodyMarkdown: "old root", BodyHTML: "<p>old root</p>", Status: domain.CommentApproved, TripcodeKind: domain.TripcodeNone},
		{ID: rootNew, SiteID: site.ID, PageID: page.ID, AuthorName: "Root New", AuthorDisplayName: "Root New", BodyMarkdown: "new root", BodyHTML: "<p>new root</p>", Status: domain.CommentApproved, TripcodeKind: domain.TripcodeNone},
		{ID: replyOld, SiteID: site.ID, PageID: page.ID, ParentID: &rootOld, RootID: &rootOld, Depth: 1, AuthorName: "Reply Old", AuthorDisplayName: "Reply Old", BodyMarkdown: "old reply", BodyHTML: "<p>old reply</p>", Status: domain.CommentApproved, TripcodeKind: domain.TripcodeNone},
		{ID: replyOldTwo, SiteID: site.ID, PageID: page.ID, ParentID: &rootOld, RootID: &rootOld, Depth: 1, AuthorName: "Reply Old Two", AuthorDisplayName: "Reply Old Two", BodyMarkdown: "old reply two", BodyHTML: "<p>old reply two</p>", Status: domain.CommentApproved, TripcodeKind: domain.TripcodeNone},
		{ID: replyNew, SiteID: site.ID, PageID: page.ID, ParentID: &rootNew, RootID: &rootNew, Depth: 1, AuthorName: "Reply New", AuthorDisplayName: "Reply New", BodyMarkdown: "new reply", BodyHTML: "<p>new reply</p>", Status: domain.CommentApproved, TripcodeKind: domain.TripcodeNone},
	} {
		if err := deps.comments.Create(context.Background(), comment); err != nil {
			t.Fatal(err)
		}
		rootID := comment.ID
		if comment.RootID != nil {
			rootID = *comment.RootID
		}
		path := repository.CommentPath(nil, comment.ID)
		if comment.ParentID != nil {
			path = repository.CommentPath(&domain.Comment{Path: rootID}, comment.ID)
		}
		if err := deps.comments.UpdateTreeFields(context.Background(), comment.ID, &rootID, comment.Depth, path); err != nil {
			t.Fatal(err)
		}
	}

	times := map[string]string{
		rootOld:     "2026-01-01T00:00:00Z",
		replyOld:    "2026-01-01T00:01:00Z",
		rootNew:     "2026-01-01T00:02:00Z",
		replyNew:    "2026-01-01T00:03:00Z",
		replyOldTwo: "2026-01-01T00:04:00Z",
	}
	for id, at := range times {
		if _, err := deps.db.ExecContext(context.Background(), `UPDATE comments SET created_at=?, updated_at=? WHERE id=?`, at, at, id); err != nil {
			t.Fatal(err)
		}
	}

	comments, err := deps.comments.ApprovedByPage(context.Background(), page.ID, domain.CommentSortOldest)
	if err != nil {
		t.Fatal(err)
	}
	tree := BuildTree(comments)
	if len(tree) != 2 {
		t.Fatalf("expected two root threads, got %d", len(tree))
	}
	if tree[0].ID != rootOld || len(tree[0].Children) != 2 || tree[0].Children[0].ID != replyOld || tree[0].Children[1].ID != replyOldTwo {
		t.Fatalf("expected old root thread first with its reply, got %#v", tree[0])
	}
	if tree[1].ID != rootNew || len(tree[1].Children) != 1 || tree[1].Children[0].ID != replyNew {
		t.Fatalf("expected new root thread second with its reply, got %#v", tree[1])
	}

	comments, err = deps.comments.ApprovedByPage(context.Background(), page.ID, domain.CommentSortNewest)
	if err != nil {
		t.Fatal(err)
	}
	tree = BuildTree(comments)
	if tree[0].ID != rootNew || tree[1].ID != rootOld {
		t.Fatalf("expected newest root thread first, got %s then %s", tree[0].ID, tree[1].ID)
	}

	comments, err = deps.comments.ApprovedByPage(context.Background(), page.ID, domain.CommentSortBest)
	if err != nil {
		t.Fatal(err)
	}
	tree = BuildTree(comments)
	if tree[0].ID != rootOld {
		t.Fatalf("expected most active root thread first for best sort, got %s", tree[0].ID)
	}
}

func TestBannedIPRejected(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	ipHash := HashValue("secret", "203.0.113.10")
	if err := deps.modSvc.AddIPBan(context.Background(), &domain.IPBan{SiteID: &site.ID, IPHash: ipHash}); err != nil {
		t.Fatal(err)
	}
	input := validInput(nil)
	input.IP = "203.0.113.10"
	comment, reason, err := deps.commentSvc.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if comment.Status != domain.CommentRejected || reason != "ip banned" {
		t.Fatalf("expected rejected banned IP, got %s %q", comment.Status, reason)
	}
	stored, err := deps.comments.ByID(context.Background(), comment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ModerationReason == nil || *stored.ModerationReason != "ip banned" {
		t.Fatalf("expected stored moderation reason ip banned, got %v", stored.ModerationReason)
	}
}

func TestRecentIPRateLimitRejectsWithReason(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)

	for i := 0; i < recentIPCommentLimit; i++ {
		input := validInput(nil)
		input.PageKey = "/posts/rate-limit"
		input.BodyMarkdown = "Unique comment " + strconv.Itoa(i)
		comment, reason, err := deps.commentSvc.Create(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if comment.Status != domain.CommentApproved || reason != "auto moderation" {
			t.Fatalf("expected approved before limit, got %s %q at %d", comment.Status, reason, i)
		}
	}

	limited := validInput(nil)
	limited.PageKey = "/posts/rate-limit"
	limited.BodyMarkdown = "One comment too many"
	result, err := deps.commentSvc.CreateDetailed(context.Background(), limited)
	if err != nil {
		t.Fatal(err)
	}
	if result.Comment.Status != domain.CommentRejected || result.Reason != "rate limit" {
		t.Fatalf("expected rejected rate limit, got %s %q", result.Comment.Status, result.Reason)
	}
	if result.RetryAfter <= 0 || result.Limit != recentIPCommentLimit || result.Window != recentIPWindow {
		t.Fatalf("expected retry metadata, got retry=%s limit=%d window=%s", result.RetryAfter, result.Limit, result.Window)
	}
}

func TestReservedIdentityUsesHigherRateLimit(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	createReservedIdentity(t, deps, &site.ID, "UT3USW", "correct-secret")

	for i := 0; i < recentIPCommentLimit+1; i++ {
		input := validInput(nil)
		input.PageKey = "/posts/reserved-rate-limit"
		input.AuthorName = "UT3USW##correct-secret"
		input.BodyMarkdown = "Reserved identity comment " + strconv.Itoa(i)
		comment, reason, err := deps.commentSvc.Create(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if comment.Status != domain.CommentApproved || reason != "auto moderation" {
			t.Fatalf("expected reserved identity to pass default limit, got %s %q at %d", comment.Status, reason, i)
		}
		if comment.IdentityID == nil {
			t.Fatalf("expected reserved identity id at %d", i)
		}
	}
}

func TestCommentStatusEventWritesDurableEventAndAuditHistory(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	comment, _, err := deps.commentSvc.Create(context.Background(), validInput(nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.commentSvc.SetStatus(context.Background(), comment.ID, domain.CommentSpam); err != nil {
		t.Fatal(err)
	}
	events, err := deps.events.List(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	var sawStatusEvent bool
	for _, event := range events {
		if event.Type == domain.EventCommentStatusSet && event.CommentID != nil && *event.CommentID == comment.ID {
			sawStatusEvent = true
		}
	}
	if !sawStatusEvent {
		t.Fatalf("expected durable %s event for comment %s", domain.EventCommentStatusSet, comment.ID)
	}
	history, err := deps.modSvc.EventsForComment(context.Background(), comment.ID)
	if err != nil {
		t.Fatal(err)
	}
	var sawSpamAudit bool
	for _, event := range history {
		if event.Action == string(domain.CommentSpam) {
			sawSpamAudit = true
		}
	}
	if !sawSpamAudit {
		t.Fatalf("expected audit handler to write spam moderation history, got %#v", history)
	}
}

func TestPlainNameHasNoTripcode(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	comment, _, err := deps.commentSvc.Create(context.Background(), validInput(nil))
	if err != nil {
		t.Fatal(err)
	}
	if comment.AuthorDisplayName != "Oleksii" || comment.AuthorName != "Oleksii" {
		t.Fatalf("expected plain display name, got author=%q display=%q", comment.AuthorName, comment.AuthorDisplayName)
	}
	if comment.TripcodeKind != domain.TripcodeNone {
		t.Fatalf("expected no tripcode, got %s", comment.TripcodeKind)
	}
	if comment.TripcodePublic != nil || comment.IdentityID != nil {
		t.Fatalf("expected no tripcode/identity, got trip=%v identity=%v", comment.TripcodePublic, comment.IdentityID)
	}
}

func TestAnonymousTripcodeCreated(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	input := validInput(nil)
	input.AuthorName = "Oleksii##some-secret"
	comment, _, err := deps.commentSvc.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if comment.AuthorDisplayName != "Oleksii" || comment.AuthorName != "Oleksii" {
		t.Fatalf("expected parsed display name, got author=%q display=%q", comment.AuthorName, comment.AuthorDisplayName)
	}
	if comment.TripcodeKind != domain.TripcodeAnonymous {
		t.Fatalf("expected anonymous tripcode, got %s", comment.TripcodeKind)
	}
	if comment.TripcodePublic == nil || len(*comment.TripcodePublic) != 10 {
		t.Fatalf("expected 10-char public tripcode, got %#v", comment.TripcodePublic)
	}
	if comment.IdentityID != nil {
		t.Fatalf("anonymous tripcode should not attach identity id")
	}
}

func TestAnonymousTripcodeStability(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	first := validInput(nil)
	first.AuthorName = "Name##same-secret"
	first.PageKey = "/posts/one"
	commentA, _, err := deps.commentSvc.Create(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	second := validInput(nil)
	second.AuthorName = "Name##same-secret"
	second.PageKey = "/posts/two"
	commentB, _, err := deps.commentSvc.Create(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if commentA.TripcodePublic == nil || commentB.TripcodePublic == nil || *commentA.TripcodePublic != *commentB.TripcodePublic {
		t.Fatalf("same secret should produce same tripcode, got %v and %v", commentA.TripcodePublic, commentB.TripcodePublic)
	}
}

func TestAnonymousTripcodeDifferentSecretDiffers(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	first := validInput(nil)
	first.AuthorName = "Name##first-secret"
	first.PageKey = "/posts/one"
	commentA, _, err := deps.commentSvc.Create(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	second := validInput(nil)
	second.AuthorName = "Name##different-secret"
	second.PageKey = "/posts/two"
	commentB, _, err := deps.commentSvc.Create(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if commentA.TripcodePublic == nil || commentB.TripcodePublic == nil || *commentA.TripcodePublic == *commentB.TripcodePublic {
		t.Fatalf("different secrets should produce different tripcodes, got %v and %v", commentA.TripcodePublic, commentB.TripcodePublic)
	}
}

func TestReservedIdentityWithoutSecretRejected(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	createReservedIdentity(t, deps, &site.ID, "UT3USW", "correct-secret")
	input := validInput(nil)
	input.AuthorName = "UT3USW"
	if _, _, err := deps.commentSvc.Create(context.Background(), input); err == nil {
		t.Fatal("expected reserved identity without secret to be rejected")
	}
}

func TestReservedIdentityWrongSecretRejected(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	createReservedIdentity(t, deps, &site.ID, "UT3USW", "correct-secret")
	input := validInput(nil)
	input.AuthorName = "UT3USW##wrong-secret"
	if _, _, err := deps.commentSvc.Create(context.Background(), input); err == nil {
		t.Fatal("expected reserved identity wrong secret to be rejected")
	}
}

func TestReservedIdentityCorrectSecretAttachesIdentity(t *testing.T) {
	deps := newTestDeps(t)
	site := createSite(t, deps, domain.ModerationAuto)
	identity := createReservedIdentity(t, deps, &site.ID, "UT3USW", "correct-secret")
	input := validInput(nil)
	input.AuthorName = "ut3usw##correct-secret"
	comment, _, err := deps.commentSvc.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if comment.IdentityID == nil || *comment.IdentityID != identity.ID {
		t.Fatalf("expected identity id %d, got %v", identity.ID, comment.IdentityID)
	}
	if comment.TripcodeKind != domain.TripcodeReserved {
		t.Fatalf("expected reserved tripcode, got %s", comment.TripcodeKind)
	}
	if comment.BadgeType == nil || *comment.BadgeType != domain.BadgeVerified {
		t.Fatalf("expected verified badge, got %v", comment.BadgeType)
	}
}

func TestSubmittedTripcodeSecretNeverStoredOnComment(t *testing.T) {
	deps := newTestDeps(t)
	createSite(t, deps, domain.ModerationAuto)
	input := validInput(nil)
	input.AuthorName = "NoStore##ultra-secret-value"
	comment, _, err := deps.commentSvc.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	var authorName, displayName, body, metadata sql.NullString
	if err := deps.db.QueryRowContext(context.Background(), `SELECT author_name, author_display_name, body_markdown, metadata_json FROM comments WHERE id=?`, comment.ID).Scan(&authorName, &displayName, &body, &metadata); err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]sql.NullString{
		"author_name":         authorName,
		"author_display_name": displayName,
		"body_markdown":       body,
		"metadata_json":       metadata,
	} {
		if value.Valid && strings.Contains(value.String, "ultra-secret-value") {
			t.Fatalf("submitted secret leaked into comments.%s: %q", field, value.String)
		}
	}
}

func validInput(parentID *string) domain.CommentCreateInput {
	return domain.CommentCreateInput{
		SiteKey:      "test-site",
		PageKey:      "/posts/hello",
		PageTitle:    "Hello",
		PageURL:      "https://blog.example/posts/hello",
		AuthorName:   "Oleksii",
		BodyMarkdown: "Nice post",
		ParentID:     parentID,
		Origin:       "https://blog.example",
		IP:           "203.0.113.2",
		UserAgent:    "test",
	}
}

func createReservedIdentity(t *testing.T, deps testDeps, siteID *int64, displayName, secret string) *domain.Identity {
	t.Helper()
	identitySvc := NewIdentityService(deps.identities, "tripcode-secret")
	identity, err := identitySvc.Create(context.Background(), domain.IdentityCreateInput{
		SiteID:      siteID,
		DisplayName: displayName,
		Secret:      secret,
		BadgeType:   domain.BadgeVerified,
	})
	if err != nil {
		t.Fatal(err)
	}
	return identity
}
