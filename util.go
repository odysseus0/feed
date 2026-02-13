package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var wsRegexp = regexp.MustCompile(`\s+`)

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q", s)
	}
	return id, nil
}

func parseDBTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", v)
}

func timeToDBString(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func formatDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}

func humanAgo(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "never"
	}
	d := time.Since(*t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func dedupGUID(link, title string, published *time.Time) string {
	if strings.TrimSpace(link) != "" {
		return strings.TrimSpace(link)
	}
	stamp := ""
	if published != nil {
		stamp = published.UTC().Format(time.RFC3339Nano)
	}
	h := sha1.Sum([]byte(strings.TrimSpace(title) + "|" + stamp))
	return "sha1:" + hex.EncodeToString(h[:])
}

func compactText(v string, max int) string {
	v = strings.TrimSpace(wsRegexp.ReplaceAllString(v, " "))
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max-1] + "..."
}

func truncate(v string, max int) string {
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max]
}
