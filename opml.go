package main

import (
	"encoding/xml"
	"io"
	"os"
	"strings"
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
	Text     string        `xml:"text,attr,omitempty"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline,omitempty"`
}

func ReadOPML(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var doc opmlDoc
	if err := xml.NewDecoder(f).Decode(&doc); err != nil {
		return nil, err
	}

	var urls []string
	var walk func([]opmlOutline)
	walk = func(outlines []opmlOutline) {
		for _, o := range outlines {
			if strings.TrimSpace(o.XMLURL) != "" {
				urls = append(urls, strings.TrimSpace(o.XMLURL))
			}
			if len(o.Outlines) > 0 {
				walk(o.Outlines)
			}
		}
	}
	walk(doc.Body.Outlines)

	return uniqueStrings(urls), nil
}

func WriteOPML(w io.Writer, feeds []Feed) error {
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
