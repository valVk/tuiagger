package tui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/storage"
)

// saveDialogState backs the 's' overlay in manual-request mode, matching
// ManualSaveDialog.tsx: a name field and a tag picker that cycles existing
// tags or drops into free-text entry for a brand-new one.
type saveDialogState struct {
	NameInput textinput.Model
	Tags      []string // "default" always leads, matching TS
	TagIndex  int

	NewTagMode  bool
	NewTagInput textinput.Model

	Focus string // "name" | "tag" | "newTag"
}

func newSaveDialogState(availableTags []string, editing *storage.SavedRequest) saveDialogState {
	tags := availableTags
	if !slices.Contains(tags, "default") {
		tags = append([]string{"default"}, tags...)
	}

	name := textinput.New()
	name.Focus()
	tagIndex := 0
	if editing != nil {
		name.SetValue(editing.Name)
		for i, t := range tags {
			if t == editing.Tag {
				tagIndex = i
				break
			}
		}
	}

	return saveDialogState{
		NameInput:   name,
		Tags:        tags,
		TagIndex:    tagIndex,
		NewTagInput: textinput.New(),
		Focus:       "name",
	}
}

func (s saveDialogState) currentTag() string {
	if s.NewTagMode {
		return s.NewTagInput.Value()
	}
	if s.TagIndex < 0 || s.TagIndex >= len(s.Tags) {
		return ""
	}
	return s.Tags[s.TagIndex]
}

func (m Model) handleSaveDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dlg := &m.Manual.SaveDialog
	key := msg.String()

	if key == "esc" {
		if dlg.NewTagMode {
			dlg.NewTagMode = false
			dlg.NewTagInput.SetValue("")
			dlg.Focus = "tag"
			return m, nil
		}
		m.Manual.ShowSaveDialog = false
		return m, nil
	}

	switch dlg.Focus {
	case "name":
		if key == "enter" || key == "tab" {
			dlg.Focus = "tag"
			dlg.NameInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		dlg.NameInput, cmd = dlg.NameInput.Update(msg)
		return m, cmd

	case "tag":
		switch key {
		case "left":
			if dlg.TagIndex > 0 {
				dlg.TagIndex--
			}
			return m, nil
		case "right":
			if dlg.TagIndex < len(dlg.Tags)-1 {
				dlg.TagIndex++
			} else {
				dlg.NewTagMode = true
				dlg.Focus = "newTag"
				dlg.NewTagInput.Focus()
			}
			return m, nil
		case "tab":
			dlg.Focus = "name"
			dlg.NameInput.Focus()
			return m, nil
		case "enter":
			name := strings.TrimSpace(dlg.NameInput.Value())
			tag := dlg.currentTag()
			if name != "" && tag != "" {
				return m.saveManualRequest(name, tag), nil
			}
			return m, nil
		}
		return m, nil

	case "newTag":
		if key == "left" && dlg.NewTagInput.Value() == "" {
			dlg.NewTagMode = false
			dlg.Focus = "tag"
			return m, nil
		}
		if key == "enter" {
			name := strings.TrimSpace(dlg.NameInput.Value())
			tag := strings.TrimSpace(dlg.NewTagInput.Value())
			if name != "" && tag != "" {
				return m.saveManualRequest(name, tag), nil
			}
			return m, nil
		}
		var cmd tea.Cmd
		dlg.NewTagInput, cmd = dlg.NewTagInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// saveManualRequest persists the in-progress manual draft, matching
// App.tsx's handleManualSaveFromDialog: creates the tag first if it's new,
// then adds or updates the SavedRequest, then returns to browse.
func (m Model) saveManualRequest(name, tag string) Model {
	manual := m.Manual
	if m.Store == nil {
		return m.exitManual()
	}

	if tag != "default" && !slices.Contains(m.Spec.Tags, tag) {
		m.Store.AddCustomTag(storage.CustomTag{Name: tag})
	}

	var queryParams, headers []storage.KeyValuePair
	for _, p := range manual.Params {
		kv := storage.KeyValuePair{ID: p.ID, Key: p.Name, Value: p.Value, Enabled: p.Enabled}
		if p.In == "header" {
			headers = append(headers, kv)
		} else {
			queryParams = append(queryParams, kv)
		}
	}

	method := manual.Method
	if method == "" {
		method = "GET"
	}

	if manual.EditingRequest != nil {
		m.Store.UpdateSavedRequest(manual.EditingRequest.ID, func(r *storage.SavedRequest) {
			r.Method = method
			r.Path = manual.Path
			r.QueryParams = queryParams
			r.Headers = headers
			r.Body = manual.Body
			r.BodyType = "json"
			r.Name = name
			r.Tag = tag
		})
	} else {
		m.Store.AddSavedRequest(storage.SavedRequest{
			ManualRequestState: storage.ManualRequestState{
				Method: method, Path: manual.Path, QueryParams: queryParams, Headers: headers,
				Body: manual.Body, BodyType: "json",
			},
			Name: name,
			Tag:  tag,
		})
	}

	return m.exitManual().refreshSavedRequests()
}

// renderSaveDialogOverlay matches ManualSaveDialog.tsx: a centered
// double-bordered box with name field, tag picker, and hint line.
func (m Model) renderSaveDialogOverlay(height, width int) string {
	dlg := m.Manual.SaveDialog

	nameLabel := "Name  "
	nameStyle := dimStyle
	if dlg.Focus == "name" {
		nameStyle = cyanStyle
	}
	nameLine := nameStyle.Render(nameLabel) + dlg.NameInput.View()

	tagLabel := "Tag   "
	tagStyle := dimStyle
	if dlg.Focus == "tag" || dlg.Focus == "newTag" {
		tagStyle = cyanStyle
	}
	var tagValue string
	if dlg.NewTagMode {
		tagValue = dlg.NewTagInput.View()
	} else {
		current := dlg.currentTag()
		if current == "" {
			current = "(no tags)"
		}
		valStyle := lipgloss.NewStyle()
		if dlg.Focus == "tag" {
			valStyle = cyanStyle
		}
		tagValue = dimStyle.Render("< ") + valStyle.Render(current) + dimStyle.Render(" > new")
	}
	tagLine := tagStyle.Render(tagLabel) + tagValue

	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(activeBorderColor).Width(56).Align(lipgloss.Center).Render("SAVE REQUEST"),
		"",
		nameLine,
		"",
		tagLine,
		"",
		dimStyle.Render("Tab: switch field  ←/→: cycle tags  Enter: save  Esc: cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(activeBorderColor).
		Padding(1, 2).
		Width(60).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderRenameTagOverlay is a small adaptation from TS, which renders the
// rename input inline in RightPanel rather than as its own overlay — see
// HANDOFF.md for why this rewrite uses a centered box instead.
func (m Model) renderRenameTagOverlay(height, width int) string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(activeBorderColor).Render("RENAME TAG"),
		"",
		dimStyle.Render("Old: ")+m.RenameTag.TagName,
		cyanStyle.Render("New: ")+m.RenameTag.Input.View(),
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
