// Package tui is the root Bubbletea model: a single Update dispatches every
// key to a state transition, replacing the TS app's 9 independently
// duplicated keyboard hooks with one place input becomes state (see the
// tuiagger-dev-go skill for why this is the point of the rewrite, not
// incidental to it).
package tui

import (
	"maps"
	"sort"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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

// tryItState holds everything scoped to one endpoint's try-it-out session.
// It's reset whenever the left-panel selection changes, matching the TS
// app's selectedItem-change effect in App.tsx.
type tryItState struct {
	ParamValues    map[string]string
	DisabledParams map[string]bool
	OverridePath   string
	OverrideMethod string

	ParamCursor  int
	ParamEditing bool
	ValueInput   textinput.Model

	// CustomParams backs the PARAMETERS table's always-present "[ + ]" row
	// — matching ParametersSection.tsx, which appends an `addNew` row
	// unconditionally in try-it-out mode regardless of whether the
	// endpoint declares any spec parameters, so the section (and its
	// hints) never disappears and custom query/path params can always be
	// added. NameInput/ParamField back the name half of custom/add-new row
	// editing (ValueInput above is shared with spec-param editing, one at
	// a time since only one row can be selected).
	CustomParams []storage.CustomParameter
	ParamField   string // "name" | "value", used while editing a custom/add-new row
	NameInput    textinput.Model
	NewParamIn   string // in-progress type ("query"/"path") for the add-new row

	// HeadersFocused/HeaderCursor/HeaderEditing back a second, independent
	// table above PARAMETERS for CustomParams entries with In=="header" —
	// matches HeadersSection.tsx, a wholly separate NAME/VALUE-only editor
	// (no TYPE/DESCRIPTION columns, no enum cycling). Editing state
	// (NameInput/ValueInput/ParamField) is shared with the PARAMETERS
	// table since only one of the two can ever be mid-edit at once.
	HeadersFocused bool
	HeaderCursor   int
	HeaderEditing  bool

	EditingPath bool
	PathInput   textinput.Model

	// Body mirrors useRightPanelKeyboard.ts's bodyTabFocused/editingBody:
	// 'j' off the last parameter row (or 'k' back) moves focus onto the
	// BODY section; 'i' there edits it (auto-scaffolding a placeholder if
	// still empty, matching the TS quirk where that path uses
	// scaffoldPlaceholder rather than the faker-driven scaffoldBody
	// enterTryIt already ran once on entry).
	Body        string
	BodyFocused bool
	EditingBody bool
	BodyInput   textarea.Model

	ShowResetConfirm bool
}

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

	Response *request.Response
	Curl     string
	Loading  bool
	Viewer   responseViewer

	HTTPClient request.HTTPClient
	Store      *storage.Store

	LeftExpanded bool // '[' toggles 30% <-> 50% left panel width, matching App.tsx

	ShowHelp   bool
	HelpScroll int

	ShowInfo     bool
	InfoSection  infoSection
	ServerCursor int
	AuthCursor   int
	EnvCursor    int
	Auth         authEditState
	Env          envEditState

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
		m.Auth.Editing || m.Env.InsertingVar || m.Env.AddingEnv
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
		m.Response = msg.response
		m.Curl = msg.curl
		if msg.response != nil {
			m.Viewer = newResponseViewer(msg.response.Body)
		}
		m.RightScroll = m.scrollToResponse()
		return m, nil
	case yankExpiredMsg:
		if msg.curl {
			m.Viewer.YankedCurl = false
		} else {
			m.Viewer.Yanked = false
		}
		return m, nil
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
	m.Response = nil
	m.Curl = ""
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

	if m.Mode == ModeManual {
		return m.handleManualKey(msg)
	}
	if m.Mode == ModeRenameTag {
		return m.handleRenameTagKey(msg)
	}
	if m.Mode == ModeTryIt {
		return m.handleTryItKey(msg)
	}

	// Matches useAppKeyboard.ts's tagDeleteConfirm intercept: takes over
	// input entirely (browse mode only) until y/n/Esc resolves it.
	if m.TagDeleteConfirm != "" {
		switch key {
		case "y":
			if m.Store != nil {
				m.Store.DeleteCustomTag(m.TagDeleteConfirm)
			}
			m.TagDeleteConfirm = ""
			return m.refreshSavedRequests(), nil
		case "n", "esc":
			m.TagDeleteConfirm = ""
			return m, nil
		}
		return m, nil
	}

	// R/D on a custom tag, E/D on a saved request, and 'e' quick-execute all
	// work regardless of which panel is active, matching
	// useAppKeyboard.ts's browse-mode handler (checked ahead of h/l/j/k
	// navigation) — 'e' in particular is bound in a useInput with
	// isActive: mode === 'browse' only, no panel check, so it must work
	// from the left panel too, not just after pressing 'l' first.
	if item := m.selectedItem(); item != nil {
		switch {
		case key == "R" && item.Type == ItemTag && m.isCustomTag(item.TagName):
			return m.enterRenameTag(item.TagName), nil
		case key == "D" && item.Type == ItemTag && m.isCustomTag(item.TagName):
			m.TagDeleteConfirm = item.TagName
			return m, nil
		case key == "E" && item.Type == ItemSavedRequest:
			return m.enterManualEdit(item.SavedRequest), nil
		case key == "D" && item.Type == ItemSavedRequest:
			if m.Store != nil {
				m.Store.DeleteSavedRequest(item.SavedRequest.ID)
			}
			return m.refreshSavedRequests(), nil
		case key == "e" && item.Type == ItemEndpoint:
			cmd := m.quickExecuteCmd(item.Endpoint)
			m.Loading = true
			return m, cmd
		case key == "e" && item.Type == ItemSavedRequest:
			cmd := m.savedRequestExecuteCmd(item.SavedRequest)
			m.Loading = true
			return m, cmd
		}
	}

	switch key {
	case "ctrl+r":
		m.SpecLoading = true
		return m, m.reloadCmd()
	case "?":
		m.ShowHelp = true
		m.HelpScroll = 0
		return m, nil
	case "i":
		return m.enterInfo(), nil
	case "h", "left":
		m.ActivePanel = PanelLeft
		return m, nil
	case "l", "right":
		m.ActivePanel = PanelRight
		return m, nil
	case "[":
		m.LeftExpanded = !m.LeftExpanded
		return m, nil
	case "t":
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			return m.enterTryIt(), nil
		}
		return m, nil
	case "m":
		return m.enterManualNew(), nil
	}

	if m.ActivePanel == PanelLeft {
		return m.handleLeftPanelKey(key)
	}

	// Response-viewer keys (J/K/G/v/y/Esc/\) take priority over generic
	// scroll (j/k/g) when a response is present — distinct key casing means
	// both coexist without a mode flag, matching the TS app. Lowercase 'g'
	// is bound by both ResponseViewer.tsx (jump response cursor to top) and
	// usePanelNavigation.ts (reset panel scroll) as two independent Ink
	// input handlers that both fire on the same keypress — replicated here
	// by routing to the viewer and then still falling through to the
	// generic handler below, rather than picking one.
	if m.Response != nil {
		switch key {
		case "J", "K", "G", "v", "y", "esc", `\`:
			var cmd tea.Cmd
			m.Viewer, cmd = m.Viewer.handleKey(key)
			return m, cmd
		case "C":
			// Go-only addition, not a TS port — yanks the generated curl
			// command to the clipboard, independent of tab/selection state.
			if m.Curl != "" {
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.yankCurl(m.Curl)
				return m, cmd
			}
		case "j", "k":
			// While actively visual-selecting, lowercase j/k also drive the
			// response cursor instead of the outer panel scroll — found via
			// a user report: without this, pressing 'v' then reaching for
			// the muscle-memory 'j'/'k' (rather than the shifted 'J'/'K'
			// the hint text actually asks for) just scrolls the panel out
			// from under the selection, which looks exactly like "can't
			// expand the selection, it just moves the viewport." Only
			// active during a selection — outside of one, lowercase j/k
			// keeps its normal job of reaching content that might be
			// scrolled out of view above the response section.
			if m.Viewer.Selecting {
				viewerKey := "J"
				if key == "k" {
					viewerKey = "K"
				}
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.handleKey(viewerKey)
				return m, cmd
			}
		case "g":
			m.Viewer, _ = m.Viewer.handleKey(key)
		}
	}

	return m.handleRightPanelKey(key)
}

func (m Model) safeLeftIndex() int {
	if len(m.FlatList) == 0 {
		return 0
	}
	if m.LeftIndex >= len(m.FlatList) {
		return len(m.FlatList) - 1
	}
	return m.LeftIndex
}

func (m Model) selectedItem() *FlatListItem {
	if len(m.FlatList) == 0 {
		return nil
	}
	return &m.FlatList[m.safeLeftIndex()]
}

func (m Model) handleLeftPanelKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.LeftIndex = min(m.safeLeftIndex()+1, len(m.FlatList)-1)
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "k", "up":
		m.LeftIndex = max(m.safeLeftIndex()-1, 0)
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "g":
		m.LeftIndex = 0
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "G":
		m.LeftIndex = max(0, len(m.FlatList)-1)
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "enter":
		if item := m.selectedItem(); item != nil && item.Type == ItemTag {
			m.toggleTag(item.TagName)
		}
		return m, nil
	case "c":
		m.ExpandedTags = make(map[string]bool)
		m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
		m.LeftIndex = 0
		return m, nil
	case "x":
		m.ExpandedTags = allExpanded(m.AllTags)
		m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
		return m, nil
	}
	return m, nil
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

	leftWidthPct := 30
	if m.LeftExpanded {
		leftWidthPct = 50
	}
	leftWidth := max(m.Width*leftWidthPct/100, 20)
	rightWidth := max(m.Width-leftWidth-2, 20)
	inner := max(rightWidth-4, 1)

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

func (m Model) handleRightPanelKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.RightScroll++
		return m, nil
	case "k", "up":
		m.RightScroll = max(0, m.RightScroll-1)
		return m, nil
	case "g":
		m.RightScroll = 0
		return m, nil
	case "/":
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			codes := sortedResponseCodes(item.Endpoint)
			if len(codes) > 0 {
				m.ResponseTab = (m.ResponseTab + 1) % len(codes)
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) toggleTag(tagName string) {
	next := make(map[string]bool, len(m.ExpandedTags))
	maps.Copy(next, m.ExpandedTags)
	next[tagName] = !next[tagName]
	m.ExpandedTags = next
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
}

func allExpanded(tags []string) map[string]bool {
	m := make(map[string]bool, len(tags))
	for _, t := range tags {
		m[t] = true
	}
	return m
}

func sortedResponseCodes(ep *openapi.ParsedEndpoint) []string {
	var codes []string
	for _, r := range ep.Operation.Responses {
		codes = append(codes, r.Status)
	}
	sort.Strings(codes)
	return codes
}
