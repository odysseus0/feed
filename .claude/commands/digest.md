# RSS Digest

Read my RSS feeds and surface what's worth reading today.

## Tools

`feed` is installed locally. Use it via Bash.

## Workflow

1. **Fetch** — Run `feed fetch` to pull latest entries.
2. **Scan** — Run `feed get entries --limit 50` to see recent unread entries (title, feed, date, summary).
3. **Triage** — Based on the titles and summaries, pick the 5-10 most high-signal posts. Prioritize: AI progress, systems engineering, developer tools, anything surprising or contrarian.
4. **Read** — For each pick, run `feed get entry <id>` to read the full post as Markdown.
5. **Synthesize** — Present a digest: for each post, give the title, source, and a 2-3 sentence summary of why it matters. Group by theme if natural clusters emerge.
6. **Mark read** — Run `feed update entries --read <id1> <id2> ...` to mark triaged entries as read.

## Notes

- Default output is a table — that's the most token-efficient format for scanning.
- `feed get entry <id>` returns full content as Markdown — read this for the actual post.
- Don't use `-o json` unless you need structured data for further processing. Table is cheaper.
- If too many entries, filter by feed: `feed get entries --feed <feed_id> --limit 20`.
- Use `feed search "<query>"` to find posts on a specific topic.
