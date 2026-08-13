package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

func TestViewRendersWithoutPanicAtVariousSizes(t *testing.T) {
	sizes := []struct{ w, h int }{
		{0, 0}, // before first WindowSizeMsg
		{80, 24},
		{40, 10}, // narrow/short terminal
		{200, 60},
	}
	for _, sz := range sizes {
		m := New(loadTestSpec(t), "PetStore")
		next, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		m = next.(Model)
		out := m.View()
		if sz.w == 0 && out == "" {
			t.Errorf("expected placeholder output before window size is known")
		}
	}
}

func TestViewShowsEndpointDetailsWhenSelected(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	m = step(m, "j", "l") // select first endpoint row, focus right panel

	out := m.View()
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint {
		t.Fatalf("expected endpoint selected")
	}
	if !strings.Contains(out, item.Endpoint.Path) {
		t.Errorf("expected view to contain endpoint path %q", item.Endpoint.Path)
	}
}

func TestActionBannerShownRegardlessOfActivePanel(t *testing.T) {
	m := firstEndpointModel(t)

	// The banner renders unconditionally now — gating it on `active` made
	// the right panel's content jump vertically every time focus toggled
	// between panels (h/l), since the banner popped in/out.
	leftOut := strings.Join(m.renderEndpointLines(m.selectedItem().Endpoint, false, 80), "\n")
	if !strings.Contains(leftOut, "Try it out (t)") || !strings.Contains(leftOut, "Quick execute (e)") {
		t.Errorf("expected action banner while left panel focused too, got:\n%s", leftOut)
	}

	rightOut := strings.Join(m.renderEndpointLines(m.selectedItem().Endpoint, true, 80), "\n")
	if !strings.Contains(rightOut, "Try it out (t)") || !strings.Contains(rightOut, "Quick execute (e)") {
		t.Errorf("expected action banner when right panel active, got:\n%s", rightOut)
	}
}

func TestStatusBarHintsColorKeysDistinctFromLabels(t *testing.T) {
	m := firstEndpointModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	out := m.renderStatusBar()
	// The key gets a cyan ANSI code (36) that the label text must not share
	// — a regression to one flat dim block would collapse this to a single
	// escape sequence with no cyan segment at all.
	if !strings.Contains(out, "\x1b[36m") {
		t.Errorf("expected status bar hint keys to carry a distinct cyan style, got:\n%q", out)
	}
}

func TestTryItActionButtonsAreColored(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	out := strings.Join(m.renderTryItLines(m.selectedItem().Endpoint, 80), "\n")
	if !strings.Contains(out, "Execute (e)") {
		t.Errorf("expected Execute button, got:\n%s", out)
	}
	if !strings.Contains(out, "\x1b[32;1m") && !strings.Contains(out, "\x1b[1;32m") {
		t.Errorf("expected Execute button to be green+bold, got:\n%q", out)
	}
}

func TestResponsesHeadingMatchesTSCasingAndNextHint(t *testing.T) {
	m := firstEndpointModel(t) // left panel focused -> not "active"
	item := m.selectedItem()
	inactiveOut := strings.Join(m.renderEndpointLines(item.Endpoint, false, 80), "\n")
	if !strings.Contains(inactiveOut, "Responses") {
		t.Errorf("expected 'Responses' (TS casing, not 'RESPONSES'), got:\n%s", inactiveOut)
	}
	if strings.Contains(inactiveOut, "/:next") {
		t.Errorf("expected no /:next hint while inactive, got:\n%s", inactiveOut)
	}

	if len(item.Endpoint.Operation.Responses) < 2 {
		t.Skip("endpoint doesn't have multiple response codes to test /:next")
	}
	activeOut := strings.Join(m.renderEndpointLines(item.Endpoint, true, 80), "\n")
	if !strings.Contains(activeOut, "/:next") {
		t.Errorf("expected /:next hint when active with multiple response codes, got:\n%s", activeOut)
	}
}

func TestOnlyActiveResponseTabGetsStatusColor(t *testing.T) {
	m := firstEndpointModel(t)
	item := m.selectedItem()
	if len(item.Endpoint.Operation.Responses) < 2 {
		t.Skip("endpoint doesn't have multiple response codes")
	}
	out := strings.Join(renderResponseTabs(item.Endpoint.Operation, 0, true), "\n")
	// The active tab (first, since activeTab=0) is bold+reverse+colored in
	// one combined SGR sequence; inactive tabs render with no escape codes
	// at all (TS: color only when isTab — replicated exactly).
	if !strings.Contains(out, ";7;") { // reverse video (SGR 7) only on the active tab
		t.Errorf("expected active tab to include reverse-video SGR code, got:\n%q", out)
	}
	if !strings.Contains(out, " 400 ") { // inactive tab: bare text, no escape prefix
		t.Errorf("expected inactive tab rendered as plain text, got:\n%q", out)
	}
}

func TestLeftPanelShowsOverrideIndicator(t *testing.T) {
	m := firstEndpointModel(t)
	m = m.WithServices(nil, newTestStore(t))
	item := m.selectedItem()
	ep := item.Endpoint

	before := m.renderListRow(*item, false, 80)
	if strings.Contains(before, "~") {
		t.Errorf("expected no override indicator before any override saved, got %q", before)
	}

	if err := m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, storage.EndpointOverride{
		Params: map[string]string{"a": "b"}, CustomParams: []storage.CustomParameter{}, DisabledParams: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	after := m.renderListRow(*item, false, 80)
	if !strings.Contains(after, "~") {
		t.Errorf("expected '~' override indicator after saving an override, got %q", after)
	}
}

func TestViewShowsEmptySelectionPrompt(t *testing.T) {
	// A spec with no endpoints selects nothing meaningful on the right —
	// guard against a nil-selection panic explicitly.
	m := Model{ActivePanel: PanelRight}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	out := m.View()
	if !strings.Contains(out, "Select an endpoint") {
		t.Errorf("expected empty-selection prompt, got:\n%s", out)
	}
}

func TestViewRendersTryItModeWithoutPanic(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "t")
	out := m.View()
	if !strings.Contains(out, "Execute") {
		t.Errorf("expected try-it hints in view, got:\n%s", out)
	}
}

func TestViewRendersResponseBlockWithoutPanic(t *testing.T) {
	// View() only exposes whatever fits the current scroll window, and the
	// response block sits after a long params/body/responses section for
	// most endpoints — assert on the block renderer directly rather than
	// depending on scroll position within the full View() output.
	m := firstEndpointModel(t)
	next, _ := m.Update(responseMsg{
		response: &request.Response{Status: 200, StatusText: "OK", Body: "{\n  \"a\": 1\n}", TimeMs: 12},
		curl:     "curl 'http://x'",
	})
	m = next.(Model)

	out := m.View() // still must not panic at whatever scroll position
	if out == "" {
		t.Errorf("expected non-empty view")
	}

	block := strings.Join(m.renderResponseBlock(80), "\n")
	if !strings.Contains(block, "200") {
		t.Errorf("expected status code in response block, got:\n%s", block)
	}
}

// TS's ResponseViewer never renders response.error (grepped — zero call
// sites); a network failure just leaves status=0/"Error" and an empty body,
// rendered through the exact same status-line path as any other response.
// This looks like a gap in the TS app, but "strictly follow the TS UX
// decisions" means replicating it rather than silently improving on it —
// flagged in HANDOFF.md as a candidate the user may want to diverge from.
func TestViewRendersErrorResponseMatchesTSSilentBehavior(t *testing.T) {
	m := firstEndpointModel(t)
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 0, StatusText: "Error"}})
	m = next.(Model)

	out := m.View()
	if out == "" {
		t.Errorf("expected non-empty view")
	}

	block := strings.Join(m.renderResponseBlock(80), "\n")
	if !strings.Contains(block, "Error") {
		t.Errorf("expected bare '0 Error' status text, got:\n%s", block)
	}
}

func TestViewQuittingRendersEmpty(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	next, _ = m.Update(key("q"))
	m = next.(Model)
	if m.View() != "" {
		t.Errorf("expected empty view once quitting")
	}
}
