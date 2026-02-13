package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	defaultStaleMinutes    = 30
	defaultFetchConcurrent = 10
	defaultHTTPTimeoutSec  = 20
)

const (
	defaultUserAgent  = "feed/0.1"
	configFolderName  = "feed"
	configFileName    = "config.toml"
	configPathEnvName = "XDG_CONFIG_HOME"
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
		DBPath:           defaultDB,
		StaleAfter:       defaultStaleMinutes * time.Minute,
		FetchConcurrency: defaultFetchConcurrent,
		RetentionDays:    0,
		HTTPTimeout:      defaultHTTPTimeoutSec * time.Second,
		UserAgent:        defaultUserAgent,
	}

	configPath, hasConfig, err := findConfigPath(home)
	if err != nil {
		return Config{}, err
	}
	if hasConfig {
		fileCfg, err := loadFileConfig(configPath)
		if err != nil {
			return Config{}, err
		}
		applyFileConfig(&cfg, fileCfg)
	}

	applyEnvOverrides(&cfg)

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

type fileConfig struct {
	DBPath           *string `toml:"db_path"`
	StaleMinutes     *int    `toml:"stale_minutes"`
	FetchConcurrency *int    `toml:"fetch_concurrency"`
	RetentionDays    *int    `toml:"retention_days"`
}

func findConfigPath(home string) (string, bool, error) {
	candidates := make([]string, 0, 2)
	if xdgConfigHome := strings.TrimSpace(os.Getenv(configPathEnvName)); xdgConfigHome != "" {
		candidates = append(candidates, filepath.Join(xdgConfigHome, configFolderName, configFileName))
	}
	candidates = append(candidates, filepath.Join(home, ".config", configFolderName, configFileName))

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return "", false, fmt.Errorf("config path %q is a directory; expected a file", candidate)
			}
			return candidate, true, nil
		}
		if os.IsNotExist(err) {
			continue
		}
		return "", false, fmt.Errorf("failed to read config path %q: %w", candidate, err)
	}
	return "", false, nil
}

func loadFileConfig(path string) (fileConfig, error) {
	var cfg fileConfig
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return fileConfig{}, fmt.Errorf("invalid config file %q: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		unknown := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			unknown = append(unknown, key.String())
		}
		sort.Strings(unknown)
		return fileConfig{}, fmt.Errorf("invalid config file %q: unknown key(s): %s", path, strings.Join(unknown, ", "))
	}
	if err := validateFileConfig(path, cfg); err != nil {
		return fileConfig{}, err
	}
	return cfg, nil
}

func validateFileConfig(path string, cfg fileConfig) error {
	if cfg.DBPath != nil && strings.TrimSpace(*cfg.DBPath) == "" {
		return fmt.Errorf("invalid config file %q: db_path must be non-empty when provided", path)
	}
	if cfg.StaleMinutes != nil && *cfg.StaleMinutes <= 0 {
		return fmt.Errorf("invalid config file %q: stale_minutes must be > 0", path)
	}
	if cfg.FetchConcurrency != nil && *cfg.FetchConcurrency < 1 {
		return fmt.Errorf("invalid config file %q: fetch_concurrency must be >= 1", path)
	}
	if cfg.RetentionDays != nil && *cfg.RetentionDays < 0 {
		return fmt.Errorf("invalid config file %q: retention_days must be >= 0", path)
	}
	return nil
}

func applyFileConfig(cfg *Config, fileCfg fileConfig) {
	if fileCfg.DBPath != nil {
		cfg.DBPath = *fileCfg.DBPath
	}
	if fileCfg.StaleMinutes != nil {
		cfg.StaleAfter = time.Duration(*fileCfg.StaleMinutes) * time.Minute
	}
	if fileCfg.FetchConcurrency != nil {
		cfg.FetchConcurrency = *fileCfg.FetchConcurrency
	}
	if fileCfg.RetentionDays != nil {
		cfg.RetentionDays = *fileCfg.RetentionDays
	}
}

func applyEnvOverrides(cfg *Config) {
	if v, ok := os.LookupEnv("FEED_DB_PATH"); ok && v != "" {
		cfg.DBPath = v
	}
	if v, ok := os.LookupEnv("FEED_STALE_MINUTES"); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.StaleAfter = time.Duration(n) * time.Minute
		}
	}
	if v, ok := os.LookupEnv("FEED_FETCH_CONCURRENCY"); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			cfg.FetchConcurrency = n
		}
	}
	if v, ok := os.LookupEnv("FEED_RETENTION_DAYS"); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.RetentionDays = n
		}
	}
	if v, ok := os.LookupEnv("FEED_HTTP_TIMEOUT_SECONDS"); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.HTTPTimeout = time.Duration(n) * time.Second
		}
	}
	if v, ok := os.LookupEnv("FEED_USER_AGENT"); ok && v != "" {
		cfg.UserAgent = v
	}
}
