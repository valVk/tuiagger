package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/valVK/tuiagger/internal/storage"
)

// manualState holds one manual-request-builder session — either a fresh
// draft ('m') or an in-progress edit of a saved request ('E'), matching
// App.tsx's ManualState. Query/path and header custom params live in one
// Params list (distinguished by .In) but are split into two independent
// views by splitCustomParams, matching TS's own
// nonHeaderParams/headerParams filtering of one customParams array.
//
// Focus follows useManualPanelKeyboard.ts's actual model — confirmed by
// reading it, not assumed — which is the *same* up/down-boundary-crossing
// scheme useRightPanelKeyboard.ts uses for try-it-out (HeadersFocused/
// BodyFocused booleans, no Tab-driven section cycling at all). An earlier
// version of this file used a homegrown Tab-cycling manualFocus enum that
// didn't exist in TS; replaced to match.
type manualState struct {
	Path           string
	Method         string
	Params         []storage.CustomParameter
	Body           string
	EditingRequest *storage.SavedRequest

	EditingPath bool
	PathInput   textinput.Model

	// HeaderTable backs the HEADERS table (In=="header" params only) —
	// mirrors tryItState's field exactly, see headertable.go.
	HeaderTable headerTableState

	// ParamCursor/ParamEditing address the PARAMETERS table (In!="header"
	// params) — the *default* focus whenever nothing else claims it, same
	// as try-it-out. Name/value editing widgets live on HeaderTable, shared
	// with HEADERS editing since only one table is ever mid-edit at once.
	ParamCursor  int // 0..len(non-header Params); == len is the "add new" row
	ParamEditing bool
	ParamAddNew  bool   // editing the add-new row rather than an existing one
	NewParamIn   string // in-progress type ("query"/"path") for the PARAMETERS add-new row

	BodyFocused bool
	EditingBody bool
	BodyInput   textarea.Model
	// BodyScrollFloor mirrors tryItState's field of the same name (see
	// its doc comment) — RightScroll's value at the moment BODY was
	// focused, so 'k' can scroll back up through whatever 'j' scrolled
	// past before unfocusing back to PARAMETERS.
	BodyScrollFloor int
	// ContentTypeTab indexes manualContentTypes below — cycled by 'c'
	// while BODY is focused, same key/shape as tryItState.ContentTypeTab.
	// Unlike try-it-out, the manual builder has no spec/schema to
	// enumerate declared content types from (a hand-built request can be
	// anything), so this is a fixed 3-way toggle instead of a
	// sortedContentTypes(...) lookup.
	ContentTypeTab int

	ShowSaveDialog bool
	SaveDialog     saveDialogState
}

func newManualState() manualState {
	return manualState{
		Method:     "GET",
		NewParamIn: "query",
		PathInput:  textinput.New(),
		HeaderTable: headerTableState{
			NameInput:  textinput.New(),
			ValueInput: textinput.New(),
		},
		BodyInput: newBodyTextarea(),
	}
}

// enterManualNew starts a blank manual request draft — matches
// useAppKeyboard.ts's 'm' handler.
func (m Model) enterManualNew() Model {
	m.Manual = newManualState()
	m.Mode = ModeManual
	m.ActivePanel = PanelRight
	m.Viewer = responseViewer{}
	// Matches enterTryIt: without this, whatever scroll offset was left
	// over from browsing (or a previous manual session) carries into the
	// new draft instead of starting at the top.
	m.RightScroll = 0
	return m
}

// enterManualEdit loads an existing saved request into the builder for
// editing — matches useAppKeyboard.ts's 'E' handler.
func (m Model) enterManualEdit(sr *storage.SavedRequest) Model {
	state := newManualState()
	state.Path = sr.Path
	state.Method = sr.Method
	state.Body = sr.Body
	state.ContentTypeTab = indexOfContentType(sr.BodyType)
	state.EditingRequest = sr
	for _, p := range sr.QueryParams {
		state.Params = append(state.Params, storage.CustomParameter{ID: p.ID, Name: p.Key, Value: p.Value, In: "query", Enabled: p.Enabled})
	}
	for _, h := range sr.Headers {
		state.Params = append(state.Params, storage.CustomParameter{ID: h.ID, Name: h.Key, Value: h.Value, In: "header", Enabled: h.Enabled})
	}
	m.Manual = state
	m.Mode = ModeManual
	m.ActivePanel = PanelRight
	m.Viewer = responseViewer{}
	m.RightScroll = 0
	return m
}

func (m Model) exitManual() Model {
	m.Mode = ModeBrowse
	m.Viewer = responseViewer{}
	return m
}

func indexOfMethod(method string) int {
	for i, m := range httpMethods {
		if strings.EqualFold(m, method) {
			return i
		}
	}
	return 0
}

// manualContentTypes is the manual builder's fixed content-type cycle —
// see manualState.ContentTypeTab's doc comment for why this is a static
// list rather than sortedContentTypes(...) over a spec.
var manualContentTypes = []string{"application/json", "application/x-www-form-urlencoded", "application/xml"}

// manualSelectedContentType resolves a ContentTypeTab index to its type
// string, wrapping out-of-range values the same way selectedContentType
// does for try-it-out.
func manualSelectedContentType(tab int) string {
	idx := ((tab % len(manualContentTypes)) + len(manualContentTypes)) % len(manualContentTypes)
	return manualContentTypes[idx]
}

// indexOfContentType maps a persisted content-type string (SavedRequest's
// repurposed BodyType field) back to its tab index — mirrors enterTryIt's
// override.ContentType restore. Old saved requests with the pre-refactor
// literal "json" value (see saveManualRequest's prior hardcoded writes)
// don't match any entry here and fall back to index 0
// (application/json), preserving their existing behavior.
func indexOfContentType(contentType string) int {
	for i, ct := range manualContentTypes {
		if ct == contentType {
			return i
		}
	}
	return 0
}
