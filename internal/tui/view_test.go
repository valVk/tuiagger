package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
