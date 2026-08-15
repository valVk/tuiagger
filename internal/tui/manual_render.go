package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/storage"
)

// renderManualPanel renders ManualRequestPanel.tsx's editor: the bordered
// panel chrome plus a scrolled slice of renderManualLines' content.
func (m Model) renderManualPanel(height, width int) string {
	// ManualRequestPanel.tsx is always isActive while ModeManual is active
	// (there's no left/right panel split to lose focus to), so it always
	// gets the bold/thick border — but the *weight* still needs to match
	// panelBorderStyle's thick chars, not just NormalBorder in cyan.
	borderStyle, borderColor := panelBorderStyle(true)
	lines := m.renderManualLines(width)

	visibleHeight := max(height-2, 1)
	start := min(m.RightScroll, max(len(lines)-1, 0))
	end := min(start+visibleHeight, len(lines))
	visible := lines[start:end]
	for len(visible) < visibleHeight {
		visible = append(visible, "")
	}

	return lipgloss.NewStyle().
		Width(width-2).
		Height(height).
		Padding(0, 1).
		BorderStyle(borderStyle).
		BorderForeground(borderColor).
		Render(strings.Join(visible, "\n"))
}

// renderManualLines builds ManualRequestPanel.tsx's content — method/path
// header, action buttons, a merged query+header PARAMS table, and (for
// write methods) a multi-line BODY box shared with try-it-out via
// bodybox.go. Split from renderManualPanel (which applies scroll + border
// chrome on top) so rightPanelLineCount can measure this content without
// duplicating it — same shape as renderTryItLines/renderEndpointLines.
func (m Model) renderManualLines(width int) []string {
	// Available columns inside the box: width minus the border (1 each
	// side) minus Padding(0,1) (1 each side) — Width() below already
	// counts padding internally (lipgloss), so only the border needs
	// subtracting from the outer width passed in. Same fix as
	// renderLeftPanel/renderRightPanel in view.go.
	inner := max(width-4, 1)

	manual := m.Manual
	var lines []string

	title := " MANUAL REQUEST "
	if manual.EditingRequest != nil {
		title = " EDITING: " + manual.EditingRequest.Name + " "
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Background(activeBorderColor).Foreground(lipgloss.Color("0"))

	method := manual.Method
	if method == "" {
		method = "GET"
	}
	header := titleStyle.Render(title) + " " + MethodBadge(method) + dimStyle.Render(" (m)") + " "
	if manual.EditingPath {
		header += manual.PathInput.View()
	} else {
		path := manual.Path
		pathStyle := boldStyle
		if path == "" {
			path = "/path/required"
			pathStyle = lipgloss.NewStyle().Foreground(color5xx)
		}
		header += pathStyle.Render(path) + dimStyle.Render(" (p)")
	}
	// Rule under the heading is cyan, not gray — ManualRequestPanel.tsx's
	// heading Box uses borderColor="cyan" (RightPanel.tsx's equivalent
	// heading uses gray); a real, deliberate visual distinction, not an
	// oversight to normalize away.
	headingRule := cyanStyle.Render(strings.Repeat("─", inner))
	// No blank line between the heading box and the button row (TS has no
	// marginTop there); the section below the buttons does get one
	// (marginTop={1} on ManualRequestPanel.tsx's PARAMS-wrapping Box).
	lines = append(lines, header, headingRule)

	// Matches ManualRequestPanel.tsx's hand-composed banner: brackets touch
	// directly ("][", no gap), Delete only appears while editing a saved
	// request, and Cancel's bracket has a trailing space TS never trims.
	banner := dimStyle.Render("[ ") + greenBoldStyle.Render("Execute (e)") + dimStyle.Render(" ][ ") +
		cyanStyle.Render("Save (s)") + dimStyle.Render(" ]")
	if manual.EditingRequest != nil {
		banner += dimStyle.Render("[ ") + lipgloss.NewStyle().Foreground(color5xx).Render("Delete (d)") + dimStyle.Render(" ]")
	}
	banner += dimStyle.Render("[ Cancel (Esc) ] ")
	lines = append(lines, lipgloss.NewStyle().Width(inner).Align(lipgloss.Right).Render(banner), "")

	headerParams, params := splitCustomParams(manual.Params)

	// HeadersSection.tsx renders above ParametersSection here too, matching
	// try-it-out's renderTryItLines (same stacked HEADERS -> PARAMETERS ->
	// BODY focus order per useManualPanelKeyboard.ts).
	headersLines, _ := renderHeadersSection(headerParams, manual.HeaderTable.Cursor, manual.HeaderTable.Focused, manual.HeaderTable.Editing, manual.HeaderTable.ParamField, manual.HeaderTable.NameInput, manual.HeaderTable.ValueInput)
	lines = append(lines, headersLines...)

	// PARAMETERS is the default focus: "active" (cursor highlighting)
	// whenever HEADERS/BODY haven't claimed it, matching
	// ParametersSection.tsx's isActive={isActive && !bodyTabFocused &&
	// !editingBody && !headersFocused}.
	paramsActive := !manual.HeaderTable.Focused && !manual.BodyFocused

	// Matches ParametersSection.tsx's heading exactly (this section is
	// reused verbatim for the manual builder's non-header custom params) —
	// "PARAMETERS", not "PARAMS", same hint wording as try-it-out's.
	lines = append(lines, "", boldStyle.Render("PARAMETERS")+dimStyle.Render(" j/k: move | i: edit | d: toggle | x: del | c: type"))
	widgets := paramEditWidgets{Field: manual.HeaderTable.ParamField, NameInput: manual.HeaderTable.NameInput, ValueInput: manual.HeaderTable.ValueInput, NewParamIn: manual.NewParamIn}
	lines = append(lines, paramTableHeader())
	lines = append(lines, dimStyle.Render(strings.Repeat("─", inner)))
	for i, p := range params {
		selected := paramsActive && i == manual.ParamCursor
		editing := selected && manual.ParamEditing && !manual.ParamAddNew
		lines = append(lines, renderCustomParamRow(p, selected, editing, widgets))
	}
	addSelected := paramsActive && manual.ParamCursor == len(params)
	lines = append(lines, renderAddParamRow(addSelected, addSelected && manual.ParamEditing && manual.ParamAddNew, widgets))

	if isWriteMethod(method) {
		contentType := manualSelectedContentType(manual.ContentTypeTab)
		lines = append(lines, "")
		lines = append(lines, renderBodyHeading(manualContentTypes, contentType, manual.BodyFocused, false)...)
		// Manual has no schema to scaffold an empty-body placeholder from
		// (a hand-built request can be anything), so unlike TryIt's box it
		// shows the same "i: edit" hint whether focused or not — passing
		// the same string for both EmptyHint fields reproduces that
		// exactly rather than introducing TryIt's "j: focus" distinction
		// where Manual never had it.
		lines = append(lines, renderBodyBox(bodyBoxState{
			Width:              inner,
			Focused:            manual.BodyFocused,
			Editing:            manual.EditingBody,
			Body:               manual.Body,
			BodyInput:          manual.BodyInput,
			EmptyHintUnfocused: "i: edit",
			EmptyHintFocused:   "i: edit",
		})...)
	}

	if m.Viewer.Response != nil {
		lines = append(lines, m.renderResponseBlock(inner)...)
	}

	return lines
}

// paramEditWidgets bundles the name/value text inputs shared by any
// custom-param row editor — both the manual builder's PARAMS table
// (manual_render.go) and try-it-out's PARAMETERS table (tryit_render.go)
// edit custom query/path params the same way, so they share this rendering
// rather than each keeping their own copy of
// renderCustomParamRow/renderAddParamRow.
type paramEditWidgets struct {
	Field      string // "name" | "value"
	NameInput  textinput.Model
	ValueInput textinput.Model
	NewParamIn string
}

// renderCustomParamRow matches CustomParamRow.tsx: cursor, name, value,
// type (with a '(c)' hint when selected).
func renderCustomParamRow(p storage.CustomParameter, selected, editing bool, w paramEditWidgets) string {
	cursor := strings.Repeat(" ", paramCursorWidth)
	if selected {
		cursor = cyanStyle.Render(padRight("> ", paramCursorWidth))
	}

	nameStyle := lipgloss.NewStyle()
	if selected {
		nameStyle = cyanStyle
	}
	if !p.Enabled {
		nameStyle = nameStyle.Foreground(inactiveBorderColor).Strikethrough(true)
	}
	var nameCell string
	if editing && w.Field == "name" {
		nameCell = padRight(w.NameInput.View(), paramNameWidth) // pre-styled widget view, can't repad
	} else {
		plain := p.Name
		if plain == "" {
			plain = "-"
		}
		nameCell = nameStyle.Render(padRight(truncateTS(plain, paramNameWidth), paramNameWidth))
	}

	valueStyle := lipgloss.NewStyle().Foreground(color2xx)
	if selected {
		valueStyle = cyanStyle
	}
	if !p.Enabled {
		valueStyle = lipgloss.NewStyle().Foreground(inactiveBorderColor)
	}
	var valueCell string
	if editing && w.Field == "value" {
		valueCell = padRight(w.ValueInput.View(), paramValueWidth) // pre-styled widget view, can't repad
	} else {
		plain := p.Value
		if plain == "" {
			plain = "-"
		}
		valueCell = valueStyle.Render(padRight(truncateTS(plain, paramValueWidth), paramValueWidth))
	}

	typePlain := padRight(p.In, paramTypeWidth)
	typeCell := yellowStyle.Render(typePlain)
	if selected {
		typeCell += dimStyle.Render(" (c)")
	}

	return cursor + nameCell + valueCell + typeCell + dimStyle.Render("(custom)")
}

// renderAddParamRow matches AddNewParamRow.tsx's "[ i: add parameter ]" /
// "[ + ]" affordance.
func renderAddParamRow(selected, editing bool, w paramEditWidgets) string {
	cursor := strings.Repeat(" ", paramCursorWidth)
	if selected {
		cursor = cyanStyle.Render(padRight("> ", paramCursorWidth))
	}
	if editing {
		name := w.NameInput.View()
		if w.Field != "name" {
			name = w.NameInput.Value()
			if name == "" {
				name = "-"
			}
			name = cyanStyle.Render(name)
		}
		value := w.ValueInput.View()
		if w.Field != "value" {
			value = w.ValueInput.Value()
			if value == "" {
				value = "-"
			}
			value = dimStyle.Render(value)
		}
		return cursor + padRight(name, paramNameWidth) + padRight(value, paramValueWidth) + yellowStyle.Render(w.NewParamIn)
	}
	if !selected {
		return cursor + dimStyle.Render("[ + ]")
	}
	nameCell := cyanStyle.Render(padRight("[ i: add parameter ]", paramNameWidth))
	valueCell := dimStyle.Render(padRight("c: type", paramValueWidth))
	newParamIn := w.NewParamIn
	if newParamIn == "" {
		newParamIn = "query"
	}
	return cursor + nameCell + valueCell + yellowStyle.Render(newParamIn)
}
