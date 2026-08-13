package tui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
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

// TestManualDefaultFocusIsParametersWithBoundaryCrossing replaces the old
// Tab-cycling test: useManualPanelKeyboard.ts (confirmed by reading it, not
// assumed) has no Tab-driven section cycling at all — PARAMETERS is the
// default focus, and 'k'/'j' at its top/bottom boundary cross into
// HEADERS/BODY, matching useRightPanelKeyboard.ts's try-it-out model
// exactly (see tryit.go's handleTryItKey).
func TestManualDefaultFocusIsParametersWithBoundaryCrossing(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m")
	if m.Manual.HeadersFocused || m.Manual.BodyFocused {
		t.Fatalf("expected PARAMETERS to be the default focus")
	}

	// 'k' at the first (only) PARAMETERS row moves focus up into HEADERS.
	m = step(m, "k")
	if !m.Manual.HeadersFocused {
		t.Errorf("expected 'k' at param row 0 to focus HEADERS")
	}
	m = step(m, "j") // exit HEADERS back down (boundary crossing both ways)
	if m.Manual.HeadersFocused {
		t.Errorf("expected 'j' at the HEADERS add-row to exit back to PARAMETERS")
	}

	// GET has no body section, so 'j' at the last param row is a no-op.
	m = step(m, "j")
	if m.Manual.BodyFocused {
		t.Errorf("expected 'j' to be a no-op with no BODY section (GET)")
	}

	m = step(m, "m") // cycle method to POST, which does have a body
	m = step(m, "j")
	if !m.Manual.BodyFocused {
		t.Errorf("expected 'j' at the last param row to focus BODY for POST")
	}
}

func TestManualAddParamRow(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m") // default focus is already PARAMETERS, cursor on the add-new row
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
	m = step(m, "m", "i")
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

// TestManualAddHeaderViaHeadersSection is the manual-builder counterpart to
// tryit_test.go's TestAddHeaderViaHeadersSection: 'k' at the default
// PARAMETERS focus (row 0) enters HEADERS, 'i' adds a new header row, and
// the result must be excluded from the PARAMETERS table's own row math.
func TestManualAddHeaderViaHeadersSection(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "k")
	if !m.Manual.HeadersFocused {
		t.Fatalf("expected 'k' at param row 0 to focus HEADERS")
	}
	m = step(m, "i")
	if !m.Manual.HeaderEditing {
		t.Fatalf("expected 'i' to start editing the add-new header row")
	}
	m = typeText(m, "X-Test")
	m = step(m, "tab")
	m = typeText(m, "hello")
	m = step(m, "enter")

	if m.Manual.HeaderEditing {
		t.Errorf("expected editing to end after Enter")
	}
	headerParams, params := splitCustomParams(m.Manual.Params)
	if len(headerParams) != 1 || headerParams[0].Name != "X-Test" || headerParams[0].Value != "hello" {
		t.Fatalf("expected one header param X-Test=hello, got %+v", headerParams)
	}
	if headerParams[0].In != "header" {
		t.Errorf("expected new header param to have In=\"header\", got %q", headerParams[0].In)
	}
	if len(params) != 0 {
		t.Errorf("expected the header param excluded from PARAMETERS, got %+v", params)
	}
}

// TestManualCycleParamType matches ParametersSection.tsx's AddNewParamRow
// type field (`'query' | 'path'` — confirmed by reading it): 'c' cycles
// only query<->path in PARAMETERS now that header-typed params live in
// their own HEADERS section (added a header can only ever be created
// there, always with In="header", never via cycling — see
// TestManualAddHeaderViaHeadersSection).
func TestManualCycleParamType(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "i")
	m = typeText(m, "a")
	m = step(m, "tab")
	m = typeText(m, "b")
	m = step(m, "enter")

	m = step(m, "c")
	if m.Manual.Params[0].In != "path" {
		t.Errorf("expected type to cycle to path, got %q", m.Manual.Params[0].In)
	}
}

// TestManualBodyEditorSupportsMultipleLines is a regression test for the
// Phase 7 unification: the manual builder's body field used to be a
// single-line bubbles/textinput (Phase 4 scope cut), unlike try-it-out's
// bubbles/textarea. Enter must insert a newline (not commit/close editing)
// and Esc must be what ends editing, matching tryit.go's body editor
// exactly now that both share the same widget.
func TestManualBodyEditorSupportsMultipleLines(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "m") // cycle to POST, which has a BODY section
	m = step(m, "j", "i") // 'j' at the (empty) last param row focuses BODY, 'i' starts editing
	if !m.Manual.EditingBody {
		t.Fatalf("expected body editing to start")
	}
	m = typeText(m, "line1")
	m = step(m, "enter")
	m = typeText(m, "line2")
	if m.Manual.EditingBody != true {
		t.Fatalf("expected Enter to insert a newline, not end editing")
	}
	if !strings.Contains(m.Manual.BodyInput.Value(), "\n") {
		t.Errorf("expected a newline in the textarea value, got %q", m.Manual.BodyInput.Value())
	}

	m = step(m, "esc")
	if m.Manual.EditingBody {
		t.Errorf("expected Esc to end editing")
	}
	if m.Manual.Body != "line1\nline2" {
		t.Errorf("expected multi-line body committed, got %q", m.Manual.Body)
	}
}

// TestManualBodyEditorShowsDoneHint mirrors
// TestBodyEditingShowsDoneHint/TestBodyEditingHintNotShownWhenIdle in
// tryit_body_test.go for the now-shared body editor.
func TestManualBodyEditorShowsDoneHint(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "m", "j", "i")
	out := m.renderManualPanel(60, 150)
	if !strings.Contains(out, "Esc: done") {
		t.Errorf("expected an 'Esc: done' hint while editing the manual body, got:\n%s", out)
	}
}

// TestManualExecuteSendsRealHTTPRequest is a coverage-gap fix: 'e' in the
// manual builder returned a Loading=true + non-nil cmd, and every existing
// test stopped there, verified via state assertions only. Nothing actually
// invoked the returned tea.Cmd, so manualExecuteCmd/runRequestCmd — the
// code that builds and sends the real HTTP request — had 0% test coverage
// despite the feature being fully wired. Found via `go test -cover` during
// the Phase 7 hardening pass, not a reported bug.
func TestManualExecuteSendsRealHTTPRequest(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := modelWithStore(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), m.Store)
	m = step(m, "m", "m") // cycle to POST
	m = step(m, "p")
	m = typeText(m, "/widgets")
	m = step(m, "enter")  // commit path
	m = step(m, "j", "i") // 'j' at the empty last param row focuses BODY
	m = typeText(m, `{"ok":true}`)
	m = step(m, "esc") // commit body (still BodyFocused, matches TS: 'e' is swallowed there too)
	m = step(m, "k")   // exit BodyFocused back to PARAMETERS so 'e' reaches the top-level execute binding

	next, cmd := m.Update(key("e"))
	m = next.(Model)
	if !m.Loading || cmd == nil {
		t.Fatalf("expected Loading=true and a non-nil command")
	}
	msg := cmd()
	next2, _ := m.Update(msg)
	m = next2.(Model)

	if m.Response == nil || m.Response.Status != 200 {
		t.Fatalf("expected a successful response, got %+v", m.Response)
	}
	if gotMethod != "POST" || gotPath != "/widgets" || gotBody != `{"ok":true}` {
		t.Errorf("expected POST /widgets with the typed body, got method=%q path=%q body=%q", gotMethod, gotPath, gotBody)
	}
}

// TestSavedRequestQuickExecuteSendsRealHTTPRequest covers the other 0%
// execute path found the same way: browse-mode 'e' on a saved request.
func TestSavedRequestQuickExecuteSendsRealHTTPRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("X-Api-Key")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := modelWithStore(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), m.Store)

	m = step(m, "m", "p")
	m = typeText(m, "/gadgets")
	m = step(m, "enter") // commit path
	// 'k' at the default PARAMETERS focus (row 0) moves up into HEADERS —
	// a header param can now only be created there, matching
	// HeadersSection.tsx (see TestManualCycleParamType's doc comment).
	m = step(m, "k", "i")
	m = typeText(m, "X-Api-Key")
	m = step(m, "tab")
	m = typeText(m, "secret")
	m = step(m, "enter") // commit the header
	// Top-level bindings like 's' don't fire while HeadersFocused (matches
	// useManualPanelKeyboard.ts's `if (headersFocused) return;`) — exit
	// headers focus first.
	m = step(m, "esc")
	m = step(m, "s")
	m = typeText(m, "Gadget Request")
	m = step(m, "enter", "enter") // name confirm, default tag, save

	// A freshly created tag starts collapsed (matches TS) — expand it so
	// the saved-request row is reachable, same as saveOneRequest does.
	if !m.ExpandedTags["default"] {
		m.toggleTag("default")
	}

	item := findSavedRequestItem(m)
	if item == nil {
		t.Fatalf("setup: expected a saved request in the flat list")
	}
	m.LeftIndex = indexOf(m.FlatList, item)
	m = step(m, "l")

	next, cmd := m.Update(key("e"))
	m = next.(Model)
	if cmd == nil {
		t.Fatalf("expected a non-nil execute command")
	}
	msg := cmd()
	next2, _ := m.Update(msg)
	m = next2.(Model)

	if m.Response == nil || m.Response.Status != 200 {
		t.Fatalf("expected a successful response, got %+v", m.Response)
	}
	if gotMethod != "GET" || gotPath != "/gadgets" || gotHeader != "secret" {
		t.Errorf("expected GET /gadgets with X-Api-Key header, got method=%q path=%q header=%q", gotMethod, gotPath, gotHeader)
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
	m = step(m, "j", "i") // 'j' at the empty last param row focuses BODY
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

// TestSaveDialogAndRenameTagOverlaysRenderViaFullView is a coverage-gap
// fix: renderSaveDialogOverlay and renderRenameTagOverlay are only ever
// invoked from View()'s mode switch, and no existing test called View()
// while either mode was active — both had 0% coverage despite the app
// wiring them in correctly. Found via `go test -cover`.
func TestSaveDialogAndRenameTagOverlaysRenderViaFullView(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "p")
	m = typeText(m, "/x")
	m = step(m, "enter", "s")
	out := m.View()
	if !strings.Contains(out, "SAVE REQUEST") {
		t.Errorf("expected the save dialog to render via View(), got:\n%s", out)
	}

	m2 := modelWithStore(t)
	if err := m2.Store.AddCustomTag(storage.CustomTag{Name: "mytag"}); err != nil {
		t.Fatal(err)
	}
	m2 = m2.refreshSavedRequests()
	idx := -1
	for i, it := range m2.FlatList {
		if it.Type == ItemTag && it.TagName == "mytag" {
			idx = i
		}
	}
	m2.LeftIndex = idx
	m2 = step(m2, "R")
	out2 := m2.View()
	if !strings.Contains(out2, "RENAME TAG") {
		t.Errorf("expected the rename-tag overlay to render via View(), got:\n%s", out2)
	}
}

// TestManualPanelRendersCustomParamRowsViaFullView is a coverage-gap fix
// for renderCustomParamRow (manual.go) — every existing param test drove
// state transitions directly and asserted on m.Manual.Params, never called
// View()/renderManualPanel with a saved param present.
func TestManualPanelRendersCustomParamRowsViaFullView(t *testing.T) {
	m := modelWithStore(t)
	m = step(m, "m", "i")
	m = typeText(m, "X-Test")
	m = step(m, "tab")
	m = typeText(m, "value1")
	m = step(m, "enter")

	out := m.View()
	if !strings.Contains(out, "X-Test") || !strings.Contains(out, "value1") {
		t.Errorf("expected the custom param row to render via View(), got:\n%s", out)
	}
}
