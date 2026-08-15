package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sizedModel(t *testing.T) Model {
	t.Helper()
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return next.(Model)
}

// --- Help popup ---

func TestHelpPopupOpensAndCloses(t *testing.T) {
	m := sizedModel(t)
	m = step(m, "?")
	if !m.ShowHelp {
		t.Fatalf("expected help popup open")
	}
	out := m.View()
	if !strings.Contains(out, "KEYBOARD SHORTCUTS") {
		t.Errorf("expected help content, got:\n%s", out)
	}
	m = step(m, "?")
	if m.ShowHelp {
		t.Errorf("expected help popup closed by second '?'")
	}
}

func TestHelpPopupClosesOnEsc(t *testing.T) {
	m := step(sizedModel(t), "?", "esc")
	if m.ShowHelp {
		t.Errorf("expected esc to close help")
	}
}

func TestHelpPopupBlocksOtherKeysWhileOpen(t *testing.T) {
	m := sizedModel(t)
	before := m.ActivePanel
	m = step(m, "?", "l")
	if m.ActivePanel != before {
		t.Errorf("expected panel-switch key to be swallowed while help is open")
	}
	if !m.ShowHelp {
		t.Errorf("expected help to stay open")
	}
}

func TestHelpPopupScrollClamped(t *testing.T) {
	m := step(sizedModel(t), "?")
	m = step(m, "k") // already at top, must clamp not go negative
	if m.HelpScroll != 0 {
		t.Errorf("expected scroll clamped at 0, got %d", m.HelpScroll)
	}
	m = step(m, "G")
	afterG := m.HelpScroll
	m = step(m, "j")
	if m.HelpScroll != afterG {
		t.Errorf("expected scroll clamped at max, got %d want %d", m.HelpScroll, afterG)
	}
}

// --- Info popup ---

func TestInfoPopupOpensAndCloses(t *testing.T) {
	m := sizedModel(t)
	m = step(m, "i")
	if !m.ShowInfo {
		t.Fatalf("expected info popup open")
	}
	if m.Info.Section != infoServers {
		t.Errorf("expected to open on the Servers section")
	}
	out := m.View()
	if !strings.Contains(out, "SERVERS") {
		t.Errorf("expected servers section, got:\n%s", out)
	}
	m = step(m, "i")
	if m.ShowInfo {
		t.Errorf("expected 'i' to close the info popup (matches InfoPopup.tsx's own 'i' close binding)")
	}
}

func TestInfoPopupClosesOnEsc(t *testing.T) {
	m := step(sizedModel(t), "i", "esc")
	if m.ShowInfo {
		t.Errorf("expected esc to close info popup")
	}
}

func TestInfoPopupServerSelectionUpdatesSelectedServerAndCloses(t *testing.T) {
	m := sizedModel(t)
	if len(m.Spec.Spec.Servers) < 1 {
		t.Skip("fixture has no servers")
	}
	m = step(m, "i", "enter")
	if m.ShowInfo {
		t.Errorf("expected Enter to close the info popup, matching useServersKeyboard.ts")
	}
	if m.SelectedServer != 0 {
		t.Errorf("expected server 0 selected, got %d", m.SelectedServer)
	}
}

func TestInfoPopupTabCyclesSections(t *testing.T) {
	m := step(sizedModel(t), "i")
	first := m.Info.Section
	m = step(m, "tab")
	if m.Info.Section == first {
		t.Errorf("expected Tab to move to a different section")
	}
}

func TestInfoPopupSkipsAuthWhenNoSecuritySchemes(t *testing.T) {
	m := sizedModel(t)
	if m.Spec.Spec.Components != nil && len(m.Spec.Spec.Components.SecuritySchemes) > 0 {
		t.Skip("fixture has security schemes; this test wants none")
	}
	m = step(m, "i", "tab")
	if m.Info.Section == infoAuth {
		t.Errorf("expected auth section to be skipped when there are no security schemes")
	}
}

// --- Left panel width toggle interacting with popups ---

func TestBracketDoesNothingWhilePopupsOpen(t *testing.T) {
	m := step(sizedModel(t), "?")
	m = step(m, "[")
	if m.LeftExpanded {
		t.Errorf("expected '[' to be swallowed by the help popup")
	}
}

// --- Ctrl+R reload ---

func TestReloadSetsLoadingAndReturnsCmd(t *testing.T) {
	m := sizedModel(t).WithSource("../openapi/testdata/petstore.json")
	next, cmd := m.Update(key("ctrl+r"))
	m = next.(Model)
	if !m.SpecLoading {
		t.Fatalf("expected SpecLoading set")
	}
	if cmd == nil {
		t.Fatalf("expected a reload command")
	}
	out := m.View()
	if !strings.Contains(out, "Loading") {
		t.Errorf("expected full-screen loading view, got:\n%s", out)
	}
}

func TestReloadSuccessRebuildsSpecAndReturnsToBrowse(t *testing.T) {
	m := sizedModel(t)
	m = step(m, "enter", "j", "t") // expand first tag, select an endpoint, enter try-it to prove reload resets it
	if m.Mode != ModeTryIt {
		t.Fatalf("expected try-it mode")
	}
	spec := loadTestSpec(t)
	next, _ := m.Update(reloadMsg{spec: spec})
	m = next.(Model)
	if m.SpecLoading {
		t.Errorf("expected loading cleared")
	}
	if m.Mode != ModeBrowse {
		t.Errorf("expected reload to return to browse mode")
	}
	if len(m.FlatList) == 0 {
		t.Errorf("expected flat list rebuilt from the reloaded spec")
	}
}

func TestReloadErrorShowsFullScreenErrorAndBlocksOtherKeys(t *testing.T) {
	m := sizedModel(t)
	next, _ := m.Update(reloadMsg{err: errTest{"boom"}})
	m = next.(Model)
	if m.SpecError == "" {
		t.Fatalf("expected SpecError set")
	}
	out := m.View()
	if !strings.Contains(out, "boom") || !strings.Contains(out, "Error loading OpenAPI specification") {
		t.Errorf("expected full-screen error view, got:\n%s", out)
	}

	before := m.ActivePanel
	m2 := step(m, "l")
	if m2.ActivePanel != before {
		t.Errorf("expected non-retry keys to be swallowed on the error screen")
	}
}

func TestReloadErrorRetryViaCtrlR(t *testing.T) {
	m := sizedModel(t)
	next, _ := m.Update(reloadMsg{err: errTest{"boom"}})
	m = next.(Model)

	next, cmd := m.Update(key("ctrl+r"))
	m = next.(Model)
	if m.SpecError != "" {
		t.Errorf("expected error cleared on retry")
	}
	if !m.SpecLoading || cmd == nil {
		t.Errorf("expected retry to start loading again")
	}
}

func TestQuitAlwaysWorksEvenOnErrorScreen(t *testing.T) {
	m := sizedModel(t)
	next, _ := m.Update(reloadMsg{err: errTest{"boom"}})
	m = next.(Model)
	next, cmd := m.Update(key("q"))
	m = next.(Model)
	if !m.Quitting || cmd == nil {
		t.Errorf("expected 'q' to quit even on the error screen")
	}
}

type errTest struct{ msg string }

func (e errTest) Error() string { return e.msg }
