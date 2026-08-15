// Package bodyformat serializes a scaffolded request body value tree
// (whatever openapi.ScaffoldFakeBody/ScaffoldPlaceholder produced —
// map[string]any/[]any/string/float64/bool/nil, not format-specific) as
// human-editable text for a given content type, and converts that text to
// the actual bytes that belong on the wire.
//
// The two steps are deliberately separate: Encode's output is what a user
// sees and edits in the tui's BODY box, WireEncode's output is what
// actually gets sent. For JSON and XML those are the same text — Encode
// already produces real wire format. For application/x-www-form-urlencoded
// they differ on purpose: Encode produces plain, unescaped "key=value"
// lines (hand-editing a real percent-encoded query string in a textarea is
// painful — spaces as "+", punctuation as "%XX" — found via a user
// report), and WireEncode percent-encodes that text once, at send time.
//
// This package has no dependency on the tui or openapi packages — it's a
// pure text transform, usable from any caller that has a value tree and a
// content-type string.
package bodyformat

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Encode serializes v as human-editable text for contentType. Falls back
// to JSON for any type it doesn't specifically recognize (including an
// empty string), matching the wire-format default request.Build() itself
// falls back to.
func Encode(contentType string, v any) string {
	switch contentType {
	case "application/x-www-form-urlencoded":
		return encodeFormURLEncoded(v)
	case "application/xml", "text/xml":
		return encodeXML(v, "root")
	default:
		return jsonPretty(v)
	}
}

// WireEncode converts Encode's human-editable text into the actual bytes
// that belong on the wire for contentType. A no-op for JSON/XML — Encode
// already produced wire-ready text for those — so every caller can run
// body text through WireEncode unconditionally right before sending,
// without a content-type special case of its own.
func WireEncode(contentType, text string) string {
	if contentType == "application/x-www-form-urlencoded" {
		return formTextToQueryString(text)
	}
	return text
}

func jsonPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func jsonCompact(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// encodeFormURLEncoded serializes v as human-editable "key=value" text —
// plain and unescaped, one field per line. Only top-level object keys
// become fields (the format has no natural nested-object encoding). A
// plain-valued array (strings/numbers/bools) repeats its key once per
// item — matching how HTML forms and net/url.Values themselves already
// encode multi-value fields, so formTextToQueryString's line-by-line
// url.Values.Add just works without bracket-notation machinery. An
// object, or an array containing one, falls back to a single compact-JSON
// line for that key (jsonCompact, not jsonPretty — an indented value
// would span multiple lines and get misread as separate key=value pairs
// by formTextToQueryString) rather than being silently dropped or
// exploded into per-field lines nobody asked for.
func encodeFormURLEncoded(v any) string {
	obj, ok := v.(map[string]any)
	if !ok {
		// Not an object at the top level (e.g. a bare array/scalar
		// schema) — nothing sensible to key form fields by.
		return "value=" + stringify(v)
	}
	var lines []string
	for _, k := range sortedAnyKeys(obj) {
		lines = append(lines, formFieldLines(k, obj[k])...)
	}
	return strings.Join(lines, "\n")
}

func formFieldLines(key string, v any) []string {
	if arr, ok := v.([]any); ok && isPlainValueArray(arr) {
		lines := make([]string, len(arr))
		for i, item := range arr {
			lines[i] = key + "=" + stringify(item)
		}
		return lines
	}
	switch v.(type) {
	case map[string]any, []any:
		return []string{key + "=" + jsonCompact(v)}
	default:
		return []string{key + "=" + stringify(v)}
	}
}

// isPlainValueArray reports whether every element is a scalar (string,
// number, bool, nil) rather than a nested object/array — the shape
// formFieldLines can repeat one key=value line per item for.
func isPlainValueArray(arr []any) bool {
	for _, item := range arr {
		switch item.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

// formTextToQueryString converts encodeFormURLEncoded's human-editable
// "key=value" lines into the actual percent-encoded
// application/x-www-form-urlencoded wire format. Blank lines and lines
// without "=" are skipped (typing/editing artifacts, not errors worth
// surfacing here — same "trust the user's hand-typed body" model already
// in place for JSON). Repeated keys accumulate as a multi-value field
// (url.Values.Add), matching formFieldLines' one-line-per-array-item
// encoding on the way in.
func formTextToQueryString(text string) string {
	values := url.Values{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		values.Add(key, value)
	}
	return values.Encode()
}

// encodeXML serializes v as XML, recursively: object -> nested
// <key>...</key> per property (sorted for deterministic output, same
// reason jsonPretty's json.MarshalIndent sorts map keys — plain Go map
// iteration isn't ordered), array -> one repeated tag per item, primitive
// -> escaped text content. root names the top-level element since a
// request body's Schema carries no name of its own.
func encodeXML(v any, root string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>` + "\n")
	writeXMLElement(&b, root, v, 0)
	return b.String()
}

func writeXMLElement(b *strings.Builder, name string, v any, indent int) {
	pad := strings.Repeat("  ", indent)
	switch t := v.(type) {
	case map[string]any:
		b.WriteString(pad + "<" + name + ">\n")
		for _, k := range sortedAnyKeys(t) {
			writeXMLElement(b, k, t[k], indent+1)
		}
		b.WriteString(pad + "</" + name + ">\n")
	case []any:
		for _, item := range t {
			writeXMLElement(b, name, item, indent)
		}
	default:
		b.WriteString(pad + "<" + name + ">" + xmlEscape(stringify(t)) + "</" + name + ">\n")
	}
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

func sortedAnyKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// stringify matches JS's `.toString()` coercion, same semantics as the
// tui package's own toStr — duplicated here (rather than imported) to
// keep this package dependency-free of tui: strings pass through, other
// JSON scalars are stringified, nil becomes "".
func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}
