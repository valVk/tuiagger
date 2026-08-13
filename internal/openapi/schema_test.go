package openapi

import (
	"strings"
	"testing"
)

func TestFormatSchemaObjectWithRequiredMarker(t *testing.T) {
	s := &Schema{
		Type:     []string{"object"},
		Required: []string{"id"},
		Properties: []Property{
			{Name: "id", Schema: &Schema{Type: []string{"string"}}},
			{Name: "count", Schema: &Schema{Type: []string{"integer"}}},
		},
	}
	out := FormatSchema(s, 0)
	if !strings.Contains(out, `"id" *: string`) {
		t.Errorf("expected required marker on id, got:\n%s", out)
	}
	if !strings.Contains(out, `"count": integer`) {
		t.Errorf("expected count without marker, got:\n%s", out)
	}
}

func TestFormatSchemaEnum(t *testing.T) {
	s := &Schema{Type: []string{"string"}, Enum: []any{"a", "b"}}
	out := FormatSchema(s, 0)
	if out != "string enum: [a, b]" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestScaffoldPlaceholderObject(t *testing.T) {
	s := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "name", Schema: &Schema{Type: []string{"string"}}},
			{Name: "age", Schema: &Schema{Type: []string{"integer"}}},
			{Name: "active", Schema: &Schema{Type: []string{"boolean"}}},
		},
	}
	out := ScaffoldPlaceholder(s).(map[string]any)
	if out["name"] != "<string>" {
		t.Errorf("expected <string>, got %v", out["name"])
	}
	if out["age"] != 0 {
		t.Errorf("expected 0, got %v", out["age"])
	}
	if out["active"] != false {
		t.Errorf("expected false, got %v", out["active"])
	}
}

func TestScaffoldPlaceholderEnum(t *testing.T) {
	s := &Schema{Type: []string{"string"}, Enum: []any{"available", "pending", "sold"}}
	out := ScaffoldPlaceholder(s)
	if out != "available | pending | sold" {
		t.Errorf("unexpected: %v", out)
	}
}

func TestHTMLToPlainTextStripsTagsAndCollapsesBlankLines(t *testing.T) {
	got := HTMLToPlainText("<p>Hello</p><p>World</p>\n\n\n<b>!</b>")
	want := "Hello\nWorld\n!"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestHTMLToPlainTextSkipsStyleScriptAndTitle is a regression test for a
// real HTML error page (Python's http.server 501 response — reported by a
// user seeing a raw ":root { color-scheme: ... }" block and a duplicated
// "Error response" heading in the response viewer). <style>/<script>
// content and <title> text are never part of a rendered page and must not
// leak into the plain-text output.
func TestHTMLToPlainTextSkipsStyleScriptAndTitle(t *testing.T) {
	html := `<!DOCTYPE HTML>
<html>
<head>
<meta charset="utf-8">
<title>Error response</title>
<style type="text/css">
    :root { color-scheme: light dark; }
</style>
</head>
<body>
<h1>Error response</h1>
<p>Error code: 501</p>
<p>Message: Unsupported method ('POST').</p>
</body>
</html>`

	got := HTMLToPlainText(html)
	if strings.Contains(got, "color-scheme") {
		t.Errorf("expected CSS content stripped, got:\n%s", got)
	}
	if strings.Count(got, "Error response") != 1 {
		t.Errorf("expected the <title>-duplicated heading removed (exactly one 'Error response'), got:\n%s", got)
	}
	if !strings.Contains(got, "Error code: 501") {
		t.Errorf("expected real body text preserved, got:\n%s", got)
	}
}

func TestHTMLToPlainTextSkipsScriptContent(t *testing.T) {
	got := HTMLToPlainText(`<p>before</p><script>if (1 < 2) { alert('x'); }</script><p>after</p>`)
	if strings.Contains(got, "alert") {
		t.Errorf("expected script content stripped, got %q", got)
	}
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
