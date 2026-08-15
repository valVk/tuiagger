package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// authPanelState backs the info popup's AUTH section — matches
// AuthSection.tsx/useAuthKeyboard.ts. Editing/Scheme/Input back
// editingScheme/editValue: Enter on a selected scheme starts editing its
// credential. A nested widget under infoPopupState, not a Mode-routed
// component itself — see serversPanelState's doc comment.
type authPanelState struct {
	Cursor  int
	Editing bool
	Scheme  string
	Input   textinput.Model
}

func newAuthPanelState() authPanelState {
	return authPanelState{Input: textinput.New()}
}

// UpdateNav handles row navigation and entering edit mode; UpdateEdit
// (below) takes over once Editing is true — mirrors the two-phase split
// every other custom-row editor in this package uses (see paramtable.go).
func (a authPanelState) UpdateNav(key string, names []string, store *storage.Store) authPanelState {
	switch key {
	case "j", "down":
		a.Cursor = min(a.Cursor+1, max(len(names)-1, 0))
	case "k", "up":
		a.Cursor = max(a.Cursor-1, 0)
	case "enter":
		if len(names) == 0 {
			return a
		}
		name := names[a.Cursor]
		value := ""
		if store != nil {
			value = store.LoadAuth().Credentials[name]
		}
		a.Editing = true
		a.Scheme = name
		a.Input.SetValue(value)
		a.Input.Focus()
	}
	return a
}

// UpdateEdit matches useAuthKeyboard.ts's editingScheme branch: Esc/Enter
// both commit and exit — there's no "cancel without saving", the TS
// source treats the two identically.
func (a authPanelState) UpdateEdit(msg tea.KeyMsg, store *storage.Store) (authPanelState, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		if store != nil {
			store.SetCredential(a.Scheme, a.Input.Value())
		}
		a.Editing = false
		return a, nil
	}
	var cmd tea.Cmd
	a.Input, cmd = a.Input.Update(msg)
	return a, cmd
}

// authSchemeNames matches InfoPopup.tsx's
// `Object.entries(spec.components.securitySchemes)`, which iterates in the
// spec's original declaration order, not alphabetically.
func authSchemeNames(spec *openapi.ParsedSpec) []string {
	if spec.Spec.Components == nil {
		return nil
	}
	return spec.Spec.Components.SecuritySchemeOrder
}

func (a authPanelState) View(spec *openapi.ParsedSpec, store *storage.Store, active bool, width int) []string {
	title := boldStyle.Render("AUTH")
	if active {
		title = boldStyle.Foreground(activeBorderColor).Render("AUTH") + "  " +
			dimStyle.Render("Tab: switch  j/k: move  Enter: edit  Esc: close")
	}
	lines := []string{title, dimStyle.Render(strings.Repeat("─", width))}

	var creds map[string]string
	if store != nil {
		creds = store.LoadAuth().Credentials
	}
	for i, name := range authSchemeNames(spec) {
		scheme := spec.Spec.Components.SecuritySchemes[name]
		cursor := "  "
		style := lipgloss.NewStyle()
		if active && i == a.Cursor {
			cursor = cyanStyle.Render("> ")
			style = cyanStyle
		}
		label := authSchemeLabel(scheme)

		isEditing := active && a.Editing && a.Scheme == name
		var valueText string
		switch {
		case isEditing:
			valueText = a.Input.View()
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
