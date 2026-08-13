package tui

import (
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

type ItemType int

const (
	ItemTag ItemType = iota
	ItemEndpoint
	ItemSavedRequest
)

// FlatListItem is one row of the left panel's flattened tag/endpoint/saved-
// request tree. Matches usePanelNavigation.ts's FlatListItem.
type FlatListItem struct {
	Type         ItemType
	ID           string
	TagName      string
	Endpoint     *openapi.ParsedEndpoint
	SavedRequest *storage.SavedRequest
}

// buildFlatList expands allTags into tag rows, interleaving each expanded
// tag's endpoints and saved requests directly beneath it — matches
// usePanelNavigation.ts's buildFlatList.
func buildFlatList(allTags []string, endpointsByTag map[string][]openapi.ParsedEndpoint, savedRequestsByTag map[string][]storage.SavedRequest, expandedTags map[string]bool) []FlatListItem {
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
		for i := range savedRequestsByTag[tag] {
			sr := savedRequestsByTag[tag][i]
			items = append(items, FlatListItem{
				Type:         ItemSavedRequest,
				ID:           "saved-" + sr.ID,
				TagName:      tag,
				SavedRequest: &sr,
			})
		}
	}
	return items
}
