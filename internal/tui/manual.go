package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

// manualFocus mirrors useManualPanelKeyboard.ts's editingPath/bodyTabFocused
// state as a single enum: which section of ManualRequestPanel.tsx 'Tab'
// currently lands on.
type manualFocus int

const (
	manualFocusPath manualFocus = iota
	manualFocusParams
	manualFocusBody
)

// manualState holds one manual-request-builder session — either a fresh
// draft ('m') or an in-progress edit of a saved request ('E'), matching
// App.tsx's ManualState. Query and header custom params are merged into one
// Params list (distinguished by .In) rather than TS's separate
// ParametersSection/HeadersSection — a deliberate simplification, see
// HANDOFF.md.
type manualState struct {
	Path           string
	Method         string
	Params         []storage.CustomParameter
	Body           string
	EditingRequest *storage.SavedRequest

	Focus manualFocus

	EditingPath bool
	PathInput   textinput.Model

	ParamCursor  int // 0..len(Params); == len(Params) is the "add new" row
	ParamEditing bool
	ParamAddNew  bool   // editing the add-new row rather than an existing one
	ParamField   string // "name" | "value"
	NameInput    textinput.Model
	ValueInput   textinput.Model
	NewParamIn   string // in-progress type for the add-new row

	EditingBody bool
	BodyInput   textinput.Model

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
		Focus:      manualFocusPath,
		NewParamIn: "query",
		PathInput:  textinput.New(),
		NameInput:  textinput.New(),
		ValueInput: textinput.New(),
		BodyInput:  textinput.New(),
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

func (m Model) handleManualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Manual.ShowSaveDialog {
		return m.handleSaveDialogKey(msg)
	}
	if m.Manual.EditingPath {
		return m.handleManualPathKey(msg)
	}
	if m.Manual.EditingBody {
		return m.handleManualBodyKey(msg)
	}
	if m.Manual.ParamEditing {
		return m.handleManualParamEditKey(msg)
	}

	key := msg.String()

	// Top-level actions work regardless of current Focus, matching
	// useManualPanelKeyboard.ts's 'm'/'p' bindings and useAppKeyboard.ts's
	// 'e'/'s'/'d'/Esc handlers for manual mode.
	switch key {
	case "esc":
		return m.exitManual(), nil
	case "tab":
		m.Manual.Focus = m.nextManualFocus()
		return m, nil
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

	if m.Manual.Focus == manualFocusParams {
		return m.handleManualParamsKey(key)
	}
	if m.Manual.Focus == manualFocusBody && key == "i" {
		m.Manual.EditingBody = true
		m.Manual.BodyInput.SetValue(m.Manual.Body)
		m.Manual.BodyInput.Focus()
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

// nextManualFocus cycles Path -> Params -> Body (skipped for methods with no
// body) -> Path, matching "Tab: Move to next section".
func (m Model) nextManualFocus() manualFocus {
	switch m.Manual.Focus {
	case manualFocusPath:
		return manualFocusParams
	case manualFocusParams:
		if isWriteMethod(m.Manual.Method) {
			return manualFocusBody
		}
		return manualFocusPath
	default:
		return manualFocusPath
	}
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

func (m Model) handleManualBodyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Manual.Body = m.Manual.BodyInput.Value()
		m.Manual.EditingBody = false
		return m, nil
	}
	var cmd tea.Cmd
	m.Manual.BodyInput, cmd = m.Manual.BodyInput.Update(msg)
	return m, cmd
}

func (m Model) handleManualParamsKey(key string) (tea.Model, tea.Cmd) {
	rows := len(m.Manual.Params) + 1 // + add-new row
	switch key {
	case "j", "down":
		if m.Manual.ParamCursor < rows-1 {
			m.Manual.ParamCursor++
		}
		return m, nil
	case "k", "up":
		if m.Manual.ParamCursor > 0 {
			m.Manual.ParamCursor--
		}
		return m, nil
	case "i":
		return m.enterManualParamEdit(), nil
	case "x":
		if m.Manual.ParamCursor < len(m.Manual.Params) {
			m.Manual.Params = append(m.Manual.Params[:m.Manual.ParamCursor], m.Manual.Params[m.Manual.ParamCursor+1:]...)
			if m.Manual.ParamCursor > 0 && m.Manual.ParamCursor >= len(m.Manual.Params)+1 {
				m.Manual.ParamCursor--
			}
		}
		return m, nil
	case "c":
		types := []string{"query", "header", "path"}
		cycle := func(cur string) string {
			idx := 0
			for i, t := range types {
				if t == cur {
					idx = i
				}
			}
			return types[(idx+1)%len(types)]
		}
		if m.Manual.ParamCursor < len(m.Manual.Params) {
			p := &m.Manual.Params[m.Manual.ParamCursor]
			p.In = cycle(p.In)
		} else {
			m.Manual.NewParamIn = cycle(m.Manual.NewParamIn)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) enterManualParamEdit() Model {
	if m.Manual.ParamCursor < len(m.Manual.Params) {
		p := m.Manual.Params[m.Manual.ParamCursor]
		m.Manual.ParamAddNew = false
		m.Manual.NameInput.SetValue(p.Name)
		m.Manual.ValueInput.SetValue(p.Value)
	} else {
		m.Manual.ParamAddNew = true
		m.Manual.NameInput.SetValue("")
		m.Manual.ValueInput.SetValue("")
		m.Manual.NewParamIn = "query"
	}
	m.Manual.ParamField = "name"
	m.Manual.ParamEditing = true
	m.Manual.NameInput.Focus()
	m.Manual.ValueInput.Blur()
	return m
}

func (m Model) handleManualParamEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Manual.ParamEditing = false
		return m, nil
	case "tab":
		if m.Manual.ParamField == "name" {
			m.Manual.ParamField = "value"
			m.Manual.NameInput.Blur()
			m.Manual.ValueInput.Focus()
		} else {
			m.Manual.ParamField = "name"
			m.Manual.ValueInput.Blur()
			m.Manual.NameInput.Focus()
		}
		return m, nil
	case "enter":
		if m.Manual.ParamAddNew {
			name := strings.TrimSpace(m.Manual.NameInput.Value())
			if name != "" {
				m.Manual.Params = append(m.Manual.Params, storage.CustomParameter{
					ID: uuid.NewString(), Name: name, Value: m.Manual.ValueInput.Value(),
					In: m.Manual.NewParamIn, Enabled: true,
				})
				m.Manual.ParamCursor = len(m.Manual.Params) - 1
				m.Manual.ParamEditing = false
			}
		} else {
			m.Manual.Params[m.Manual.ParamCursor].Name = m.Manual.NameInput.Value()
			m.Manual.Params[m.Manual.ParamCursor].Value = m.Manual.ValueInput.Value()
			m.Manual.ParamEditing = false
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.Manual.ParamField == "name" {
		m.Manual.NameInput, cmd = m.Manual.NameInput.Update(msg)
	} else {
		m.Manual.ValueInput, cmd = m.Manual.ValueInput.Update(msg)
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

		collector := &request.ParameterCollector{CustomParams: params}
		spec := request.Spec{
			Method:            method,
			BaseURL:           baseURL,
			Path:              collector.ApplyPathParams(path),
			QueryParams:       collector.QueryParams(),
			HeaderParams:      collector.HeaderParams(),
			Body:              body,
			OperationSecurity: security,
			SecuritySchemes:   securitySchemes,
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
	borderColor := activeBorderColor

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
	lines = append(lines, header, "")

	buttons := []button{{"Execute (e)", greenBoldStyle}, {"Save (s)", cyanStyle}}
	if manual.EditingRequest != nil {
		buttons = append(buttons, button{"Delete (d)", lipgloss.NewStyle().Foreground(color5xx)})
	}
	buttons = append(buttons, button{"Cancel (Esc)", dimStyle})
	lines = append(lines, lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(renderButtons(buttons)), "")

	lines = append(lines, boldStyle.Render("PARAMS")+"  "+dimStyle.Render("query + header, cycle type with 'c'"))
	if manual.Focus == manualFocusParams {
		lines[len(lines)-1] += "  " + renderHints([]hint{{"j/k", "move"}, {"i", "edit"}, {"x", "del"}, {"c", "type"}})
	}
	lines = append(lines, paramTableHeader())
	for i, p := range manual.Params {
		selected := manual.Focus == manualFocusParams && i == manual.ParamCursor
		editing := selected && manual.ParamEditing && !manual.ParamAddNew
		lines = append(lines, renderCustomParamRow(p, selected, editing, manual))
	}
	addSelected := manual.Focus == manualFocusParams && manual.ParamCursor == len(manual.Params)
	lines = append(lines, renderAddParamRow(addSelected, addSelected && manual.ParamEditing && manual.ParamAddNew, manual))

	if isWriteMethod(method) {
		lines = append(lines, "", boldStyle.Render("BODY")+" "+dimStyle.Render("application/json"))
		bodyBoxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		if manual.Focus == manualFocusBody {
			bodyBoxStyle = bodyBoxStyle.BorderForeground(activeBorderColor)
		} else {
			bodyBoxStyle = bodyBoxStyle.BorderForeground(inactiveBorderColor)
		}
		var bodyContent string
		switch {
		case manual.EditingBody:
			bodyContent = manual.BodyInput.View()
		case manual.Body != "":
			bodyContent = manual.Body
		default:
			bodyContent = dimStyle.Render("i: edit")
		}
		lines = append(lines, bodyBoxStyle.Render(bodyContent))
	}

	if m.Response != nil {
		lines = append(lines, m.renderResponseBlock(width)...)
	}

	visibleHeight := max(height-2, 1)
	start := min(m.RightScroll, max(len(lines)-1, 0))
	end := min(start+visibleHeight, len(lines))
	visible := lines[start:end]
	for len(visible) < visibleHeight {
		visible = append(visible, "")
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Render(strings.Join(visible, "\n"))
}

// renderCustomParamRow matches CustomParamRow.tsx: cursor, name, value,
// type (with a '(c)' hint when selected).
func renderCustomParamRow(p storage.CustomParameter, selected, editing bool, manual manualState) string {
	cursor := "  "
	if selected {
		cursor = cyanStyle.Render("> ")
	}

	nameCell := p.Name
	if nameCell == "" {
		nameCell = "-"
	}
	if editing && manual.ParamField == "name" {
		nameCell = manual.NameInput.View()
	}
	nameStyle := lipgloss.NewStyle()
	if selected {
		nameStyle = cyanStyle
	}
	if !p.Enabled {
		nameStyle = nameStyle.Foreground(inactiveBorderColor).Strikethrough(true)
	}

	valueCell := p.Value
	if valueCell == "" {
		valueCell = "-"
	}
	if editing && manual.ParamField == "value" {
		valueCell = manual.ValueInput.View()
	}
	valueStyle := lipgloss.NewStyle().Foreground(color2xx)
	if selected {
		valueStyle = cyanStyle
	}
	if !p.Enabled {
		valueStyle = lipgloss.NewStyle().Foreground(inactiveBorderColor)
	}

	typeCell := yellowStyle.Render(p.In)
	if selected {
		typeCell += dimStyle.Render(" (c)")
	}

	return cursor +
		padRight(nameStyle.Render(truncateTS(nameCell, paramNameWidth)), paramNameWidth) +
		padRight(valueStyle.Render(truncateTS(valueCell, paramValueWidth)), paramValueWidth) +
		padRight(typeCell, paramTypeWidth) +
		dimStyle.Render("(custom)")
}

// renderAddParamRow matches AddNewParamRow.tsx's "[ i: add parameter ]" /
// "[ + ]" affordance.
func renderAddParamRow(selected, editing bool, manual manualState) string {
	cursor := "  "
	if selected {
		cursor = cyanStyle.Render("> ")
	}
	if editing {
		name := manual.NameInput.View()
		if manual.ParamField != "name" {
			name = manual.NameInput.Value()
			if name == "" {
				name = "-"
			}
			name = cyanStyle.Render(name)
		}
		value := manual.ValueInput.View()
		if manual.ParamField != "value" {
			value = manual.ValueInput.Value()
			if value == "" {
				value = "-"
			}
			value = dimStyle.Render(value)
		}
		return cursor + padRight(name, paramNameWidth) + padRight(value, paramValueWidth) + yellowStyle.Render(manual.NewParamIn)
	}
	label := dimStyle.Render("[ + ]")
	if selected {
		label = cyanStyle.Render("[ i: add parameter ]")
	}
	return cursor + label
}
