package main

import (
	"strings"

	"golang.org/x/net/html"
)

var blockedTags = map[string]struct{}{
	"base":     {},
	"embed":    {},
	"form":     {},
	"iframe":   {},
	"input":    {},
	"link":     {},
	"meta":     {},
	"noscript": {},
	"object":   {},
	"script":   {},
	"style":    {},
	"textarea": {},
}

func SanitizeHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	doc, err := html.Parse(strings.NewReader("<body>" + raw + "</body>"))
	if err != nil {
		return raw
	}

	body := findBodyNode(doc)
	if body == nil {
		return raw
	}

	var b strings.Builder
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		sanitized := sanitizeNode(c)
		if sanitized == nil {
			continue
		}
		_ = html.Render(&b, sanitized)
	}
	return strings.TrimSpace(b.String())
}

func findBodyNode(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && strings.EqualFold(n.Data, "body") {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBodyNode(c); b != nil {
			return b
		}
	}
	return nil
}

func sanitizeNode(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	switch n.Type {
	case html.TextNode:
		return &html.Node{Type: html.TextNode, Data: n.Data}
	case html.CommentNode:
		return nil
	case html.ElementNode:
		tag := strings.ToLower(strings.TrimSpace(n.Data))
		if _, blocked := blockedTags[tag]; blocked {
			return nil
		}
		clone := &html.Node{Type: html.ElementNode, Data: n.Data, Namespace: n.Namespace}
		for _, a := range n.Attr {
			k := strings.ToLower(strings.TrimSpace(a.Key))
			if k == "" || strings.HasPrefix(k, "on") || k == "style" || k == "srcdoc" {
				continue
			}
			if isURLAttr(k) && !isSafeURL(a.Val, tag, k) {
				continue
			}
			clone.Attr = append(clone.Attr, a)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := sanitizeNode(c)
			if child != nil {
				clone.AppendChild(child)
			}
		}
		return clone
	case html.DocumentNode:
		clone := &html.Node{Type: html.DocumentNode}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := sanitizeNode(c)
			if child != nil {
				clone.AppendChild(child)
			}
		}
		return clone
	default:
		clone := &html.Node{Type: n.Type, Data: n.Data, Namespace: n.Namespace}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := sanitizeNode(c)
			if child != nil {
				clone.AppendChild(child)
			}
		}
		return clone
	}
}

func isURLAttr(k string) bool {
	switch k {
	case "href", "src", "poster", "cite", "action", "formaction", "data":
		return true
	default:
		return false
	}
}

func isSafeURL(v, tag, attr string) bool {
	u := strings.TrimSpace(strings.ToLower(v))
	if u == "" {
		return true
	}
	if strings.HasPrefix(u, "javascript:") || strings.HasPrefix(u, "vbscript:") {
		return false
	}
	if strings.HasPrefix(u, "data:") {
		if tag == "img" && attr == "src" {
			return strings.HasPrefix(u, "data:image/")
		}
		return false
	}
	return true
}
