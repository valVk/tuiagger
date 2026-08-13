package tui

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.NewStore("", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// firstEndpointModel returns a Model with the left panel positioned on the
// first endpoint row (index 1 — index 0 is that endpoint's tag).
func firstEndpointModel(t *testing.T) Model {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	m = step(m, "j") // move onto the first endpoint row
	if item := m.selectedItem(); item == nil || item.Type != ItemEndpoint {
		t.Fatalf("expected an endpoint selected, got %+v", item)
	}
	return m
}

// endpointWithParamsModel finds and selects an endpoint that has at least
// one spec parameter, skipping the test if the fixture has none.
func endpointWithParamsModel(t *testing.T) Model {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	for i, item := range m.FlatList {
		if item.Type == ItemEndpoint && len(item.Endpoint.Operation.Parameters) > 0 {
			m.LeftIndex = i
			return m
		}
	}
	t.Skip("no endpoint with parameters in fixture")
	return m
}

// endpointWithBodyModel finds and selects an endpoint that declares a
// requestBody, skipping the test if the fixture has none.
func endpointWithBodyModel(t *testing.T) Model {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	for i, item := range m.FlatList {
		if item.Type == ItemEndpoint && item.Endpoint.Operation.RequestBody != nil {
			m.LeftIndex = i
			return m
		}
	}
	t.Skip("no endpoint with a request body in fixture")
	return m
}

func TestEnterTryItSwitchesModeAndPanel(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	if m.Mode != ModeTryIt {
		t.Fatalf("expected ModeTryIt")
	}
	if m.ActivePanel != PanelRight {
		t.Errorf("expected right panel focused")
	}
}

func TestEnterTryItIgnoredWithoutEndpointSelected(t *testing.T) {
	m := New(loadTestSpec(t), "") // starts on a tag row
	m = step(m, "t")
	if m.Mode != ModeBrowse {
		t.Errorf("expected 't' to be a no-op on a tag row")
	}
}

func TestEnterTryItLoadsSavedOverride(t *testing.T) {
	m := firstEndpointModel(t)
	m = m.WithServices(nil, newTestStore(t))
	item := m.selectedItem()
	ep := item.Endpoint

	override := storage.EndpointOverride{
		Params:         map[string]string{"status": "available"},
		CustomParams:   []storage.CustomParameter{},
		DisabledParams: []string{},
	}
	if err := m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, override); err != nil {
		t.Fatal(err)
	}

	m = step(m, "t")
	if m.TryIt.ParamValues["status"] != "available" {
		t.Errorf("expected saved override to load, got %+v", m.TryIt.ParamValues)
	}
}

func TestExitTryItOnEsc(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t", "esc")
	if m.Mode != ModeBrowse {
		t.Errorf("expected Esc to return to browse mode")
	}
}

func TestParamCursorNavigation(t *testing.T) {
	m := endpointWithParamsModel(t)
	m = step(m, "t")
	item := m.selectedItem()
	params := sortedParameters(item.Endpoint.Operation.Parameters)
	m = step(m, "j")
	if len(params) > 1 && m.TryIt.ParamCursor != 1 {
		t.Errorf("expected cursor to advance, got %d", m.TryIt.ParamCursor)
	}
	m = step(m, "k")
	if m.TryIt.ParamCursor != 0 {
		t.Errorf("expected cursor back at 0, got %d", m.TryIt.ParamCursor)
	}
}

func TestParamEditSetsValueOnEnter(t *testing.T) {
	m := endpointWithParamsModel(t)
	m = step(m, "t")
	item := m.selectedItem()
	params := sortedParameters(item.Endpoint.Operation.Parameters)
	name := params[0].Name

	m = step(m, "i")
	if !m.TryIt.ParamEditing {
		t.Fatalf("expected param editing to start")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = next.(Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = next.(Model)
	m = step(m, "enter")

	if m.TryIt.ParamEditing {
		t.Errorf("expected editing to end on enter")
	}
	if m.TryIt.ParamValues[name] != "xy" {
		t.Errorf("expected value 'xy', got %q", m.TryIt.ParamValues[name])
	}
}

func TestParamEditEscDiscardsNothingButExitsEditing(t *testing.T) {
	m := endpointWithParamsModel(t)
	m = step(m, "t")
	m = step(m, "i")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	m = next.(Model)
	m = step(m, "esc")
	if m.TryIt.ParamEditing {
		t.Errorf("expected esc to leave edit mode")
	}
}

func TestDisableToggle(t *testing.T) {
	m := endpointWithParamsModel(t)
	m = step(m, "t")
	item := m.selectedItem()
	params := sortedParameters(item.Endpoint.Operation.Parameters)
	name := params[0].Name

	m = step(m, "d")
	if !m.TryIt.DisabledParams[name] {
		t.Errorf("expected param disabled")
	}
	m = step(m, "d")
	if m.TryIt.DisabledParams[name] {
		t.Errorf("expected param re-enabled")
	}
}

func TestMethodCyclesThroughHTTPMethods(t *testing.T) {
	m := firstEndpointModel(t)
	item := m.selectedItem()
	baseMethod := strings.ToUpper(string(item.Endpoint.Method))
	baseIdx := slices.Index(httpMethods, baseMethod)

	m = step(m, "t")
	m = step(m, "m")
	want := httpMethods[(baseIdx+1)%len(httpMethods)]
	if m.TryIt.OverrideMethod != want {
		t.Errorf("expected cycle from %s to land on %s, got %q", baseMethod, want, m.TryIt.OverrideMethod)
	}
}

func TestPathEditSubmitsOnEnter(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t", "p")
	if !m.TryIt.EditingPath {
		t.Fatalf("expected path editing to start")
	}
	// Simulate typing by directly setting the textinput value then submitting.
	m.TryIt.PathInput.SetValue("/custom/path")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.TryIt.EditingPath {
		t.Errorf("expected path editing to end")
	}
	if m.TryIt.OverridePath != "/custom/path" {
		t.Errorf("expected override path saved, got %q", m.TryIt.OverridePath)
	}
}

func TestResetConfirmFlow(t *testing.T) {
	m := firstEndpointModel(t)
	m = m.WithServices(nil, newTestStore(t))
	item := m.selectedItem()
	ep := item.Endpoint
	m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, storage.EndpointOverride{
		Params: map[string]string{"a": "b"}, CustomParams: []storage.CustomParameter{}, DisabledParams: []string{},
	})

	m = step(m, "t", "r")
	if !m.TryIt.ShowResetConfirm {
		t.Fatalf("expected reset confirm prompt")
	}
	m = step(m, "n")
	if m.TryIt.ShowResetConfirm {
		t.Errorf("expected 'n' to dismiss confirm")
	}

	m = step(m, "r", "y")
	if m.TryIt.ShowResetConfirm {
		t.Errorf("expected confirm to close after 'y'")
	}
	if len(m.TryIt.ParamValues) != 0 {
		t.Errorf("expected params cleared after reset, got %+v", m.TryIt.ParamValues)
	}
	if got := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); got != nil {
		t.Errorf("expected override deleted from store, got %+v", got)
	}
}

func TestQuickExecuteFromBrowseUsesStubClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := firstEndpointModel(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), newTestStore(t))

	m = step(m, "l", "e")
	if !m.Loading {
		t.Fatalf("expected Loading to be set immediately after 'e'")
	}
}

func TestEnterTryItResetsRightScroll(t *testing.T) {
	// Regression: entering try-it-out used to leave whatever RightScroll
	// was left over from browsing, which — combined with the body now
	// auto-scaffolding (making the content longer) — could land the view
	// mid-document instead of at the top. Matches
	// useAppKeyboard.ts's 't' handler calling panelNav.setRightScroll(0).
	m := firstEndpointModel(t)
	m = step(m, "l", "j", "j", "j", "j", "j", "h") // scroll right panel while browsing
	if m.RightScroll == 0 {
		t.Fatalf("setup: expected a nonzero scroll before entering try-it-out")
	}
	m = step(m, "t")
	if m.RightScroll != 0 {
		t.Errorf("expected RightScroll reset to 0 on entering try-it-out, got %d", m.RightScroll)
	}
}

func TestTryItAutoScrollsToKeepSelectedParamVisible(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20}) // small height forces scrolling
	m = next.(Model)
	found := false
	for i, item := range m.FlatList {
		if item.Type == ItemEndpoint && len(item.Endpoint.Operation.Parameters) >= 2 {
			m.LeftIndex = i
			found = true
			break
		}
	}
	if !found {
		t.Skip("no endpoint with 2+ params in fixture")
	}
	m = step(m, "t")

	params := sortedParameters(m.selectedItem().Endpoint.Operation.Parameters)
	for range len(params) - 1 {
		m = step(m, "j")
	}
	if m.TryIt.ParamCursor != len(params)-1 {
		t.Fatalf("setup: expected cursor on the last param, got %d", m.TryIt.ParamCursor)
	}

	out := m.View()
	if !strings.Contains(out, "Execute (e)") {
		t.Skip("terminal too small to assert on visible content in this environment")
	}
	// The selected row's cursor marker ("> ") must actually render — if the
	// auto-scroll didn't kick in, the cursor row would be scrolled off the
	// bottom of a height-20 terminal.
	if !strings.Contains(out, "\x1b[36m>") {
		t.Errorf("expected the selected param row's cursor to be visible, got:\n%s", out)
	}
}

// TestResponseViewerWorksInTryItOutMode is a regression test: a user
// executing a request from inside try-it-out (the most natural place to
// do it — fill params, press 'e') previously had no way to visually-select
// or yank the response at all. v/y/J/K/G were completely unrouted in
// ModeTryIt (TS's own ResponseViewer.tsx is wired isActive={isActive &&
// !isTryItMode}, so it has the identical restriction — this is a
// deliberate improvement over that, not a port of it, since there's no
// real keybinding conflict and "execute, then copy" is a very common
// workflow to lock behind switching modes).
func TestResponseViewerWorksInTryItOutMode(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	body := "line0\nline1\nline2\nline3\nline4"
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200, Body: body}, curl: "curl x"})
	m = next.(Model)
	if m.Mode != ModeTryIt {
		t.Fatalf("setup: expected to stay in try-it-out mode after the response arrives")
	}

	m = step(m, "v")
	if !m.Viewer.Selecting {
		t.Fatalf("expected 'v' to start visual selection in try-it-out mode")
	}
	m = step(m, "J", "J")
	if m.Viewer.Cursor != 2 {
		t.Fatalf("expected 'J' to move the response cursor in try-it-out mode, got cursor=%d", m.Viewer.Cursor)
	}
	if got := m.Viewer.selectedText(); got != "line0\nline1\nline2" {
		t.Errorf("expected 3-line selection, got %q", got)
	}
}

// TestResponseViewerEscInTryItCancelsSelectionFirst matches the priority
// this rewrite chose when enabling the response viewer inside try-it-out:
// Esc cancels an in-progress visual selection before it means "exit
// try-it-out", so it doesn't swallow one gesture with the other.
func TestResponseViewerEscInTryItCancelsSelectionFirst(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200, Body: "a\nb\nc"}, curl: ""})
	m = next.(Model)
	m = step(m, "v")
	if !m.Viewer.Selecting {
		t.Fatalf("setup: expected selecting")
	}

	m = step(m, "esc")
	if m.Viewer.Selecting {
		t.Errorf("expected Esc to cancel the selection")
	}
	if m.Mode != ModeTryIt {
		t.Errorf("expected to remain in try-it-out after canceling a selection, got mode %v", m.Mode)
	}

	m = step(m, "esc")
	if m.Mode != ModeBrowse {
		t.Errorf("expected a second Esc (nothing selected) to exit try-it-out, got mode %v", m.Mode)
	}
}

// TestLowercaseJKExtendsSelectionInTryItMode is the try-it-out half of the
// same fix as TestLowercaseJKExtendsSelectionInBrowseMode: lowercase j/k
// are try-it-out's own param-row navigation keys, so without this a
// selecting user's 'j' would move the PARAMETERS cursor instead of
// extending the response selection.
func TestLowercaseJKExtendsSelectionInTryItMode(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200, Body: "l0\nl1\nl2\nl3"}})
	m = next.(Model)

	m = step(m, "v")
	beforeParamCursor := m.TryIt.ParamCursor
	m = step(m, "j", "j")
	if m.TryIt.ParamCursor != beforeParamCursor {
		t.Errorf("expected lowercase j to extend the selection, not move ParamCursor (changed from %d to %d)", beforeParamCursor, m.TryIt.ParamCursor)
	}
	if got := m.Viewer.selectedText(); got != "l0\nl1\nl2" {
		t.Errorf("expected 3-line selection via lowercase j in try-it-out, got %q", got)
	}
}

func TestExecuteMsgClearsLoadingAndSetsResponse(t *testing.T) {
	m := firstEndpointModel(t)
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200}, curl: "curl ..."})
	m = next.(Model)
	if m.Loading {
		t.Errorf("expected Loading cleared")
	}
	if m.Response == nil || m.Response.Status != 200 {
		t.Errorf("expected response set, got %+v", m.Response)
	}
	if m.Curl != "curl ..." {
		t.Errorf("expected curl set")
	}
}
