package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tengjizhang/feed/internal/config"
)

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
	for _, key := range []string{
		"HOME",
		"XDG_CONFIG_HOME",
		"FEED_DB_PATH",
		"FEED_STALE_MINUTES",
		"FEED_FETCH_CONCURRENCY",
		"FEED_RETENTION_DAYS",
		"FEED_HTTP_TIMEOUT_SECONDS",
		"FEED_USER_AGENT",
	} {
		unsetEnvForTest(t, key)
	}
}

func writeConfigFile(t *testing.T, home string, body string) string {
	t.Helper()
	path := filepath.Join(home, ".config", "feed", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}

func TestRootCommand_DBFlagOverridesEnvAndConfig(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	setEnvForTest(t, "HOME", home)

	configDB := filepath.Join(t.TempDir(), "from-config.db")
	writeConfigFile(t, home, `db_path = "`+configDB+`"`+"\n")

	envDB := filepath.Join(t.TempDir(), "from-env.db")
	setEnvForTest(t, "FEED_DB_PATH", envDB)

	cfg, err := config.LoadConfig()
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
