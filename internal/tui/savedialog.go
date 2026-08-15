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
	name.Placeholder = "request name"
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

	newTag := textinput.New()
	newTag.Placeholder = "new tag name"

	return saveDialogState{
		NameInput:   name,
		Tags:        tags,
		TagIndex:    tagIndex,
		NewTagInput: newTag,
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

// saveDialogResult signals what Update decided the parent should do —
// close on cancel, or persist+close on a successful save. Mirrors
// headerTableState's "report the change, let the parent apply it"
// convention (see headertable.go), since saving needs Manual-level
// (Model-owned: Store, exitManual) side effects this component can't
// perform itself.
type saveDialogResult int

const (
	saveDialogNone saveDialogResult = iota
	saveDialogCancelled
	saveDialogSaved
)

func (s saveDialogState) Update(msg tea.KeyMsg) (saveDialogState, saveDialogResult, tea.Cmd) {
	key := msg.String()

	if key == "esc" {
		if s.NewTagMode {
			s.NewTagMode = false
			s.NewTagInput.SetValue("")
			s.Focus = "tag"
			return s, saveDialogNone, nil
		}
		return s, saveDialogCancelled, nil
	}

	switch s.Focus {
	case "name":
		if key == "enter" || key == "tab" {
			s.Focus = "tag"
			s.NameInput.Blur()
			return s, saveDialogNone, nil
		}
		var cmd tea.Cmd
		s.NameInput, cmd = s.NameInput.Update(msg)
		return s, saveDialogNone, cmd

	case "tag":
		switch key {
		case "left":
			if s.TagIndex > 0 {
				s.TagIndex--
			}
			return s, saveDialogNone, nil
		case "right":
			if s.TagIndex < len(s.Tags)-1 {
				s.TagIndex++
			} else {
				s.NewTagMode = true
				s.Focus = "newTag"
				s.NewTagInput.Focus()
			}
			return s, saveDialogNone, nil
		case "tab":
			s.Focus = "name"
			s.NameInput.Focus()
			return s, saveDialogNone, nil
		case "enter":
			if strings.TrimSpace(s.NameInput.Value()) != "" && s.currentTag() != "" {
				return s, saveDialogSaved, nil
			}
			return s, saveDialogNone, nil
		}
		return s, saveDialogNone, nil

	case "newTag":
		if key == "left" && s.NewTagInput.Value() == "" {
			s.NewTagMode = false
			s.Focus = "tag"
			return s, saveDialogNone, nil
		}
		if key == "enter" {
			if strings.TrimSpace(s.NameInput.Value()) != "" && strings.TrimSpace(s.NewTagInput.Value()) != "" {
				return s, saveDialogSaved, nil
			}
			return s, saveDialogNone, nil
		}
		var cmd tea.Cmd
		s.NewTagInput, cmd = s.NewTagInput.Update(msg)
		return s, saveDialogNone, cmd
	}

	return s, saveDialogNone, nil
}

func (m Model) handleSaveDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var result saveDialogResult
	var cmd tea.Cmd
	m.Manual.SaveDialog, result, cmd = m.Manual.SaveDialog.Update(msg)
	switch result {
	case saveDialogCancelled:
		m.Manual.ShowSaveDialog = false
	case saveDialogSaved:
		name := strings.TrimSpace(m.Manual.SaveDialog.NameInput.Value())
		tag := m.Manual.SaveDialog.currentTag()
		return m.saveManualRequest(name, tag), nil
	}
	return m, cmd
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

// View matches ManualSaveDialog.tsx: a centered double-bordered box with
// name field, tag picker, and hint line.
func (s saveDialogState) View(height, width int) string {
	nameLabel := "Name  "
	nameStyle := dimStyle
	if s.Focus == "name" {
		nameStyle = cyanStyle
	}
	nameLine := nameStyle.Render(nameLabel) + s.NameInput.View()

	tagLabel := "Tag   "
	tagStyle := dimStyle
	if s.Focus == "tag" || s.Focus == "newTag" {
		tagStyle = cyanStyle
	}
	var tagValue string
	if s.NewTagMode {
		tagValue = s.NewTagInput.View()
	} else {
		current := s.currentTag()
		if current == "" {
			current = "(no tags)"
		}
		valStyle := lipgloss.NewStyle()
		if s.Focus == "tag" {
			valStyle = cyanStyle
		}
		tagValue = dimStyle.Render("← ") + valStyle.Render(current) + dimStyle.Render(" → new")
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
