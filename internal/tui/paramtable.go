package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/valVK/tuiagger/internal/storage"
)

// enterParamRowEdit populates the shared name/value widgets (see
// headertable.go — ParamField/NameInput/ValueInput are reused between the
// HEADERS table and PARAMETERS' custom/add-new row editing) for a custom
// or add-new PARAMETERS row edit. Caller sets its own
// ParamEditing/ParamAddNew flags, matching SpecParamRow.tsx/
// CustomParamRow.tsx/AddNewParamRow.tsx's edit-mode entry.
func (h headerTableState) enterParamRowEdit(name, value string) headerTableState {
	h.ParamField = "name"
	h.NameInput.SetValue(name)
	h.NameInput.Focus()
	h.ValueInput.SetValue(value)
	h.ValueInput.Blur()
	return h
}

// handleParamRowEditKey matches useParamNavigation.ts's insertMode branch
// for custom/add-new PARAMETERS rows: Tab toggles which field is focused,
// Enter commits (adding a new CustomParameter for the add-new row, or
// updating the existing one), Esc cancels without saving. Shared by
// try-it-out's handleCustomParamEditKey (non-spec-param branch) and the
// manual builder's handleManualParamEditKey — try-it-out's spec-param
// editing (enum cycling, disabled-row guard) has no manual-builder
// equivalent and stays in tryit.go.
//
// Returns the updated headerTableState, the updated custom-param slice
// (nil if unchanged — Esc/Tab/a keystroke that doesn't commit, or a
// rejected empty-name add-new), whether editing is now done (Esc, or a
// successful Enter commit — caller should clear its own ParamEditing
// flag and, for an add-new commit, move its cursor onto the new row),
// and any textinput cmd.
func (h headerTableState) handleParamRowEditKey(msg tea.KeyMsg, isAddNew bool, idx int, custom []storage.CustomParameter, newParamIn string) (headerTableState, []storage.CustomParameter, bool, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return h, nil, true, nil
	case "tab":
		if h.ParamField == "name" {
			h.ParamField = "value"
			h.NameInput.Blur()
			h.ValueInput.Focus()
		} else {
			h.ParamField = "name"
			h.ValueInput.Blur()
			h.NameInput.Focus()
		}
		return h, nil, false, nil
	case "enter":
		if isAddNew {
			name := strings.TrimSpace(h.NameInput.Value())
			if name == "" {
				return h, nil, false, nil
			}
			in := newParamIn
			if in == "" {
				in = "query"
			}
			updated := append(custom, storage.CustomParameter{
				ID: uuid.NewString(), Name: name, Value: h.ValueInput.Value(),
				In: in, Enabled: true,
			})
			return h, updated, true, nil
		}
		custom[idx].Name = h.NameInput.Value()
		custom[idx].Value = h.ValueInput.Value()
		return h, custom, true, nil
	}

	var cmd tea.Cmd
	if h.ParamField == "name" {
		h.NameInput, cmd = h.NameInput.Update(msg)
	} else {
		h.ValueInput, cmd = h.ValueInput.Update(msg)
	}
	return h, nil, false, cmd
}
