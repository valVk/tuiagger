package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// renderBodyHeading renders the "BODY" label plus its optional required
// marker, "c:cycle" hint, and content-type tab line — shared by TryIt
// (where required and types come from the endpoint's schema) and Manual
// (where required is always false and types is the fixed
// manualContentTypes list).
func renderBodyHeading(types []string, activeType string, focused, required bool) []string {
	heading := boldStyle.Render("BODY")
	if required {
		heading += lipgloss.NewStyle().Foreground(color5xx).Render(" *")
	}
	if len(types) > 1 && focused {
		heading += dimStyle.Render(" c:cycle")
	}
	lines := []string{heading}
	if len(types) > 0 {
		lines = append(lines, renderContentTypeTabLine(types, activeType))
	}
	return lines
}

// renderContentTypeTabLine renders BODY's content-type selector — same
// visual shape as renderResponseTabs' status-code tab line (reverse+bold
// on the active entry), minus the status-color lookup that doesn't apply
// here.
func renderContentTypeTabLine(types []string, active string) string {
	var b strings.Builder
	for _, ct := range types {
		style := lipgloss.NewStyle()
		if ct == active {
			style = style.Reverse(true).Bold(true)
		}
		b.WriteString(style.Render(" " + ct + " "))
		b.WriteString(" ")
	}
	return b.String()
}

// bodyBoxState is what TryIt's and Manual's BODY boxes have in common: a
// rounded border colored by idle/focused/editing, a hint that changes with
// focus, the live bubbles/textarea view while editing, or the plain body
// content otherwise. What differs between the two callers — whether
// there's a schema to scaffold a placeholder from, whether "required"
// applies — stays with the caller; only the shared box mechanics live
// here.
type bodyBoxState struct {
	Width     int
	Focused   bool
	Editing   bool
	Body      string
	BodyInput textarea.Model

	// EmptyLines is the placeholder content shown when Body == "" &&
	// !Editing (TryIt's scaffolded preview, already colorized where
	// applicable) — nil for a caller with no scaffold to show (Manual).
	EmptyLines []string
	// EmptyHintUnfocused/EmptyHintFocused are the hint lines shown below
	// EmptyLines (or alone, if EmptyLines is nil) — separate
	// focused/unfocused text lets TryIt say "j: focus" vs "i: edit" while
	// Manual (which shows the same "i: edit" either way) passes the same
	// string for both.
	EmptyHintUnfocused string
	EmptyHintFocused   string
}

// renderBodyBox renders one BODY box's bordered content, already split
// into one terminal row per element (matches every other entry in the
// caller's flat []string — a lipgloss.Render() of a bordered box is
// itself a multi-line string, and appending it as a single element
// under-counts its real height, letting the total rendered output exceed
// the panel's row budget and overflow the terminal instead of scrolling
// through it correctly).
func renderBodyBox(s bodyBoxState) []string {
	borderColor := inactiveBorderColor
	switch {
	case s.Editing:
		borderColor = color2xx
	case s.Focused:
		borderColor = activeBorderColor
	}

	var content []string
	switch {
	case !s.Editing && s.Body == "":
		hint := s.EmptyHintUnfocused
		if s.Focused {
			hint = s.EmptyHintFocused
		}
		content = append(content, s.EmptyLines...)
		content = append(content, dimStyle.Render(hint))
	case s.Editing:
		// Matches RightPanel.tsx's `{editingBody && <Text dimColor>Enter:
		// done | Shift+Enter: newline | Esc: cancel</Text>}` hint below the
		// textarea — but with corrected key semantics for this widget, not
		// a verbatim copy of the TS wording. bubbles/textarea's default
		// keymap binds plain Enter to insert a newline (there's no distinct
		// Shift+Enter binding — most terminals can't even reliably tell the
		// two apart), the opposite of what TS's TextArea does. Esc is what
		// actually ends editing here, so the hint says that instead of
		// "cancel" (TS's own wording is a bit inaccurate too: body is
		// already committed to state on every keystroke via onChange, so
		// Esc doesn't truly cancel anything there either — just stops
		// editing, same as this rewrite).
		content = []string{s.BodyInput.View(), dimStyle.Render("Enter: newline  Esc: done")}
	case s.Focused:
		// Deliberate divergence from TS, not a port of it: RightPanel.tsx's
		// hint text only ever renders inside the empty-body branch above —
		// once the body has content (the common case, since try-it-out
		// auto-scaffolds one on entry), the 'i'/'k' shortcuts go silent
		// with no way to discover them while BODY is actually focused.
		// Show the same focus hint here too — found via a user report
		// ("Body does not show the shortcut i if active"). Just "i: edit",
		// not "| k: back to params" — a user found that half redundant/
		// extra once shown alongside actual content.
		content = append(strings.Split(s.Body, "\n"), dimStyle.Render("i: edit"))
	default:
		content = strings.Split(s.Body, "\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(max(s.Width-4, 4)).
		Render(strings.Join(content, "\n"))
	return strings.Split(box, "\n")
}
