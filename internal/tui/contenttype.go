package tui

import (
	"sort"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// contentTypeCycle resolves a ContentTypeTab index against a fixed list of
// content types — the shared shape behind both TryIt's spec-derived list
// (sortedContentTypes) and Manual's fixed list (manualContentTypes): both
// just cycle-and-select over a []string, differing only in where that
// list comes from.
type contentTypeCycle struct {
	types []string
}

// Selected resolves tab to a content type, wrapping out-of-range values
// (positive or negative) the same way a modulo cycle always has.
// "application/json" for an empty list, matching Build()'s own
// application/json fallback.
func (c contentTypeCycle) Selected(tab int) string {
	if len(c.types) == 0 {
		return "application/json"
	}
	idx := ((tab % len(c.types)) + len(c.types)) % len(c.types)
	return c.types[idx]
}

// Next advances tab to the following entry, wrapping to 0 past the end.
// A no-op (returns tab unchanged) for a list of 0 or 1 entries — there's
// nothing to cycle to.
func (c contentTypeCycle) Next(tab int) int {
	if len(c.types) <= 1 {
		return tab
	}
	return (tab + 1) % len(c.types)
}

// IndexOf finds contentType's tab index, or 0 (the default tab) if it's
// not in the list — the shape a persisted content-type string needs to
// restore into a tab index.
func (c contentTypeCycle) IndexOf(contentType string) int {
	for i, ct := range c.types {
		if ct == contentType {
			return i
		}
	}
	return 0
}

// unsupportedContentTypes are declared-but-not-yet-encodable request body
// formats, filtered out of sortedContentTypes below so selecting one can't
// silently produce an empty/wrong body: multipart/form-data needs a
// file-attach UI and a generated boundary (deliberately deferred, see
// CLAUDE.md's Out of Scope — "File uploads / FormData"); application/
// octet-stream is raw binary, not something a schema-driven scaffolder can
// represent at all.
var unsupportedContentTypes = map[string]bool{
	"multipart/form-data":      true,
	"application/octet-stream": true,
}

// sortedContentTypes matches sortedResponseCodes' shape (browse.go): the
// declared content-type keys, sorted for deterministic cycling, minus
// formats this app can't yet encode.
func sortedContentTypes(content map[string]openapi.MediaType) []string {
	var types []string
	for ct := range content {
		if !unsupportedContentTypes[ct] {
			types = append(types, ct)
		}
	}
	sort.Strings(types)
	return types
}

// selectedContentType resolves a tryItState.ContentTypeTab index to an
// actual content-type string for the given endpoint — "application/json"
// when the endpoint has no request body at all, otherwise
// contentTypeCycle over sortedContentTypes(...).
func selectedContentType(ep *openapi.ParsedEndpoint, tab int) string {
	if ep == nil || ep.Operation.RequestBody == nil {
		return "application/json"
	}
	return contentTypeCycle{types: sortedContentTypes(ep.Operation.RequestBody.Content)}.Selected(tab)
}

// rawContentType is what actually gets persisted into
// storage.EndpointOverride.ContentType: "" at the default tab (index 0),
// matching isEmptyOverride's "nothing worth persisting" contract — an
// untouched session shouldn't mark the endpoint "*saved params" just for
// resolving to its own default content type, the same reasoning that keeps
// an auto-scaffolded-but-untouched Body from tripping isEmptyOverride
// either (see exitTryIt's doc comment). Only an explicit non-default
// selection is worth writing to disk.
func rawContentType(ep *openapi.ParsedEndpoint, tab int) string {
	if tab == 0 {
		return storage.DefaultContentType
	}
	return selectedContentType(ep, tab)
}

// manualContentTypes is the manual builder's fixed content-type cycle —
// unlike TryIt, the manual builder has no spec/schema to enumerate
// declared content types from (a hand-built request can be anything), so
// this is a fixed 3-way toggle instead of a sortedContentTypes(...)
// lookup.
var manualContentTypes = []string{"application/json", "application/x-www-form-urlencoded", "application/xml"}

func manualContentTypeCycle() contentTypeCycle {
	return contentTypeCycle{types: manualContentTypes}
}

// manualSelectedContentType resolves a manualState.ContentTypeTab index to
// its type string.
func manualSelectedContentType(tab int) string {
	return manualContentTypeCycle().Selected(tab)
}

// indexOfContentType maps a persisted content-type string (SavedRequest's
// repurposed BodyType field) back to its tab index — mirrors
// selectedContentType's override.ContentType restore in enterTryIt. Old
// saved requests with the pre-refactor literal "json" value (see
// saveManualRequest's prior hardcoded writes) don't match any entry here
// and fall back to index 0 (application/json), preserving their existing
// behavior.
func indexOfContentType(contentType string) int {
	return manualContentTypeCycle().IndexOf(contentType)
}
