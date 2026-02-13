# feed

RSS gives you the content you chose, not what an algorithm chose. But 92 feeds produce 2600 entries a week, and you can't read them all.

`feed` is a headless RSS engine that lets you pipe your feeds through an AI agent — your own algorithm.

```bash
feed import hn-popular-blogs-2025.opml         # 92 curated tech blogs
feed fetch                                      # 2600+ entries in seconds
feed get entries -o json --limit 50 \
  | claude -p "What's worth reading for a systems engineer? Give me the top 5 with one sentence each on why."
```

Single Go binary. Local SQLite database you can query directly. No server, no daemon, no UI. Just structured data out of stdout, ready to pipe anywhere.

```bash
# It's just a SQLite file — bring your own queries
sqlite3 ~/.local/share/feed/feed.db "SELECT title, url FROM entries ORDER BY published_at DESC LIMIT 10"
```

<!-- TODO: terminal recording GIF here -->

## Install

```bash
go install github.com/odysseus0/feed/cmd/feed@latest
```

Or build from source:

```bash
go build -o feed ./cmd/feed
```

Includes a starter OPML with [92 popular tech blogs](hn-popular-blogs-2025.opml) curated from Hacker News discussions.

## Usage

```bash
# Add a feed (auto-discovers feed URL from any webpage)
feed add feed https://simonwillison.net

# Fetch all feeds (or just one by ID)
feed fetch
feed fetch 42

# Browse entries
feed get entries                    # unread, newest first
feed get entries --status all       # everything
feed get entries --feed 1 -o json   # one feed, as JSON

# Read a full post (rendered as Markdown)
feed get entry 446

# Search across everything
feed search "rust async"

# Triage
feed update entry 446 --read
feed update entry 446 --starred
feed update entries --read 100 101 102 103   # batch

# Manage feeds
feed get feeds              # list all with unread counts
feed remove feed 42
feed import feeds.opml
feed export > backup.opml

# Stats
feed get stats
```

### Output modes

Every command supports `-o table` (default), `-o json`, or `-o wide`. Status messages go to stderr, data to stdout — pipe-friendly by design.

## Why this exists

Platform algorithms are optimized for the platform — engagement, ads, time on site. An LLM you run locally is optimized for you. RSS got you off the content treadmill. `feed` + an agent gives you control back.

Newsboat, miniflux, and NetNewsWire are readers. This is plumbing. They have UIs; this has clean table output for agents and structured data for scripts. `feed` is the missing layer between RSS and whatever you want to do with it — LLM triage, vault ingestion, notification pipelines, or scripts you haven't written yet.

Bring your own algorithm.

## How it works

- **Feed discovery** — `feed add https://example.com` parses `<link rel="alternate">` tags. No need to find the feed URL yourself.
- **Concurrent fetching** — 10 workers by default with conditional requests (ETag/If-Modified-Since). Polite and fast.
- **Pre-computed Markdown** — HTML content is converted to Markdown at fetch time. `feed get entry <id>` renders instantly.
- **Full-text search** — SQLite FTS5 across titles, summaries, and content.
- **Auto-fetch on staleness** — `feed get entries` fetches automatically if feeds are >30min stale. Skip with `--no-fetch`.
- **Batch state management** — Mark 50 entries as read in one command. Essential for agent triage workflows.

## Configuration

`feed` works out of the box with sane defaults. Optional config via `~/.config/feed/config.toml` or environment variables.

| Setting | Env var | Default |
|---------|---------|---------|
| Database path | `FEED_DB_PATH` | `~/.local/share/feed/feed.db` |
| Staleness threshold | `FEED_STALE_MINUTES` | `30` |
| Fetch workers | `FEED_FETCH_CONCURRENCY` | `10` |
| Retention (days) | `FEED_RETENTION_DAYS` | `0` (keep all) |
| HTTP timeout | `FEED_HTTP_TIMEOUT_SECONDS` | `20` |

Precedence: CLI flags > env vars > config file > defaults.

## Origin

Inspired by [Karpathy's RSS revival tweet](https://x.com/karpathy/status/2018043254986703167) (Feb 2026): "download a client, or vibe code one." We vibe coded the headless engine for agents.

## Development

```bash
go test ./...
go vet ./...
```

Pure Go, no CGO. `go install` works on any platform.

## License

MIT
