package app

import (
	"context"
	"database/sql"

	"deadcomments/internal/domain"
	"deadcomments/internal/repository"
	"deadcomments/internal/service"
)

func SeedDevelopment(ctx context.Context, database *sql.DB, cfg Config) error {
	if !cfg.DevSeed {
		return nil
	}
	sites := repository.NewSiteRepository(database)
	count, err := sites.Count(ctx)
	if err != nil || count > 0 {
		return err
	}
	siteSvc := service.NewSiteService(sites)
	return siteSvc.Create(ctx, &domain.Site{
		Key:                   "docs-demo",
		Name:                  "Docusaurus Demo",
		AllowedOrigins:        []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001"},
		DefaultModerationMode: domain.ModerationAuto,
		DefaultPageState:      domain.PageOpen,
		DefaultTheme:          domain.ThemeAuto,
		MaxCommentLength:      5000,
		AllowReplies:          true,
	})
}
