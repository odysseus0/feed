package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

func NormalizeURL(raw string) (string, error) {
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

func DiscoverFeedURL(ctx context.Context, client *http.Client, parser *gofeed.Parser, rawURL, userAgent string) (string, error) {
	normalized, err := NormalizeURL(rawURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = "feed/0.1"
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/xml, application/atom+xml, application/rss+xml, application/feed+json, text/xml, text/html, */*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if len(body) == 0 {
		return "", fmt.Errorf("empty response body from %s", normalized)
	}

	effectiveURL := normalized
	if resp.Request != nil && resp.Request.URL != nil {
		effectiveURL = resp.Request.URL.String()
	}

	if _, err := parser.Parse(bytes.NewReader(body)); err == nil {
		return effectiveURL, nil
	}

	base, err := url.Parse(effectiveURL)
	if err != nil {
		return "", err
	}
	candidates := discoverFeedCandidates(body, base)
	if len(candidates) > 0 {
		return candidates[0], nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed: %s", resp.Status)
	}
	return "", fmt.Errorf("no feed discovered at %s", effectiveURL)
}

func discoverFeedCandidates(body []byte, base *url.URL) []string {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	resolvedBase := base
	if baseHref := findBaseHref(root); baseHref != "" {
		if u, err := url.Parse(baseHref); err == nil {
			resolvedBase = base.ResolveReference(u)
		}
	}

	out := make([]string, 0)
	seen := map[string]struct{}{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "link") {
			attrs := attrMap(n)
			if !isAlternateRel(attrs["rel"]) {
				goto children
			}
			href := strings.TrimSpace(attrs["href"])
			if href == "" {
				goto children
			}
			typeAttr := strings.ToLower(strings.TrimSpace(attrs["type"]))
			if !isFeedLinkType(typeAttr, href) {
				goto children
			}
			if typeAttr == "application/json" && strings.Contains(strings.ToLower(href), "/wp-json/") {
				goto children
			}
			if u, err := url.Parse(href); err == nil {
				abs := resolvedBase.ResolveReference(u).String()
				if _, ok := seen[abs]; !ok {
					seen[abs] = struct{}{}
					out = append(out, abs)
				}
			}
		}
	children:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

func findBaseHref(root *html.Node) string {
	var baseHref string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if baseHref != "" {
			return
		}
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "base") {
			attrs := attrMap(n)
			if href := strings.TrimSpace(attrs["href"]); href != "" {
				baseHref = href
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return baseHref
}

func isAlternateRel(rel string) bool {
	for _, token := range strings.Fields(strings.ToLower(rel)) {
		if token == "alternate" {
			return true
		}
	}
	return false
}

func isFeedLinkType(typeAttr, href string) bool {
	typeAttr = strings.ToLower(strings.TrimSpace(typeAttr))
	switch typeAttr {
	case "application/rss+xml", "application/atom+xml", "application/feed+json", "application/json", "application/xml", "text/xml":
		return true
	}
	if typeAttr != "" {
		return strings.Contains(typeAttr, "rss") || strings.Contains(typeAttr, "atom") || strings.Contains(typeAttr, "feed")
	}

	h := strings.ToLower(strings.TrimSpace(href))
	checkPath := h
	if u, err := url.Parse(href); err == nil && u.Path != "" {
		checkPath = strings.ToLower(u.Path)
	}
	ext := path.Ext(checkPath)
	if ext == ".rss" || ext == ".atom" || ext == ".xml" || ext == ".json" {
		return true
	}
	return strings.Contains(h, "/feed") || strings.Contains(h, "rss") || strings.Contains(h, "atom")
}

func attrMap(n *html.Node) map[string]string {
	m := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		m[strings.ToLower(a.Key)] = a.Val
	}
	return m
}
