package main

import "database/sql"

type App struct {
	cfg      Config
	db       *sql.DB
	store    *Store
	renderer *Renderer
	fetcher  *Fetcher
}

func NewApp(cfg Config, dbPath string) (*App, error) {
	cfg.DBPath = dbPath
	db, err := openDB(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	store := NewStore(db)
	renderer := NewRenderer()
	fetcher := NewFetcher(store, renderer, cfg)

	return &App{
		cfg:      cfg,
		db:       db,
		store:    store,
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
