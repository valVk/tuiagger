package tui

import (
	"net/url"
	"strings"
	"testing"

	"github.com/valVK/tuiagger/internal/openapi"
)

func TestEncodeFormURLEncodedFlattensTopLevelObject(t *testing.T) {
	got := encodeFormURLEncoded(map[string]any{"name": "doggie", "id": float64(10)})
	if got != "id=10\nname=doggie" {
		t.Errorf("expected sorted, human-readable key=value lines, got %q", got)
	}
}

func TestEncodeFormURLEncodedRepeatsKeyForPlainArray(t *testing.T) {
	got := encodeFormURLEncoded(map[string]any{"tags": []any{"a", "b"}})
	if got != "tags=a\ntags=b" {
		t.Errorf("expected one line per array item, got %q", got)
	}
}

func TestEncodeFormURLEncodedFallsBackToJSONForObjectArray(t *testing.T) {
	got := encodeFormURLEncoded(map[string]any{
		"tags": []any{map[string]any{"id": float64(1), "name": "x"}},
	})
	if strings.Contains(got, "\n") {
		t.Errorf("expected a single compact-JSON line for an object array, got %q", got)
	}
	if !strings.Contains(got, `"id":1`) {
		t.Errorf("expected the object array embedded as JSON, got %q", got)
	}
}

func TestFormTextToQueryStringEncodesForWire(t *testing.T) {
	got := formTextToQueryString("name=doggie & co\nid=10\n\nnot-a-field\ntags=a\ntags=b")
	values, err := url.ParseQuery(got)
	if err != nil {
		t.Fatalf("expected a valid query string, got %q: %v", got, err)
	}
	if values.Get("name") != "doggie & co" || values.Get("id") != "10" {
		t.Errorf("expected round-tripped field values, got %+v", values)
	}
	if got2 := values["tags"]; len(got2) != 2 || got2[0] != "a" || got2[1] != "b" {
		t.Errorf("expected repeated 'tags' lines to become a multi-value field, got %+v", got2)
	}
	if _, ok := values["not-a-field"]; ok {
		t.Errorf("expected a line without '=' to be skipped, got %+v", values)
	}
	if !strings.Contains(got, "%20") && !strings.Contains(got, "+") {
		t.Errorf("expected the space in 'doggie & co' actually percent/plus-encoded on the wire, got %q", got)
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
	if got != "value=just a string" {
		t.Errorf("expected a single, unescaped 'value' field, got %q", got)
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
