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

// HTMLToPlainText strips HTML tags from a description field, matching
// parser.ts's htmlToPlainText: text nodes are concatenated, each closing tag
// adds a newline, and runs of blank lines collapse to one.
func HTMLToPlainText(input string) string {
	var b strings.Builder
	tokenizer := html.NewTokenizer(strings.NewReader(input))

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			text := multiNewline.ReplaceAllString(b.String(), "\n")
			return strings.TrimSpace(text)
		case html.TextToken:
			b.WriteString(string(tokenizer.Text()))
		case html.EndTagToken:
			b.WriteByte('\n')
		}
	}
}
