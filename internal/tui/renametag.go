package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
