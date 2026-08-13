// Package tui is the root Bubbletea model: a single Update dispatches every
// key to a state transition, replacing the TS app's 9 independently
// duplicated keyboard hooks with one place input becomes state (see the
// tuiagger-dev-go skill for why this is the point of the rewrite, not
// incidental to it).
package tui

import (
	"maps"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
)

type ActivePanel int

const (
	PanelLeft ActivePanel = iota
	PanelRight
)

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

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

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
	}

	if m.ActivePanel == PanelLeft {
		return m.handleLeftPanelKey(key)
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
