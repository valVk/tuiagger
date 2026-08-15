package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// renameTagState backs 'R' on a custom tag row — a single text field,
// matching App.tsx's RenameTagState (rendered as its own mode rather than
// inline in the list row, per HANDOFF.md's noted adaptation).
type renameTagState struct {
	TagName string
	Input   textinput.Model
}

func (m Model) enterRenameTag(tagName string) Model {
	input := textinput.New()
	input.SetValue(tagName)
	input.Focus()
	m.RenameTag = renameTagState{TagName: tagName, Input: input}
	m.Mode = ModeRenameTag
	return m
}

// renameTagResult signals what Update decided the parent should do —
// mirrors saveDialogState's convention (see savedialog.go), since
// committing needs Model-owned (Store, refreshSavedRequests) side
// effects this component can't perform itself.
type renameTagResult int

const (
	renameTagNone renameTagResult = iota
	renameTagCancelled
	renameTagCommitted
)

func (s renameTagState) Update(msg tea.KeyMsg) (renameTagState, renameTagResult, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return s, renameTagCancelled, nil
	case "enter":
		return s, renameTagCommitted, nil
	}
	var cmd tea.Cmd
	s.Input, cmd = s.Input.Update(msg)
	return s, renameTagNone, cmd
}

func (m Model) handleRenameTagKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var result renameTagResult
	var cmd tea.Cmd
	m.RenameTag, result, cmd = m.RenameTag.Update(msg)
	switch result {
	case renameTagCancelled:
		m.Mode = ModeBrowse
	case renameTagCommitted:
		newName := strings.TrimSpace(m.RenameTag.Input.Value())
		if newName != "" && newName != m.RenameTag.TagName && m.Store != nil {
			m.Store.RenameCustomTag(m.RenameTag.TagName, newName)
		}
		m.Mode = ModeBrowse
		return m.refreshSavedRequests(), nil
	}
	return m, cmd
}

// View is a small adaptation from TS, which renders the rename input
// inline in RightPanel rather than as its own overlay — see HANDOFF.md for
// why this rewrite uses a centered box instead.
func (s renameTagState) View(height, width int) string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(activeBorderColor).Render("RENAME TAG"),
		"",
		dimStyle.Render("Old: ")+s.TagName,
		cyanStyle.Render("New: ")+s.Input.View(),
		"",
		dimStyle.Render("Enter: save  Esc: cancel"),
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(activeBorderColor).
		Padding(1, 2).
		Width(50).
		Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
