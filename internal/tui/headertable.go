package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/valVK/tuiagger/internal/storage"
)

// headerTableState backs the HEADERS table shared by try-it-out
// (tryItState) and the manual request builder (manualState) — matches
// HeadersSection.tsx, a NAME/VALUE-only editor (no TYPE/DESCRIPTION
// columns, no enum cycling) for CustomParams entries with In=="header".
//
// ParamField/NameInput/ValueInput are also reused by the PARAMETERS
// table's custom/add-new row editing (only one of the two tables can ever
// be mid-edit at once, so there's no need for two live sets of widgets).
type headerTableState struct {
	Focused bool
	Cursor  int
	Editing bool

	ParamField string // "name" | "value"
	NameInput  textinput.Model
	ValueInput textinput.Model
}

// handleFocusedKey matches useHeadersNavigation.ts's non-insert-mode
// branch: j/k move within the HEADERS table (NAME/VALUE rows + one
// always-present add-new row), overflowing either boundary exits headers
// focus entirely (TS's onTabOut/onTabBack both just clear headersFocused,
// landing back on whichever PARAMETERS row was active before).
//
// Returns the updated state and, only when headers actually changed
// (delete/toggle), the merged CustomParams slice the caller should assign
// to CustomParams — nil means "unchanged".
func (h headerTableState) handleFocusedKey(msg tea.KeyMsg, headers, others []storage.CustomParameter) (headerTableState, []storage.CustomParameter, tea.Cmd) {
	totalRows := len(headers) + 1
	switch msg.String() {
	case "j", "down":
		if h.Cursor < totalRows-1 {
			h.Cursor++
		} else {
			h.Focused = false
		}
		return h, nil, nil
	case "k", "up":
		if h.Cursor > 0 {
			h.Cursor--
		} else {
			h.Focused = false
		}
		return h, nil, nil
	case "tab", "esc":
		h.Focused = false
		return h, nil, nil
	case "i":
		h.Editing = true
		h.ParamField = "name"
		if h.Cursor < len(headers) {
			p := headers[h.Cursor]
			h.NameInput.SetValue(p.Name)
			h.ValueInput.SetValue(p.Value)
		} else {
			h.NameInput.SetValue("")
			h.ValueInput.SetValue("")
		}
		h.NameInput.Focus()
		h.ValueInput.Blur()
		return h, nil, nil
	case "x":
		if h.Cursor < len(headers) {
			idx := h.Cursor
			headers = append(headers[:idx:idx], headers[idx+1:]...)
			merged := mergeCustomParams(headers, others)
			if h.Cursor > len(headers) {
				h.Cursor = len(headers)
			}
			return h, merged, nil
		}
		return h, nil, nil
	case "d":
		if h.Cursor < len(headers) {
			headers[h.Cursor].Enabled = !headers[h.Cursor].Enabled
			return h, mergeCustomParams(headers, others), nil
		}
		return h, nil, nil
	}
	return h, nil, nil
}

// handleEditKey matches useHeadersNavigation.ts's insertMode branch: Tab
// toggles name/value focus, Enter commits, Esc cancels — no enum/type
// cycling, since header values are always plain strings.
//
// Same nil-means-unchanged CustomParams convention as handleFocusedKey.
func (h headerTableState) handleEditKey(msg tea.KeyMsg, headers, others []storage.CustomParameter) (headerTableState, []storage.CustomParameter, tea.Cmd) {
	isAddNew := h.Cursor >= len(headers)

	switch msg.String() {
	case "esc":
		h.Editing = false
		return h, nil, nil
	case "tab":
		if h.ParamField == "name" {
			h.ParamField = "value"
			h.NameInput.Blur()
			h.ValueInput.Focus()
		} else {
			h.ParamField = "name"
			h.ValueInput.Blur()
			h.NameInput.Focus()
		}
		return h, nil, nil
	case "enter":
		if isAddNew {
			name := strings.TrimSpace(h.NameInput.Value())
			if name != "" {
				headers = append(headers, storage.CustomParameter{
					ID: uuid.NewString(), Name: name, Value: h.ValueInput.Value(),
					In: "header", Enabled: true,
				})
				merged := mergeCustomParams(headers, others)
				h.Cursor = len(headers) - 1
				h.Editing = false
				return h, merged, nil
			}
			return h, nil, nil
		}
		idx := h.Cursor
		headers[idx].Name = h.NameInput.Value()
		headers[idx].Value = h.ValueInput.Value()
		merged := mergeCustomParams(headers, others)
		h.Editing = false
		return h, merged, nil
	}

	var cmd tea.Cmd
	if h.ParamField == "name" {
		h.NameInput, cmd = h.NameInput.Update(msg)
	} else {
		h.ValueInput, cmd = h.ValueInput.Update(msg)
	}
	return h, nil, cmd
}

const (
	headerCursorWidth = 3
	headerNameWidth   = 25
	headerValueWidth  = 28 // HeadersSection.tsx's own VALUE_WIDTH, distinct from paramValueWidth
)

// renderHeadersSection matches HeadersSection.tsx: a NAME/VALUE-only table
// (no TYPE/DESCRIPTION, no enum cycling) for CustomParams entries with
// In=="header", plus an always-present add-new row. Returns the rendered
// lines and the line index of the current cursor row (for auto-scroll),
// matching renderTryItLines's cursorLine convention.
func renderHeadersSection(headers []storage.CustomParameter, cursor int, focused, editing bool, field string, nameInput, valueInput textinput.Model) ([]string, int) {
	hint := ""
	if focused {
		if editing {
			hint = " Tab: switch field | Enter: confirm | Esc: cancel"
		} else {
			hint = " j/k: move | i: edit | d: toggle | x: del"
		}
	}
	lines := []string{
		boldStyle.Render("HEADERS") + dimStyle.Render(hint),
		strings.Repeat(" ", headerCursorWidth) + dimStyle.Bold(true).Render(padRight("NAME", headerNameWidth)+"VALUE"),
	}

	cursorLine := -1
	for i, h := range headers {
		selected := focused && i == cursor
		isEditingThis := focused && editing && i == cursor
		if selected {
			cursorLine = len(lines)
		}
		rowCursor := strings.Repeat(" ", headerCursorWidth)
		if selected {
			rowCursor = cyanStyle.Render(padRight("> ", headerCursorWidth))
		}
		var nameCell, valueCell string
		if isEditingThis && field == "name" {
			nameCell = padRight(nameInput.View(), headerNameWidth)
		} else {
			nameStyle := lipgloss.NewStyle()
			if selected {
				nameStyle = cyanStyle
			}
			plain := h.Name
			if plain == "" {
				plain = "-"
			}
			nameCell = nameStyle.Render(padRight(truncateTS(plain, headerNameWidth), headerNameWidth))
		}
		if isEditingThis && field == "value" {
			valueCell = valueInput.View()
		} else {
			plain := h.Value
			if plain == "" {
				plain = "-"
			}
			valueStyle := lipgloss.NewStyle().Foreground(color2xx)
			if !h.Enabled {
				valueStyle = lipgloss.NewStyle().Foreground(inactiveBorderColor)
			}
			valueCell = valueStyle.Render(truncateTS(plain, headerValueWidth))
		}
		lines = append(lines, rowCursor+nameCell+valueCell)
	}

	addSelected := focused && cursor == len(headers)
	isAddingNew := addSelected && editing
	if addSelected {
		cursorLine = len(lines)
	}
	addCursor := strings.Repeat(" ", headerCursorWidth)
	if addSelected {
		addCursor = cyanStyle.Render(padRight("> ", headerCursorWidth))
	}
	switch {
	case isAddingNew:
		var nameCell, valueCell string
		if field == "name" {
			nameCell = padRight(nameInput.View(), headerNameWidth)
		} else {
			plain := nameInput.Value()
			if plain == "" {
				plain = "-"
			}
			nameCell = cyanStyle.Render(padRight(truncateTS(plain, headerNameWidth), headerNameWidth))
		}
		if field == "value" {
			valueCell = valueInput.View()
		} else {
			plain := valueInput.Value()
			if plain == "" {
				plain = "-"
			}
			valueCell = dimStyle.Render(truncateTS(plain, headerValueWidth))
		}
		lines = append(lines, addCursor+nameCell+valueCell)
	case addSelected:
		lines = append(lines, addCursor+cyanStyle.Render("[ i: add header ]"))
	default:
		lines = append(lines, addCursor+dimStyle.Render("[ + ]"))
	}

	return lines, cursorLine
}
