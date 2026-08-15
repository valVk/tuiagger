package tui

import (
	"testing"

	"github.com/valVK/tuiagger/internal/openapi"
)

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
