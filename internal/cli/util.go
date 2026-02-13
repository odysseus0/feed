package cli

import (
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

func compactText(v string, max int) string {
	v = strings.TrimSpace(wsRegexp.ReplaceAllString(v, " "))
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max-1] + "..."
}

func fallback(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}
