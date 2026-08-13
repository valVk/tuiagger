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
