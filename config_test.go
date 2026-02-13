package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var configEnvKeys = []string{
	"HOME",
	configPathEnvName,
	"FEED_DB_PATH",
	"FEED_STALE_MINUTES",
	"FEED_FETCH_CONCURRENCY",
	"FEED_RETENTION_DAYS",
	"FEED_HTTP_TIMEOUT_SECONDS",
	"FEED_USER_AGENT",
}

func setEnvForTest(t *testing.T, key, value string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvKeys {
		unsetEnvForTest(t, key)
	}
}

func writeConfigFile(t *testing.T, home string, body string) string {
	t.Helper()
	path := filepath.Join(home, ".config", configFolderName, configFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}

func TestLoadConfig_NoConfigFileUsesDefaults(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	setEnvForTest(t, "HOME", home)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	wantDB := filepath.Join(home, ".local", "share", "feed", "feed.db")
	if cfg.DBPath != wantDB {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, wantDB)
	}
	if cfg.StaleAfter != defaultStaleMinutes*time.Minute {
		t.Fatalf("StaleAfter = %s, want %s", cfg.StaleAfter, defaultStaleMinutes*time.Minute)
	}
	if cfg.FetchConcurrency != defaultFetchConcurrent {
		t.Fatalf("FetchConcurrency = %d, want %d", cfg.FetchConcurrency, defaultFetchConcurrent)
	}
	if cfg.RetentionDays != 0 {
		t.Fatalf("RetentionDays = %d, want 0", cfg.RetentionDays)
	}
	if cfg.HTTPTimeout != defaultHTTPTimeoutSec*time.Second {
		t.Fatalf("HTTPTimeout = %s, want %s", cfg.HTTPTimeout, defaultHTTPTimeoutSec*time.Second)
	}
	if cfg.UserAgent != defaultUserAgent {
		t.Fatalf("UserAgent = %q, want %q", cfg.UserAgent, defaultUserAgent)
	}
}

func TestLoadConfig_ConfigFileValuesApplied(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	setEnvForTest(t, "HOME", home)

	wantDB := filepath.Join(t.TempDir(), "cfg.db")
	writeConfigFile(t, home, `
db_path = "`+wantDB+`"
stale_minutes = 45
fetch_concurrency = 4
retention_days = 7
`)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.DBPath != wantDB {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, wantDB)
	}
	if cfg.StaleAfter != 45*time.Minute {
		t.Fatalf("StaleAfter = %s, want 45m", cfg.StaleAfter)
	}
	if cfg.FetchConcurrency != 4 {
		t.Fatalf("FetchConcurrency = %d, want 4", cfg.FetchConcurrency)
	}
	if cfg.RetentionDays != 7 {
		t.Fatalf("RetentionDays = %d, want 7", cfg.RetentionDays)
	}
	if cfg.HTTPTimeout != defaultHTTPTimeoutSec*time.Second {
		t.Fatalf("HTTPTimeout = %s, want %s", cfg.HTTPTimeout, defaultHTTPTimeoutSec*time.Second)
	}
	if cfg.UserAgent != defaultUserAgent {
		t.Fatalf("UserAgent = %q, want %q", cfg.UserAgent, defaultUserAgent)
	}
}

func TestLoadConfig_XDGConfigPreferredOverHomeConfig(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	xdg := t.TempDir()
	setEnvForTest(t, "HOME", home)
	setEnvForTest(t, configPathEnvName, xdg)

	writeConfigFile(t, home, `
stale_minutes = 15
`)
	xdgPath := filepath.Join(xdg, configFolderName, configFileName)
	if err := os.MkdirAll(filepath.Dir(xdgPath), 0o755); err != nil {
		t.Fatalf("mkdir xdg config dir: %v", err)
	}
	if err := os.WriteFile(xdgPath, []byte("stale_minutes = 60\n"), 0o644); err != nil {
		t.Fatalf("write xdg config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.StaleAfter != 60*time.Minute {
		t.Fatalf("StaleAfter = %s, want 60m", cfg.StaleAfter)
	}
}

func TestLoadConfig_EnvOverridesConfigFile(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	setEnvForTest(t, "HOME", home)

	writeConfigFile(t, home, `
db_path = "/tmp/from-config.db"
stale_minutes = 20
fetch_concurrency = 2
retention_days = 5
`)

	envDB := filepath.Join(t.TempDir(), "from-env.db")
	setEnvForTest(t, "FEED_DB_PATH", envDB)
	setEnvForTest(t, "FEED_STALE_MINUTES", "25")
	setEnvForTest(t, "FEED_FETCH_CONCURRENCY", "6")
	setEnvForTest(t, "FEED_RETENTION_DAYS", "11")
	setEnvForTest(t, "FEED_HTTP_TIMEOUT_SECONDS", "9")
	setEnvForTest(t, "FEED_USER_AGENT", "feed-test/2.0")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.DBPath != envDB {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, envDB)
	}
	if cfg.StaleAfter != 25*time.Minute {
		t.Fatalf("StaleAfter = %s, want 25m", cfg.StaleAfter)
	}
	if cfg.FetchConcurrency != 6 {
		t.Fatalf("FetchConcurrency = %d, want 6", cfg.FetchConcurrency)
	}
	if cfg.RetentionDays != 11 {
		t.Fatalf("RetentionDays = %d, want 11", cfg.RetentionDays)
	}
	if cfg.HTTPTimeout != 9*time.Second {
		t.Fatalf("HTTPTimeout = %s, want 9s", cfg.HTTPTimeout)
	}
	if cfg.UserAgent != "feed-test/2.0" {
		t.Fatalf("UserAgent = %q, want %q", cfg.UserAgent, "feed-test/2.0")
	}
}

func TestLoadConfig_InvalidOrEmptyEnvDoesNotOverrideConfigFile(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	setEnvForTest(t, "HOME", home)

	configDB := filepath.Join(t.TempDir(), "from-config.db")
	writeConfigFile(t, home, `
db_path = "`+configDB+`"
stale_minutes = 42
fetch_concurrency = 7
retention_days = 13
`)

	setEnvForTest(t, "FEED_DB_PATH", "")
	setEnvForTest(t, "FEED_STALE_MINUTES", "abc")
	setEnvForTest(t, "FEED_FETCH_CONCURRENCY", "0")
	setEnvForTest(t, "FEED_RETENTION_DAYS", "-1")
	setEnvForTest(t, "FEED_HTTP_TIMEOUT_SECONDS", "0")
	setEnvForTest(t, "FEED_USER_AGENT", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.DBPath != configDB {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, configDB)
	}
	if cfg.StaleAfter != 42*time.Minute {
		t.Fatalf("StaleAfter = %s, want 42m", cfg.StaleAfter)
	}
	if cfg.FetchConcurrency != 7 {
		t.Fatalf("FetchConcurrency = %d, want 7", cfg.FetchConcurrency)
	}
	if cfg.RetentionDays != 13 {
		t.Fatalf("RetentionDays = %d, want 13", cfg.RetentionDays)
	}
	if cfg.HTTPTimeout != defaultHTTPTimeoutSec*time.Second {
		t.Fatalf("HTTPTimeout = %s, want %s", cfg.HTTPTimeout, defaultHTTPTimeoutSec*time.Second)
	}
	if cfg.UserAgent != defaultUserAgent {
		t.Fatalf("UserAgent = %q, want %q", cfg.UserAgent, defaultUserAgent)
	}
}

func TestLoadConfig_InvalidConfigReturnsError(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantSnippet string
	}{
		{
			name:        "stale_minutes non-positive",
			body:        "stale_minutes = 0\n",
			wantSnippet: "stale_minutes must be > 0",
		},
		{
			name:        "fetch_concurrency too small",
			body:        "fetch_concurrency = 0\n",
			wantSnippet: "fetch_concurrency must be >= 1",
		},
		{
			name:        "retention_days negative",
			body:        "retention_days = -1\n",
			wantSnippet: "retention_days must be >= 0",
		},
		{
			name:        "db_path empty",
			body:        "db_path = \"   \"\n",
			wantSnippet: "db_path must be non-empty",
		},
		{
			name:        "unknown key",
			body:        "timeout_seconds = 10\n",
			wantSnippet: "unknown key(s): timeout_seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			home := t.TempDir()
			setEnvForTest(t, "HOME", home)
			path := writeConfigFile(t, home, tt.body)

			_, err := LoadConfig()
			if err == nil {
				t.Fatalf("LoadConfig() error = nil, want error")
			}
			msg := err.Error()
			if !strings.Contains(msg, tt.wantSnippet) {
				t.Fatalf("error %q does not contain %q", msg, tt.wantSnippet)
			}
			if !strings.Contains(msg, path) {
				t.Fatalf("error %q does not contain path %q", msg, path)
			}
		})
	}
}

func TestRootCommand_DBFlagOverridesEnvAndConfig(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	setEnvForTest(t, "HOME", home)

	configDB := filepath.Join(t.TempDir(), "from-config.db")
	writeConfigFile(t, home, `db_path = "`+configDB+`"`+"\n")

	envDB := filepath.Join(t.TempDir(), "from-env.db")
	setEnvForTest(t, "FEED_DB_PATH", envDB)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DBPath != envDB {
		t.Fatalf("LoadConfig DBPath = %q, want %q", cfg.DBPath, envDB)
	}

	flagDB := filepath.Join(t.TempDir(), "from-flag.db")
	root := NewRootCmd(cfg)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--db", flagDB, "get", "stats", "-o", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute: %v (stderr: %s)", err, stderr.String())
	}

	if _, err := os.Stat(flagDB); err != nil {
		t.Fatalf("expected flag DB at %q: %v", flagDB, err)
	}
	if _, err := os.Stat(envDB); !os.IsNotExist(err) {
		t.Fatalf("expected env DB not to be opened, stat err: %v", err)
	}
	if _, err := os.Stat(configDB); !os.IsNotExist(err) {
		t.Fatalf("expected config DB not to be opened, stat err: %v", err)
	}
}
