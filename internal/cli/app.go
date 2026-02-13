package cli

import (
	"database/sql"

	"github.com/tengjizhang/feed/internal/config"
	feedpkg "github.com/tengjizhang/feed/internal/fetch"
	"github.com/tengjizhang/feed/internal/store"
)

type App struct {
	cfg      config.Config
	db       *sql.DB
	store    *store.Store
	renderer *feedpkg.Renderer
	fetcher  *feedpkg.Fetcher
}

func NewApp(cfg config.Config, dbPath string) (*App, error) {
	cfg.DBPath = dbPath
	db, err := store.OpenDB(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	s := store.NewStore(db)
	renderer := feedpkg.NewRenderer()
	fetcher := feedpkg.NewFetcher(s, renderer, cfg)

	return &App{
		cfg:      cfg,
		db:       db,
		store:    s,
		renderer: renderer,
		fetcher:  fetcher,
	}, nil
}

func (a *App) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
