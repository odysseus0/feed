package opml

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/odysseus0/feed/internal/model"
	"golang.org/x/net/html/charset"
)

type opmlDoc struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr,omitempty"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

type opmlHead struct {
	Title string `xml:"title,omitempty"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text         string        `xml:"text,attr,omitempty"`
	Title        string        `xml:"title,attr,omitempty"`
	Type         string        `xml:"type,attr,omitempty"`
	XMLURL       string        `xml:"xmlUrl,attr,omitempty"`
	XMLURLLower  string        `xml:"xmlurl,attr,omitempty"`
	HTMLURL      string        `xml:"htmlUrl,attr,omitempty"`
	HTMLURLLower string        `xml:"htmlurl,attr,omitempty"`
	Outlines     []opmlOutline `xml:"outline,omitempty"`
}

func ReadOPML(path string) ([]string, error) {
	r, err := openOPML(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var doc opmlDoc
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}

	var urls []string
	var walk func([]opmlOutline)
	walk = func(outlines []opmlOutline) {
		for _, o := range outlines {
			if feedURL := o.FeedURL(); feedURL != "" {
				urls = append(urls, feedURL)
			}
			if len(o.Outlines) > 0 {
				walk(o.Outlines)
			}
		}
	}
	walk(doc.Body.Outlines)

	return uniqueStrings(urls), nil
}

func openOPML(path string) (io.ReadCloser, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetch %s: %s", path, resp.Status)
		}
		return resp.Body, nil
	}
	return os.Open(path)
}

func WriteOPML(w io.Writer, feeds []model.Feed) error {
	outlines := make([]opmlOutline, 0, len(feeds))
	for _, f := range feeds {
		outlines = append(outlines, opmlOutline{
			Text:    fallback(strings.TrimSpace(f.Title), f.URL),
			Title:   fallback(strings.TrimSpace(f.Title), f.URL),
			Type:    "rss",
			XMLURL:  f.URL,
			HTMLURL: f.SiteURL,
		})
	}

	doc := opmlDoc{
		Version: "2.0",
		Head: opmlHead{
			Title: "feed export",
		},
		Body: opmlBody{
			Outlines: []opmlOutline{{
				Text:     "Subscriptions",
				Title:    "Subscriptions",
				Outlines: outlines,
			}},
		},
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	if err := enc.Encode(doc); err != nil {
		return err
	}
	return enc.Flush()
}

func fallback(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (o opmlOutline) FeedURL() string {
	if v := strings.TrimSpace(o.XMLURL); v != "" {
		return v
	}
	if v := strings.TrimSpace(o.XMLURLLower); v != "" {
		return v
	}
	return ""
}
