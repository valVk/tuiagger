package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
)

// serversPanelState backs the info popup's SERVERS section — matches
// useServersKeyboard.ts. A nested widget under infoPopupState (see
// infopopup.go), not a Mode-routed component itself — same convention as
// headerTableState (headertable.go): richer, section-specific signatures
// rather than a bare tea.Msg entry point.
type serversPanelState struct {
	Cursor int
}

// Update moves the cursor, or on Enter reports which server index the
// caller should make the active one (selected >= 0) and that the popup
// should close. SelectedServer is root-owned (also read by the header bar
// and every execute path), so this panel can't just set it itself — same
// "report the change, let the parent apply it" shape as
// headerTableState.handleFocusedKey reporting a merged CustomParams slice.
func (s serversPanelState) Update(key string, servers []openapi.Server) (next serversPanelState, selected int, closePopup bool) {
	next = s
	selected = -1
	switch key {
	case "j", "down":
		next.Cursor = min(next.Cursor+1, len(servers)-1)
	case "k", "up":
		next.Cursor = max(next.Cursor-1, 0)
	case "enter":
		selected = next.Cursor
		closePopup = true
	}
	return next, selected, closePopup
}

func (s serversPanelState) View(servers []openapi.Server, selectedServer int, active bool, width int) []string {
	title := boldStyle.Render("SERVERS")
	if active {
		title = boldStyle.Foreground(activeBorderColor).Render("SERVERS") + "  " +
			dimStyle.Render("Tab: switch  j/k: move  Enter: select  Esc: close")
	}
	lines := []string{title, dimStyle.Render(strings.Repeat("─", width))}
	for i, srv := range servers {
		cursor := "  "
		lineStyle := lipgloss.NewStyle()
		if active && i == s.Cursor {
			cursor = cyanStyle.Render("> ")
			lineStyle = cyanStyle
		}
		selected := i == selectedServer
		urlStyle := lineStyle
		if selected {
			urlStyle = urlStyle.Bold(true)
		}
		line := cursor + urlStyle.Render(srv.URL)
		if srv.Description != "" {
			line += dimStyle.Render("  " + srv.Description)
		}
		if selected {
			line += lipgloss.NewStyle().Foreground(color2xx).Render("  active")
		}
		lines = append(lines, line)
	}
	return lines
}
