package tui

import (
	"sort"
	"strings"

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

// enterInfo opens the info popup, matching App.tsx's 'i' handler. Servers
// section is fully functional (Enter selects + closes, matching
// useServersKeyboard.ts exactly); Auth and Environments are read-only
// summaries backed by the real persisted stores — editing credentials/envs
// is Phase 5 scope (AuthSection.tsx/EnvironmentsSection.tsx's insert-mode
// text editing isn't ported yet), flagged explicitly in the UI rather than
// silently omitted.
func (m Model) enterInfo() Model {
	m.ShowInfo = true
	m.InfoSection = infoServers
	m.ServerCursor = m.SelectedServer
	m.AuthCursor = 0
	m.EnvCursor = 0
	return m
}

func (m Model) handleInfoKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "i":
		m.ShowInfo = false
		return m, nil
	case "tab":
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
	}
	return m, nil
}

func (m Model) handleEnvironmentsKey(key string) (tea.Model, tea.Cmd) {
	count := 0
	if m.Store != nil {
		count = len(m.Store.LoadEnvironments().Environments)
	}
	switch key {
	case "j", "down":
		m.EnvCursor = min(m.EnvCursor+1, max(count-1, 0))
	case "k", "up":
		m.EnvCursor = max(m.EnvCursor-1, 0)
	}
	return m, nil
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
			dimStyle.Render("Tab: switch  j/k: move  Esc: close  (editing not yet available)")
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
		valueText := dimStyle.Render("not set")
		if v := creds[name]; v != "" {
			display := v
			if len(display) > 20 {
				display = display[:20] + "…"
			}
			valueText = lipgloss.NewStyle().Foreground(color2xx).Render(display)
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
		title = boldStyle.Foreground(activeBorderColor).Render("ENVIRONMENTS") + "  " +
			dimStyle.Render("Tab: switch  j/k: move  Esc: close  (editing not yet available)")
	}
	lines := []string{title}

	var envs []struct{ Name string }
	activeIdx := -1
	if m.Store != nil {
		store := m.Store.LoadEnvironments()
		activeIdx = store.ActiveIndex
		for _, e := range store.Environments {
			envs = append(envs, struct{ Name string }{e.Name})
		}
	}
	if len(envs) == 0 {
		lines = append(lines, dimStyle.Render("  no environments yet"))
		return lines
	}
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
	return lines
}
