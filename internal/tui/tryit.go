package tui

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		}
	}
	state.ValueInput = textinput.New()
	state.PathInput = textinput.New()

	m.Mode = ModeTryIt
	m.ActivePanel = PanelRight
	m.TryIt = state
	return m
}

func (m Model) exitTryIt() Model {
	m.Mode = ModeBrowse
	return m
}

func (m Model) handleTryItKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint {
		return m.exitTryIt(), nil
	}
	ep := item.Endpoint
	params := sortedParameters(ep.Operation.Parameters)

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

	if m.TryIt.ParamEditing {
		return m.handleParamEditKey(msg, params)
	}

	switch key {
	case "esc":
		return m.exitTryIt(), nil
	case "j", "down":
		if m.TryIt.ParamCursor < len(params)-1 {
			m.TryIt.ParamCursor++
		}
		return m, nil
	case "k", "up":
		if m.TryIt.ParamCursor > 0 {
			m.TryIt.ParamCursor--
		}
		return m, nil
	case "i":
		return m.enterParamEdit(params), nil
	case "d":
		if len(params) > 0 {
			name := params[m.TryIt.ParamCursor].Name
			if m.TryIt.DisabledParams[name] {
				delete(m.TryIt.DisabledParams, name)
			} else {
				m.TryIt.DisabledParams[name] = true
			}
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

func (m Model) enterParamEdit(params []openapi.Parameter) Model {
	if len(params) == 0 || m.TryIt.ParamCursor >= len(params) {
		return m
	}
	p := params[m.TryIt.ParamCursor]
	if m.TryIt.DisabledParams[p.Name] {
		return m
	}
	m.TryIt.ParamEditing = true
	m.TryIt.ValueInput.SetValue(m.TryIt.ParamValues[p.Name])
	m.TryIt.ValueInput.Focus()
	return m
}

func (m Model) handleParamEditKey(msg tea.KeyMsg, params []openapi.Parameter) (tea.Model, tea.Cmd) {
	if m.TryIt.ParamCursor >= len(params) {
		m.TryIt.ParamEditing = false
		return m, nil
	}
	p := params[m.TryIt.ParamCursor]

	switch msg.String() {
	case "esc", "enter":
		m.TryIt.ParamValues[p.Name] = m.TryIt.ValueInput.Value()
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
	m.TryIt.ValueInput, cmd = m.TryIt.ValueInput.Update(msg)
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
	m.TryIt.OverridePath = ""
	m.TryIt.OverrideMethod = ""
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
// background, matching App.tsx's executeCurrentEndpoint. Body editing isn't
// wired yet (deferred — see HANDOFF.md), so a scaffolded placeholder body
// is sent automatically for write methods that define a request body, same
// source (openapi.ScaffoldPlaceholder) the read-only view already renders.
//
// Shared by try-it-out's 'e' (uses the in-progress edit state) and browse
// mode's quick-execute 'e' (uses whatever was last saved to disk) — see
// quickExecuteCmd.
func (m Model) executeCmd(ep *openapi.ParsedEndpoint) tea.Cmd {
	tryIt := m.TryIt
	return m.executeWithOverride(ep, tryIt.ParamValues, tryIt.DisabledParams, tryIt.OverridePath, tryIt.OverrideMethod)
}

// quickExecuteCmd runs a request from browse mode using the endpoint's
// saved override (if any), without entering try-it-out — matches
// CLAUDE.md's "e - Quick execute (reuses saved overrides)".
func (m Model) quickExecuteCmd(ep *openapi.ParsedEndpoint) tea.Cmd {
	paramValues := map[string]string{}
	disabled := map[string]bool{}
	overridePath, overrideMethod := "", ""
	if m.Store != nil {
		if override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); override != nil {
			paramValues = override.Params
			for _, d := range override.DisabledParams {
				disabled[d] = true
			}
			overridePath = override.OverridePath
			overrideMethod = override.OverrideMethod
		}
	}
	return m.executeWithOverride(ep, paramValues, disabled, overridePath, overrideMethod)
}

func (m Model) executeWithOverride(ep *openapi.ParsedEndpoint, values map[string]string, disabledSet map[string]bool, overridePath, overrideMethod string) tea.Cmd {
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
				CustomParams:   []storage.CustomParameter{},
				DisabledParams: disabled,
				OverridePath:   overridePath,
				OverrideMethod: overrideMethod,
			}
			store.SaveEndpointOverride(string(method), ep.Path, override)
		}

		envVars, authCreds := loadEnvAndAuth(store)

		collector := &request.ParameterCollector{
			SpecParams:      specParams,
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

		body := ""
		if requestBody != nil && isWriteMethod(effectiveMethod) {
			for _, mt := range requestBody.Content {
				if mt.Schema != nil {
					body = jsonPretty(openapi.ScaffoldPlaceholder(mt.Schema))
				}
				break
			}
		}
		if body != "" {
			body = request.Interpolate(body, envVars)
		}

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
func (m Model) renderTryItLines(ep *openapi.ParsedEndpoint, width int) []string {
	op := ep.Operation
	var lines []string

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
	lines = append(lines, header, "")

	buttons := []button{}
	if methodModified || pathModified {
		buttons = append(buttons, button{"Reset (r)", yellowStyle})
	}
	buttons = append(buttons, button{"Execute (e)", greenBoldStyle}, button{"Cancel (Esc)", dimStyle})
	lines = append(lines, lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(renderButtons(buttons)), "")

	if op.Summary != "" {
		lines = append(lines, boldStyle.Render(op.Summary))
	}
	if op.Description != "" {
		lines = append(lines, wrapLines(openapi.HTMLToPlainText(op.Description), width)...)
	}

	params := sortedParameters(op.Parameters)
	if len(params) > 0 {
		lines = append(lines, "", boldStyle.Render("PARAMETERS")+"  "+renderHints([]hint{
			{"j/k", "move"}, {"i", "edit"}, {"d", "disable"}, {"←/→", "enum"},
		}))
		lines = append(lines, paramTableHeader())
		for i, p := range params {
			selected := i == m.TryIt.ParamCursor
			editing := selected && m.TryIt.ParamEditing
			editingView := ""
			if editing {
				editingView = m.TryIt.ValueInput.View()
			}
			lines = append(lines, renderParamRow(paramRowState{
				param: p, value: m.TryIt.ParamValues[p.Name], selected: selected,
				editing: editing, disabled: m.TryIt.DisabledParams[p.Name], editingView: editingView,
			}, width)...)
		}
	}

	if op.RequestBody != nil {
		lines = append(lines, "", boldStyle.Render("BODY")+" "+dimStyle.Render(contentTypesOf(op.RequestBody.Content)+" — auto-filled from schema on execute"))
		schema := firstSchema(op.RequestBody.Content)
		if schema != nil {
			for l := range strings.SplitSeq(openapi.FormatSchema(schema, 0), "\n") {
				lines = append(lines, dimStyle.Render(l))
			}
		}
	}

	// Matches ResponsesSection.tsx's isActive={isActive && !isTryItMode} —
	// this section is never "active" while in try-it-out mode, so its
	// '/:next' hint and per-tab status color never show here.
	lines = append(lines, renderResponseTabs(op, m.ResponseTab, false)...)
	lines = append(lines, m.renderResponseBlock(width)...)

	return lines
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
