package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
)

// renderTryItBodySection matches RightPanel.tsx's isTryItMode BODY box: a
// rounded border (gray/cyan/green for idle/focused/editing), a scaffolded
// placeholder preview with a contextual hint when empty and unfocused, the
// live bubbles/textarea view while editing, or the plain saved/scaffolded
// content otherwise.
func (m Model) renderTryItBodySection(op *openapi.Operation, width int) []string {
	var types []string
	var required bool
	if op.RequestBody != nil {
		types = sortedContentTypes(op.RequestBody.Content)
		required = op.RequestBody.Required
	}
	contentType := selectedContentType(m.TryIt.Endpoint, m.TryIt.ContentTypeTab)

	heading := boldStyle.Render("BODY")
	if required {
		heading += lipgloss.NewStyle().Foreground(color5xx).Render(" *")
	}
	if len(types) > 1 && m.TryIt.BodyFocused {
		heading += dimStyle.Render(" c:cycle")
	}
	lines := []string{heading}
	if len(types) > 0 {
		lines = append(lines, renderContentTypeTabLine(types, contentType))
	}

	borderColor := inactiveBorderColor
	switch {
	case m.TryIt.EditingBody:
		borderColor = color2xx
	case m.TryIt.BodyFocused:
		borderColor = activeBorderColor
	}

	var content []string
	switch {
	case !m.TryIt.EditingBody && m.TryIt.Body == "":
		var placeholderLines []string
		if op.RequestBody != nil {
			if scaffold := scaffoldFor(op.RequestBody.Content, contentType, false); scaffold != "" {
				placeholderLines = strings.Split(scaffold, "\n")
			}
		}
		if len(placeholderLines) > 0 {
			for _, l := range placeholderLines {
				if contentType == "application/json" {
					content = append(content, colorizeJSONLine(l))
				} else {
					content = append(content, l)
				}
			}
			hint := "j: focus"
			if m.TryIt.BodyFocused {
				hint = "i: edit"
			}
			content = append(content, dimStyle.Render(hint))
		} else {
			hint := "j to focus, i to edit"
			if m.TryIt.BodyFocused {
				hint = "i: edit"
			}
			content = append(content, dimStyle.Render(hint))
		}
	case m.TryIt.EditingBody:
		// Matches RightPanel.tsx's `{editingBody && <Text dimColor>Enter:
		// done | Shift+Enter: newline | Esc: cancel</Text>}` hint below the
		// textarea — but with corrected key semantics for this widget, not
		// a verbatim copy of the TS wording. bubbles/textarea's default
		// keymap binds plain Enter to insert a newline (there's no distinct
		// Shift+Enter binding — most terminals can't even reliably tell the
		// two apart), the opposite of what TS's TextArea does. Esc is what
		// actually ends editing here (see handleBodyEditKey), so the hint
		// says that instead of "cancel" (TS's own wording is a bit
		// inaccurate too: body is already committed to state on every
		// keystroke via onChange, so Esc doesn't truly cancel anything
		// there either — just stops editing, same as this rewrite).
		content = []string{m.TryIt.BodyInput.View(), dimStyle.Render("Enter: newline  Esc: done")}
	case m.TryIt.BodyFocused:
		// Deliberate divergence from TS, not a port of it: RightPanel.tsx's
		// hint text only ever renders inside the `!editingBody && !body`
		// branch above — once the body has content (the common case,
		// since try-it-out auto-scaffolds one on entry, see enterTryIt),
		// the 'i'/'k' shortcuts go silent with no way to discover them
		// while BODY is actually focused. Show the same focus hint here
		// too — found via a user report ("Body does not show the
		// shortcut i if active"). Just "i: edit", not "| k: back to
		// params" — a user found that half redundant/extra once shown
		// alongside actual content.
		content = append(strings.Split(m.TryIt.Body, "\n"), dimStyle.Render("i: edit"))
	default:
		content = strings.Split(m.TryIt.Body, "\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(max(width-4, 4)).
		Render(strings.Join(content, "\n"))
	// box is itself a multi-line string (the rendered border + its
	// content) — every other entry in this flat []string is exactly one
	// terminal row, and downstream code (renderRightPanel's scroll/pad
	// math, cursorLine tracking) assumes that invariant. Appending box as
	// a single element under-counted its real height, letting the total
	// rendered output exceed the panel's row budget and overflow the
	// terminal — which is what actually caused the "jumps to bottom, can't
	// scroll back" bug: real terminal-native scroll from writing more rows
	// than the screen height, not a scroll-offset calculation bug.
	lines = append(lines, strings.Split(box, "\n")...)
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

// renderTryItLines renders the try-it-out variant of the endpoint detail
// view: an editable method/path header and PARAMETERS table, sharing
// summary/description/body/responses rendering with the browse-mode view.
// The second return value is the line index of whatever's currently
// focused (a param row, or the BODY section) — try-it-out has no manual
// scroll key (j/k drive param/body navigation instead, matching TS), so
// renderRightPanel uses this to auto-scroll the focused row into view
// instead of relying on m.RightScroll, matching TS's scrollToParamRow.
func (m Model) renderTryItLines(ep *openapi.ParsedEndpoint, width int) ([]string, int) {
	op := ep.Operation
	var lines []string
	cursorLine := 0

	if m.TryIt.ShowResetConfirm {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).
			Render("Reset all overrides for this endpoint? (y/n)"), "")
	}

	displayMethod := string(ep.Method)
	if m.TryIt.OverrideMethod != "" {
		displayMethod = m.TryIt.OverrideMethod
	}
	displayPath := ep.Path
	if m.TryIt.OverridePath != "" {
		displayPath = m.TryIt.OverridePath
	}
	methodModified := m.TryIt.OverrideMethod != "" && !strings.EqualFold(m.TryIt.OverrideMethod, string(ep.Method))
	pathModified := m.TryIt.OverridePath != "" && m.TryIt.OverridePath != ep.Path

	header := MethodBadge(displayMethod)
	if methodModified {
		header += yellowStyle.Render("*")
	}
	header += dimStyle.Render(" (m)") + " "
	if m.TryIt.EditingPath {
		header += m.TryIt.PathInput.View()
	} else {
		pathStyle := boldStyle
		if pathModified {
			pathStyle = yellowStyle.Bold(true)
		}
		header += pathStyle.Render(displayPath)
		if pathModified {
			header += yellowStyle.Render("*")
		}
		header += dimStyle.Render(" (p)")
	}
	// No blank line before or after the button row — RightPanel.tsx renders
	// it as `marginTop={0}` directly under the heading's border-bottom, and
	// the summary/description block right after it has no explicit gap
	// either (its own Box just starts immediately).
	lines = append(lines, header, dimStyle.Render(strings.Repeat("─", width)))

	buttons := []button{}
	if methodModified || pathModified {
		buttons = append(buttons, button{"Reset (r)", yellowStyle})
	}
	buttons = append(buttons, button{"Execute (e)", greenBoldStyle}, button{"Cancel (Esc)", dimStyle})
	lines = append(lines, lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(renderButtons(buttons)))

	if op.Summary != "" {
		lines = append(lines, boldStyle.Render(op.Summary))
	}
	if op.Description != "" {
		lines = append(lines, wrapLines(openapi.HTMLToPlainText(op.Description), width)...)
	}

	// Matches ParametersSection.tsx: this section (header, hints, column
	// header, and the always-present add-new row) renders unconditionally
	// in try-it-out mode, even for an endpoint with zero spec parameters
	// (e.g. a POST whose only input is its body) — TS's rows array always
	// has at least the addNew entry. Previously this whole block was
	// skipped when len(params)==0, which silently dropped both the hints
	// and any way to add a custom query/path param for such endpoints.
	params := sortedParameters(op.Parameters)
	headerParams, custom := splitCustomParams(m.TryIt.CustomParams)
	widgets := paramEditWidgets{Field: m.TryIt.HeaderTable.ParamField, NameInput: m.TryIt.HeaderTable.NameInput, ValueInput: m.TryIt.HeaderTable.ValueInput, NewParamIn: m.TryIt.NewParamIn}

	// HeadersSection.tsx renders above ParametersSection, matching TS's
	// stacked focus order (up from PARAMETERS row 0 enters HEADERS).
	headersLines, headerCursorLine := renderHeadersSection(headerParams, m.TryIt.HeaderTable.Cursor, m.TryIt.HeaderTable.Focused, m.TryIt.HeaderTable.Editing, m.TryIt.HeaderTable.ParamField, m.TryIt.HeaderTable.NameInput, m.TryIt.HeaderTable.ValueInput)
	lines = append(lines, "")
	headersStart := len(lines)
	lines = append(lines, headersLines...)
	if m.TryIt.HeaderTable.Focused && headerCursorLine >= 0 {
		cursorLine = headersStart + headerCursorLine
	}

	// Matches ParametersSection.tsx's hint line exactly: one uniformly-dim
	// string (not per-key cyan highlighting like renderHints), "toggle" not
	// "disable", " | "-separated.
	lines = append(lines, "", boldStyle.Render("PARAMETERS")+dimStyle.Render(" j/k: move | i: edit | d: toggle | x: del | c: type"))
	lines = append(lines, paramTableHeader())
	lines = append(lines, dimStyle.Render(strings.Repeat("─", width)))
	// paramsActive matches manual_render.go's own paramsActive gate — a
	// PARAMETERS row shouldn't show as selected while HEADERS or BODY
	// currently owns focus, the same as HEADERS' own renderHeadersSection
	// requires `focused` before highlighting a row (found via a user
	// report that PARAMETERS looked "always active" regardless of which
	// section was actually focused).
	paramsActive := !m.TryIt.HeaderTable.Focused && !m.TryIt.BodyFocused
	for i, p := range params {
		selected := paramsActive && i == m.TryIt.ParamCursor
		editing := selected && m.TryIt.ParamEditing
		editingView := ""
		if editing {
			editingView = m.TryIt.HeaderTable.ValueInput.View()
		}
		if selected {
			cursorLine = len(lines)
		}
		lines = append(lines, renderParamRow(paramRowState{
			param: p, value: m.TryIt.ParamValues[p.Name], selected: selected,
			editing: editing, disabled: m.TryIt.DisabledParams[p.Name], editingView: editingView,
		}, width)...)
	}
	for i, p := range custom {
		rowIndex := len(params) + i
		selected := paramsActive && rowIndex == m.TryIt.ParamCursor
		editing := selected && m.TryIt.ParamEditing
		if selected {
			cursorLine = len(lines)
		}
		lines = append(lines, renderCustomParamRow(p, selected, editing, widgets))
	}
	addRowIndex := len(params) + len(custom)
	addSelected := paramsActive && addRowIndex == m.TryIt.ParamCursor
	if addSelected {
		cursorLine = len(lines)
	}
	lines = append(lines, renderAddParamRow(addSelected, addSelected && m.TryIt.ParamEditing, widgets))

	if m.tryItHasBodySection(ep) {
		lines = append(lines, "")
		if m.TryIt.BodyFocused || m.TryIt.EditingBody {
			cursorLine = len(lines)
		}
		lines = append(lines, m.renderTryItBodySection(op, width)...)
	}

	// Matches ResponsesSection.tsx's isActive={isActive && !isTryItMode} —
	// this section is never "active" while in try-it-out mode, so its
	// '/:next' hint and per-tab status color never show here.
	lines = append(lines, renderResponseTabs(op, m.ResponseTab, false)...)
	lines = append(lines, m.renderResponseBlock(width)...)

	return lines, cursorLine
}
