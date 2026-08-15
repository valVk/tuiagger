// Package tui is the root Bubbletea model: a single Update dispatches every
// key to a state transition, replacing the TS app's 9 independently
// duplicated keyboard hooks with one place input becomes state (see the
// tuiagger-dev-go skill for why this is the point of the rewrite, not
// incidental to it).
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

type ActivePanel int

const (
	PanelLeft ActivePanel = iota
	PanelRight
)

type Mode int

const (
	ModeBrowse Mode = iota
	ModeTryIt
	ModeManual
	ModeRenameTag
)

type Model struct {
	Spec           *openapi.ParsedSpec
	CollectionName string
	SelectedServer int

	AllTags            []string
	EndpointsByTag     map[string][]openapi.ParsedEndpoint
	SavedRequests      []storage.SavedRequest
	SavedRequestsByTag map[string][]storage.SavedRequest
	CustomTags         []storage.CustomTag
	ExpandedTags       map[string]bool
	FlatList           []FlatListItem

	ActivePanel ActivePanel
	LeftIndex   int
	RightScroll int
	ResponseTab int // index into the selected endpoint's sorted response codes

	Mode             Mode
	TryIt            tryItState
	Manual           manualState
	RenameTag        renameTagState
	TagDeleteConfirm string // custom tag name pending 'D' confirmation, "" = none

	Loading bool
	Viewer  responseViewer

	HTTPClient request.HTTPClient
	Store      *storage.Store

	LeftExpanded bool // '[' toggles 30% <-> 50% left panel width, matching App.tsx

	ShowHelp bool
	Help     helpPopupState

	ShowInfo bool
	Info     infoPopupState

	// Source is the collection/path/URL the spec was loaded from, kept for
	// Ctrl+R reload (re-runs openapi.ParseOpenAPISpec against it).
	Source      string
	SpecLoading bool
	SpecError   string

	Width, Height int
	Quitting      bool
}

// New builds the initial Model with every tag collapsed. Deliberate
// divergence from TS's usePanelNavigation.ts, which starts with every tag
// expanded (`useState(() => new Set(allTags))`) — a large spec otherwise
// dumps every endpoint into view on first launch instead of just its tags.
func New(spec *openapi.ParsedSpec, collectionName string) Model {
	endpointsByTag := openapi.GetEndpointsByTag(spec.Endpoints)
	m := Model{
		Spec:           spec,
		CollectionName: collectionName,
		AllTags:        spec.Tags,
		EndpointsByTag: endpointsByTag,
		ExpandedTags:   make(map[string]bool, len(spec.Tags)),
		ActivePanel:    PanelLeft,
	}
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
	return m
}

// WithServices injects the HTTP client and persistence store — split from
// New so tests can build a Model without a real store/client when neither
// is exercised.
func (m Model) WithServices(client request.HTTPClient, store *storage.Store) Model {
	m.HTTPClient = client
	m.Store = store
	return m.refreshSavedRequests()
}

// refreshSavedRequests reloads saved requests/custom tags from disk and
// recomputes AllTags/FlatList, matching useSavedRequests.ts's store-backed
// getAllTags — called after any CRUD on saved requests or custom tags.
func (m Model) refreshSavedRequests() Model {
	if m.Store == nil {
		return m
	}
	store := m.Store.LoadSavedRequests()
	m.SavedRequests = store.Requests
	m.CustomTags = store.CustomTags
	m.SavedRequestsByTag = groupByTag(store.Requests)
	m.AllTags = computeAllTags(m.Spec.Tags, store.CustomTags, store.Requests)
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
	m.LeftIndex = m.safeLeftIndex()
	return m
}

func groupByTag(reqs []storage.SavedRequest) map[string][]storage.SavedRequest {
	out := map[string][]storage.SavedRequest{}
	for i := range reqs {
		out[reqs[i].Tag] = append(out[reqs[i].Tag], reqs[i])
	}
	return out
}

// computeAllTags matches useSavedRequests.ts's getAllTags:
// [...new Set([...specTags, ...customTagNames, ...requestTagNames])].
func computeAllTags(specTags []string, customTags []storage.CustomTag, reqs []storage.SavedRequest) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range specTags {
		add(t)
	}
	for _, t := range customTags {
		add(t.Name)
	}
	for _, r := range reqs {
		add(r.Tag)
	}
	return out
}

// isEditingText reports whether a textinput is currently focused anywhere
// in the app — used to keep single-character bindings like 'q' from being
// swallowed as a global shortcut while the user is typing.
func (m Model) isEditingText() bool {
	return m.TryIt.EditingPath || m.TryIt.ParamEditing || m.TryIt.EditingBody ||
		m.Manual.EditingPath || m.Manual.ParamEditing || m.Manual.EditingBody ||
		m.Manual.ShowSaveDialog || m.Mode == ModeRenameTag ||
		m.Info.Auth.Editing || m.Info.Environments.InsertingVar || m.Info.Environments.AddingEnv
}

func (m Model) isCustomTag(name string) bool {
	for _, t := range m.CustomTags {
		if t.Name == name {
			return true
		}
	}
	return false
}

// WithSource records where the spec was loaded from, enabling Ctrl+R reload.
func (m Model) WithSource(source string) Model {
	m.Source = source
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// responseMsg carries a completed request's result back into Update — the
// tea.Cmd pattern that keeps HTTP execution out of Update itself.
type responseMsg struct {
	response *request.Response
	curl     string
}

// reloadMsg carries the result of a Ctrl+R reload, matching
// useOpenAPI.ts's reload(): on success the whole spec is swapped in; on
// failure the TS app discards its working UI in favor of a full-screen
// error (App.tsx renders the error branch whenever `error` is set,
// regardless of whether a previous `spec` still exists) — replicated in
// View() via SpecError taking priority over everything else.
type reloadMsg struct {
	spec *openapi.ParsedSpec
	err  error
}

func (m Model) reloadCmd() tea.Cmd {
	source := m.Source
	return func() tea.Msg {
		parsed, err := openapi.ParseOpenAPISpec(source)
		return reloadMsg{spec: parsed, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		return m, nil
	case responseMsg:
		m.Loading = false
		var cmd tea.Cmd
		m.Viewer, cmd = m.Viewer.Update(msg)
		m.RightScroll = m.scrollToResponse()
		return m, cmd
	case yankExpiredMsg:
		var cmd tea.Cmd
		m.Viewer, cmd = m.Viewer.Update(msg)
		return m, cmd
	case reloadMsg:
		return m.applyReload(msg), nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) applyReload(msg reloadMsg) Model {
	m.SpecLoading = false
	if msg.err != nil {
		m.SpecError = msg.err.Error()
		return m
	}
	m.SpecError = ""
	m.Spec = msg.spec
	m.AllTags = computeAllTags(msg.spec.Tags, m.CustomTags, m.SavedRequests)
	m.EndpointsByTag = openapi.GetEndpointsByTag(msg.spec.Endpoints)
	// Preserve which tags were expanded across the reload, same as TS
	// (usePanelNavigation's expandedTags is untouched by a spec refresh).
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
	m.LeftIndex = m.safeLeftIndex()
	m.Mode = ModeBrowse
	m.Viewer = responseViewer{}
	m.Loading = false
	m.RightScroll = 0
	return m
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// 'q' always quits, even mid-reload or on the full-screen error view —
	// matches the TS app's onQuit binding staying mounted regardless of
	// what App.tsx is currently rendering. The one exception: while a
	// textinput is focused, 'q' must reach it as a literal character
	// (typing a path/param/body containing the letter 'q') rather than
	// quitting the app out from under the user.
	if key == "q" && !m.isEditingText() {
		m.Quitting = true
		return m, tea.Quit
	}

	// A failed reload replaces the whole UI with an error view in TS
	// (App.tsx: `if (error || !spec) return <ErrorScreen/>`, evaluated
	// before Header/panels/StatusBar render at all) — only retry (Ctrl+R)
	// and quit make sense here.
	if m.SpecError != "" {
		if key == "ctrl+r" {
			m.SpecLoading, m.SpecError = true, ""
			return m, m.reloadCmd()
		}
		return m, nil
	}
	if m.SpecLoading {
		return m, nil
	}

	if m.ShowHelp {
		return m.handleHelpKey(key)
	}
	if m.ShowInfo {
		return m.handleInfoKey(msg)
	}

	// Every other mode has its own Update entry point; browse (the
	// default/fallthrough Mode) has handleBrowseKey in browse.go —
	// everything past this dispatch used to live inline here, unreachable
	// once any of the cases above already returned.
	switch m.Mode {
	case ModeManual:
		return m.handleManualKey(msg)
	case ModeRenameTag:
		return m.handleRenameTagKey(msg)
	case ModeTryIt:
		return m.handleTryItKey(msg)
	default:
		return m.handleBrowseKey(msg)
	}
}

// rightPanelLayout replicates View()/renderRightPanel's layout math
// (leftWidth/rightWidth/inner/visibleHeight) — shared by scrollToResponse
// and clampRightScroll below so both compute scroll positions against the
// exact same content width/height the next render will actually use.
func (m Model) rightPanelLayout() (inner, visibleHeight int) {
	contentHeight := max(m.Height-8, 10)
	leftWidthPct := 30
	if m.LeftExpanded {
		leftWidthPct = 50
	}
	leftWidth := max(m.Width*leftWidthPct/100, 20)
	rightWidth := max(m.Width-leftWidth-2, 20)
	return max(rightWidth-4, 1), max(contentHeight-2, 1)
}

// rightPanelLineCount returns how many lines the right panel would render
// right now for clampRightScroll below — deliberately re-derived rather
// than cached, the same content-dependent recompute scrollToResponse
// already does for the same reason (cheap relative to a keypress).
func (m Model) rightPanelLineCount(inner int) int {
	if m.Mode == ModeTryIt {
		if m.TryIt.Endpoint == nil {
			return 0
		}
		lines, _ := m.renderTryItLines(m.TryIt.Endpoint, inner)
		return len(lines)
	}
	item := m.selectedItem()
	if item == nil {
		return 0
	}
	switch item.Type {
	case ItemTag:
		return len(m.renderTagLines(item.TagName, inner))
	case ItemEndpoint:
		return len(m.renderEndpointLines(item.Endpoint, m.ActivePanel == PanelRight, inner))
	}
	return 0
}

// clampRightScroll bounds RightScroll to the range render will actually
// use. Scroll keys increment/decrement it freely (matching the existing
// "clamp at render time" pattern already in renderRightPanel), but without
// this, repeatedly scrolling past the bottom (or, in try-it-out's
// BODY-focused case, past the top) accumulates a hidden offset that then
// silently eats an equal number of presses in the other direction before
// the view visibly moves — found via a user report ("press down 5
// times... have to press up 5 times up to begin scroll up").
func (m Model) clampRightScroll() int {
	inner, visibleHeight := m.rightPanelLayout()
	total := m.rightPanelLineCount(inner)
	return min(max(m.RightScroll, 0), max(total-visibleHeight, 0))
}

// scrollToResponse computes the right-panel scroll offset that brings the
// just-arrived response into view, replicating renderRightPanel's own
// layout math (leftWidth/rightWidth/inner) so Update can decide the target
// offset without View having any say in state. Deliberate improvement over
// TS, not a port of it: App.tsx just resets scroll to 0 (top of the
// endpoint doc) after executing, which doesn't actually reveal the response
// unless it happens to fit above the fold — found via a user report that a
// fresh response wasn't visible without manually scrolling down every time.
// Falls back to 0 for anything this can't compute a real answer for (no
// endpoint selected, manual builder — see the doc comment on why manual
// mode isn't covered yet).
func (m Model) scrollToResponse() int {
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint || m.Mode == ModeManual {
		return 0
	}

	inner, _ := m.rightPanelLayout()

	var lines []string
	if m.Mode == ModeTryIt {
		lines, _ = m.renderTryItLines(item.Endpoint, inner)
	} else {
		lines = m.renderEndpointLines(item.Endpoint, m.ActivePanel == PanelRight, inner)
	}
	responseLines := m.renderResponseBlock(inner)
	if len(responseLines) == 0 {
		return 0
	}
	responseStart := max(len(lines)-len(responseLines), 0)
	// renderResponseBlock always leads with a blank separator line before
	// the actual "RESPONSE ..." heading (visual spacing from whatever's
	// above it) — skip past it so the heading itself lands at the top of
	// the viewport, not one blank row below the top.
	if responseLines[0] == "" {
		responseStart++
	}
	// Deliberately not scrollToShow: that's minimal-motion "nudge just
	// enough to bring one line into view" tracking, meant for following a
	// cursor that moves a row at a time (try-it-out's param rows). Landing
	// the response's first line at the *bottom* edge of the viewport
	// technically satisfies "in view" but shows almost none of it — the
	// user wants the response section itself at the top, like a jump, not
	// a nudge.
	//
	// Also deliberately NOT clamped to max(len(lines)-visibleHeight, 0):
	// that clamp exists so a plain scroll never leaves dangling blank rows
	// at the bottom of the viewport, but applied here it pulls the start
	// backward whenever the response is short (found via a user
	// screenshot: a short response left several lines of the *preceding*
	// section's content bleeding in above "RESPONSE ..." instead of the
	// heading sitting flush under the box's top border). A short response
	// leaving blank rows at the bottom is the right tradeoff — the heading
	// must always be the first visible line. renderRightPanel's own
	// rendering already pads a too-short `visible` slice with blank lines,
	// so this is safe.
	return responseStart
}
