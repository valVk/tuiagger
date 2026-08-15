package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/storage"
)

type envView int

const (
	envViewList envView = iota
	envViewEdit
)

// environmentsPanelState backs the info popup's ENVIRONMENTS section —
// matches useEnvironmentsKeyboard.ts's whole state bundle: the list/edit
// sub-view, the variable table cursor and its own insert mode, and the
// "add a new environment" text entry. A nested widget under
// infoPopupState, not a Mode-routed component itself — see
// serversPanelState's doc comment.
//
// SubView (not "View") to avoid colliding with this type's own View
// render method.
type environmentsPanelState struct {
	Cursor  int // which environment row is selected in the list sub-view
	SubView envView

	VarCursor    int
	InsertingVar bool
	VarField     string // "key" | "value"
	EditingKey   string // which existing var is being edited
	IsNewVar     bool   // editing the add-new-variable row rather than an existing one
	KeyInput     textinput.Model
	ValueInput   textinput.Model

	AddingEnv    bool
	NewNameInput textinput.Model
}

func newEnvironmentsPanelState() environmentsPanelState {
	return environmentsPanelState{
		KeyInput:     textinput.New(),
		ValueInput:   textinput.New(),
		NewNameInput: textinput.New(),
	}
}

type envEntry struct {
	Name      string
	Variables map[string]string
}

func loadEnvEntries(store *storage.Store) []envEntry {
	if store == nil {
		return nil
	}
	s := store.LoadEnvironments()
	out := make([]envEntry, len(s.Environments))
	for i, e := range s.Environments {
		out[i] = envEntry{e.Name, e.Variables}
	}
	return out
}

func activeEnvIndex(store *storage.Store) int {
	if store == nil {
		return -1
	}
	return store.LoadEnvironments().ActiveIndex
}

// UpdateListKey matches useEnvironmentsKeyboard.ts's list-view branch
// (envView === 'list').
func (e environmentsPanelState) UpdateListKey(key string, store *storage.Store) environmentsPanelState {
	if e.SubView == envViewEdit {
		return e.updateEditNavKey(key, store)
	}

	envs := loadEnvEntries(store)
	switch key {
	case "j", "down":
		e.Cursor = min(e.Cursor+1, max(len(envs)-1, 0))
	case "k", "up":
		e.Cursor = max(e.Cursor-1, 0)
	case "enter":
		if len(envs) > 0 && store != nil {
			store.SetActiveEnvironment(e.Cursor)
		}
	case "e":
		if len(envs) > 0 {
			e.SubView = envViewEdit
			e.VarCursor = 0
		}
	case "n":
		e.AddingEnv = true
		e.NewNameInput.SetValue("")
		e.NewNameInput.Focus()
	case "x":
		if len(envs) > 0 && store != nil {
			store.DeleteEnvironment(e.Cursor)
			e.Cursor = max(0, e.Cursor-1)
		}
	}
	return e
}

func (e environmentsPanelState) UpdateNewNameKey(msg tea.KeyMsg, store *storage.Store) (environmentsPanelState, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(e.NewNameInput.Value())
		if name != "" && store != nil {
			store.AddEnvironment(name)
			e.Cursor = len(loadEnvEntries(store)) - 1
		}
		e.AddingEnv = false
		return e, nil
	case "esc":
		e.AddingEnv = false
		return e, nil
	}
	var cmd tea.Cmd
	e.NewNameInput, cmd = e.NewNameInput.Update(msg)
	return e, cmd
}

// updateEditNavKey matches useEnvironmentsKeyboard.ts's envView === 'edit'
// branch (row navigation over an environment's variables, not currently
// inserting). Esc-to-go-back-to-the-list isn't in the TS source (it has no
// escape case here at all — envView can only change via Tab bouncing to
// another section and back, which leaves you stuck in 'edit' with no
// visible way out), but CLAUDE.md/HelpPopup.tsx both document "Esc: back
// to list" as the intended shortcut, so it's implemented here rather than
// left dead.
func (e environmentsPanelState) updateEditNavKey(key string, store *storage.Store) environmentsPanelState {
	envs := loadEnvEntries(store)
	if e.Cursor >= len(envs) {
		e.SubView = envViewList
		return e
	}
	varCount := len(envs[e.Cursor].Variables)
	totalRows := varCount + 1 // + add-new row

	switch key {
	case "esc":
		e.SubView = envViewList
	case "j", "down":
		e.VarCursor = min(e.VarCursor+1, totalRows-1)
	case "k", "up":
		e.VarCursor = max(e.VarCursor-1, 0)
	case "i":
		keys := sortedVarKeys(envs[e.Cursor].Variables)
		if e.VarCursor < len(keys) {
			k := keys[e.VarCursor]
			e.EditingKey = k
			e.IsNewVar = false
			e.KeyInput.SetValue(k)
			e.ValueInput.SetValue(envs[e.Cursor].Variables[k])
		} else {
			e.EditingKey = ""
			e.IsNewVar = true
			e.KeyInput.SetValue("")
			e.ValueInput.SetValue("")
		}
		e.VarField = "key"
		e.InsertingVar = true
		e.KeyInput.Focus()
		e.ValueInput.Blur()
	case "x":
		keys := sortedVarKeys(envs[e.Cursor].Variables)
		if e.VarCursor < len(keys) && store != nil {
			store.DeleteEnvironmentVariable(e.Cursor, keys[e.VarCursor])
			e.VarCursor = max(0, e.VarCursor-1)
		}
	}
	return e
}

func (e environmentsPanelState) UpdateVarEditKey(msg tea.KeyMsg, store *storage.Store) (environmentsPanelState, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if e.VarField == "key" {
			e.VarField = "value"
			e.KeyInput.Blur()
			e.ValueInput.Focus()
		} else {
			e.VarField = "key"
			e.ValueInput.Blur()
			e.KeyInput.Focus()
		}
		return e, nil
	case "esc":
		e.InsertingVar = false
		return e, nil
	case "enter":
		key := strings.TrimSpace(e.KeyInput.Value())
		if key != "" && store != nil {
			store.SetEnvironmentVariable(e.Cursor, key, e.ValueInput.Value())
		}
		e.InsertingVar = false
		return e, nil
	}

	var cmd tea.Cmd
	if e.VarField == "key" {
		e.KeyInput, cmd = e.KeyInput.Update(msg)
	} else {
		e.ValueInput, cmd = e.ValueInput.Update(msg)
	}
	return e, cmd
}

func sortedVarKeys(vars map[string]string) []string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (e environmentsPanelState) View(store *storage.Store, active bool, width int) []string {
	title := boldStyle.Render("ENVIRONMENTS")
	if active {
		hint := "Tab: switch  j/k: move  Enter: activate  e: edit  n: new  x: del"
		if e.SubView == envViewEdit {
			hint = "j/k: move  i: edit  x: del  Esc: back"
		}
		title = boldStyle.Foreground(activeBorderColor).Render("ENVIRONMENTS") + "  " + dimStyle.Render(hint)
	}
	lines := []string{title, dimStyle.Render(strings.Repeat("─", width))}

	if active && e.SubView == envViewEdit {
		return append(lines, e.viewVarTable(store)...)
	}
	return append(lines, e.viewList(store, active)...)
}

func (e environmentsPanelState) viewList(store *storage.Store, active bool) []string {
	envs := loadEnvEntries(store)
	activeIdx := activeEnvIndex(store)
	var lines []string
	for i, entry := range envs {
		cursor := "  "
		style := lipgloss.NewStyle()
		if active && i == e.Cursor {
			cursor = cyanStyle.Render("> ")
			style = cyanStyle
		}
		line := cursor + style.Render(entry.Name)
		if i == activeIdx {
			line += lipgloss.NewStyle().Foreground(color2xx).Render("  active")
		}
		lines = append(lines, line)
	}

	if active {
		if e.AddingEnv {
			lines = append(lines, cyanStyle.Render("> ")+e.NewNameInput.View())
		} else {
			lines = append(lines, dimStyle.Render("  [ n: new environment ]"))
		}
	}
	if len(envs) == 0 && !e.AddingEnv {
		lines = append(lines, dimStyle.Render("  no environments yet"))
	}
	return lines
}

func (e environmentsPanelState) viewVarTable(store *storage.Store) []string {
	envs := loadEnvEntries(store)
	if e.Cursor >= len(envs) {
		return nil
	}
	env := envs[e.Cursor]
	keys := sortedVarKeys(env.Variables)

	lines := []string{
		boldStyle.Render("Variables for ") + cyanStyle.Bold(true).Render(env.Name),
		strings.Repeat(" ", 3) + dimStyle.Bold(true).Render(padRight("NAME", 24)+padRight("VALUE", 30)),
	}

	for i, k := range keys {
		isCur := i == e.VarCursor
		isEditing := e.InsertingVar && !e.IsNewVar && e.EditingKey == k
		cursor := "  "
		nameStyle := lipgloss.NewStyle()
		if isCur {
			cursor = cyanStyle.Render("> ")
			nameStyle = cyanStyle
		}
		name := k
		if isEditing && e.VarField == "key" {
			name = e.KeyInput.View()
		}
		value := env.Variables[k]
		if value == "" {
			value = "-"
		}
		valueStyle := lipgloss.NewStyle().Foreground(color2xx)
		if isEditing && e.VarField == "value" {
			value = e.ValueInput.View()
			valueStyle = lipgloss.NewStyle()
		}
		lines = append(lines, cursor+padRight(nameStyle.Render(name), 24)+padRight(valueStyle.Render(value), 30))
	}

	isCurAdd := e.VarCursor == len(keys)
	isAdding := e.InsertingVar && e.IsNewVar
	cursor := "  "
	if isCurAdd {
		cursor = cyanStyle.Render("> ")
	}
	if isAdding {
		name := e.KeyInput.View()
		if e.VarField != "key" {
			name = cyanStyle.Render(displayOr(e.KeyInput.Value(), "-"))
		}
		value := dimStyle.Render("-")
		if e.VarField == "value" {
			value = e.ValueInput.View()
		}
		lines = append(lines, cursor+padRight(name, 24)+padRight(value, 30))
	} else {
		label := dimStyle.Render("[ + ]")
		if isCurAdd {
			label = cyanStyle.Render("[ i: add variable ]")
		}
		lines = append(lines, cursor+label)
	}

	return lines
}

func displayOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
