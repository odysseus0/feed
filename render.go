package main

import (
	"strings"

	markdown "github.com/JohannesKaufmann/html-to-markdown"
)

type Renderer struct {
	converter *markdown.Converter
}

func NewRenderer() *Renderer {
	c := markdown.NewConverter("", true, nil)
	return &Renderer{converter: c}
}

func (r *Renderer) HTMLToMarkdown(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	out, err := r.converter.ConvertString(html)
	if err != nil {
		return compactText(html, 4000)
	}
	return strings.TrimSpace(out)
}
