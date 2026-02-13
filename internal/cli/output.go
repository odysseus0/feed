package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeEntriesTable(out io.Writer, entries []Entry, wide bool) {
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	if wide {
		fmt.Fprintln(tw, "ID\tFEED_ID\tFEED\tTITLE\tDATE\tREAD\tSTAR\tURL\tSUMMARY")
		for _, e := range entries {
			fmt.Fprintf(
				tw,
				"%d\t%d\t%s\t%s\t%s\t%t\t%t\t%s\t%s\n",
				e.ID,
				e.FeedID,
				compactText(e.FeedTitle, 24),
				compactText(displayEntryTitle(e), 56),
				formatDate(e.PublishedAt),
				e.Read,
				e.Starred,
				compactText(e.URL, 48),
				compactText(oneLine(e.Summary), 90),
			)
		}
	} else {
		fmt.Fprintln(tw, "ID\tFEED\tTITLE\tDATE\tSUMMARY")
		for _, e := range entries {
			fmt.Fprintf(
				tw,
				"%d\t%s\t%s\t%s\t%s\n",
				e.ID,
				compactText(e.FeedTitle, 24),
				compactText(displayEntryTitle(e), 56),
				formatDate(e.PublishedAt),
				compactText(oneLine(e.Summary), 90),
			)
		}
	}
	_ = tw.Flush()
}

func writeFeedsTable(out io.Writer, feeds []Feed, wide bool) {
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	if wide {
		fmt.Fprintln(tw, "ID\tTITLE\tUNREAD\tTOTAL\tLAST_FETCH\tERRORS\tURL\tSITE_URL\tLAST_ERROR")
		for _, f := range feeds {
			fmt.Fprintf(
				tw,
				"%d\t%s\t%d\t%d\t%s\t%d\t%s\t%s\t%s\n",
				f.ID,
				compactText(fallback(f.Title, f.URL), 30),
				f.UnreadCount,
				f.TotalCount,
				humanAgo(f.LastFetchedAt),
				f.ErrorCount,
				compactText(f.URL, 46),
				compactText(f.SiteURL, 46),
				compactText(oneLine(f.LastError), 70),
			)
		}
	} else {
		fmt.Fprintln(tw, "ID\tTITLE\tUNREAD\tLAST_FETCH\tERRORS\tLAST_ERROR\tURL")
		for _, f := range feeds {
			fmt.Fprintf(
				tw,
				"%d\t%s\t%d\t%s\t%d\t%s\t%s\n",
				f.ID,
				compactText(fallback(f.Title, f.URL), 30),
				f.UnreadCount,
				humanAgo(f.LastFetchedAt),
				f.ErrorCount,
				compactText(oneLine(f.LastError), 42),
				compactText(f.URL, 56),
			)
		}
	}
	_ = tw.Flush()
}

func writeStatsTable(out io.Writer, st Stats) {
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "METRIC\tVALUE")
	fmt.Fprintf(tw, "feeds\t%d\n", st.Feeds)
	fmt.Fprintf(tw, "unread\t%d\n", st.Unread)
	fmt.Fprintf(tw, "starred\t%d\n", st.Starred)
	fmt.Fprintf(tw, "total\t%d\n", st.Total)
	_ = tw.Flush()
}

func writeFetchReportTable(out io.Writer, rep FetchReport) {
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "FEED_ID\tFEED\tNEW\tUPDATED\tNOT_MODIFIED\tERROR")
	for _, r := range rep.Results {
		fmt.Fprintf(
			tw,
			"%d\t%s\t%d\t%d\t%t\t%s\n",
			r.FeedID,
			compactText(r.FeedTitle, 30),
			r.NewEntries,
			r.Updated,
			r.NotModified,
			compactText(oneLine(r.Error), 70),
		)
	}
	_ = tw.Flush()
}

func oneLine(v string) string {
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	return strings.TrimSpace(v)
}

func displayEntryTitle(e Entry) string {
	if strings.TrimSpace(e.Title) != "" {
		return e.Title
	}
	if strings.TrimSpace(e.URL) != "" {
		return e.URL
	}
	return "(untitled)"
}
