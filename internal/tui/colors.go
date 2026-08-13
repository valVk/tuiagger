package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSI 4-bit color numbers, matching the TS app's named Ink colors
// (see src/utils/colors.ts) so the palette carries over exactly.
var methodColors = map[string]lipgloss.Color{
	"get":     lipgloss.Color("4"), // blue
	"post":    lipgloss.Color("2"), // green
	"put":     lipgloss.Color("3"), // yellow
	"delete":  lipgloss.Color("1"), // red
	"patch":   lipgloss.Color("6"), // cyan
	"options": lipgloss.Color("8"), // gray
	"head":    lipgloss.Color("5"), // magenta
	"trace":   lipgloss.Color("8"), // gray
}

// MethodColor maps an HTTP method to its display color, matching
// colors.ts's getMethodColor (default: gray).
func MethodColor(method string) lipgloss.Color {
	if c, ok := methodColors[strings.ToLower(method)]; ok {
		return c
	}
	return lipgloss.Color("8")
}

var (
	color2xx = lipgloss.Color("2") // green
	color3xx = lipgloss.Color("6") // cyan
	color4xx = lipgloss.Color("3") // yellow
	color5xx = lipgloss.Color("1") // red
	colorDef = lipgloss.Color("8") // gray
)

// StatusColor maps an HTTP status code to its display color, matching
// colors.ts's getStatusColor.
func StatusColor(status int) lipgloss.Color {
	switch {
	case status >= 200 && status < 300:
		return color2xx
	case status >= 300 && status < 400:
		return color3xx
	case status >= 400 && status < 500:
		return color4xx
	case status >= 500:
		return color5xx
	default:
		return colorDef
	}
}

// MethodBadge renders a method as a colored, padded badge, matching
// MethodBadge.tsx exactly: method.toUpperCase().padEnd(8), prefixed with
// one space (a 9-character interior).
func MethodBadge(method string) string {
	style := lipgloss.NewStyle().
		Background(MethodColor(method)).
		Foreground(lipgloss.Color("15")).
		Bold(true)
	return style.Render(" " + padRight(strings.ToUpper(method), 8))
}
