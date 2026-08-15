package tui

import (
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

// infoPopupState composes the info popup's three independent sections —
// SERVERS, AUTH, ENVIRONMENTS — each its own nested component (own state,
// own Update, own View: serversPanelState/authPanelState/
// environmentsPanelState in infopopup_servers.go/infopopup_auth.go/
// infopopup_env.go). This parent owns only which section is active and
// dispatches to whichever one — replaces what was one 656-line file
// tangling all three concerns behind flat Model fields.
type infoPopupState struct {
	Section      infoSection
	Servers      serversPanelState
	Auth         authPanelState
	Environments environmentsPanelState
}

func newInfoPopupState(selectedServer int) infoPopupState {
	return infoPopupState{
		Section:      infoServers,
		Servers:      serversPanelState{Cursor: selectedServer},
		Auth:         newAuthPanelState(),
		Environments: newEnvironmentsPanelState(),
	}
}

// enterInfo opens the info popup, matching App.tsx's 'i' handler.
func (m Model) enterInfo() Model {
	m.ShowInfo = true
	m.Info = newInfoPopupState(m.SelectedServer)
	return m
}

// handleInfoKey is InfoPopup's own Update — the actual Mode-routed
// component entry point; Servers/Auth/Environments underneath are nested
// widgets with their own richer per-section signatures (see
// serversPanelState's doc comment for why).
func (m Model) handleInfoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any in-progress text edit gets first refusal at every keystroke —
	// matches useAuthKeyboard.ts/useEnvironmentsKeyboard.ts, whose own
	// `if (editingX) { ...; return; }` guards run before InfoPopup.tsx's
	// Tab/Esc section-switching ever sees the keystroke, so a value like
	// "internal" doesn't close the popup on its 'i', switch sections on its
	// (nonexistent) Tab, etc.
	if m.Info.Auth.Editing {
		var cmd tea.Cmd
		m.Info.Auth, cmd = m.Info.Auth.UpdateEdit(msg, m.Store)
		return m, cmd
	}
	if m.Info.Environments.InsertingVar {
		var cmd tea.Cmd
		m.Info.Environments, cmd = m.Info.Environments.UpdateVarEditKey(msg, m.Store)
		return m, cmd
	}
	if m.Info.Environments.AddingEnv {
		var cmd tea.Cmd
		m.Info.Environments, cmd = m.Info.Environments.UpdateNewNameKey(msg, m.Store)
		return m, cmd
	}

	key := msg.String()
	inEnvEdit := m.Info.Section == infoEnvironments && m.Info.Environments.SubView == envViewEdit
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
		m.Info.Section = m.nextInfoSection()
		return m, nil
	}

	switch m.Info.Section {
	case infoServers:
		var selected int
		var closePopup bool
		m.Info.Servers, selected, closePopup = m.Info.Servers.Update(key, m.servers())
		if selected >= 0 {
			m.SelectedServer = selected
		}
		if closePopup {
			m.ShowInfo = false
		}
	case infoAuth:
		m.Info.Auth = m.Info.Auth.UpdateNav(key, authSchemeNames(m.Spec), m.Store)
	case infoEnvironments:
		m.Info.Environments = m.Info.Environments.UpdateListKey(key, m.Store)
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
		if s == m.Info.Section {
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

func (m Model) renderInfoPopup(height, width int) string {
	var lines []string

	header := boldStyle.Render(m.Spec.Spec.Info.Title) + dimStyle.Render("  v"+m.Spec.Spec.Info.Version+"  OpenAPI "+m.Spec.Spec.OpenAPI)
	if m.CollectionName != "" {
		header = yellowStyle.Bold(true).Render("["+m.CollectionName+"]  ") + header
	}
	lines = append(lines, header)
	if m.Spec.Spec.Info.Description != "" {
		// InfoPopup.tsx's <Text wrap="truncate"> only truncates each line
		// that's too WIDE for the terminal — it doesn't collapse the
		// description to one line. A multi-paragraph OpenAPI description
		// (common: literal '\n's between paragraphs/link lists) renders as
		// that many rows in TS, so split on '\n' first, truncate each row
		// individually, rather than truncating the whole string to one row.
		for l := range strings.SplitSeq(m.Spec.Spec.Info.Description, "\n") {
			lines = append(lines, dimStyle.Render(truncate(l, width-4)))
		}
	}

	inner := max(width-4, 1)

	lines = append(lines, "")
	lines = append(lines, m.Info.Servers.View(m.servers(), m.SelectedServer, m.Info.Section == infoServers, inner)...)

	if m.Spec.Spec.Components != nil && len(m.Spec.Spec.Components.SecuritySchemes) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.Info.Auth.View(m.Spec, m.Store, m.Info.Section == infoAuth, inner)...)
	}

	lines = append(lines, "")
	lines = append(lines, m.Info.Environments.View(m.Store, m.Info.Section == infoEnvironments, inner)...)

	return lipgloss.NewStyle().
		Width(width-2).
		Height(height).
		Padding(0, 1).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(activeBorderColor).
		Render(strings.Join(lines, "\n"))
}
