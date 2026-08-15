package tui

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

// httpMethods is the cycle order for the 'm' key in try-it-out mode,
// matching useRightPanelKeyboard.ts's HTTP_METHODS.
var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

// enterTryIt switches to try-it-out mode for the selected endpoint, loading
// any previously saved override — matches App.tsx's selectedItem-change
// effect (override load) plus useAppKeyboard.ts's 't' handler (mode switch).
func (m Model) enterTryIt() Model {
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint {
		return m
	}
	ep := item.Endpoint

	state := tryItState{
		ParamValues:    map[string]string{},
		DisabledParams: map[string]bool{},
	}
	if m.Store != nil {
		if override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); override != nil {
			state.ParamValues = maps.Clone(override.Params)
			for _, d := range override.DisabledParams {
				state.DisabledParams[d] = true
			}
			state.OverridePath = override.OverridePath
			state.OverrideMethod = override.OverrideMethod
			state.Body = override.Body
			state.CustomParams = slices.Clone(override.CustomParams)
		}
	}
	// Matches useAppKeyboard.ts's 't' handler: auto-fill the body with
	// realistic fake data (scaffoldBody, not the placeholder-style
	// scaffoldPlaceholder) the moment try-it-out opens, but only if there's
	// nothing there already (a saved override's body always wins).
	if state.Body == "" && ep.Operation.RequestBody != nil {
		if schema := applicationJSONSchema(ep.Operation.RequestBody.Content); schema != nil {
			if scaffolded := openapi.ScaffoldFakeBody(schema); scaffolded != nil {
				state.Body = jsonPretty(scaffolded)
			}
		}
	}
	state.HeaderTable.ValueInput = textinput.New()
	state.HeaderTable.NameInput = textinput.New()
	state.NewParamIn = "query"
	state.PathInput = textinput.New()
	state.BodyInput = newBodyTextarea()

	m.Mode = ModeTryIt
	m.ActivePanel = PanelRight
	m.TryIt = state
	// Matches useAppKeyboard.ts's 't' handler: panelNav.setRightScroll(0).
	// Without this, whatever scroll offset was left over from browsing the
	// endpoint's docs (or a previous response) carries into try-it-out,
	// which can land the view mid-way through the (now longer, since the
	// body auto-scaffolds) content instead of at the top.
	m.RightScroll = 0
	return m
}

func newBodyTextarea() textarea.Model {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.SetHeight(10)
	return ta
}

// applicationJSONSchema looks up the "application/json" media type
// specifically, matching useAppKeyboard.ts's body-scaffold trigger — unlike
// firstSchema (used for read-only docs display, where any declared content
// type is a reasonable thing to show), scaffolding a body a user will
// actually send shouldn't depend on Go's nondeterministic map iteration
// order picking, say, "multipart/form-data" instead.
func applicationJSONSchema(content map[string]openapi.MediaType) *openapi.Schema {
	if mt, ok := content["application/json"]; ok {
		return mt.Schema
	}
	return nil
}

// exitTryIt persists the in-progress edit (params, disabled set, body,
// path/method overrides) before returning to browse mode, matching
// App.tsx's Esc handler — TS saves on Esc exit, not just on execute, so
// scaffolding or hand-editing a body and then backing out without pressing
// 'e' doesn't lose the work.
//
// Deliberate divergence from TS, not a port of it: TS's Esc handler calls
// saveOverride() unconditionally, even when every field is empty — which
// resurrects an empty-but-present override (and its "*saved params"/"~"
// indicators) right after 'r' (reset) clears everything and the user exits
// normally afterward, making a reset look like it didn't take. Deleting any
// existing override instead of writing an empty one when there's nothing
// left to save closes that gap without changing the "save on exit" behavior
// for every other case (an untouched write-method endpoint's auto-scaffolded
// Body is never empty, so it still gets saved as before).
func (m Model) exitTryIt() Model {
	if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint && m.Store != nil {
		ep := item.Endpoint
		override := storage.EndpointOverride{
			Params:         m.TryIt.ParamValues,
			CustomParams:   m.TryIt.CustomParams,
			DisabledParams: disabledSlice(m.TryIt.DisabledParams),
			Body:           m.TryIt.Body,
			OverridePath:   m.TryIt.OverridePath,
			OverrideMethod: m.TryIt.OverrideMethod,
		}
		if isEmptyOverride(override) {
			m.Store.DeleteEndpointOverride(string(ep.Method), ep.Path)
		} else {
			m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, override)
		}
	}
	m.Mode = ModeBrowse
	return m
}

// isEmptyOverride reports whether an override has nothing worth persisting
// — every field at its zero value. See exitTryIt's doc comment for why this
// matters (a reset followed by a normal exit must not resurrect an
// empty-but-present override).
func isEmptyOverride(o storage.EndpointOverride) bool {
	return len(o.Params) == 0 && len(o.CustomParams) == 0 && len(o.DisabledParams) == 0 &&
		o.Body == "" && o.OverridePath == "" && o.OverrideMethod == ""
}

// tryItTotalRows matches ParametersSection.tsx's rows array: required specs,
// then optional specs (already what sortedParameters returns), then custom
// params, then one always-present "addNew" row — present even with zero
// spec parameters, which is why the section (and its hints) must never be
// skipped just because an endpoint like a POST with only a body has no
// query/path parameters of its own.
func tryItTotalRows(params []openapi.Parameter, custom []storage.CustomParameter) int {
	return len(params) + len(custom) + 1
}

// splitCustomParams separates HeadersSection.tsx's header-typed entries from
// everything ParametersSection.tsx shows (query/path) — same underlying
// list, filtered into two independent views, matching TS's
// nonHeaderParams/headerParams split.
func splitCustomParams(all []storage.CustomParameter) (headers, others []storage.CustomParameter) {
	for _, p := range all {
		if p.In == "header" {
			headers = append(headers, p)
		} else {
			others = append(others, p)
		}
	}
	return headers, others
}

// mergeCustomParams recombines a modified headers or non-header slice with
// the untouched other group, matching TS's
// `[...nonHeaderParams, ...updated]` / `[...updated, ...headerParams]`
// recombination on every HeadersSection/ParametersSection change.
func mergeCustomParams(headers, others []storage.CustomParameter) []storage.CustomParameter {
	merged := make([]storage.CustomParameter, 0, len(headers)+len(others))
	merged = append(merged, headers...)
	merged = append(merged, others...)
	return merged
}

func (m Model) handleTryItKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint {
		return m.exitTryIt(), nil
	}
	ep := item.Endpoint
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
	if m.Response != nil {
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
			if m.Curl != "" {
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.yankCurl(m.Curl)
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

func cycleQueryPath(in string) string {
	if in == "query" {
		return "path"
	}
	return "query"
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
		m.TryIt.HeaderTable.ParamField = "name"
		m.TryIt.HeaderTable.NameInput.SetValue(p.Name)
		m.TryIt.HeaderTable.NameInput.Focus()
		m.TryIt.HeaderTable.ValueInput.SetValue(p.Value)
		m.TryIt.HeaderTable.ValueInput.Blur()
	default: // add-new row
		m.TryIt.ParamEditing = true
		m.TryIt.HeaderTable.ParamField = "name"
		m.TryIt.HeaderTable.NameInput.SetValue("")
		m.TryIt.HeaderTable.NameInput.Focus()
		m.TryIt.HeaderTable.ValueInput.SetValue("")
		m.TryIt.HeaderTable.ValueInput.Blur()
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

// handleCustomParamEditKey matches useParamNavigation.ts's insertMode
// branch for 'custom'/'addNew' rows: Tab toggles which field is focused,
// Enter commits (adding a new CustomParameter for the add-new row, or
// updating the existing one), Esc cancels without saving.
func (m Model) handleCustomParamEditKey(msg tea.KeyMsg, params []openapi.Parameter, headerParams, custom []storage.CustomParameter, cursor int) (tea.Model, tea.Cmd) {
	isAddNew := cursor >= len(params)+len(custom)

	switch msg.String() {
	case "esc":
		m.TryIt.ParamEditing = false
		return m, nil
	case "tab":
		if m.TryIt.HeaderTable.ParamField == "name" {
			m.TryIt.HeaderTable.ParamField = "value"
			m.TryIt.HeaderTable.NameInput.Blur()
			m.TryIt.HeaderTable.ValueInput.Focus()
		} else {
			m.TryIt.HeaderTable.ParamField = "name"
			m.TryIt.HeaderTable.ValueInput.Blur()
			m.TryIt.HeaderTable.NameInput.Focus()
		}
		return m, nil
	case "enter":
		if isAddNew {
			name := strings.TrimSpace(m.TryIt.HeaderTable.NameInput.Value())
			if name != "" {
				in := m.TryIt.NewParamIn
				if in == "" {
					in = "query"
				}
				custom = append(custom, storage.CustomParameter{
					ID: uuid.NewString(), Name: name, Value: m.TryIt.HeaderTable.ValueInput.Value(),
					In: in, Enabled: true,
				})
				m.TryIt.CustomParams = mergeCustomParams(headerParams, custom)
				m.TryIt.ParamCursor = len(params) + len(custom) - 1
				m.TryIt.ParamEditing = false
			}
		} else {
			idx := cursor - len(params)
			custom[idx].Name = m.TryIt.HeaderTable.NameInput.Value()
			custom[idx].Value = m.TryIt.HeaderTable.ValueInput.Value()
			m.TryIt.CustomParams = mergeCustomParams(headerParams, custom)
			m.TryIt.ParamEditing = false
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.TryIt.HeaderTable.ParamField == "name" {
		m.TryIt.HeaderTable.NameInput, cmd = m.TryIt.HeaderTable.NameInput.Update(msg)
	} else {
		m.TryIt.HeaderTable.ValueInput, cmd = m.TryIt.HeaderTable.ValueInput.Update(msg)
	}
	return m, cmd
}

// renderTryItBodySection matches RightPanel.tsx's isTryItMode BODY box: a
// rounded border (gray/cyan/green for idle/focused/editing), a scaffolded
// placeholder preview with a contextual hint when empty and unfocused, the
// live bubbles/textarea view while editing, or the plain saved/scaffolded
// content otherwise.
func (m Model) renderTryItBodySection(op *openapi.Operation, width int) []string {
	var contentTypes string
	var required bool
	if op.RequestBody != nil {
		contentTypes = contentTypesOf(op.RequestBody.Content)
		required = op.RequestBody.Required
	}
	heading := boldStyle.Render("BODY") + " " + dimStyle.Render(contentTypes)
	if required {
		heading += lipgloss.NewStyle().Foreground(color5xx).Render(" *")
	}
	lines := []string{heading}

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
		var schema *openapi.Schema
		if op.RequestBody != nil {
			schema = applicationJSONSchema(op.RequestBody.Content)
		}
		var placeholderLines []string
		if schema != nil {
			if scaffold := openapi.ScaffoldPlaceholder(schema); scaffold != nil {
				placeholderLines = strings.Split(jsonPretty(scaffold), "\n")
			}
		}
		if len(placeholderLines) > 0 {
			for _, l := range placeholderLines {
				content = append(content, dimStyle.Render(l))
			}
			hint := "j: focus"
			if m.TryIt.BodyFocused {
				hint = "i: edit | k: back"
			}
			content = append(content, dimStyle.Render(hint))
		} else {
			hint := "j to focus, i to edit"
			if m.TryIt.BodyFocused {
				hint = "i: edit | k: back to params"
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
	// Matches TS: no hint is shown once the body is non-empty and not
	// being edited — RightPanel.tsx's hint text only ever renders inside
	// the `!editingBody && !body` branch above.
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
	switch msg.String() {
	case "i":
		if item := m.selectedItem(); m.TryIt.Body == "" && item != nil && item.Type == ItemEndpoint && item.Endpoint.Operation.RequestBody != nil {
			if schema := applicationJSONSchema(item.Endpoint.Operation.RequestBody.Content); schema != nil {
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
	case "e":
		// Matches useAppKeyboard.ts's global 'e' handler: it lives in a
		// separate hook from useRightPanelKeyboard.ts's bodyTabFocused
		// branch and only checks rightPanelNormalMode (unaffected by body
		// focus), so execute still works here — unlike 'm'/'p'/'r', which
		// are local to the focused hook and correctly swallowed below.
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			cmd := m.executeCmd(item.Endpoint)
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

func (m Model) resetOverride(ep *openapi.ParsedEndpoint) Model {
	if m.Store != nil {
		m.Store.DeleteEndpointOverride(string(ep.Method), ep.Path)
	}
	m.TryIt.ParamValues = map[string]string{}
	m.TryIt.DisabledParams = map[string]bool{}
	m.TryIt.CustomParams = nil
	m.TryIt.ParamCursor = 0
	m.TryIt.OverridePath = ""
	m.TryIt.OverrideMethod = ""
	m.TryIt.Body = ""
	m.TryIt.BodyFocused = false
	m.TryIt.ShowResetConfirm = false
	return m
}

func enumValues(p openapi.Parameter) []string {
	if p.Schema == nil {
		return nil
	}
	out := make([]string, 0, len(p.Schema.Enum))
	for _, e := range p.Schema.Enum {
		out = append(out, toStr(e))
	}
	return out
}

// toStr matches JS's `.toString()` coercion used throughout the TS app for
// enum/example/default values: strings pass through, other JSON scalars are
// stringified, nil becomes "".
func toStr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// executeCmd saves the current overrides and runs the request in the
// background, matching App.tsx's executeCurrentEndpoint — including the
// in-progress (possibly hand-edited) body, scaffolded once on entering
// try-it-out (see enterTryIt) rather than regenerated on every execute.
//
// Shared by try-it-out's 'e' (uses the in-progress edit state) and browse
// mode's quick-execute 'e' (uses whatever was last saved to disk) — see
// quickExecuteCmd.
func (m Model) executeCmd(ep *openapi.ParsedEndpoint) tea.Cmd {
	tryIt := m.TryIt
	return m.executeWithOverride(ep, tryIt.ParamValues, tryIt.DisabledParams, tryIt.CustomParams, tryIt.OverridePath, tryIt.OverrideMethod, tryIt.Body)
}

// quickExecuteCmd runs a request from browse mode using the endpoint's
// saved override (if any), without entering try-it-out — matches
// CLAUDE.md's "e - Quick execute (reuses saved overrides)".
func (m Model) quickExecuteCmd(ep *openapi.ParsedEndpoint) tea.Cmd {
	paramValues := map[string]string{}
	disabled := map[string]bool{}
	var customParams []storage.CustomParameter
	overridePath, overrideMethod, body := "", "", ""
	if m.Store != nil {
		if override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); override != nil {
			paramValues = override.Params
			for _, d := range override.DisabledParams {
				disabled[d] = true
			}
			customParams = override.CustomParams
			overridePath = override.OverridePath
			overrideMethod = override.OverrideMethod
			body = override.Body
		}
	}
	return m.executeWithOverride(ep, paramValues, disabled, customParams, overridePath, overrideMethod, body)
}

func (m Model) executeWithOverride(ep *openapi.ParsedEndpoint, values map[string]string, disabledSet map[string]bool, customParams []storage.CustomParameter, overridePath, overrideMethod, body string) tea.Cmd {
	method := ep.Method
	paramValues := maps.Clone(values)
	disabled := disabledSlice(disabledSet)
	specParams := ep.Operation.Parameters
	requestBody := ep.Operation.RequestBody
	security := ep.Operation.Security
	if len(security) == 0 {
		security = m.Spec.Spec.Security
	}
	var securitySchemes map[string]openapi.SecurityScheme
	if m.Spec.Spec.Components != nil {
		securitySchemes = m.Spec.Spec.Components.SecuritySchemes
	}
	servers := m.Spec.Spec.Servers
	selectedServer := m.SelectedServer
	client := m.HTTPClient
	store := m.Store

	return func() tea.Msg {
		baseURL := "http://localhost"
		if len(servers) > 0 {
			idx := selectedServer
			if idx < 0 || idx >= len(servers) {
				idx = 0
			}
			baseURL = servers[idx].URL
		}

		if store != nil {
			override := storage.EndpointOverride{
				Params:         paramValues,
				CustomParams:   customParams,
				DisabledParams: disabled,
				Body:           body,
				OverridePath:   overridePath,
				OverrideMethod: overrideMethod,
			}
			// Matches exitTryIt's isEmptyOverride check — without it, every
			// execute unconditionally persisted an override (matching
			// App.tsx's own unconditional saveOverride() call before
			// executing), so even a browse-mode quick-execute with nothing
			// ever touched (no try-it-out session, no saved override to
			// begin with) would mark the endpoint "~"/"*saved params" from
			// the request alone. Found via a user report ("if I execute
			// request even if I did not change anything... it marked as
			// overridden"). Deliberate divergence from TS, not a port of
			// it, same reasoning as exitTryIt's fix.
			if isEmptyOverride(override) {
				store.DeleteEndpointOverride(string(method), ep.Path)
			} else {
				store.SaveEndpointOverride(string(method), ep.Path, override)
			}
		}

		envVars, authCreds := loadEnvAndAuth(store)

		collector := &request.ParameterCollector{
			SpecParams:      specParams,
			CustomParams:    customParams,
			DisabledParams:  disabled,
			ParameterValues: paramValues,
			EnvVars:         envVars,
		}

		path := ep.Path
		if overridePath != "" {
			path = overridePath
		}
		effectiveMethod := string(method)
		if overrideMethod != "" {
			effectiveMethod = overrideMethod
		}

		// Fallback for callers that never went through enterTryIt (browse
		// mode's quick-execute 'e' on an endpoint with no saved override
		// yet) — same realistic-data scaffold, just generated here instead
		// of once up front.
		if body == "" && requestBody != nil && isWriteMethod(effectiveMethod) {
			if schema := applicationJSONSchema(requestBody.Content); schema != nil {
				body = jsonPretty(openapi.ScaffoldFakeBody(schema))
			}
		}
		body = request.Interpolate(body, envVars)

		spec := request.Spec{
			Method:            effectiveMethod,
			BaseURL:           baseURL,
			Path:              collector.ApplyPathParams(path),
			QueryParams:       collector.QueryParams(),
			HeaderParams:      collector.HeaderParams(),
			Body:              body,
			OperationSecurity: security,
			SecuritySchemes:   securitySchemes,
			AuthCredentials:   authCreds,
		}

		if client == nil {
			return responseMsg{response: &request.Response{Error: "no HTTP client configured"}}
		}
		resp, curl := request.Execute(client, spec)
		return responseMsg{response: resp, curl: curl}
	}
}

func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH":
		return true
	}
	return false
}

// loadEnvAndAuth reads the active environment's variables and stored auth
// credentials at execute time — shared by tryit.go and manual.go so both
// request paths get {{envVar}} interpolation and auth injection now that
// Phase 5 wired credential/environment editing into the info popup.
func loadEnvAndAuth(store *storage.Store) (envVars, authCreds map[string]string) {
	if store == nil {
		return nil, nil
	}
	envStore := store.LoadEnvironments()
	if envStore.ActiveIndex >= 0 && envStore.ActiveIndex < len(envStore.Environments) {
		envVars = envStore.Environments[envStore.ActiveIndex].Variables
	}
	authCreds = store.LoadAuth().Credentials
	return envVars, authCreds
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
	for i, p := range params {
		selected := i == m.TryIt.ParamCursor
		editing := selected && m.TryIt.ParamEditing
		editingView := ""
		if editing {
			editingView = m.TryIt.HeaderTable.ValueInput.View()
		}
		if selected && !m.TryIt.BodyFocused {
			cursorLine = len(lines)
		}
		lines = append(lines, renderParamRow(paramRowState{
			param: p, value: m.TryIt.ParamValues[p.Name], selected: selected,
			editing: editing, disabled: m.TryIt.DisabledParams[p.Name], editingView: editingView,
		}, width)...)
	}
	for i, p := range custom {
		rowIndex := len(params) + i
		selected := rowIndex == m.TryIt.ParamCursor
		editing := selected && m.TryIt.ParamEditing
		if selected && !m.TryIt.BodyFocused {
			cursorLine = len(lines)
		}
		lines = append(lines, renderCustomParamRow(p, selected, editing, widgets))
	}
	addRowIndex := len(params) + len(custom)
	addSelected := addRowIndex == m.TryIt.ParamCursor
	if addSelected && !m.TryIt.BodyFocused {
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

func jsonPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func disabledSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
