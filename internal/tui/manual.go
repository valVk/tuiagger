package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

// manualState holds one manual-request-builder session — either a fresh
// draft ('m') or an in-progress edit of a saved request ('E'), matching
// App.tsx's ManualState. Query/path and header custom params live in one
// Params list (distinguished by .In) but are split into two independent
// views by splitCustomParams, matching TS's own
// nonHeaderParams/headerParams filtering of one customParams array.
//
// Focus follows useManualPanelKeyboard.ts's actual model — confirmed by
// reading it, not assumed — which is the *same* up/down-boundary-crossing
// scheme useRightPanelKeyboard.ts uses for try-it-out (HeadersFocused/
// BodyFocused booleans, no Tab-driven section cycling at all). An earlier
// version of this file used a homegrown Tab-cycling manualFocus enum that
// didn't exist in TS; replaced to match.
type manualState struct {
	Path           string
	Method         string
	Params         []storage.CustomParameter
	Body           string
	EditingRequest *storage.SavedRequest

	EditingPath bool
	PathInput   textinput.Model

	// HeaderTable backs the HEADERS table (In=="header" params only) —
	// mirrors tryItState's field exactly, see headertable.go.
	HeaderTable headerTableState

	// ParamCursor/ParamEditing address the PARAMETERS table (In!="header"
	// params) — the *default* focus whenever nothing else claims it, same
	// as try-it-out. Name/value editing widgets live on HeaderTable, shared
	// with HEADERS editing since only one table is ever mid-edit at once.
	ParamCursor  int // 0..len(non-header Params); == len is the "add new" row
	ParamEditing bool
	ParamAddNew  bool   // editing the add-new row rather than an existing one
	NewParamIn   string // in-progress type ("query"/"path") for the PARAMETERS add-new row

	BodyFocused bool
	EditingBody bool
	BodyInput   textarea.Model

	ShowSaveDialog bool
	SaveDialog     saveDialogState
}

// renameTagState backs 'R' on a custom tag row — a single text field,
// matching App.tsx's RenameTagState (rendered as its own mode rather than
// inline in the list row, per HANDOFF.md's noted adaptation).
type renameTagState struct {
	TagName string
	Input   textinput.Model
}

func newManualState() manualState {
	return manualState{
		Method:     "GET",
		NewParamIn: "query",
		PathInput:  textinput.New(),
		HeaderTable: headerTableState{
			NameInput:  textinput.New(),
			ValueInput: textinput.New(),
		},
		BodyInput: newBodyTextarea(),
	}
}

// enterManualNew starts a blank manual request draft — matches
// useAppKeyboard.ts's 'm' handler.
func (m Model) enterManualNew() Model {
	m.Manual = newManualState()
	m.Mode = ModeManual
	m.ActivePanel = PanelRight
	m.Response = nil
	m.Curl = ""
	return m
}

// enterManualEdit loads an existing saved request into the builder for
// editing — matches useAppKeyboard.ts's 'E' handler.
func (m Model) enterManualEdit(sr *storage.SavedRequest) Model {
	state := newManualState()
	state.Path = sr.Path
	state.Method = sr.Method
	state.Body = sr.Body
	state.EditingRequest = sr
	for _, p := range sr.QueryParams {
		state.Params = append(state.Params, storage.CustomParameter{ID: p.ID, Name: p.Key, Value: p.Value, In: "query", Enabled: p.Enabled})
	}
	for _, h := range sr.Headers {
		state.Params = append(state.Params, storage.CustomParameter{ID: h.ID, Name: h.Key, Value: h.Value, In: "header", Enabled: h.Enabled})
	}
	m.Manual = state
	m.Mode = ModeManual
	m.ActivePanel = PanelRight
	m.Response = nil
	m.Curl = ""
	return m
}

func (m Model) exitManual() Model {
	m.Mode = ModeBrowse
	m.Response = nil
	m.Curl = ""
	return m
}

// handleManualKey follows useManualPanelKeyboard.ts's actual dispatch order
// exactly: editingPath, then editingBody, then header/params insert modes,
// then headersFocused (swallows everything, HeadersSection's own useInput
// handles it), then bodyTabFocused, then the top-level m/p/e/s/d/Esc
// bindings, then — falling through all of the above — PARAMETERS is the
// default focus.
func (m Model) handleManualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Manual.ShowSaveDialog {
		return m.handleSaveDialogKey(msg)
	}

	key := msg.String()
	headerParams, params := splitCustomParams(m.Manual.Params)

	if m.Manual.EditingPath {
		return m.handleManualPathKey(msg)
	}
	if m.Manual.EditingBody {
		return m.handleManualBodyKey(msg)
	}
	if m.Manual.HeaderTable.Editing {
		var merged []storage.CustomParameter
		var cmd tea.Cmd
		m.Manual.HeaderTable, merged, cmd = m.Manual.HeaderTable.handleEditKey(msg, headerParams, params)
		if merged != nil {
			m.Manual.Params = merged
		}
		return m, cmd
	}
	if m.Manual.HeaderTable.Focused {
		var merged []storage.CustomParameter
		var cmd tea.Cmd
		m.Manual.HeaderTable, merged, cmd = m.Manual.HeaderTable.handleFocusedKey(msg, headerParams, params)
		if merged != nil {
			m.Manual.Params = merged
		}
		return m, cmd
	}
	if m.Manual.BodyFocused {
		switch key {
		case "i":
			m.Manual.EditingBody = true
			m.Manual.BodyInput.SetValue(m.Manual.Body)
			m.Manual.BodyInput.Focus()
		case "k", "up", "esc":
			m.Manual.BodyFocused = false
		}
		return m, nil
	}
	if m.Manual.ParamEditing {
		return m.handleManualParamEditKey(msg, headerParams, params)
	}

	// Top-level actions work regardless of focus, matching
	// useManualPanelKeyboard.ts's 'm'/'p' bindings and useAppKeyboard.ts's
	// 'e'/'s'/'d'/Esc handlers for manual mode.
	switch key {
	case "esc":
		return m.exitManual(), nil
	case "m":
		idx := indexOfMethod(m.Manual.Method)
		m.Manual.Method = httpMethods[(idx+1)%len(httpMethods)]
		return m, nil
	case "p":
		m.Manual.EditingPath = true
		m.Manual.PathInput.SetValue(m.Manual.Path)
		m.Manual.PathInput.Focus()
		return m, nil
	case "e":
		if m.Manual.Path == "" {
			return m, nil
		}
		cmd := m.manualExecuteCmd()
		m.Loading = true
		return m, cmd
	case "s":
		m.Manual.ShowSaveDialog = true
		m.Manual.SaveDialog = newSaveDialogState(m.AllTags, m.Manual.EditingRequest)
		return m, nil
	case "d":
		if m.Manual.EditingRequest != nil && m.Store != nil {
			m.Store.DeleteSavedRequest(m.Manual.EditingRequest.ID)
			return m.exitManual().refreshSavedRequests(), nil
		}
		return m, nil
	}

	// Default focus: PARAMETERS. 'k' at row 0 moves up into HEADERS; 'j'
	// past the last row moves down into BODY (write methods only) — same
	// boundary-crossing model as try-it-out's handleTryItKey.
	rows := len(params) + 1
	switch key {
	case "j", "down":
		if m.Manual.ParamCursor < rows-1 {
			m.Manual.ParamCursor++
		} else if isWriteMethod(m.Manual.Method) {
			m.Manual.BodyFocused = true
		}
		return m, nil
	case "k", "up":
		if m.Manual.ParamCursor > 0 {
			m.Manual.ParamCursor--
		} else {
			m.Manual.HeaderTable.Focused = true
		}
		return m, nil
	case "i":
		return m.enterManualParamEdit(params), nil
	case "x":
		if m.Manual.ParamCursor < len(params) {
			idx := m.Manual.ParamCursor
			params = append(params[:idx:idx], params[idx+1:]...)
			m.Manual.Params = mergeCustomParams(headerParams, params)
			if m.Manual.ParamCursor > 0 && m.Manual.ParamCursor >= len(params)+1 {
				m.Manual.ParamCursor--
			}
		}
		return m, nil
	case "c":
		if m.Manual.ParamCursor < len(params) {
			params[m.Manual.ParamCursor].In = cycleQueryPath(params[m.Manual.ParamCursor].In)
			m.Manual.Params = mergeCustomParams(headerParams, params)
		} else if m.Manual.ParamCursor == len(params) {
			m.Manual.NewParamIn = cycleQueryPath(m.Manual.NewParamIn)
		}
		return m, nil
	}
	return m, nil
}

func indexOfMethod(method string) int {
	for i, m := range httpMethods {
		if strings.EqualFold(m, method) {
			return i
		}
	}
	return 0
}

func (m Model) handleManualPathKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Manual.Path = m.Manual.PathInput.Value()
		m.Manual.EditingPath = false
		return m, nil
	}
	var cmd tea.Cmd
	m.Manual.PathInput, cmd = m.Manual.PathInput.Update(msg)
	return m, cmd
}

// handleManualBodyKey matches tryit.go's handleBodyEditKey: only Esc ends
// editing (Enter inserts a newline, per bubbles/textarea's default keymap —
// see tryit.go's body-editing hint for why that's the right binding for
// this widget rather than a literal port of the TS "Enter: done" text).
func (m Model) handleManualBodyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.Manual.Body = m.Manual.BodyInput.Value()
		m.Manual.EditingBody = false
		return m, nil
	}
	var cmd tea.Cmd
	m.Manual.BodyInput, cmd = m.Manual.BodyInput.Update(msg)
	m.Manual.Body = m.Manual.BodyInput.Value()
	return m, cmd
}

func (m Model) enterManualParamEdit(params []storage.CustomParameter) Model {
	if m.Manual.ParamCursor < len(params) {
		p := params[m.Manual.ParamCursor]
		m.Manual.ParamAddNew = false
		m.Manual.HeaderTable = m.Manual.HeaderTable.enterParamRowEdit(p.Name, p.Value)
	} else {
		m.Manual.ParamAddNew = true
		m.Manual.HeaderTable = m.Manual.HeaderTable.enterParamRowEdit("", "")
		m.Manual.NewParamIn = "query"
	}
	m.Manual.ParamEditing = true
	return m
}

// handleManualParamEditKey routes through the shared
// headerTableState.handleParamRowEditKey — see paramtable.go.
func (m Model) handleManualParamEditKey(msg tea.KeyMsg, headerParams, params []storage.CustomParameter) (tea.Model, tea.Cmd) {
	var updated []storage.CustomParameter
	var done bool
	var cmd tea.Cmd
	m.Manual.HeaderTable, updated, done, cmd = m.Manual.HeaderTable.handleParamRowEditKey(msg, m.Manual.ParamAddNew, m.Manual.ParamCursor, params, m.Manual.NewParamIn)
	if updated != nil {
		m.Manual.Params = mergeCustomParams(headerParams, updated)
		if m.Manual.ParamAddNew {
			m.Manual.ParamCursor = len(updated) - 1
		}
	}
	if done {
		m.Manual.ParamEditing = false
	}
	return m, cmd
}

func (m Model) enterRenameTag(tagName string) Model {
	input := textinput.New()
	input.SetValue(tagName)
	input.Focus()
	m.RenameTag = renameTagState{TagName: tagName, Input: input}
	m.Mode = ModeRenameTag
	return m
}

func (m Model) handleRenameTagKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Mode = ModeBrowse
		return m, nil
	case "enter":
		newName := strings.TrimSpace(m.RenameTag.Input.Value())
		if newName != "" && newName != m.RenameTag.TagName && m.Store != nil {
			m.Store.RenameCustomTag(m.RenameTag.TagName, newName)
		}
		m.Mode = ModeBrowse
		return m.refreshSavedRequests(), nil
	}
	var cmd tea.Cmd
	m.RenameTag.Input, cmd = m.RenameTag.Input.Update(msg)
	return m, cmd
}

// manualExecuteCmd builds and runs the in-progress manual draft, matching
// App.tsx's handleManualExecuteFromState. No {{env}} interpolation yet —
// consistent with executeWithOverride in tryit.go, environments aren't
// wired into the TUI until Phase 5.
func (m Model) manualExecuteCmd() tea.Cmd {
	manual := m.Manual
	return m.runRequestCmd(manual.Method, manual.Path, manual.Params, manual.Body)
}

// savedRequestExecuteCmd runs a persisted saved request directly from
// browse mode, matching App.tsx's executeCurrentEndpoint's savedRequest
// branch (browse-mode 'e' quick-execute).
func (m Model) savedRequestExecuteCmd(sr *storage.SavedRequest) tea.Cmd {
	var params []storage.CustomParameter
	for _, p := range sr.QueryParams {
		params = append(params, storage.CustomParameter{ID: p.ID, Name: p.Key, Value: p.Value, In: "query", Enabled: p.Enabled})
	}
	for _, h := range sr.Headers {
		params = append(params, storage.CustomParameter{ID: h.ID, Name: h.Key, Value: h.Value, In: "header", Enabled: h.Enabled})
	}
	return m.runRequestCmd(sr.Method, sr.Path, params, sr.Body)
}

func (m Model) runRequestCmd(method, path string, params []storage.CustomParameter, body string) tea.Cmd {
	specServers := m.Spec.Spec.Servers
	selectedServer := m.SelectedServer
	client := m.HTTPClient
	store := m.Store
	security := m.Spec.Spec.Security
	var securitySchemes map[string]openapi.SecurityScheme
	if m.Spec.Spec.Components != nil {
		securitySchemes = m.Spec.Spec.Components.SecuritySchemes
	}

	if method == "" {
		method = "GET"
	}

	return func() tea.Msg {
		baseURL := "http://localhost"
		if len(specServers) > 0 {
			idx := selectedServer
			if idx < 0 || idx >= len(specServers) {
				idx = 0
			}
			baseURL = specServers[idx].URL
		}

		envVars, authCreds := loadEnvAndAuth(store)

		collector := &request.ParameterCollector{CustomParams: params, EnvVars: envVars}
		spec := request.Spec{
			Method:            method,
			BaseURL:           baseURL,
			Path:              collector.ApplyPathParams(path),
			QueryParams:       collector.QueryParams(),
			HeaderParams:      collector.HeaderParams(),
			Body:              request.Interpolate(body, envVars),
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

// renderManualPanel renders ManualRequestPanel.tsx's editor: method/path
// header, action buttons, a merged query+header PARAMS table, and (for
// write methods) a single-line BODY field — no bubbles/textarea vendored
// yet, so multi-line body editing is a deliberate scope cut, see
// HANDOFF.md.
func (m Model) renderManualPanel(height, width int) string {
	// ManualRequestPanel.tsx is always isActive while ModeManual is active
	// (there's no left/right panel split to lose focus to), so it always
	// gets the bold/thick border — but the *weight* still needs to match
	// panelBorderStyle's thick chars, not just NormalBorder in cyan.
	borderStyle, borderColor := panelBorderStyle(true)

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
		lines = append(lines, "", boldStyle.Render("BODY")+" "+dimStyle.Render("application/json"))
		// Matches renderTryItBodySection's explicit Width — without it the
		// box shrinks to fit whatever's currently typed instead of
		// spanning the panel, which looked inconsistent once this editor
		// became a multi-line textarea (a pre-existing cosmetic gap from
		// the original single-line textinput, fixed while unifying the two
		// body editors in Phase 7).
		bodyBoxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(max(inner-4, 4))
		switch {
		case manual.EditingBody:
			bodyBoxStyle = bodyBoxStyle.BorderForeground(color2xx)
		case manual.BodyFocused:
			bodyBoxStyle = bodyBoxStyle.BorderForeground(activeBorderColor)
		default:
			bodyBoxStyle = bodyBoxStyle.BorderForeground(inactiveBorderColor)
		}
		var bodyContent string
		switch {
		case manual.EditingBody:
			// Matches tryit.go's renderTryItBodySection hint — see its doc
			// comment for why this is "Enter: newline  Esc: done" and not a
			// literal port of the TS wording.
			bodyContent = manual.BodyInput.View() + "\n" + dimStyle.Render("Enter: newline  Esc: done")
		case manual.Body != "":
			bodyContent = manual.Body
		default:
			bodyContent = dimStyle.Render("i: edit")
		}
		// Same fix as renderTryItBodySection in tryit.go: bodyBoxStyle's
		// render is a multi-line string (the bordered box), so it must be
		// split into individual rows before joining the flat lines slice —
		// appending it as one element under-counts its real height and
		// overflows the panel's row budget.
		lines = append(lines, strings.Split(bodyBoxStyle.Render(bodyContent), "\n")...)
	}

	if m.Response != nil {
		lines = append(lines, m.renderResponseBlock(inner)...)
	}

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

// paramEditWidgets bundles the name/value text inputs shared by any
// custom-param row editor — both the manual builder's PARAMS table
// (manual.go) and try-it-out's PARAMETERS table (tryit.go) edit custom
// query/path params the same way, so they share this rendering rather than
// each keeping their own copy of renderCustomParamRow/renderAddParamRow.
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
