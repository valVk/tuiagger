package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
)

type infoSection int

const (
	infoServers infoSection = iota
	infoAuth
	infoEnvironments
)

type envView int

const (
	envViewList envView = iota
	envViewEdit
)

// authEditState backs AuthSection.tsx's editingScheme/editValue — 'enter'
// on a selected scheme starts editing its credential.
type authEditState struct {
	Editing bool
	Scheme  string
	Input   textinput.Model
}

// envEditState backs useEnvironmentsKeyboard.ts's whole state bundle: the
// list/edit sub-view, the variable table cursor and its own insert mode,
// and the "add a new environment" text entry.
type envEditState struct {
	View envView

	VarCursor    int
	InsertingVar bool
	VarField     string // "key" | "value"
	EditingKey   string // which existing var is being edited
	IsNewVar     bool   // editing the add-new-variable row rather than an existing one
	KeyInput     textinput.Model
	ValueInput   textinput.Model

	AddingEnv    bool
	NewNameInput textinput.Model
}

// enterInfo opens the info popup, matching App.tsx's 'i' handler. Servers
// section is fully functional (Enter selects + closes, matching
// useServersKeyboard.ts exactly); Auth and Environments are now fully
// editable (Phase 5), matching AuthSection.tsx/EnvironmentsSection.tsx.
func (m Model) enterInfo() Model {
	m.ShowInfo = true
	m.InfoSection = infoServers
	m.ServerCursor = m.SelectedServer
	m.AuthCursor = 0
	m.EnvCursor = 0
	m.Auth = authEditState{Input: textinput.New()}
	m.Env = envEditState{
		View:         envViewList,
		KeyInput:     textinput.New(),
		ValueInput:   textinput.New(),
		NewNameInput: textinput.New(),
	}
	return m
}

func (m Model) handleInfoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any in-progress text edit gets first refusal at every keystroke —
	// matches useAuthKeyboard.ts/useEnvironmentsKeyboard.ts, whose own
	// `if (editingX) { ...; return; }` guards run before InfoPopup.tsx's
	// Tab/Esc section-switching ever sees the keystroke, so a value like
	// "internal" doesn't close the popup on its 'i', switch sections on its
	// (nonexistent) Tab, etc.
	if m.Auth.Editing {
		return m.handleAuthEditKey(msg)
	}
	if m.Env.InsertingVar {
		return m.handleEnvVarEditKey(msg)
	}
	if m.Env.AddingEnv {
		return m.handleEnvNewNameKey(msg)
	}

	key := msg.String()
	inEnvEdit := m.InfoSection == infoEnvironments && m.Env.View == envViewEdit
	// TS's InfoPopup.tsx closes on 'i' everywhere except inside the
	// environment variable table (`key.escape || (input === 'i' &&
	// !(section === 'environments' && envView === 'edit'))`) — but its Esc
	// always closes the *whole* popup, even from env edit view, because
	// useEnvironmentsKeyboard.ts has no escape case there at all. That
	// leaves no way back to the environment list short of closing and
	// reopening the popup. CLAUDE.md/HelpPopup.tsx both document "Esc: back
	// to list" as the intended shortcut, so — deliberately diverging from
	// the TS source here — Esc backs out of edit view first and only closes
	// the popup on a second press.
	if key == "i" && !inEnvEdit {
		m.ShowInfo = false
		return m, nil
	}
	if key == "esc" && !inEnvEdit {
		m.ShowInfo = false
		return m, nil
	}
	if key == "tab" {
		m.InfoSection = m.nextInfoSection()
		return m, nil
	}

	switch m.InfoSection {
	case infoServers:
		return m.handleServersKey(key)
	case infoAuth:
		return m.handleAuthKey(key)
	case infoEnvironments:
		return m.handleEnvironmentsKey(key)
	}
	return m, nil
}

func (m Model) nextInfoSection() infoSection {
	order := []infoSection{infoServers}
	if m.Spec.Spec.Components != nil && len(m.Spec.Spec.Components.SecuritySchemes) > 0 {
		order = append(order, infoAuth)
	}
	order = append(order, infoEnvironments)
	for i, s := range order {
		if s == m.InfoSection {
			return order[(i+1)%len(order)]
		}
	}
	return infoServers
}

func (m Model) servers() []openapi.Server {
	if len(m.Spec.Spec.Servers) == 0 {
		return []openapi.Server{{URL: "http://localhost", Description: "Default"}}
	}
	return m.Spec.Spec.Servers
}

func (m Model) handleServersKey(key string) (tea.Model, tea.Cmd) {
	servers := m.servers()
	switch key {
	case "j", "down":
		m.ServerCursor = min(m.ServerCursor+1, len(servers)-1)
	case "k", "up":
		m.ServerCursor = max(m.ServerCursor-1, 0)
	case "enter":
		m.SelectedServer = m.ServerCursor
		m.ShowInfo = false
	}
	return m, nil
}

func (m Model) authSchemeNames() []string {
	if m.Spec.Spec.Components == nil {
		return nil
	}
	names := make([]string, 0, len(m.Spec.Spec.Components.SecuritySchemes))
	for name := range m.Spec.Spec.Components.SecuritySchemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m Model) handleAuthKey(key string) (tea.Model, tea.Cmd) {
	names := m.authSchemeNames()
	switch key {
	case "j", "down":
		m.AuthCursor = min(m.AuthCursor+1, max(len(names)-1, 0))
	case "k", "up":
		m.AuthCursor = max(m.AuthCursor-1, 0)
	case "enter":
		if len(names) == 0 {
			return m, nil
		}
		name := names[m.AuthCursor]
		value := ""
		if m.Store != nil {
			value = m.Store.LoadAuth().Credentials[name]
		}
		m.Auth.Editing = true
		m.Auth.Scheme = name
		m.Auth.Input.SetValue(value)
		m.Auth.Input.Focus()
	}
	return m, nil
}

// handleAuthEditKey matches useAuthKeyboard.ts's editingScheme branch:
// Esc/Enter both commit and exit (there's no "cancel without saving" —
// the TS source treats the two identically).
func (m Model) handleAuthEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		if m.Store != nil {
			m.Store.SetCredential(m.Auth.Scheme, m.Auth.Input.Value())
		}
		m.Auth.Editing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.Auth.Input, cmd = m.Auth.Input.Update(msg)
	return m, cmd
}

func (m Model) environments() []struct {
	Name      string
	Variables map[string]string
} {
	if m.Store == nil {
		return nil
	}
	store := m.Store.LoadEnvironments()
	out := make([]struct {
		Name      string
		Variables map[string]string
	}, len(store.Environments))
	for i, e := range store.Environments {
		out[i] = struct {
			Name      string
			Variables map[string]string
		}{e.Name, e.Variables}
	}
	return out
}

func (m Model) activeEnvIndex() int {
	if m.Store == nil {
		return -1
	}
	return m.Store.LoadEnvironments().ActiveIndex
}

// activeEnvName matches App.tsx's `envs.activeEnv?.name`, shown as a badge
// in the header bar.
func (m Model) activeEnvName() string {
	if m.Store == nil {
		return ""
	}
	store := m.Store.LoadEnvironments()
	if store.ActiveIndex < 0 || store.ActiveIndex >= len(store.Environments) {
		return ""
	}
	return store.Environments[store.ActiveIndex].Name
}

// handleEnvironmentsKey matches useEnvironmentsKeyboard.ts's list-view
// branch (envView === 'list').
func (m Model) handleEnvironmentsKey(key string) (tea.Model, tea.Cmd) {
	if m.Env.View == envViewEdit {
		return m.handleEnvEditNavKey(key)
	}

	envs := m.environments()
	switch key {
	case "j", "down":
		m.EnvCursor = min(m.EnvCursor+1, max(len(envs)-1, 0))
	case "k", "up":
		m.EnvCursor = max(m.EnvCursor-1, 0)
	case "enter":
		if len(envs) > 0 && m.Store != nil {
			m.Store.SetActiveEnvironment(m.EnvCursor)
		}
	case "e":
		if len(envs) > 0 {
			m.Env.View = envViewEdit
			m.Env.VarCursor = 0
		}
	case "n":
		m.Env.AddingEnv = true
		m.Env.NewNameInput.SetValue("")
		m.Env.NewNameInput.Focus()
	case "x":
		if len(envs) > 0 && m.Store != nil {
			m.Store.DeleteEnvironment(m.EnvCursor)
			m.EnvCursor = max(0, m.EnvCursor-1)
		}
	}
	return m, nil
}

func (m Model) handleEnvNewNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.Env.NewNameInput.Value())
		if name != "" && m.Store != nil {
			m.Store.AddEnvironment(name)
			m.EnvCursor = len(m.environments()) - 1
		}
		m.Env.AddingEnv = false
		return m, nil
	case "esc":
		m.Env.AddingEnv = false
		return m, nil
	}
	var cmd tea.Cmd
	m.Env.NewNameInput, cmd = m.Env.NewNameInput.Update(msg)
	return m, cmd
}

// handleEnvEditNavKey matches useEnvironmentsKeyboard.ts's envView ===
// 'edit' branch (row navigation over an environment's variables, not
// currently inserting). Esc-to-go-back-to-the-list isn't in the TS source
// (it has no escape case here at all — envView can only change via Tab
// bouncing to another section and back, which leaves you stuck in 'edit'
// with no visible way out), but CLAUDE.md/HelpPopup.tsx both document "Esc:
// back to list" as the intended shortcut, so it's implemented here rather
// than left dead.
func (m Model) handleEnvEditNavKey(key string) (tea.Model, tea.Cmd) {
	envs := m.environments()
	if m.EnvCursor >= len(envs) {
		m.Env.View = envViewList
		return m, nil
	}
	varCount := len(envs[m.EnvCursor].Variables)
	totalRows := varCount + 1 // + add-new row

	switch key {
	case "esc":
		m.Env.View = envViewList
	case "j", "down":
		m.Env.VarCursor = min(m.Env.VarCursor+1, totalRows-1)
	case "k", "up":
		m.Env.VarCursor = max(m.Env.VarCursor-1, 0)
	case "i":
		keys := sortedVarKeys(envs[m.EnvCursor].Variables)
		if m.Env.VarCursor < len(keys) {
			k := keys[m.Env.VarCursor]
			m.Env.EditingKey = k
			m.Env.IsNewVar = false
			m.Env.KeyInput.SetValue(k)
			m.Env.ValueInput.SetValue(envs[m.EnvCursor].Variables[k])
		} else {
			m.Env.EditingKey = ""
			m.Env.IsNewVar = true
			m.Env.KeyInput.SetValue("")
			m.Env.ValueInput.SetValue("")
		}
		m.Env.VarField = "key"
		m.Env.InsertingVar = true
		m.Env.KeyInput.Focus()
		m.Env.ValueInput.Blur()
	case "x":
		keys := sortedVarKeys(envs[m.EnvCursor].Variables)
		if m.Env.VarCursor < len(keys) && m.Store != nil {
			m.Store.DeleteEnvironmentVariable(m.EnvCursor, keys[m.Env.VarCursor])
			m.Env.VarCursor = max(0, m.Env.VarCursor-1)
		}
	}
	return m, nil
}

func (m Model) handleEnvVarEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.Env.VarField == "key" {
			m.Env.VarField = "value"
			m.Env.KeyInput.Blur()
			m.Env.ValueInput.Focus()
		} else {
			m.Env.VarField = "key"
			m.Env.ValueInput.Blur()
			m.Env.KeyInput.Focus()
		}
		return m, nil
	case "esc":
		m.Env.InsertingVar = false
		return m, nil
	case "enter":
		key := strings.TrimSpace(m.Env.KeyInput.Value())
		if key != "" && m.Store != nil {
			m.Store.SetEnvironmentVariable(m.EnvCursor, key, m.Env.ValueInput.Value())
		}
		m.Env.InsertingVar = false
		return m, nil
	}

	var cmd tea.Cmd
	if m.Env.VarField == "key" {
		m.Env.KeyInput, cmd = m.Env.KeyInput.Update(msg)
	} else {
		m.Env.ValueInput, cmd = m.Env.ValueInput.Update(msg)
	}
	return m, cmd
}

func sortedVarKeys(vars map[string]string) []string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (m Model) renderInfoPopup(height, width int) string {
	var lines []string

	header := boldStyle.Render(m.Spec.Spec.Info.Title) + dimStyle.Render("  v"+m.Spec.Spec.Info.Version+"  OpenAPI "+m.Spec.Spec.OpenAPI)
	if m.CollectionName != "" {
		header = yellowStyle.Bold(true).Render("["+m.CollectionName+"]  ") + header
	}
	lines = append(lines, header)
	if m.Spec.Spec.Info.Description != "" {
		lines = append(lines, dimStyle.Render(truncate(m.Spec.Spec.Info.Description, width-4)))
	}

	lines = append(lines, "")
	lines = append(lines, m.renderServersSection()...)

	if m.Spec.Spec.Components != nil && len(m.Spec.Spec.Components.SecuritySchemes) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.renderAuthSection()...)
	}

	lines = append(lines, "")
	lines = append(lines, m.renderEnvironmentsSection()...)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(activeBorderColor).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderServersSection() []string {
	active := m.InfoSection == infoServers
	title := boldStyle.Render("SERVERS")
	if active {
		title = boldStyle.Foreground(activeBorderColor).Render("SERVERS") + "  " +
			dimStyle.Render("Tab: switch  j/k: move  Enter: select  Esc: close")
	}
	lines := []string{title}
	for i, s := range m.servers() {
		cursor := "  "
		lineStyle := lipgloss.NewStyle()
		if active && i == m.ServerCursor {
			cursor = cyanStyle.Render("> ")
			lineStyle = cyanStyle
		}
		selected := i == m.SelectedServer
		urlStyle := lineStyle
		if selected {
			urlStyle = urlStyle.Bold(true)
		}
		line := cursor + urlStyle.Render(s.URL)
		if s.Description != "" {
			line += dimStyle.Render("  " + s.Description)
		}
		if selected {
			line += lipgloss.NewStyle().Foreground(color2xx).Render("  active")
		}
		lines = append(lines, line)
	}
	return lines
}

func (m Model) renderAuthSection() []string {
	active := m.InfoSection == infoAuth
	title := boldStyle.Render("AUTH")
	if active {
		title = boldStyle.Foreground(activeBorderColor).Render("AUTH") + "  " +
			dimStyle.Render("Tab: switch  j/k: move  Enter: edit  Esc: close")
	}
	lines := []string{title}

	var creds map[string]string
	if m.Store != nil {
		creds = m.Store.LoadAuth().Credentials
	}
	for i, name := range m.authSchemeNames() {
		scheme := m.Spec.Spec.Components.SecuritySchemes[name]
		cursor := "  "
		style := lipgloss.NewStyle()
		if active && i == m.AuthCursor {
			cursor = cyanStyle.Render("> ")
			style = cyanStyle
		}
		label := authSchemeLabel(scheme)

		isEditing := active && m.Auth.Editing && m.Auth.Scheme == name
		var valueText string
		switch {
		case isEditing:
			valueText = m.Auth.Input.View()
		case creds[name] != "":
			display := creds[name]
			if len(display) > 20 {
				display = display[:20] + "…"
			}
			valueText = lipgloss.NewStyle().Foreground(color2xx).Render(display)
		default:
			valueText = dimStyle.Render("not set")
		}
		lines = append(lines, cursor+style.Bold(true).Render(name)+dimStyle.Render("  "+label+"  ")+valueText)
	}
	return lines
}

func authSchemeLabel(scheme openapi.SecurityScheme) string {
	switch scheme.Type {
	case "http":
		label := "http"
		if scheme.Scheme != "" {
			label = scheme.Scheme
			if scheme.BearerFormat != "" {
				label += " (" + scheme.BearerFormat + ")"
			}
		}
		return label
	case "apiKey":
		return "apiKey in " + scheme.In + " as " + scheme.Name
	default:
		return scheme.Type
	}
}

func (m Model) renderEnvironmentsSection() []string {
	active := m.InfoSection == infoEnvironments
	title := boldStyle.Render("ENVIRONMENTS")
	if active {
		hint := "Tab: switch  j/k: move  Enter: activate  e: edit  n: new  x: del"
		if m.Env.View == envViewEdit {
			hint = "j/k: move  i: edit  x: del  Esc: back"
		}
		title = boldStyle.Foreground(activeBorderColor).Render("ENVIRONMENTS") + "  " + dimStyle.Render(hint)
	}
	lines := []string{title}

	if active && m.Env.View == envViewEdit {
		return append(lines, m.renderEnvVarTable()...)
	}
	return append(lines, m.renderEnvList(active)...)
}

func (m Model) renderEnvList(active bool) []string {
	envs := m.environments()
	activeIdx := m.activeEnvIndex()
	var lines []string
	for i, e := range envs {
		cursor := "  "
		style := lipgloss.NewStyle()
		if active && i == m.EnvCursor {
			cursor = cyanStyle.Render("> ")
			style = cyanStyle
		}
		line := cursor + style.Render(e.Name)
		if i == activeIdx {
			line += lipgloss.NewStyle().Foreground(color2xx).Render("  active")
		}
		lines = append(lines, line)
	}

	if active {
		if m.Env.AddingEnv {
			lines = append(lines, cyanStyle.Render("> ")+m.Env.NewNameInput.View())
		} else {
			lines = append(lines, dimStyle.Render("  [ n: new environment ]"))
		}
	}
	if len(envs) == 0 && !m.Env.AddingEnv {
		lines = append(lines, dimStyle.Render("  no environments yet"))
	}
	return lines
}

func (m Model) renderEnvVarTable() []string {
	envs := m.environments()
	if m.EnvCursor >= len(envs) {
		return nil
	}
	env := envs[m.EnvCursor]
	keys := sortedVarKeys(env.Variables)

	lines := []string{
		boldStyle.Render("Variables for ") + cyanStyle.Bold(true).Render(env.Name),
		strings.Repeat(" ", 3) + dimStyle.Bold(true).Render(padRight("NAME", 24)+padRight("VALUE", 30)),
	}

	for i, k := range keys {
		isCur := i == m.Env.VarCursor
		isEditing := m.Env.InsertingVar && !m.Env.IsNewVar && m.Env.EditingKey == k
		cursor := "  "
		nameStyle := lipgloss.NewStyle()
		if isCur {
			cursor = cyanStyle.Render("> ")
			nameStyle = cyanStyle
		}
		name := k
		if isEditing && m.Env.VarField == "key" {
			name = m.Env.KeyInput.View()
		}
		value := env.Variables[k]
		if value == "" {
			value = "-"
		}
		valueStyle := lipgloss.NewStyle().Foreground(color2xx)
		if isEditing && m.Env.VarField == "value" {
			value = m.Env.ValueInput.View()
			valueStyle = lipgloss.NewStyle()
		}
		lines = append(lines, cursor+padRight(nameStyle.Render(name), 24)+padRight(valueStyle.Render(value), 30))
	}

	isCurAdd := m.Env.VarCursor == len(keys)
	isAdding := m.Env.InsertingVar && m.Env.IsNewVar
	cursor := "  "
	if isCurAdd {
		cursor = cyanStyle.Render("> ")
	}
	if isAdding {
		name := m.Env.KeyInput.View()
		if m.Env.VarField != "key" {
			name = cyanStyle.Render(displayOr(m.Env.KeyInput.Value(), "-"))
		}
		value := dimStyle.Render("-")
		if m.Env.VarField == "value" {
			value = m.Env.ValueInput.View()
		}
		lines = append(lines, cursor+padRight(name, 24)+padRight(value, 30))
	} else {
		label := dimStyle.Render("[ + ]")
		if isCurAdd {
			label = cyanStyle.Render("[ i: add variable ]")
		}
		lines = append(lines, cursor+label)
	}

	return lines
}

func displayOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
