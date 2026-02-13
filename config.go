package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	defaultStaleMinutes    = 30
	defaultFetchConcurrent = 10
	defaultHTTPTimeoutSec  = 20
)

type Config struct {
	DBPath           string
	StaleAfter       time.Duration
	FetchConcurrency int
	RetentionDays    int
	HTTPTimeout      time.Duration
	UserAgent        string
}

func LoadConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	defaultDB := filepath.Join(home, ".local", "share", "feed", "feed.db")

	cfg := Config{
		DBPath:           envOr("FEED_DB_PATH", defaultDB),
		StaleAfter:       time.Duration(envIntOr("FEED_STALE_MINUTES", defaultStaleMinutes)) * time.Minute,
		FetchConcurrency: envIntOr("FEED_FETCH_CONCURRENCY", defaultFetchConcurrent),
		RetentionDays:    envIntOr("FEED_RETENTION_DAYS", 0),
		HTTPTimeout:      time.Duration(envIntOr("FEED_HTTP_TIMEOUT_SECONDS", defaultHTTPTimeoutSec)) * time.Second,
		UserAgent:        envOr("FEED_USER_AGENT", "feed/0.1"),
	}
	if cfg.FetchConcurrency < 1 {
		cfg.FetchConcurrency = defaultFetchConcurrent
	}
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = defaultStaleMinutes * time.Minute
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = defaultHTTPTimeoutSec * time.Second
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
