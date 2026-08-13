package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/request"
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

func TestActionBannerShownOnlyWhenRightPanelActive(t *testing.T) {
	m := firstEndpointModel(t)

	// Left panel focused: no action banner (matches TS's `isActive` gate).
	leftOut := strings.Join(m.renderEndpointLines(m.selectedItem().Endpoint, false, 80), "\n")
	if strings.Contains(leftOut, "Try it out") {
		t.Errorf("expected no action banner while left panel focused")
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

func TestViewRendersErrorResponseWithoutPanic(t *testing.T) {
	m := firstEndpointModel(t)
	next, _ := m.Update(responseMsg{response: &request.Response{Error: "connection refused"}})
	m = next.(Model)

	out := m.View()
	if out == "" {
		t.Errorf("expected non-empty view")
	}

	block := strings.Join(m.renderResponseBlock(80), "\n")
	if !strings.Contains(block, "connection refused") {
		t.Errorf("expected error text in response block, got:\n%s", block)
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
