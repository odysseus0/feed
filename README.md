# feed

A local-first RSS CLI built in Go.

`feed` is a single-binary, headless RSS engine for people who want scriptable feed workflows without running a server. It fetches feeds, stores entries in SQLite, tracks read/star state, and supports full-text search.

## Highlights

- Single binary (`go build`), no daemon, no web UI.
- SQLite storage (`modernc.org/sqlite`, no CGO required).
- RSS/Atom/JSON Feed parsing.
- Feed discovery from website HTML (`<link rel="alternate">`).
- Pre-computed Markdown content for fast reads.
- Entry state management (`read`, `starred`) including batch updates.
- Full-text search via SQLite FTS5.
- OPML import/export.
- Human-readable table output plus JSON output for automation.

## Install

```bash
go build -o feed .
```

Optional: move `feed` into your PATH.

## Quickstart

```bash
# Import starter subscriptions
./feed import hn-popular-blogs-2025.opml

# Fetch all feeds
./feed fetch

# Browse unread entries
./feed get entries --limit=20

# Read one entry
./feed get entry <id>

# Mark state
./feed update entry <id> --read
./feed update entry <id> --starred

# Search
./feed search "rust"
```

## CLI Grammar

```text
feed <verb> <resource> [id] [flags]
```

Core commands:

- `feed get entries|entry|feeds|stats`
- `feed add feed <url>`
- `feed remove feed <id>`
- `feed update entry|entries ...`
- `feed fetch [id]`
- `feed import <file.opml>`
- `feed export`
- `feed search <query>`

Run `./feed --help` and `./feed <command> --help` for full flag details.

## Output Modes

- Default: table
- `-o json`: machine-readable JSON
- `-o wide`: expanded table columns

Status/progress messages are written to `stderr` so `stdout` remains pipe-friendly.

## Data Location

Default database path:

```text
~/.local/share/feed/feed.db
```

Override with:

```bash
./feed --db /path/to/feed.db ...
# or
export FEED_DB_PATH=/path/to/feed.db
```

## Configuration

Configuration precedence (highest to lowest):

1. CLI flags
2. Environment variables
3. Config file
4. Built-in defaults

Optional config file lookup order:

1. `$XDG_CONFIG_HOME/feed/config.toml`
2. `~/.config/feed/config.toml`

If no config file exists, `feed` continues with env/default values.

Supported config file keys:

- `db_path`
- `stale_minutes`
- `fetch_concurrency`
- `retention_days`

Minimal `config.toml` example:

```toml
db_path = "/Users/you/.local/share/feed/feed.db"
stale_minutes = 30
fetch_concurrency = 10
retention_days = 0
```

Unknown config keys are rejected.

Environment variables:

- `FEED_DB_PATH`: database file path
- `FEED_STALE_MINUTES`: auto-fetch staleness threshold (default `30`)
- `FEED_FETCH_CONCURRENCY`: concurrent feed fetch workers (default `10`)
- `FEED_RETENTION_DAYS`: prune read, unstarred entries older than N days (`0` = disabled)
- `FEED_HTTP_TIMEOUT_SECONDS`: HTTP timeout (default `20`)
- `FEED_USER_AGENT`: outgoing user-agent string

## Architecture

- Command layer: `cmd_*.go`
- Fetch/discovery/rendering: `fetch.go`, `discover.go`, `render.go`, `sanitize.go`
- Storage/query layer: `store_*.go`, `db.go`
- OPML support: `opml.go`

## Development

```bash
go test ./...
go test -cover ./...
go vet ./...
```

Current regression suite covers discovery, fetch behavior (including conditional requests), sanitization, OPML parsing, state transitions, and pruning logic.

## Scope

This project is intentionally local and non-daemonized:

- No background service
- No cross-device sync
- No built-in TUI/web UI
- No bundled article scraping engine
