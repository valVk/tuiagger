package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// handleTryItKey is TryIt's Update — ep comes from the endpoint snapshot
// enterTryIt captured on entry (m.TryIt.Endpoint), not a fresh
// m.selectedItem() lookup on every keystroke: left-panel navigation is
// unreachable while Mode == ModeTryIt (this handler owns every key until
// Esc), so the selection can't change out from under an in-progress
// session.
func (m Model) handleTryItKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	ep := m.TryIt.Endpoint
	if ep == nil {
		return m.exitTryIt(), nil
	}
	params := sortedParameters(ep.Operation.Parameters)
	headerParams, custom := splitCustomParams(m.TryIt.CustomParams)
	totalRows := tryItTotalRows(params, custom)

	if m.TryIt.ShowResetConfirm {
		switch key {
		case "y", "Y":
			return m.resetOverride(ep), nil
		case "n", "N", "esc":
			m.TryIt.ShowResetConfirm = false
			return m, nil
		}
		return m, nil
	}

	if m.TryIt.EditingPath {
		return m.handlePathEditKey(msg)
	}

	if m.TryIt.EditingBody {
		return m.handleBodyEditKey(msg)
	}

	if m.TryIt.BodyFocused {
		return m.handleBodyFocusedKey(msg)
	}

	if m.TryIt.HeaderTable.Editing {
		var merged []storage.CustomParameter
		var cmd tea.Cmd
		m.TryIt.HeaderTable, merged, cmd = m.TryIt.HeaderTable.handleEditKey(msg, headerParams, custom)
		if merged != nil {
			m.TryIt.CustomParams = merged
		}
		return m, cmd
	}

	if m.TryIt.HeaderTable.Focused {
		var merged []storage.CustomParameter
		var cmd tea.Cmd
		m.TryIt.HeaderTable, merged, cmd = m.TryIt.HeaderTable.handleFocusedKey(msg, headerParams, custom)
		if merged != nil {
			m.TryIt.CustomParams = merged
		}
		return m, cmd
	}

	if m.TryIt.ParamEditing {
		return m.handleParamEditKey(msg, params, headerParams, custom)
	}

	// Response-viewer keys (visual-select/yank/scroll on a response from a
	// previous execute in this session). TS's ResponseViewer.tsx is wired
	// isActive={isActive && !isTryItMode} — its own useInput hook never
	// fires at all while in try-it-out, so there's no way to copy a
	// response without leaving try-it-out first. That's a real usability
	// gap, not a TS quirk worth replicating: there's no keybinding
	// conflict (try-it-out's own switch below never uses v/y/J/K/G), and
	// "execute, then copy the result" is an extremely common workflow to
	// lock behind switching modes. Deliberately enabling it here — see
	// HANDOFF.md.
	if m.Viewer.Response != nil {
		switch key {
		case "J", "K", "G", "v", "y", `\`:
			var cmd tea.Cmd
			m.Viewer, cmd = m.Viewer.handleKey(key)
			return m, cmd
		case "C":
			// Go-only addition, not a TS port — yanks the generated curl
			// command to the clipboard, independent of tab/selection state.
			// Doesn't collide with try-it-out's own 'c' (cycle param type,
			// lowercase, further down this switch).
			if m.Viewer.Curl != "" {
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.yankCurl(m.Viewer.Curl)
				return m, cmd
			}
		case "j", "k":
			// While actively visual-selecting, lowercase j/k also drive the
			// response cursor — see the matching case in model.go's browse-
			// mode routing for why. In try-it-out specifically, lowercase
			// j/k are the param-row navigation keys, so without this a
			// user's muscle-memory 'j' after 'v' would silently move the
			// PARAMETERS cursor instead of extending the selection.
			if m.Viewer.Selecting {
				viewerKey := "J"
				if key == "k" {
					viewerKey = "K"
				}
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.handleKey(viewerKey)
				return m, cmd
			}
		case "g":
			m.Viewer, _ = m.Viewer.handleKey(key)
		case "esc":
			// Esc cancels an in-progress visual selection first, same as
			// browse mode; only exits try-it-out once nothing's selected,
			// so it doesn't swallow the "back out of try-it-out" gesture.
			if m.Viewer.Selecting {
				m.Viewer, _ = m.Viewer.handleKey(key)
				return m, nil
			}
		}
	}

	switch key {
	case "esc":
		return m.exitTryIt(), nil
	case "j", "down":
		if m.TryIt.ParamCursor < totalRows-1 {
			m.TryIt.ParamCursor++
		} else if m.tryItHasBodySection(ep) {
			m.TryIt.BodyFocused = true
		}
		return m, nil
	case "k", "up":
		if m.TryIt.ParamCursor > 0 {
			m.TryIt.ParamCursor--
		} else {
			// Matches ParametersSection.tsx's onTabBack: 'k' at the first
			// PARAMETERS row moves focus up into HEADERS.
			m.TryIt.HeaderTable.Focused = true
		}
		return m, nil
	case "i":
		return m.enterParamEdit(params, custom), nil
	case "d":
		if m.TryIt.ParamCursor < len(params) {
			name := params[m.TryIt.ParamCursor].Name
			if m.TryIt.DisabledParams[name] {
				delete(m.TryIt.DisabledParams, name)
			} else {
				m.TryIt.DisabledParams[name] = true
			}
		}
		return m, nil
	case "x":
		if m.TryIt.ParamCursor >= len(params) && m.TryIt.ParamCursor < len(params)+len(custom) {
			idx := m.TryIt.ParamCursor - len(params)
			custom = append(custom[:idx:idx], custom[idx+1:]...)
			m.TryIt.CustomParams = mergeCustomParams(headerParams, custom)
			if m.TryIt.ParamCursor > 0 && m.TryIt.ParamCursor >= tryItTotalRows(params, custom) {
				m.TryIt.ParamCursor--
			}
		}
		return m, nil
	case "c":
		if m.TryIt.ParamCursor >= len(params) && m.TryIt.ParamCursor < len(params)+len(custom) {
			idx := m.TryIt.ParamCursor - len(params)
			custom[idx].In = cycleQueryPath(custom[idx].In)
			m.TryIt.CustomParams = mergeCustomParams(headerParams, custom)
		} else if m.TryIt.ParamCursor == len(params)+len(custom) {
			m.TryIt.NewParamIn = cycleQueryPath(m.TryIt.NewParamIn)
		}
		return m, nil
	case "m":
		base := string(ep.Method)
		current := m.TryIt.OverrideMethod
		if current == "" {
			current = strings.ToUpper(base)
		}
		idx := slices.Index(httpMethods, current)
		m.TryIt.OverrideMethod = httpMethods[(idx+1)%len(httpMethods)]
		return m, nil
	case "p":
		m.TryIt.EditingPath = true
		path := m.TryIt.OverridePath
		if path == "" {
			path = ep.Path
		}
		m.TryIt.PathInput.SetValue(path)
		m.TryIt.PathInput.Focus()
		return m, nil
	case "r":
		m.TryIt.ShowResetConfirm = true
		return m, nil
	case "e":
		cmd := m.executeCmd(ep)
		m.Loading = true
		return m, cmd
	}
	return m, nil
}

// enterParamEdit dispatches to the right editor based on which row the
// cursor is on: a spec param (single ValueInput, matching SpecParamRow.tsx
// including enum-cycle-via-arrows), or a custom/add-new row (name+value
// two-field editor, matching CustomParamRow.tsx/AddNewParamRow.tsx).
func (m Model) enterParamEdit(params []openapi.Parameter, custom []storage.CustomParameter) Model {
	cursor := m.TryIt.ParamCursor
	switch {
	case cursor < len(params):
		p := params[cursor]
		if m.TryIt.DisabledParams[p.Name] {
			return m
		}
		m.TryIt.ParamEditing = true
		m.TryIt.HeaderTable.ValueInput.SetValue(m.TryIt.ParamValues[p.Name])
		m.TryIt.HeaderTable.ValueInput.Focus()
	case cursor < len(params)+len(custom):
		p := custom[cursor-len(params)]
		m.TryIt.ParamEditing = true
		m.TryIt.HeaderTable = m.TryIt.HeaderTable.enterParamRowEdit(p.Name, p.Value)
	default: // add-new row
		m.TryIt.ParamEditing = true
		m.TryIt.HeaderTable = m.TryIt.HeaderTable.enterParamRowEdit("", "")
	}
	return m
}

func (m Model) handleParamEditKey(msg tea.KeyMsg, params []openapi.Parameter, headerParams, custom []storage.CustomParameter) (tea.Model, tea.Cmd) {
	cursor := m.TryIt.ParamCursor
	if cursor < len(params) {
		return m.handleSpecParamEditKey(msg, params[cursor])
	}
	return m.handleCustomParamEditKey(msg, params, headerParams, custom, cursor)
}

func (m Model) handleSpecParamEditKey(msg tea.KeyMsg, p openapi.Parameter) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.TryIt.ParamValues[p.Name] = m.TryIt.HeaderTable.ValueInput.Value()
		m.TryIt.ParamEditing = false
		return m, nil
	case "left", "right":
		if enum := enumValues(p); len(enum) > 0 {
			current := m.TryIt.ParamValues[p.Name]
			idx := slices.Index(enum, current)
			if msg.String() == "left" {
				idx = (idx - 1 + len(enum)) % len(enum)
			} else {
				idx = (idx + 1) % len(enum)
			}
			m.TryIt.ParamValues[p.Name] = enum[idx]
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.TryIt.HeaderTable.ValueInput, cmd = m.TryIt.HeaderTable.ValueInput.Update(msg)
	return m, cmd
}

// handleCustomParamEditKey handles the non-spec (custom/add-new) rows of
// the PARAMETERS table via the shared headerTableState.handleParamRowEditKey
// — see paramtable.go.
func (m Model) handleCustomParamEditKey(msg tea.KeyMsg, params []openapi.Parameter, headerParams, custom []storage.CustomParameter, cursor int) (tea.Model, tea.Cmd) {
	isAddNew := cursor >= len(params)+len(custom)
	idx := cursor - len(params)

	var updated []storage.CustomParameter
	var done bool
	var cmd tea.Cmd
	m.TryIt.HeaderTable, updated, done, cmd = m.TryIt.HeaderTable.handleParamRowEditKey(msg, isAddNew, idx, custom, m.TryIt.NewParamIn)
	if updated != nil {
		m.TryIt.CustomParams = mergeCustomParams(headerParams, updated)
		if isAddNew {
			m.TryIt.ParamCursor = len(params) + len(updated) - 1
		}
	}
	if done {
		m.TryIt.ParamEditing = false
	}
	return m, cmd
}

// tryItHasBodySection matches RightPanel.tsx's condition for showing (and
// being able to Tab/j into) the BODY section: either the operation declares
// a request body, or the effective method is one that conventionally
// carries one — a POST/PUT/PATCH endpoint with no declared schema still
// gets an editable freeform body box in TS.
func (m Model) tryItHasBodySection(ep *openapi.ParsedEndpoint) bool {
	if ep.Operation.RequestBody != nil {
		return true
	}
	method := string(ep.Method)
	if m.TryIt.OverrideMethod != "" {
		method = m.TryIt.OverrideMethod
	}
	return isWriteMethod(method)
}

// handleBodyFocusedKey matches useRightPanelKeyboard.ts's bodyTabFocused
// branch.
func (m Model) handleBodyFocusedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ep := m.TryIt.Endpoint
	switch msg.String() {
	case "i":
		if m.TryIt.Body == "" && ep != nil && ep.Operation.RequestBody != nil {
			if schema := applicationJSONSchema(ep.Operation.RequestBody.Content); schema != nil {
				if scaffolded := openapi.ScaffoldPlaceholder(schema); scaffolded != nil {
					m.TryIt.Body = jsonPretty(scaffolded)
				}
			}
		}
		m.TryIt.EditingBody = true
		m.TryIt.BodyInput.SetValue(m.TryIt.Body)
		m.TryIt.BodyInput.Focus()
		return m, nil
	case "k", "up":
		m.TryIt.BodyFocused = false
		return m, nil
	case "j", "down":
		// Scrolls past the BODY box (e.g. to reach RESPONSES below it, or
		// see the rest of a body taller than one screen) — see
		// scrollToShowBelow's doc comment for why this doesn't get
		// snapped back on the next render.
		m.RightScroll++
		return m, nil
	case "e":
		// Matches useAppKeyboard.ts's global 'e' handler: it lives in a
		// separate hook from useRightPanelKeyboard.ts's bodyTabFocused
		// branch and only checks rightPanelNormalMode (unaffected by body
		// focus), so execute still works here — unlike 'm'/'p'/'r', which
		// are local to the focused hook and correctly swallowed below.
		if ep != nil {
			cmd := m.executeCmd(ep)
			m.Loading = true
			return m, cmd
		}
		return m, nil
	case "esc":
		// Esc while body-focused (but not editing) exits try-it-out
		// entirely rather than just unfocusing — a real TS quirk, not a
		// choice made here: useRightPanelKeyboard.ts's own bodyTabFocused
		// branch treats Escape as "unfocus" (matching its hint text), but
		// useAppKeyboard.ts's separate global Esc handler fires on the same
		// keystroke too, since bodyTabFocused isn't part of its
		// rightPanelNormalMode gate (only editingPath/editingBody/insert
		// modes are) — so it unconditionally exits to browse at the same
		// time. Ink runs both useInput hooks per keystroke, so the net
		// effect a real user sees is "exits", not "unfocuses". Replicated
		// as the actual observed behavior.
		return m.exitTryIt(), nil
	}
	return m, nil
}

func (m Model) handleBodyEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.TryIt.Body = m.TryIt.BodyInput.Value()
		m.TryIt.EditingBody = false
		return m, nil
	}
	var cmd tea.Cmd
	m.TryIt.BodyInput, cmd = m.TryIt.BodyInput.Update(msg)
	m.TryIt.Body = m.TryIt.BodyInput.Value()
	return m, cmd
}

func (m Model) handlePathEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.TryIt.OverridePath = m.TryIt.PathInput.Value()
		m.TryIt.EditingPath = false
		return m, nil
	}
	var cmd tea.Cmd
	m.TryIt.PathInput, cmd = m.TryIt.PathInput.Update(msg)
	return m, cmd
}
