package tui

import "github.com/valVK/tuiagger/internal/openapi"

type ItemType int

const (
	ItemTag ItemType = iota
	ItemEndpoint
)

// FlatListItem is one row of the left panel's flattened tag/endpoint tree.
// Matches usePanelNavigation.ts's FlatListItem, minus the 'savedRequest'
// variant — saved requests are wired in Phase 4 (manual request builder).
type FlatListItem struct {
	Type     ItemType
	ID       string
	TagName  string
	Endpoint *openapi.ParsedEndpoint
}

// buildFlatList expands allTags into tag rows, interleaving each expanded
// tag's endpoints directly beneath it — matches usePanelNavigation.ts's
// buildFlatList.
func buildFlatList(allTags []string, endpointsByTag map[string][]openapi.ParsedEndpoint, expandedTags map[string]bool) []FlatListItem {
	var items []FlatListItem
	for _, tag := range allTags {
		items = append(items, FlatListItem{Type: ItemTag, ID: "tag-" + tag, TagName: tag})
		if !expandedTags[tag] {
			continue
		}
		for i := range endpointsByTag[tag] {
			ep := endpointsByTag[tag][i]
			items = append(items, FlatListItem{
				Type:     ItemEndpoint,
				ID:       "endpoint-" + string(ep.Method) + "-" + ep.Path,
				TagName:  tag,
				Endpoint: &ep,
			})
		}
	}
	return items
}
