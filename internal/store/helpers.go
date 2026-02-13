package store

import (
	"fmt"
	"strings"
	"time"
)

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

func truncate(v string, max int) string {
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max]
}
