package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid url %q", raw)
	}
	return u.String(), nil
}

func DiscoverFeedURL(ctx context.Context, client *http.Client, parser *gofeed.Parser, rawURL string) (string, error) {
	normalized, err := normalizeURL(rawURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}

	if _, err := parser.Parse(bytes.NewReader(body)); err == nil {
		return normalized, nil
	}

	base, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}
	candidates := discoverFeedCandidates(body, base)
	if len(candidates) == 0 {
		return "", fmt.Errorf("no feed discovered at %s", normalized)
	}
	return candidates[0], nil
}

func discoverFeedCandidates(body []byte, base *url.URL) []string {
	types := map[string]struct{}{
		"application/rss+xml":   {},
		"application/atom+xml":  {},
		"application/feed+json": {},
		"application/xml":       {},
		"text/xml":              {},
	}

	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	out := make([]string, 0)
	seen := map[string]struct{}{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "link") {
			attrs := attrMap(n)
			rel := strings.ToLower(attrs["rel"])
			typeAttr := strings.ToLower(attrs["type"])
			href := strings.TrimSpace(attrs["href"])
			if href != "" && strings.Contains(rel, "alternate") {
				_, typeAllowed := types[typeAttr]
				if typeAllowed || strings.Contains(typeAttr, "rss") || strings.Contains(typeAttr, "atom") || strings.Contains(typeAttr, "json") {
					if u, err := url.Parse(href); err == nil {
						abs := base.ResolveReference(u).String()
						if _, ok := seen[abs]; !ok {
							seen[abs] = struct{}{}
							out = append(out, abs)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

func attrMap(n *html.Node) map[string]string {
	m := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		m[strings.ToLower(a.Key)] = a.Val
	}
	return m
}
