// Package tui is the root Bubbletea model: a single Update dispatches every
// key to a state transition, replacing the TS app's 9 independently
// duplicated keyboard hooks with one place input becomes state (see the
// tuiagger-dev-go skill for why this is the point of the rewrite, not
// incidental to it).
package tui

import (
	"maps"
	"sort"

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

	EditingPath bool
	PathInput   textinput.Model

	ShowResetConfirm bool
}

type Model struct {
	Spec           *openapi.ParsedSpec
	CollectionName string
	SelectedServer int

	AllTags        []string
	EndpointsByTag map[string][]openapi.ParsedEndpoint
	ExpandedTags   map[string]bool
	FlatList       []FlatListItem

	ActivePanel ActivePanel
	LeftIndex   int
	RightScroll int
	ResponseTab int // index into the selected endpoint's sorted response codes

	Mode  Mode
	TryIt tryItState

	Response *request.Response
	Curl     string
	Loading  bool
	Viewer   responseViewer

	HTTPClient request.HTTPClient
	Store      *storage.Store

	Width, Height int
	Quitting      bool
}

// New builds the initial Model with every tag expanded, matching
// usePanelNavigation.ts's initial state (`useState(() => new Set(allTags))`).
func New(spec *openapi.ParsedSpec, collectionName string) Model {
	endpointsByTag := openapi.GetEndpointsByTag(spec.Endpoints)
	expanded := make(map[string]bool, len(spec.Tags))
	for _, t := range spec.Tags {
		expanded[t] = true
	}
	m := Model{
		Spec:           spec,
		CollectionName: collectionName,
		AllTags:        spec.Tags,
		EndpointsByTag: endpointsByTag,
		ExpandedTags:   expanded,
		ActivePanel:    PanelLeft,
	}
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.ExpandedTags)
	return m
}

// WithServices injects the HTTP client and persistence store — split from
// New so tests can build a Model without a real store/client when neither
// is exercised.
func (m Model) WithServices(client request.HTTPClient, store *storage.Store) Model {
	m.HTTPClient = client
	m.Store = store
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// responseMsg carries a completed request's result back into Update — the
// tea.Cmd pattern that keeps HTTP execution out of Update itself.
type responseMsg struct {
	response *request.Response
	curl     string
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
		m.RightScroll = 0
		if msg.response != nil {
			m.Viewer = newResponseViewer(msg.response.Body)
		}
		return m, nil
	case yankExpiredMsg:
		m.Viewer.Yanked = false
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.Mode == ModeTryIt {
		return m.handleTryItKey(msg)
	}

	switch key {
	case "q":
		m.Quitting = true
		return m, tea.Quit
	case "h", "left":
		m.ActivePanel = PanelLeft
		return m, nil
	case "l", "right":
		m.ActivePanel = PanelRight
		return m, nil
	case "t":
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			return m.enterTryIt(), nil
		}
		return m, nil
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
		m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.ExpandedTags)
		m.LeftIndex = 0
		return m, nil
	case "x":
		m.ExpandedTags = allExpanded(m.AllTags)
		m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.ExpandedTags)
		return m, nil
	}
	return m, nil
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
	case "e":
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			cmd := m.quickExecuteCmd(item.Endpoint)
			m.Loading = true
			return m, cmd
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
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.ExpandedTags)
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
