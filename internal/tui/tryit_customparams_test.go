package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
)

// endpointWithNoParamsModel finds an endpoint with zero spec parameters
// (e.g. a POST whose only input is its body) — the case that used to make
// the whole PARAMETERS section, hints included, disappear entirely.
func endpointWithNoParamsModel(t *testing.T) Model {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	m = next.(Model)
	for i, item := range m.FlatList {
		if item.Type == ItemEndpoint && len(item.Endpoint.Operation.Parameters) == 0 {
			m.LeftIndex = i
			return m
		}
	}
	t.Skip("no zero-parameter endpoint in fixture")
	return m
}

func TestParametersSectionAlwaysRendersEvenWithZeroSpecParams(t *testing.T) {
	m := endpointWithNoParamsModel(t)
	m = step(m, "t")

	lines, _ := m.renderTryItLines(m.selectedItem().Endpoint, 150)
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "PARAMETERS") {
		t.Fatalf("expected PARAMETERS header to render even with zero spec params, got:\n%s", out)
	}
	if !strings.Contains(out, "move") || !strings.Contains(out, "edit") || !strings.Contains(out, "disable") {
		t.Errorf("expected PARAMETERS hint text present, got:\n%s", out)
	}
	if !strings.Contains(out, "add parameter") && !strings.Contains(out, "[ + ]") {
		t.Errorf("expected the always-present add-new row, got:\n%s", out)
	}
}

func TestAddCustomParamOnZeroParamEndpoint(t *testing.T) {
	m := endpointWithNoParamsModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	if m.TryIt.ParamCursor != 0 {
		t.Fatalf("expected cursor to start on the add-new row (index 0), got %d", m.TryIt.ParamCursor)
	}

	m = step(m, "i")
	if !m.TryIt.ParamEditing {
		t.Fatalf("expected 'i' on the add-new row to start editing")
	}
	m = typeText(m, "api_key")
	m = step(m, "tab")
	m = typeText(m, "secret123")
	m = step(m, "enter")

	if len(m.TryIt.CustomParams) != 1 {
		t.Fatalf("expected one custom param added, got %d", len(m.TryIt.CustomParams))
	}
	p := m.TryIt.CustomParams[0]
	if p.Name != "api_key" || p.Value != "secret123" || p.In != "query" || !p.Enabled {
		t.Errorf("unexpected custom param: %+v", p)
	}
}

func TestEditDeleteCycleTypeCustomParam(t *testing.T) {
	m := endpointWithNoParamsModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t", "i")
	m = typeText(m, "a")
	m = step(m, "tab")
	m = typeText(m, "b")
	m = step(m, "enter")
	if len(m.TryIt.CustomParams) != 1 {
		t.Fatalf("setup: expected 1 custom param")
	}

	// cycle type query -> path
	m = step(m, "c")
	if m.TryIt.CustomParams[0].In != "path" {
		t.Errorf("expected type cycled to path, got %q", m.TryIt.CustomParams[0].In)
	}
	m = step(m, "c")
	if m.TryIt.CustomParams[0].In != "query" {
		t.Errorf("expected type cycled back to query, got %q", m.TryIt.CustomParams[0].In)
	}

	// edit existing row
	m = step(m, "i")
	if m.TryIt.ParamEditing == false {
		t.Fatalf("expected edit mode on existing custom row")
	}
	m = typeText(m, "X")
	m = step(m, "enter")
	if m.TryIt.CustomParams[0].Name != "aX" {
		t.Errorf("expected name updated to 'aX', got %q", m.TryIt.CustomParams[0].Name)
	}

	// delete it
	m = step(m, "x")
	if len(m.TryIt.CustomParams) != 0 {
		t.Errorf("expected custom param deleted, got %+v", m.TryIt.CustomParams)
	}
}

func TestCustomParamPersistsAcrossExitAndReenter(t *testing.T) {
	m := endpointWithNoParamsModel(t)
	m = m.WithServices(nil, newTestStore(t))
	ep := m.selectedItem().Endpoint
	m = step(m, "t", "i")
	m = typeText(m, "k")
	m = step(m, "tab")
	m = typeText(m, "v")
	m = step(m, "enter")
	m = step(m, "esc") // exit try-it-out, should persist via exitTryIt

	override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path)
	if override == nil || len(override.CustomParams) != 1 || override.CustomParams[0].Name != "k" {
		t.Fatalf("expected custom param persisted to override, got %+v", override)
	}

	m = step(m, "t")
	if len(m.TryIt.CustomParams) != 1 || m.TryIt.CustomParams[0].Name != "k" {
		t.Errorf("expected custom param reloaded on re-entering try-it-out, got %+v", m.TryIt.CustomParams)
	}
}

func TestExecuteSendsCustomQueryParam(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := endpointWithNoParamsModel(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), newTestStore(t))
	m = step(m, "t", "i")
	m = typeText(m, "filter")
	m = step(m, "tab")
	m = typeText(m, "active")
	m = step(m, "enter")

	next, cmd := m.Update(key("e"))
	m = next.(Model)
	if cmd == nil {
		t.Fatalf("expected execute command")
	}
	msg := cmd()
	next2, _ := m.Update(msg)
	m = next2.(Model)
	if m.Response == nil || m.Response.Status != 200 {
		t.Fatalf("expected successful response, got %+v", m.Response)
	}
	if gotQuery != "filter=active" {
		t.Errorf("expected custom query param sent, got %q", gotQuery)
	}
}

func TestQuitGuardedWhileEditingCustomParam(t *testing.T) {
	m := endpointWithNoParamsModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t", "i")
	m = typeText(m, "q")
	if !m.TryIt.ParamEditing || m.Quitting {
		t.Errorf("expected 'q' to type into the name field, not quit")
	}
	if m.TryIt.HeaderTable.NameInput.Value() != "q" {
		t.Errorf("expected literal 'q' typed, got %q", m.TryIt.HeaderTable.NameInput.Value())
	}
}
