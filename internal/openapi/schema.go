package openapi

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// FormatSchema renders a Schema as an indented pseudo-JSON outline for
// display, matching parser.ts's formatSchema. Unlike the TS version, no $ref
// resolution step is needed here — schemas are already fully resolved by
// convertSchemaProxy at parse time.
func FormatSchema(s *Schema, indent int) string {
	if s == nil {
		return "null"
	}

	spaces := strings.Repeat("  ", indent)
	primaryType := ""
	if len(s.Type) > 0 {
		primaryType = s.Type[0]
	}

	switch {
	case primaryType == "object" && len(s.Properties) > 0:
		var lines []string
		lines = append(lines, spaces+"{")
		required := make(map[string]bool, len(s.Required))
		for _, r := range s.Required {
			required[r] = true
		}
		for _, prop := range s.Properties {
			marker := ""
			if required[prop.Name] {
				marker = " *"
			}
			propStr := FormatSchema(prop.Schema, indent+1)
			lines = append(lines, fmt.Sprintf("%s  \"%s\"%s: %s", spaces, prop.Name, marker, strings.TrimLeft(propStr, " ")))
		}
		lines = append(lines, spaces+"}")
		return strings.Join(lines, "\n")

	case primaryType == "array" && s.Items != nil:
		itemsStr := FormatSchema(s.Items, indent)
		return fmt.Sprintf("%s[%s]", spaces, strings.TrimLeft(itemsStr, " "))

	case primaryType != "":
		typeStr := primaryType
		if s.Format != "" {
			typeStr += "(" + s.Format + ")"
		}
		if len(s.Enum) > 0 {
			parts := make([]string, len(s.Enum))
			for i, e := range s.Enum {
				parts[i] = fmt.Sprintf("%v", e)
			}
			typeStr += " enum: [" + strings.Join(parts, ", ") + "]"
		}
		return typeStr
	}

	return "{}"
}

// ScaffoldPlaceholder builds a representative placeholder value from a
// schema, matching parser.ts's scaffoldPlaceholder — used later (Phase 6)
// for auto-generating request bodies.
func ScaffoldPlaceholder(s *Schema) any {
	if s == nil {
		return nil
	}

	primaryType := ""
	if len(s.Type) > 0 {
		primaryType = s.Type[0]
	}

	if primaryType == "object" && len(s.Properties) > 0 {
		result := make(map[string]any, len(s.Properties))
		for _, prop := range s.Properties {
			result[prop.Name] = ScaffoldPlaceholder(prop.Schema)
		}
		return result
	}

	if primaryType == "array" && s.Items != nil {
		return []any{ScaffoldPlaceholder(s.Items)}
	}

	if len(s.Enum) > 0 {
		parts := make([]string, len(s.Enum))
		for i, e := range s.Enum {
			parts[i] = fmt.Sprintf("%v", e)
		}
		return strings.Join(parts, " | ")
	}

	switch primaryType {
	case "string":
		if s.Format != "" {
			return "<" + s.Format + ">"
		}
		return "<string>"
	case "integer", "number":
		return 0
	case "boolean":
		return false
	}

	return nil
}

var multiNewline = regexp.MustCompile(`\n\s*\n`)

// nonVisibleTags are elements whose text content is never part of a
// rendered page — <style>/<script> bodies are raw CSS/JS, <title> only
// shows in a browser tab/window chrome, never the page body. parser.ts's
// htmlToPlainText concatenates every text node with no such filtering, so
// hitting an HTML error page (a very real response body — proxies, load
// balancers, and misconfigured backends return HTML far more often than a
// clean JSON API does) leaks raw CSS into the "plain text" and duplicates
// the page's heading (once from <title>, once from <h1>). Filtering these
// out is a deliberate improvement over the TS source, not a port of it —
// flagged here per this rewrite's own convention for that.
var nonVisibleTags = map[string]bool{"style": true, "script": true, "title": true}

// HTMLToPlainText strips HTML tags from a description field or response
// body, matching parser.ts's htmlToPlainText's overall shape (text nodes
// concatenated, each closing tag adds a newline, runs of blank lines
// collapse to one) but skipping text inside nonVisibleTags — see that var's
// doc comment for why this isn't a strict line-for-line port.
func HTMLToPlainText(input string) string {
	var b strings.Builder
	tokenizer := html.NewTokenizer(strings.NewReader(input))
	skipDepth := 0

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			text := multiNewline.ReplaceAllString(b.String(), "\n")
			return strings.TrimSpace(text)
		case html.StartTagToken:
			name, _ := tokenizer.TagName()
			if nonVisibleTags[string(name)] {
				skipDepth++
			}
		case html.TextToken:
			if skipDepth == 0 {
				b.WriteString(string(tokenizer.Text()))
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			if nonVisibleTags[string(name)] && skipDepth > 0 {
				skipDepth--
			}
			b.WriteByte('\n')
		}
	}
}
