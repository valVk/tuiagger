package request

import "strings"

// HeaderPair preserves header insertion order for reproducible curl output
// (a Go map would randomize it), matching curlGenerator.ts's use of
// Object.entries on an insertion-ordered JS object.
type HeaderPair struct {
	Name  string
	Value string
}

// GenerateCurl renders a shell curl command, matching curlGenerator.ts.
func GenerateCurl(method, url string, headers []HeaderPair, body string) string {
	parts := []string{"curl -X '" + strings.ToUpper(method) + "'", "  '" + url + "'"}

	for _, h := range headers {
		parts = append(parts, "  -H '"+h.Name+": "+h.Value+"'")
	}

	if body != "" {
		escaped := strings.ReplaceAll(body, "'", `'\''`)
		parts = append(parts, "  -d '"+escaped+"'")
	}

	return strings.Join(parts, " \\\n")
}
