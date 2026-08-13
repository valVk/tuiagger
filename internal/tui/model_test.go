package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
)

func loadTestSpec(t *testing.T) *openapi.ParsedSpec {
	t.Helper()
	parsed, err := openapi.ParseOpenAPISpec("../openapi/testdata/petstore.json")
	if err != nil {
		t.Fatalf("failed to load test spec: %v", err)
	}
	return parsed
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	}
	// Any other single rune (letters, digits, punctuation) is a plain
	// KeyRunes press — covers the growing set of single-char bindings
	// (m/p/s/d/a/R/D/E/etc.) without hand-listing each one here.
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// typeText sends each rune of s as its own KeyRunes press, matching how a
// real terminal delivers typed text to a focused textinput.
func typeText(m Model, s string) Model {
	for _, r := range s {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(Model)
	}
	return m
}

func step(m Model, keys ...string) Model {
	for _, k := range keys {
		next, _ := m.Update(key(k))
		m = next.(Model)
	}
	return m
}

// TestNewCollapsesAllTagsInitially is a deliberate divergence from TS
// (usePanelNavigation.ts starts with every tag expanded) — see New's doc
// comment.
func TestNewCollapsesAllTagsInitially(t *testing.T) {
	m := New(loadTestSpec(t), "")
	for _, tag := range m.AllTags {
		if m.ExpandedTags[tag] {
			t.Errorf("expected tag %q to start collapsed", tag)
		}
	}
	// With everything collapsed, the flat list must contain only tag rows.
	if len(m.FlatList) != len(m.AllTags) {
		t.Errorf("expected flat list to include only tag rows, got %d items", len(m.FlatList))
	}
}

func TestPanelSwitching(t *testing.T) {
	m := New(loadTestSpec(t), "")
	if m.ActivePanel != PanelLeft {
		t.Fatalf("expected to start on left panel")
	}
	m = step(m, "l")
	if m.ActivePanel != PanelRight {
		t.Errorf("expected right panel after 'l'")
	}
	m = step(m, "h")
	if m.ActivePanel != PanelLeft {
		t.Errorf("expected left panel after 'h'")
	}
}

func TestLeftPanelNavigationClampsAtBounds(t *testing.T) {
	m := New(loadTestSpec(t), "")
	m = step(m, "k") // already at 0, must clamp not go negative
	if m.LeftIndex != 0 {
		t.Errorf("expected clamp at 0, got %d", m.LeftIndex)
	}
	m = step(m, "G")
	if m.LeftIndex != len(m.FlatList)-1 {
		t.Errorf("expected G to jump to last item, got %d/%d", m.LeftIndex, len(m.FlatList)-1)
	}
	m = step(m, "j") // already at last, must clamp
	if m.LeftIndex != len(m.FlatList)-1 {
		t.Errorf("expected clamp at last item, got %d", m.LeftIndex)
	}
	m = step(m, "g")
	if m.LeftIndex != 0 {
		t.Errorf("expected g to jump to first item, got %d", m.LeftIndex)
	}
}

func TestToggleTagCollapsesAndExpands(t *testing.T) {
	m := New(loadTestSpec(t), "")
	firstTag := m.FlatList[0].TagName
	initialLen := len(m.FlatList)

	m = step(m, "enter") // expand the first (selected, collapsed by default) tag row
	if !m.ExpandedTags[firstTag] {
		t.Errorf("expected tag %q to expand", firstTag)
	}
	if len(m.FlatList) <= initialLen {
		t.Errorf("expected flat list to grow after expanding a tag")
	}

	m = step(m, "enter") // collapse it back
	if m.ExpandedTags[firstTag] {
		t.Errorf("expected tag %q to re-collapse", firstTag)
	}
	if len(m.FlatList) != initialLen {
		t.Errorf("expected flat list to return to original length, got %d want %d", len(m.FlatList), initialLen)
	}
}

func TestCollapseAllAndExpandAll(t *testing.T) {
	m := New(loadTestSpec(t), "")
	m = step(m, "c")
	if len(m.FlatList) != len(m.AllTags) {
		t.Errorf("expected only tag rows after collapse-all, got %d want %d", len(m.FlatList), len(m.AllTags))
	}
	if m.LeftIndex != 0 {
		t.Errorf("expected left index reset to 0 after collapse-all")
	}

	m = step(m, "x")
	if len(m.FlatList) != len(m.AllTags)+len(m.Spec.Endpoints) {
		t.Errorf("expected all endpoints back after expand-all, got %d", len(m.FlatList))
	}
}

func TestRightPanelScroll(t *testing.T) {
	m := New(loadTestSpec(t), "")
	m = step(m, "l") // focus right panel
	m = step(m, "j", "j", "j")
	if m.RightScroll != 3 {
		t.Errorf("expected scroll 3, got %d", m.RightScroll)
	}
	m = step(m, "k")
	if m.RightScroll != 2 {
		t.Errorf("expected scroll 2, got %d", m.RightScroll)
	}
	m = step(m, "g")
	if m.RightScroll != 0 {
		t.Errorf("expected scroll reset to 0, got %d", m.RightScroll)
	}
}

func TestNavigatingLeftPanelResetsRightScrollAndResponseTab(t *testing.T) {
	m := New(loadTestSpec(t), "")
	m = step(m, "l", "j", "j", "h") // scroll right panel, then back to left
	m.RightScroll = 5
	m.ResponseTab = 2
	m = step(m, "j")
	if m.RightScroll != 0 || m.ResponseTab != 0 {
		t.Errorf("expected scroll+tab reset on left-panel navigation, got scroll=%d tab=%d", m.RightScroll, m.ResponseTab)
	}
}

func TestResponseTabCyclingOnlyOnEndpointRows(t *testing.T) {
	m := New(loadTestSpec(t), "")
	// Expand the first (collapsed by default) tag, then navigate onto its
	// first endpoint row.
	m = step(m, "enter", "j", "l")
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint {
		t.Fatalf("expected an endpoint selected, got %+v", item)
	}
	codes := sortedResponseCodes(item.Endpoint)
	if len(codes) < 2 {
		t.Skip("selected endpoint doesn't have multiple response codes to cycle")
	}
	m = step(m, "/")
	if m.ResponseTab != 1 {
		t.Errorf("expected response tab to advance to 1, got %d", m.ResponseTab)
	}
}

func TestLeftBracketTogglesLeftPanelWidth(t *testing.T) {
	m := New(loadTestSpec(t), "")
	if m.LeftExpanded {
		t.Fatalf("expected LeftExpanded false initially")
	}
	m = step(m, "[")
	if !m.LeftExpanded {
		t.Errorf("expected LeftExpanded true after '['")
	}
	m = step(m, "[")
	if m.LeftExpanded {
		t.Errorf("expected LeftExpanded false after second '['")
	}
}

func TestQuitSetsQuittingAndReturnsQuitCmd(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, cmd := m.Update(key("q"))
	nm := next.(Model)
	if !nm.Quitting {
		t.Errorf("expected Quitting to be set")
	}
	if cmd == nil {
		t.Errorf("expected a tea.Quit command")
	}
}

func TestWindowSizeMsgSetsDimensions(t *testing.T) {
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	nm := next.(Model)
	if nm.Width != 120 || nm.Height != 40 {
		t.Errorf("expected dimensions to be set, got %dx%d", nm.Width, nm.Height)
	}
}

func TestInitReturnsNoCmd(t *testing.T) {
	m := New(loadTestSpec(t), "")
	if m.Init() != nil {
		t.Errorf("expected Init to return no command")
	}
}
