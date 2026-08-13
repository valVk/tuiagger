package tui

import (
	"strings"
	"testing"
)

// gotoInfoSection opens the info popup (if not already open) and presses
// Tab until the requested section is active — avoids hard-coding how many
// sections precede it (Auth is conditionally skipped when the spec has no
// security schemes).
func gotoInfoSection(m Model, section infoSection) Model {
	if !m.ShowInfo {
		m = step(m, "i")
	}
	for i := 0; i < 5 && m.InfoSection != section; i++ {
		m = step(m, "tab")
	}
	return m
}

func TestAuthEditSetsCredentialViaEnter(t *testing.T) {
	m := modelWithStore(t)
	names := m.authSchemeNames()
	if len(names) == 0 {
		t.Skip("fixture has no security schemes")
	}
	m = gotoInfoSection(m, infoAuth)
	if m.InfoSection != infoAuth {
		t.Fatalf("expected to reach the Auth section")
	}

	m = step(m, "enter")
	if !m.Auth.Editing || m.Auth.Scheme != names[0] {
		t.Fatalf("expected editing to start for %q, got Editing=%v Scheme=%q", names[0], m.Auth.Editing, m.Auth.Scheme)
	}

	m = typeText(m, "sekret-token")
	m = step(m, "enter")
	if m.Auth.Editing {
		t.Errorf("expected editing to stop after enter")
	}
	if got := m.Store.LoadAuth().Credentials[names[0]]; got != "sekret-token" {
		t.Errorf("expected credential persisted, got %q", got)
	}
}

func TestAuthEditEscAlsoCommits(t *testing.T) {
	m := modelWithStore(t)
	names := m.authSchemeNames()
	if len(names) == 0 {
		t.Skip("fixture has no security schemes")
	}
	m = gotoInfoSection(m, infoAuth)
	m = step(m, "enter")
	m = typeText(m, "abc")
	m = step(m, "esc")
	if m.Auth.Editing {
		t.Errorf("expected Esc to exit edit mode")
	}
	if got := m.Store.LoadAuth().Credentials[names[0]]; got != "abc" {
		t.Errorf("expected Esc to commit like Enter (matches useAuthKeyboard.ts), got %q", got)
	}
	if !m.ShowInfo {
		t.Errorf("expected Esc to only exit the edit, not close the whole popup")
	}
}

func TestAuthEditingSwallowsIAndTab(t *testing.T) {
	m := modelWithStore(t)
	names := m.authSchemeNames()
	if len(names) == 0 {
		t.Skip("fixture has no security schemes")
	}
	m = gotoInfoSection(m, infoAuth)
	m = step(m, "enter")
	// 'i' and Tab must reach the textinput as literal characters, not
	// close the popup / switch sections while editing.
	m = typeText(m, "in")
	m = step(m, "tab")
	if !m.ShowInfo || !m.Auth.Editing {
		t.Fatalf("expected popup to stay open and editing to continue")
	}
	if m.InfoSection != infoAuth {
		t.Errorf("expected Tab to be swallowed by the credential input, not switch sections")
	}
	if m.Auth.Input.Value() != "in" {
		t.Errorf("expected 'i' typed into the input, got %q", m.Auth.Input.Value())
	}
}

func TestEnvironmentAddActivateDelete(t *testing.T) {
	m := modelWithStore(t)
	m = gotoInfoSection(m, infoEnvironments)
	if m.InfoSection != infoEnvironments {
		t.Fatalf("expected to reach Environments section")
	}

	m = step(m, "n")
	if !m.Env.AddingEnv {
		t.Fatalf("expected 'n' to start adding an environment")
	}
	m = typeText(m, "dev")
	m = step(m, "enter")
	if m.Env.AddingEnv {
		t.Errorf("expected add-env mode to close")
	}
	envs := m.Store.LoadEnvironments().Environments
	if len(envs) != 1 || envs[0].Name != "dev" {
		t.Fatalf("expected environment 'dev' persisted, got %+v", envs)
	}

	m = step(m, "enter") // activate it (cursor is already on the only row)
	if got := m.Store.LoadEnvironments().ActiveIndex; got != 0 {
		t.Errorf("expected env 0 activated, got %d", got)
	}
	if m.activeEnvName() != "dev" {
		t.Errorf("expected activeEnvName 'dev', got %q", m.activeEnvName())
	}

	out := m.View()
	if !strings.Contains(out, "env: dev") {
		t.Errorf("expected header to show active env badge, got:\n%s", out)
	}

	m = step(m, "x")
	if len(m.Store.LoadEnvironments().Environments) != 0 {
		t.Errorf("expected 'x' to delete the environment")
	}
}

func TestEnvironmentVariableAddEditDelete(t *testing.T) {
	m := modelWithStore(t)
	if err := m.Store.AddEnvironment("dev"); err != nil {
		t.Fatal(err)
	}
	m = gotoInfoSection(m, infoEnvironments)

	m = step(m, "e") // enter edit view for the (only, selected) environment
	if m.Env.View != envViewEdit {
		t.Fatalf("expected 'e' to enter variable-edit view")
	}

	m = step(m, "i") // cursor starts on the add-new row
	if !m.Env.InsertingVar || !m.Env.IsNewVar {
		t.Fatalf("expected add-new-variable insert mode")
	}
	m = typeText(m, "API_KEY")
	m = step(m, "tab")
	m = typeText(m, "topsecret")
	m = step(m, "enter")

	vars := m.Store.LoadEnvironments().Environments[0].Variables
	if vars["API_KEY"] != "topsecret" {
		t.Fatalf("expected variable persisted, got %+v", vars)
	}

	// Edit the existing row back to cursor 0 and re-edit its value.
	m.Env.VarCursor = 0
	m = step(m, "i")
	if m.Env.IsNewVar || m.Env.EditingKey != "API_KEY" {
		t.Fatalf("expected to edit the existing API_KEY row, got IsNewVar=%v EditingKey=%q", m.Env.IsNewVar, m.Env.EditingKey)
	}
	m = step(m, "tab") // move to value field
	m = typeText(m, "-v2")
	m = step(m, "enter")
	vars = m.Store.LoadEnvironments().Environments[0].Variables
	if vars["API_KEY"] != "topsecret-v2" {
		t.Errorf("expected value updated, got %+v", vars)
	}

	m.Env.VarCursor = 0
	m = step(m, "x")
	vars = m.Store.LoadEnvironments().Environments[0].Variables
	if len(vars) != 0 {
		t.Errorf("expected variable deleted, got %+v", vars)
	}

	m = step(m, "esc")
	if m.Env.View != envViewList {
		t.Errorf("expected Esc to return to the environment list")
	}
}

func TestQuitGuardedWhileEditingAuthOrEnvText(t *testing.T) {
	m := modelWithStore(t)
	names := m.authSchemeNames()
	if len(names) == 0 {
		t.Skip("fixture has no security schemes")
	}
	m = gotoInfoSection(m, infoAuth)
	m = step(m, "enter")
	m = typeText(m, "q")
	if !m.Auth.Editing || m.Quitting {
		t.Errorf("expected 'q' to type into the credential field, not quit")
	}
	if m.Auth.Input.Value() != "q" {
		t.Errorf("expected literal 'q' typed, got %q", m.Auth.Input.Value())
	}
}
