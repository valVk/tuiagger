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
