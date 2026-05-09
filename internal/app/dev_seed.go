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
	siteSvc := service.NewSiteService(sites)
	site, err := sites.ByKey(ctx, "docs-demo")
	if err != nil {
		return err
	}
	demo := &domain.Site{
		Key:                   "docs-demo",
		Name:                  "Docusaurus Demo",
		AllowedOrigins:        []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001"},
		DefaultModerationMode: domain.ModerationAuto,
		DefaultPageState:      domain.PageOpen,
		DefaultTheme:          domain.ThemeAuto,
		MaxCommentLength:      5000,
		AllowReplies:          true,
	}
	if site == nil {
		return siteSvc.Create(ctx, demo)
	}
	demo.ID = site.ID
	return siteSvc.Update(ctx, demo)
}
