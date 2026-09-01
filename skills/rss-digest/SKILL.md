---
name: rss-digest
description: "Agentic RSS digest using the feed CLI. Fetch, triage, and summarize RSS feeds to surface high-signal posts. Use when: (1) reading RSS feeds or catching up on news, (2) user asks for a digest, roundup, or summary of recent posts, (3) user asks what's new or interesting today, (4) user mentions feed, RSS, or blogs."
metadata: {"openclaw": {"emoji": "📡", "requires": {"bins": ["feed"]}, "install": [{"kind": "brew", "formula": "odysseus0/tap/feed", "bins": ["feed"], "label": "Install via Homebrew"}, {"kind": "go", "package": "github.com/odysseus0/feed/cmd/feed@latest", "bins": ["feed"], "label": "Install via Go"}]}}
---

# RSS Digest

Surface what's worth reading from RSS feeds. Requires `feed` CLI (`brew install odysseus0/tap/feed`).

## Workflow

1. **Scan** — `feed get entries --limit 50` for recent unread (title, feed, date, URL, summary). Auto-fetches if stale. If 0 results, run `feed get stats` — if 0 feeds, import starter set: `feed import https://github.com/odysseus0/feed/raw/main/hn-popular-blogs-2025.opml` and retry.
2. **Triage** — Pick 5-10 high-signal posts based on the user's prompt. If no specific interest given, prioritize surprising, contrarian, or unusually insightful pieces.
3. **Read + Synthesize** — For each picked entry, read the full content and summarize it briefly. Prefer fetching the URL directly (e.g. WebFetch) if available — keeps full text out of context. Otherwise use `feed get entry <id>` to read the stored content. Parallelize when possible.
4. **Present** — Compile the summaries into a digest. Group by theme if natural clusters emerge.

## Commands

```
feed get entries --limit N              # list unread entries (table)
feed get entries --feed <id> --limit N  # filter by feed
feed get entry <id>                     # read full post (markdown)
feed fetch                              # pull latest from all feeds
feed search "<query>"                   # full-text search
feed update entries --read <id> ...     # batch mark read
feed get feeds                          # list feeds with unread counts
feed get stats                          # database stats
```

## Notes

- The entries table includes full URLs. Prefer fetching URLs directly (keeps full text out of your context window). Fall back to `feed get entry <id>` if you don't have a web fetch tool.
- Do not mark entries as read. The user decides what to mark read.
- Default output is table — most token-efficient for scanning. Avoid `-o json`.
- Filter by feed if too many entries: `--feed <feed_id>`.
