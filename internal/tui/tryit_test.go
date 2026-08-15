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
// first endpoint row (tags start collapsed, so the first tag row is
// expanded first).
func firstEndpointModel(t *testing.T) Model {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	m = step(m, "enter", "j") // expand the first tag, move onto its first endpoint row
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
	m = step(m, "x") // expand all tags so every endpoint row is in FlatList
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
	m = step(m, "x") // expand all tags so every endpoint row is in FlatList
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

// TestResetThenExitDoesNotResurrectOverride is a regression test: exitTryIt
// used to unconditionally re-save an override on every Esc exit, even an
// empty one — so pressing 'r'/'y' to reset, then Esc to leave (the natural
// next step), silently recreated an empty-but-present override, bringing
// back the "*saved params"/"~" indicators right after a reset and making it
// look like the reset didn't take. Found via a live pty capture against the
// compiled binary, not a unit test failure — see HANDOFF.md.
func TestResetThenExitDoesNotResurrectOverride(t *testing.T) {
	m := firstEndpointModel(t)
	m = m.WithServices(nil, newTestStore(t))
	item := m.selectedItem()
	ep := item.Endpoint
	m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, storage.EndpointOverride{
		Params: map[string]string{"a": "b"}, CustomParams: []storage.CustomParameter{}, DisabledParams: []string{},
	})

	m = step(m, "t", "r", "y") // enter try-it, reset, confirm
	if got := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); got != nil {
		t.Fatalf("setup: expected override deleted immediately after reset, got %+v", got)
	}

	m = step(m, "esc") // exit try-it the normal way
	if got := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); got != nil {
		t.Errorf("expected override to stay deleted after a normal exit, got %+v", got)
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

// TestExecuteWithoutChangesDoesNotMarkOverridden is a regression test: every
// execute (browse quick-execute or try-it-out's 'e') used to unconditionally
// persist an EndpointOverride, matching App.tsx's own unconditional
// saveOverride() call before executing — so pressing 'e' straight from
// browse mode on an endpoint that was never entered via try-it-out, never
// customized in any way, still left it marked "~"/"*saved params" from the
// request alone. Found via a user report ("if I execute request even if I
// did not change anything... it marked as overrided").
func TestExecuteWithoutChangesDoesNotMarkOverridden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := firstEndpointModel(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), newTestStore(t))
	ep := m.selectedItem().Endpoint

	next, cmd := m.Update(key("e"))
	m = next.(Model)
	if cmd == nil {
		t.Fatalf("expected a non-nil execute command")
	}
	msg := cmd()
	next2, _ := m.Update(msg)
	m = next2.(Model)

	if m.Viewer.Response == nil {
		t.Fatalf("expected a response")
	}
	if got := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); got != nil {
		t.Errorf("expected no override persisted after executing with nothing changed, got %+v", got)
	}
}

// TestQuickExecuteWorksFromLeftPanel is a regression test: 'e' quick-execute
// used to only work after pressing 'l' to focus the right panel, since it
// was handled in handleRightPanelKey — but useAppKeyboard.ts's real 'e'
// binding (confirmed by reading it) is gated only on `mode === 'browse'`,
// with no panel check at all, so it must fire with the left panel focused
// too (the default on entry). Found via a user report ("I could tryout from
// left panel, but I could not quick execute e").
func TestQuickExecuteWorksFromLeftPanel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := firstEndpointModel(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), newTestStore(t))

	if m.ActivePanel != PanelLeft {
		t.Fatalf("setup: expected left panel focused by default")
	}
	m = step(m, "e")
	if !m.Loading {
		t.Fatalf("expected 'e' to quick-execute from the left panel without pressing 'l' first")
	}
}

// TestResponseScrollsIntoViewAfterExecute is a regression test for a
// usability gap flagged by the user: after executing, the response could
// land below the visible viewport with no indication it had arrived,
// requiring a manual scroll down every time. Deliberate improvement over
// TS, not a port of it — App.tsx just resets scroll to 0 (top of doc) after
// executing, matched exactly by TestEnterTryItResetsRightScroll's own
// entering-try-it-out case, but that doesn't reveal the response unless it
// happens to fit above the fold. See scrollToResponse's doc comment.
func TestResponseScrollsIntoViewAfterExecute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	m := endpointWithBodyModel(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), newTestStore(t))
	// A short viewport guarantees the endpoint's docs+params+body push the
	// response section below the fold, so a real scroll is required to
	// reveal it — a tall window could pass even with scrollToResponse
	// broken, if everything already fits on screen.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m = next.(Model)

	next, cmd := m.Update(key("e"))
	m = next.(Model)
	if cmd == nil {
		t.Fatalf("expected a non-nil execute command")
	}
	msg := cmd()
	next2, _ := m.Update(msg)
	m = next2.(Model)

	if m.Viewer.Response == nil {
		t.Fatalf("expected a response")
	}
	if m.RightScroll == 0 {
		t.Errorf("expected RightScroll to advance to reveal the response section, got 0")
	}

	// Regression: an earlier version landed the response a few lines below
	// the top of the viewport (bleeding in trailing content from the
	// section above it, found from a user screenshot comparison) because
	// it clamped the scroll offset to avoid dangling blank rows at the
	// bottom — the "RESPONSE ..." heading must be the *first* visible
	// line, not just somewhere on screen.
	out := m.View()
	if idx := strings.Index(out, "RESPONSE"); idx == -1 {
		t.Fatalf("expected the response heading to render at all")
	} else {
		line := strings.Count(out[:idx], "\n")
		// Header (3 rows) + panel top border (1 row) = first content row
		// is line 4 (0-indexed).
		if line != 4 {
			t.Errorf("expected 'RESPONSE' on the panel's first content row (line 4), got line %d:\n%s", line, out)
		}
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

// TestScrollPastBodyWhenFocused is a regression test: a user reported
// being unable to scroll down in try-it-out mode once the BODY section was
// focused. renderTryItLines pins its auto-scroll cursorLine to the BODY
// box's first line every render, and the old shared scrollToShow helper
// snapped the scroll position back up to that line any time it drifted
// below it — so once the box scrolled into view, 'j' had no visible
// effect. Fixed by handleBodyFocusedKey gaining a "j"/"down" case and
// renderRightPanel switching to the one-directional scrollToShowBelow
// while BODY is focused (but not being actively edited).
func TestScrollPastBodyWhenFocused(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20}) // short terminal forces scrolling
	m = next.(Model)
	m = step(m, "x") // expand all tags so every endpoint row is in FlatList
	found := false
	for i, item := range m.FlatList {
		if item.Type == ItemEndpoint && item.Endpoint.Operation.RequestBody != nil {
			m.LeftIndex = i
			found = true
			break
		}
	}
	if !found {
		t.Skip("no endpoint with a request body in fixture")
	}
	m = step(m, "t")

	// Jump straight to BODY focus (past the last param row).
	params := sortedParameters(m.selectedItem().Endpoint.Operation.Parameters)
	for range len(params) + 1 {
		m = step(m, "j")
	}
	if !m.TryIt.BodyFocused {
		t.Fatalf("setup: expected BODY to be focused, got ParamCursor=%d BodyFocused=%v", m.TryIt.ParamCursor, m.TryIt.BodyFocused)
	}

	before := m.RightScroll
	for range 20 {
		m = step(m, "j")
	}
	if m.RightScroll <= before {
		t.Fatalf("expected RightScroll to increase while scrolling past a focused BODY, stayed at %d", m.RightScroll)
	}

	// The Responses heading lives after the BODY section in the rendered
	// output — it must actually become reachable, not just increment a
	// number nothing reads.
	out := m.View()
	if !strings.Contains(out, "Execute (e)") {
		t.Skip("terminal too small to assert on visible content in this environment")
	}
	if !strings.Contains(out, "Responses") {
		t.Errorf("expected scrolling past BODY to reveal the Responses section, got:\n%s", out)
	}
}

// TestScrollBackUpAfterScrollingPastBody is a regression test: a follow-up
// user report on the fix above — scrolling down past a focused BODY
// worked, but 'k'/'up' just unfocused straight back to PARAMETERS instead
// of first scrolling back up through what 'j' had scrolled past.
// BodyScrollFloor makes 'k' undo 'j' presses one at a time (mirroring the
// scroll-down direction) and only unfocus once back at the floor.
func TestScrollBackUpAfterScrollingPastBody(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	m = next.(Model)
	m = step(m, "x")
	found := false
	for i, item := range m.FlatList {
		if item.Type == ItemEndpoint && item.Endpoint.Operation.RequestBody != nil {
			m.LeftIndex = i
			found = true
			break
		}
	}
	if !found {
		t.Skip("no endpoint with a request body in fixture")
	}
	m = step(m, "t")

	params := sortedParameters(m.selectedItem().Endpoint.Operation.Parameters)
	for range len(params) + 1 {
		m = step(m, "j")
	}
	floor := m.TryIt.BodyScrollFloor

	const downPresses = 10
	for range downPresses {
		m = step(m, "j")
	}
	if m.RightScroll != floor+downPresses {
		t.Fatalf("setup: expected RightScroll at floor+%d, got %d (floor %d)", downPresses, m.RightScroll, floor)
	}

	for i := range downPresses {
		m = step(m, "k")
		if !m.TryIt.BodyFocused {
			t.Fatalf("unfocused early after %d 'k' presses, expected %d before reaching the floor", i+1, downPresses)
		}
		want := floor + downPresses - (i + 1)
		if m.RightScroll != want {
			t.Errorf("after %d 'k' presses: RightScroll = %d, want %d", i+1, m.RightScroll, want)
		}
	}
	if m.RightScroll != floor {
		t.Fatalf("expected RightScroll back at floor %d, got %d", floor, m.RightScroll)
	}

	// One more 'k' at the floor unfocuses back to PARAMETERS, same as
	// before this fix (unchanged single-press behavior once nothing's
	// left to scroll back through).
	m = step(m, "k")
	if m.TryIt.BodyFocused {
		t.Errorf("expected 'k' at the scroll floor to unfocus BODY, still focused")
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

// TestKAtFirstParamRowEntersHeadersFocus matches ParametersSection.tsx's
// onTabBack: pressing 'k' at the first PARAMETERS row (cursor 0) moves focus
// up into the HEADERS section instead of doing nothing.
func TestKAtFirstParamRowEntersHeadersFocus(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	if m.TryIt.ParamCursor != 0 {
		t.Fatalf("expected to start on the first PARAMETERS row, got cursor %d", m.TryIt.ParamCursor)
	}
	m = step(m, "k")
	if !m.TryIt.HeaderTable.Focused {
		t.Errorf("expected 'k' at the first param row to focus HEADERS")
	}
}

// TestAddHeaderViaHeadersSection exercises the full add-header flow: enter
// HEADERS focus, 'i' to start adding, type a name, Tab to the value field,
// type a value, Enter to commit — matches useHeadersNavigation.ts's
// insertMode branch.
func TestAddHeaderViaHeadersSection(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t", "k") // enter try-it, focus HEADERS
	if !m.TryIt.HeaderTable.Focused {
		t.Fatalf("expected HEADERS focused")
	}
	m = step(m, "i")
	if !m.TryIt.HeaderTable.Editing {
		t.Fatalf("expected 'i' to start editing the add-new header row")
	}
	m = typeText(m, "X-Test")
	m = step(m, "tab")
	m = typeText(m, "hello")
	m = step(m, "enter")

	if m.TryIt.HeaderTable.Editing {
		t.Errorf("expected editing to end after Enter")
	}
	headerParams, _ := splitCustomParams(m.TryIt.CustomParams)
	if len(headerParams) != 1 || headerParams[0].Name != "X-Test" || headerParams[0].Value != "hello" {
		t.Fatalf("expected one header param X-Test=hello, got %+v", headerParams)
	}
	if headerParams[0].In != "header" {
		t.Errorf("expected new header param to have In=\"header\", got %q", headerParams[0].In)
	}
}

// TestHeaderParamsExcludedFromParametersSection ensures a header-typed
// CustomParameter never counts toward the PARAMETERS table's row math
// (cursor bounds, x/c index math) — the two sections must stay independent
// even though they share one underlying CustomParams slice.
func TestHeaderParamsExcludedFromParametersSection(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t", "k", "i")
	m = typeText(m, "X-Test")
	m = step(m, "tab")
	m = typeText(m, "hello")
	m = step(m, "enter")

	_, nonHeader := splitCustomParams(m.TryIt.CustomParams)
	if len(nonHeader) != 0 {
		t.Errorf("expected the header param to be excluded from PARAMETERS' custom list, got %+v", nonHeader)
	}
}

// TestYankCurlKeyDoesNotCollideWithCycleParamType is a regression test for
// the try-it-out response-viewer key routing: 'C' (uppercase, yank curl) is
// intercepted before the main key switch, but lowercase 'c' (cycle param
// type) must still reach it — Go's 'C'/'c' case-sensitivity means the two
// don't actually collide, but this locks that in rather than trusting it by
// inspection.
func TestYankCurlKeyDoesNotCollideWithCycleParamType(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200}, curl: "curl -X GET https://example.com"})
	m = next.(Model)

	m = step(m, "C")
	if !m.Viewer.YankedCurl {
		t.Errorf("expected 'C' to yank the curl command")
	}

	// Move the cursor onto the always-present add-new row (last row) so
	// lowercase 'c' has something to cycle, regardless of how many spec
	// parameters this fixture's endpoint happens to declare.
	item := m.selectedItem()
	params := sortedParameters(item.Endpoint.Operation.Parameters)
	_, custom := splitCustomParams(m.TryIt.CustomParams)
	m.TryIt.ParamCursor = len(params) + len(custom)

	m = step(m, "c")
	if m.TryIt.NewParamIn != "path" {
		t.Errorf("expected lowercase 'c' to still cycle the add-new row's type, got %q", m.TryIt.NewParamIn)
	}
}

func TestExecuteMsgClearsLoadingAndSetsResponse(t *testing.T) {
	m := firstEndpointModel(t)
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200}, curl: "curl ..."})
	m = next.(Model)
	if m.Loading {
		t.Errorf("expected Loading cleared")
	}
	if m.Viewer.Response == nil || m.Viewer.Response.Status != 200 {
		t.Errorf("expected response set, got %+v", m.Viewer.Response)
	}
	if m.Viewer.Curl != "curl ..." {
		t.Errorf("expected curl set")
	}
}
