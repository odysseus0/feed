package fetch

import (
	"strings"
	"testing"
)

func TestSanitizeHTML_RemovesDangerousTagsAndAttrs(t *testing.T) {
	in := `<div onclick="alert(1)"><script>alert(1)</script><a href="javascript:alert(1)" style="color:red">x</a><img src="data:image/png;base64,abcd" onerror="x"><iframe src="https://evil"></iframe></div>`
	out := SanitizeHTML(in)

	for _, bad := range []string{"<script", "onclick=", "onerror=", "style=", "<iframe", "javascript:"} {
		if strings.Contains(strings.ToLower(out), bad) {
			t.Fatalf("expected %q to be removed, got: %s", bad, out)
		}
	}
	if !strings.Contains(out, `data:image/png`) {
		t.Fatalf("expected safe data:image src to be preserved: %s", out)
	}
}

func TestSanitizeHTML_PreservesSafeMarkup(t *testing.T) {
	in := `<p>Hello <a href="https://example.com">world</a></p>`
	out := SanitizeHTML(in)
	if !strings.Contains(out, `<a href="https://example.com">world</a>`) {
		t.Fatalf("safe link should be preserved, got: %s", out)
	}
}
