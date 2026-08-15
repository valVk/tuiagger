package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/storage"
)

// handleManualKey follows useManualPanelKeyboard.ts's actual dispatch order
// exactly: editingPath, then editingBody, then header/params insert modes,
// then headersFocused (swallows everything, HeadersSection's own useInput
// handles it), then bodyTabFocused, then the top-level m/p/e/s/d/Esc
// bindings, then — falling through all of the above — PARAMETERS is the
// default focus.
func (m Model) handleManualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.Manual.ShowSaveDialog {
		return m.handleSaveDialogKey(msg)
	}

	key := msg.String()
	headerParams, params := splitCustomParams(m.Manual.Params)

	if m.Manual.EditingPath {
		return m.handleManualPathKey(msg)
	}
	if m.Manual.EditingBody {
		return m.handleManualBodyKey(msg)
	}
	if m.Manual.HeaderTable.Editing {
		var merged []storage.CustomParameter
		var cmd tea.Cmd
		m.Manual.HeaderTable, merged, cmd = m.Manual.HeaderTable.handleEditKey(msg, headerParams, params)
		if merged != nil {
			m.Manual.Params = merged
		}
		return m, cmd
	}
	if m.Manual.HeaderTable.Focused {
		var merged []storage.CustomParameter
		var cmd tea.Cmd
		m.Manual.HeaderTable, merged, cmd = m.Manual.HeaderTable.handleFocusedKey(msg, headerParams, params)
		if merged != nil {
			m.Manual.Params = merged
		}
		return m, cmd
	}
	if m.Manual.BodyFocused {
		switch key {
		case "i":
			m.Manual.EditingBody = true
			m.Manual.BodyInput = setBodyValue(m.Manual.BodyInput, m.Manual.Body)
			m.Manual.BodyInput.Focus()
		case "j", "down":
			// Mirrors try-it-out's handleBodyFocusedKey: scrolls past the
			// BODY box (e.g. to reach a response below it, or see the rest
			// of a body taller than one screen). Clamped immediately (see
			// clampRightScroll's doc comment) so scrolling past the bottom
			// doesn't need an equal number of 'k' presses to start moving
			// back up.
			m.RightScroll++
			m.RightScroll = m.clampRightScroll()
		case "k", "up":
			// Mirrors try-it-out: scroll back up first if the user
			// scrolled past the box, only unfocusing back to PARAMETERS
			// once back at the position BODY was originally focused from
			// — see BodyScrollFloor's doc comment.
			if m.RightScroll > m.Manual.BodyScrollFloor {
				m.RightScroll--
			} else {
				m.Manual.BodyFocused = false
			}
		case "esc":
			m.Manual.BodyFocused = false
		case "c":
			// Cycles manualContentTypes — a no-op check isn't needed here:
			// unlike try-it-out's spec-derived list, this fixed 3-entry
			// list always has more than one option.
			m.Manual.ContentTypeTab = (m.Manual.ContentTypeTab + 1) % len(manualContentTypes)
		}
		return m, nil
	}
	if m.Manual.ParamEditing {
		return m.handleManualParamEditKey(msg, headerParams, params)
	}

	// Top-level actions work regardless of focus, matching
	// useManualPanelKeyboard.ts's 'm'/'p' bindings and useAppKeyboard.ts's
	// 'e'/'s'/'d'/Esc handlers for manual mode.
	switch key {
	case "esc":
		return m.exitManual(), nil
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

	// Default focus: PARAMETERS. 'k' at row 0 moves up into HEADERS; 'j'
	// past the last row moves down into BODY (write methods only) — same
	// boundary-crossing model as try-it-out's handleTryItKey.
	rows := len(params) + 1
	switch key {
	case "j", "down":
		if m.Manual.ParamCursor < rows-1 {
			m.Manual.ParamCursor++
		} else if isWriteMethod(m.Manual.Method) {
			m.Manual.BodyFocused = true
			m.Manual.BodyScrollFloor = m.RightScroll
		}
		return m, nil
	case "k", "up":
		if m.Manual.ParamCursor > 0 {
			m.Manual.ParamCursor--
		} else {
			m.Manual.HeaderTable.Focused = true
		}
		return m, nil
	case "i":
		return m.enterManualParamEdit(params), nil
	case "x":
		if m.Manual.ParamCursor < len(params) {
			idx := m.Manual.ParamCursor
			params = append(params[:idx:idx], params[idx+1:]...)
			m.Manual.Params = mergeCustomParams(headerParams, params)
			if m.Manual.ParamCursor > 0 && m.Manual.ParamCursor >= len(params)+1 {
				m.Manual.ParamCursor--
			}
		}
		return m, nil
	case "c":
		if m.Manual.ParamCursor < len(params) {
			params[m.Manual.ParamCursor].In = cycleQueryPath(params[m.Manual.ParamCursor].In)
			m.Manual.Params = mergeCustomParams(headerParams, params)
		} else if m.Manual.ParamCursor == len(params) {
			m.Manual.NewParamIn = cycleQueryPath(m.Manual.NewParamIn)
		}
		return m, nil
	}
	return m, nil
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

// handleManualBodyKey matches tryit_keys.go's handleBodyEditKey: only Esc
// ends editing (Enter inserts a newline, per bubbles/textarea's default
// keymap — see that doc comment for why that's the right binding for this
// widget rather than a literal port of the TS "Enter: done" text).
func (m Model) handleManualBodyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.Manual.Body = m.Manual.BodyInput.Value()
		m.Manual.EditingBody = false
		return m, nil
	}
	var cmd tea.Cmd
	m.Manual.BodyInput, cmd = m.Manual.BodyInput.Update(msg)
	m.Manual.Body = m.Manual.BodyInput.Value()
	return m, cmd
}

func (m Model) enterManualParamEdit(params []storage.CustomParameter) Model {
	if m.Manual.ParamCursor < len(params) {
		p := params[m.Manual.ParamCursor]
		m.Manual.ParamAddNew = false
		m.Manual.HeaderTable = m.Manual.HeaderTable.enterParamRowEdit(p.Name, p.Value)
	} else {
		m.Manual.ParamAddNew = true
		m.Manual.HeaderTable = m.Manual.HeaderTable.enterParamRowEdit("", "")
		m.Manual.NewParamIn = "query"
	}
	m.Manual.ParamEditing = true
	return m
}

// handleManualParamEditKey routes through the shared
// headerTableState.handleParamRowEditKey — see paramtable.go.
func (m Model) handleManualParamEditKey(msg tea.KeyMsg, headerParams, params []storage.CustomParameter) (tea.Model, tea.Cmd) {
	var updated []storage.CustomParameter
	var done bool
	var cmd tea.Cmd
	m.Manual.HeaderTable, updated, done, cmd = m.Manual.HeaderTable.handleParamRowEditKey(msg, m.Manual.ParamAddNew, m.Manual.ParamCursor, params, m.Manual.NewParamIn)
	if updated != nil {
		m.Manual.Params = mergeCustomParams(headerParams, updated)
		if m.Manual.ParamAddNew {
			m.Manual.ParamCursor = len(updated) - 1
		}
	}
	if done {
		m.Manual.ParamEditing = false
	}
	return m, cmd
}
