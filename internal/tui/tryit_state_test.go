package tui

import (
	"strings"
	"testing"

	"github.com/valVK/tuiagger/internal/openapi"
)

func TestEncodeFormURLEncodedFlattensTopLevelObject(t *testing.T) {
	got := encodeFormURLEncoded(map[string]any{"name": "doggie", "id": float64(10)})
	if got != "id=10&name=doggie" {
		t.Errorf("expected sorted, encoded key=value pairs, got %q", got)
	}
}

func TestEncodeFormURLEncodedJSONEncodesNestedValues(t *testing.T) {
	got := encodeFormURLEncoded(map[string]any{
		"category": map[string]any{"id": float64(1), "name": "Dogs"},
	})
	if !strings.Contains(got, "category=") {
		t.Fatalf("expected a category field, got %q", got)
	}
	// The nested object must survive as JSON text inside that one field,
	// not be silently dropped or expanded into bracket notation.
	if !strings.Contains(got, "Dogs") {
		t.Errorf("expected nested object content preserved as JSON, got %q", got)
	}
}

func TestEncodeFormURLEncodedNonObjectFallsBackToSingleField(t *testing.T) {
	got := encodeFormURLEncoded("just a string")
	if got != "value=just+a+string" {
		t.Errorf("expected a single 'value' field, got %q", got)
	}
}

func TestEncodeXMLWrapsObjectPropertiesAsSortedTags(t *testing.T) {
	got := encodeXML(map[string]any{"name": "doggie", "id": float64(10)}, "root")
	if !strings.HasPrefix(got, `<?xml version="1.0"?>`) {
		t.Errorf("expected an XML declaration, got %q", got)
	}
	idIdx := strings.Index(got, "<id>")
	nameIdx := strings.Index(got, "<name>")
	if idIdx == -1 || nameIdx == -1 || idIdx > nameIdx {
		t.Errorf("expected sorted <id> before <name>, got:\n%s", got)
	}
	if !strings.Contains(got, "<id>10</id>") || !strings.Contains(got, "<name>doggie</name>") {
		t.Errorf("expected primitive values as text content, got:\n%s", got)
	}
}

func TestEncodeXMLRepeatsTagPerArrayItem(t *testing.T) {
	got := encodeXML(map[string]any{"tags": []any{"a", "b"}}, "root")
	if strings.Count(got, "<tags>") != 2 {
		t.Errorf("expected one <tags> tag per array item, got:\n%s", got)
	}
}

func TestEncodeXMLEscapesText(t *testing.T) {
	got := encodeXML(map[string]any{"note": "<a> & \"b\""}, "root")
	if strings.Contains(got, "<a>") {
		t.Errorf("expected text content to be XML-escaped, got:\n%s", got)
	}
	if !strings.Contains(got, "&amp;") {
		t.Errorf("expected '&' escaped, got:\n%s", got)
	}
}

func TestEncodeBodyDispatchesByContentType(t *testing.T) {
	v := map[string]any{"a": float64(1)}
	cases := map[string]string{
		"application/json":                  `"a": 1`,
		"application/x-www-form-urlencoded": "a=1",
		"application/xml":                   "<a>1</a>",
		"":                                  `"a": 1`, // unknown/empty falls back to JSON
	}
	for contentType, want := range cases {
		got := encodeBody(contentType, v)
		if !strings.Contains(got, want) {
			t.Errorf("encodeBody(%q, ...) = %q, expected to contain %q", contentType, got, want)
		}
	}
}

func TestSelectedSchemaLooksUpExplicitContentType(t *testing.T) {
	xmlSchema := &openapi.Schema{Type: []string{"string"}}
	content := map[string]openapi.MediaType{
		"application/json": {Schema: &openapi.Schema{Type: []string{"object"}}},
		"application/xml":  {Schema: xmlSchema},
	}
	if got := selectedSchema(content, "application/xml"); got != xmlSchema {
		t.Errorf("expected the application/xml schema, got %+v", got)
	}
	if got := selectedSchema(content, "text/plain"); got != nil {
		t.Errorf("expected nil for an undeclared content type, got %+v", got)
	}
}

func TestSortedContentTypesFiltersUnsupportedAndSorts(t *testing.T) {
	content := map[string]openapi.MediaType{
		"application/json":                  {},
		"application/xml":                   {},
		"application/x-www-form-urlencoded": {},
		"multipart/form-data":               {},
		"application/octet-stream":          {},
	}
	got := sortedContentTypes(content)
	want := []string{"application/json", "application/x-www-form-urlencoded", "application/xml"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}
