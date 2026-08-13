package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/storage"
)

func modelWithStore(t *testing.T) Model {
	t.Helper()
	m := New(loadTestSpec(t), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)
	return m.WithServices(nil, newTestStore(t))
}

func TestEnterManualNewFromBrowse(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m")
	if m.Mode != ModeManual {
		t.Fatalf("expected ModeManual, got %v", m.Mode)
	}
	if m.Manual.Method != "GET" {
		t.Errorf("expected default method GET, got %q", m.Manual.Method)
	}
	if m.Manual.EditingRequest != nil {
		t.Errorf("expected a blank draft, not an edit")
	}
}

func TestManualEscReturnsToBrowseWithoutSaving(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "esc")
	if m.Mode != ModeBrowse {
		t.Errorf("expected ModeBrowse after Esc, got %v", m.Mode)
	}
	if len(m.Store.LoadSavedRequests().Requests) != 0 {
		t.Errorf("expected no saved requests from an unsaved draft")
	}
}

func TestManualEditPath(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "p")
	if !m.Manual.EditingPath {
		t.Fatalf("expected EditingPath true after 'p'")
	}
	m = typeText(m, "/foo/bar")
	m = step(m, "enter")
	if m.Manual.EditingPath {
		t.Errorf("expected EditingPath false after enter")
	}
	if m.Manual.Path != "/foo/bar" {
		t.Errorf("expected path /foo/bar, got %q", m.Manual.Path)
	}
}

func TestManualCycleMethod(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "m")
	if m.Manual.Method != "POST" {
		t.Errorf("expected method to advance to POST, got %q", m.Manual.Method)
	}
}

func TestManualTabCyclesFocus(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m")
	if m.Manual.Focus != manualFocusPath {
		t.Fatalf("expected initial focus on path")
	}
	m = step(m, "tab")
	if m.Manual.Focus != manualFocusParams {
		t.Errorf("expected focus to move to params, got %v", m.Manual.Focus)
	}
	// GET has no body section, so Tab from params wraps back to path.
	m = step(m, "tab")
	if m.Manual.Focus != manualFocusPath {
		t.Errorf("expected focus to wrap to path for a bodyless method, got %v", m.Manual.Focus)
	}

	m = step(m, "m") // cycle method to POST, which does have a body
	m = step(m, "tab", "tab")
	if m.Manual.Focus != manualFocusBody {
		t.Errorf("expected focus to reach body for POST, got %v", m.Manual.Focus)
	}
}

func TestManualAddParamRow(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "tab") // focus params, cursor on the add-new row
	if m.Manual.ParamCursor != 0 || len(m.Manual.Params) != 0 {
		t.Fatalf("expected cursor 0 on empty params, got %d/%d", m.Manual.ParamCursor, len(m.Manual.Params))
	}
	m = step(m, "i")
	if !m.Manual.ParamEditing || !m.Manual.ParamAddNew {
		t.Fatalf("expected add-new row editing to start")
	}
	m = typeText(m, "X-Test")
	m = step(m, "tab")
	m = typeText(m, "hello")
	m = step(m, "enter")

	if len(m.Manual.Params) != 1 {
		t.Fatalf("expected one param added, got %d", len(m.Manual.Params))
	}
	p := m.Manual.Params[0]
	if p.Name != "X-Test" || p.Value != "hello" || p.In != "query" || !p.Enabled {
		t.Errorf("unexpected param: %+v", p)
	}
}

func TestManualDeleteParamRow(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "tab", "i")
	m = typeText(m, "a")
	m = step(m, "tab")
	m = typeText(m, "b")
	m = step(m, "enter")
	if len(m.Manual.Params) != 1 {
		t.Fatalf("setup: expected 1 param")
	}
	m = step(m, "x")
	if len(m.Manual.Params) != 0 {
		t.Errorf("expected param removed by 'x', got %d left", len(m.Manual.Params))
	}
}

func TestManualCycleParamType(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "tab", "i")
	m = typeText(m, "a")
	m = step(m, "tab")
	m = typeText(m, "b")
	m = step(m, "enter")

	m = step(m, "c")
	if m.Manual.Params[0].In != "header" {
		t.Errorf("expected type to cycle to header, got %q", m.Manual.Params[0].In)
	}
}

func TestManualSaveCreatesRequestUnderDefaultTag(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "p")
	m = typeText(m, "/ping")
	m = step(m, "enter", "s")
	if !m.Manual.ShowSaveDialog {
		t.Fatalf("expected save dialog open")
	}
	m = typeText(m, "My Request")
	m = step(m, "enter") // name -> tag focus
	if m.Manual.SaveDialog.Focus != "tag" {
		t.Fatalf("expected tag focus after name confirm")
	}
	m = step(m, "enter") // default tag preselected, save

	if m.Mode != ModeBrowse {
		t.Fatalf("expected return to browse after save")
	}
	reqs := m.Store.LoadSavedRequests().Requests
	if len(reqs) != 1 {
		t.Fatalf("expected 1 saved request, got %d", len(reqs))
	}
	if reqs[0].Name != "My Request" || reqs[0].Tag != "default" || reqs[0].Path != "/ping" {
		t.Errorf("unexpected saved request: %+v", reqs[0])
	}
}

func TestManualSaveWithNewTag(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "p")
	m = typeText(m, "/ping")
	m = step(m, "enter", "s")
	m = typeText(m, "Req")
	m = step(m, "enter") // -> tag focus
	// Cycle past every existing tag ("default" + spec tags) to reach the
	// new-tag slot at the end of the list.
	for i := 0; i < len(m.Manual.SaveDialog.Tags); i++ {
		m = step(m, "right")
	}
	if !m.Manual.SaveDialog.NewTagMode {
		t.Fatalf("expected new-tag mode after cycling past the last tag")
	}
	m = typeText(m, "custom-tag")
	m = step(m, "enter")

	reqs := m.Store.LoadSavedRequests().Requests
	if len(reqs) != 1 || reqs[0].Tag != "custom-tag" {
		t.Fatalf("expected request tagged custom-tag, got %+v", reqs)
	}
	tags := m.Store.LoadSavedRequests().CustomTags
	if len(tags) != 1 || tags[0].Name != "custom-tag" {
		t.Errorf("expected custom tag persisted, got %+v", tags)
	}
	if !m.isCustomTag("custom-tag") {
		t.Errorf("expected in-memory CustomTags refreshed after save")
	}
}

func findSavedRequestItem(m Model) *FlatListItem {
	for i := range m.FlatList {
		if m.FlatList[i].Type == ItemSavedRequest {
			return &m.FlatList[i]
		}
	}
	return nil
}

func saveOneRequest(t *testing.T, m Model, name, path string) Model {
	t.Helper()
	m = step(m, "m", "p")
	m = typeText(m, path)
	m = step(m, "enter", "s")
	m = typeText(m, name)
	m = step(m, "enter", "enter") // name confirm, default tag, save

	// A freshly created tag starts collapsed — matches TS's expandedTags,
	// which is seeded once at mount and never auto-includes tags that show
	// up later. Expand it so the saved-request row is reachable.
	if !m.ExpandedTags["default"] {
		m.toggleTag("default")
	}
	return m
}

func TestSavedRequestAppearsInFlatListAndCanBeEditedAndDeleted(t *testing.T) {
	m := modelWithStore(t)
	m = saveOneRequest(t, m, "Ping", "/ping")

	item := findSavedRequestItem(m)
	if item == nil {
		t.Fatalf("expected a saved-request row in the flat list")
	}
	m.LeftIndex = indexOf(m.FlatList, item)

	m = step(m, "E")
	if m.Mode != ModeManual || m.Manual.EditingRequest == nil {
		t.Fatalf("expected 'E' to enter edit mode for the saved request")
	}
	if m.Manual.Path != "/ping" {
		t.Errorf("expected editor preloaded with saved path, got %q", m.Manual.Path)
	}
	m = step(m, "esc") // discard edit, back to browse

	m.LeftIndex = indexOf(m.FlatList, findSavedRequestItem(m))
	m = step(m, "D")
	if len(m.Store.LoadSavedRequests().Requests) != 0 {
		t.Errorf("expected 'D' to delete the saved request")
	}
	if findSavedRequestItem(m) != nil {
		t.Errorf("expected saved-request row gone from the flat list")
	}
}

func indexOf(list []FlatListItem, item *FlatListItem) int {
	for i := range list {
		if list[i].ID == item.ID {
			return i
		}
	}
	return -1
}

func TestManualDeleteEditingRequestWithDKey(t *testing.T) {
	m := modelWithStore(t)
	m = saveOneRequest(t, m, "Ping", "/ping")
	item := findSavedRequestItem(m)
	m.LeftIndex = indexOf(m.FlatList, item)
	m = step(m, "E")
	m = step(m, "d")
	if m.Mode != ModeBrowse {
		t.Errorf("expected return to browse after deleting editing request")
	}
	if len(m.Store.LoadSavedRequests().Requests) != 0 {
		t.Errorf("expected saved request deleted via manual-mode 'd'")
	}
}

// TestManualViewDoesNotOverflowTerminalHeight is the manual-builder half of
// the regression covered by
// TestRenderTryItLinesEveryElementIsOneTerminalRow in tryit_body_test.go:
// renderManualPanel's BODY box had the same bug (a multi-line bordered
// render appended as one flat-lines element, under-counting its real
// height and overflowing the terminal). Assert the whole rendered frame
// never exceeds the window height it was given.
func TestManualViewDoesNotOverflowTerminalHeight(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "m") // cycle to POST, which has a BODY box
	if m.Manual.Method != "POST" {
		t.Fatalf("setup: expected method POST, got %q", m.Manual.Method)
	}
	m = step(m, "tab", "tab", "i") // focus params -> body -> start editing
	m = typeText(m, `{"a":1}`)
	m = step(m, "esc")

	out := m.View()
	rows := strings.Count(out, "\n") + 1
	if rows > m.Height {
		t.Errorf("expected rendered view to fit within %d rows, got %d", m.Height, rows)
	}
}

func TestRenameCustomTag(t *testing.T) {
	m := modelWithStore(t)
	m = saveOneRequest(t, m, "Req", "/x") // saves under "default", no custom tag yet
	if err := m.Store.AddCustomTag(storage.CustomTag{Name: "mytag"}); err != nil {
		t.Fatal(err)
	}
	m = m.refreshSavedRequests()

	idx := -1
	for i, it := range m.FlatList {
		if it.Type == ItemTag && it.TagName == "mytag" {
			idx = i
		}
	}
	if idx == -1 {
		t.Fatalf("expected custom tag row in flat list")
	}
	m.LeftIndex = idx

	m = step(m, "R")
	if m.Mode != ModeRenameTag {
		t.Fatalf("expected ModeRenameTag after 'R' on a custom tag")
	}
	// Clear the pre-filled name and type the new one.
	for range "mytag" {
		m = step(m, "backspace")
	}
	m = typeText(m, "renamed")
	m = step(m, "enter")

	if m.Mode != ModeBrowse {
		t.Errorf("expected return to browse after rename")
	}
	tags := m.Store.LoadSavedRequests().CustomTags
	if len(tags) != 1 || tags[0].Name != "renamed" {
		t.Errorf("expected tag renamed on disk, got %+v", tags)
	}
}

func TestDeleteCustomTagRequiresConfirmation(t *testing.T) {
	m := modelWithStore(t)
	if err := m.Store.AddCustomTag(storage.CustomTag{Name: "mytag"}); err != nil {
		t.Fatal(err)
	}
	m = m.refreshSavedRequests()
	idx := -1
	for i, it := range m.FlatList {
		if it.Type == ItemTag && it.TagName == "mytag" {
			idx = i
		}
	}
	m.LeftIndex = idx

	m = step(m, "D")
	if m.TagDeleteConfirm != "mytag" {
		t.Fatalf("expected delete confirmation pending, got %q", m.TagDeleteConfirm)
	}
	m = step(m, "n")
	if m.TagDeleteConfirm != "" {
		t.Errorf("expected confirmation cleared after 'n'")
	}
	if !m.isCustomTag("mytag") {
		t.Errorf("expected tag to survive 'n'")
	}

	m.LeftIndex = idx
	m = step(m, "D", "y")
	if m.isCustomTag("mytag") {
		t.Errorf("expected tag deleted after 'y'")
	}
}
