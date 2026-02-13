package fetch

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

var wsRegexp = regexp.MustCompile(`\s+`)

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

func fallback(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}
