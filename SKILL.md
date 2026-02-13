---
name: rss-digest
description: "Agentic RSS digest using the feed CLI. Fetch, triage, and summarize RSS feeds to surface high-signal posts. Use when: (1) reading RSS feeds or catching up on news, (2) user asks for a digest, roundup, or summary of recent posts, (3) user asks what's new or interesting today, (4) user mentions feed, RSS, or blogs."
metadata: {"openclaw": {"emoji": "📡", "requires": {"bins": ["feed"]}, "install": [{"kind": "brew", "formula": "odysseus0/tap/feed", "bins": ["feed"], "label": "Install via Homebrew"}, {"kind": "go", "package": "github.com/odysseus0/feed/cmd/feed@latest", "bins": ["feed"], "label": "Install via Go"}]}}
---

# RSS Digest

Surface what's worth reading from RSS feeds. Requires `feed` CLI (`brew install odysseus0/tap/feed`).

## Workflow

1. **Fetch** — `feed fetch` to pull latest entries.
2. **Scan** — `feed get entries --limit 50` for recent unread (title, feed, date, summary).
3. **Triage** — Pick 5-10 high-signal posts. Prioritize: AI progress, systems engineering, developer tools, anything surprising or contrarian.
4. **Read** — `feed get entry <id>` for each pick (full post as Markdown).
5. **Synthesize** — For each post: title, source, 2-3 sentence summary of why it matters. Group by theme if natural clusters emerge.
6. **Mark read** — `feed update entries --read <id1> <id2> ...` to mark triaged entries as read.

## Commands

```
feed fetch                              # pull latest from all feeds
feed get entries --limit N              # list unread entries (table)
feed get entries --feed <id> --limit N  # filter by feed
feed get entry <id>                     # read full post (Markdown)
feed search "<query>"                   # full-text search
feed update entries --read <id> ...     # batch mark read
feed get feeds                          # list feeds with unread counts
feed get stats                          # database stats
```

## Notes

- Default output is table — most token-efficient for scanning. Avoid `-o json`.
- `feed get entry <id>` returns Markdown — read this for the actual post content.
- Filter by feed if too many entries: `--feed <feed_id>`.
