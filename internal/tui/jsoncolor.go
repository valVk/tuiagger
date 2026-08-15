package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Hand-rolled JSON/schema-outline syntax highlighting — no external
// colorizer dependency (colorjson, chroma, ...) needed just for this,
// matching the same approach already used for curl output (see
// colorizeCurlLine in responseviewer.go) and reusing this app's existing
// palette rather than inventing a new one: cyan for keys/names (matches
// selected-row highlighting throughout), green for values/body content
// (matches paramRow's VALUE column and saved-body display), yellow for
// types (matches paramRow's TYPE column), red for the required marker
// (matches the '*' used elsewhere for required params), dim for
// punctuation/structure.
var (
	jsonKeyStyle      = cyanStyle
	jsonValueStyle    = lipgloss.NewStyle().Foreground(color2xx)
	jsonTypeStyle     = yellowStyle
	jsonPunctStyle    = dimStyle
	jsonRequiredStyle = lipgloss.NewStyle().Foreground(color5xx)
)

// jsonKeyLineRe matches a "key": value line from either real pretty-printed
// JSON or FormatSchema's pseudo-JSON outline — the optional " *" group only
// ever matches in the latter (a required-property marker; real JSON has no
// such thing).
var jsonKeyLineRe = regexp.MustCompile(`^(\s*)"([^"]*)"( \*)?:\s?(.*)$`)

// colorizeJSONLine colorizes one line of valid, pretty-printed JSON (as
// produced by jsonPretty — the auto-scaffolded try-it-out body preview):
// quoted keys cyan, string values green, numbers/true/false/null yellow,
// everything else (braces, brackets, commas, colons, whitespace) dim.
func colorizeJSONLine(line string) string {
	if m := jsonKeyLineRe.FindStringSubmatch(line); m != nil {
		indent, key, rest := m[1], m[2], m[4]
		return indent + jsonPunctStyle.Render(`"`) + jsonKeyStyle.Render(key) +
			jsonPunctStyle.Render(`":`) + " " + colorizeJSONValue(rest)
	}
	return colorizeJSONValue(line)
}

var jsonValueTokenRe = regexp.MustCompile(`"[^"]*"|-?\d+(\.\d+)?|\btrue\b|\bfalse\b|\bnull\b`)

func colorizeJSONValue(s string) string {
	var b strings.Builder
	last := 0
	for _, loc := range jsonValueTokenRe.FindAllStringIndex(s, -1) {
		b.WriteString(jsonPunctStyle.Render(s[last:loc[0]]))
		tok := s[loc[0]:loc[1]]
		if strings.HasPrefix(tok, `"`) {
			b.WriteString(jsonValueStyle.Render(tok))
		} else {
			b.WriteString(jsonTypeStyle.Render(tok))
		}
		last = loc[1]
	}
	b.WriteString(jsonPunctStyle.Render(s[last:]))
	return b.String()
}

// jsonTypeWordRe matches a bare identifier in a FormatSchema type
// expression — a type name ("string", "integer"), a format/enum word, or
// an enum member ("available", "pending"). All colored the same
// (jsonTypeStyle) rather than trying to tell type names from enum members
// apart — both describe "what this field is/can be", not the field's own
// identity (that's the key) or literal data (there isn't any in a schema
// outline).
var jsonTypeWordRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9]*`)

// colorizeSchemaLine colorizes one line of FormatSchema's pseudo-JSON
// schema outline (quoted keys, an optional " *" required marker, then a
// bare type expression instead of a real value — "integer(int64)",
// "string enum: [available, pending, sold]").
func colorizeSchemaLine(line string) string {
	if m := jsonKeyLineRe.FindStringSubmatch(line); m != nil {
		indent, key, marker, rest := m[1], m[2], m[3], m[4]
		out := indent + jsonPunctStyle.Render(`"`) + jsonKeyStyle.Render(key) + jsonPunctStyle.Render(`"`)
		if marker != "" {
			out += jsonRequiredStyle.Render(" *")
		}
		out += jsonPunctStyle.Render(":") + " " + colorizeSchemaType(rest)
		return out
	}
	return colorizeSchemaType(line)
}

func colorizeSchemaType(s string) string {
	var b strings.Builder
	last := 0
	for _, loc := range jsonTypeWordRe.FindAllStringIndex(s, -1) {
		b.WriteString(jsonPunctStyle.Render(s[last:loc[0]]))
		b.WriteString(jsonTypeStyle.Render(s[loc[0]:loc[1]]))
		last = loc[1]
	}
	b.WriteString(jsonPunctStyle.Render(s[last:]))
	return b.String()
}
