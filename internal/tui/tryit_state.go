package tui

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// tryItState holds everything scoped to one endpoint's try-it-out session.
// It's reset whenever the left-panel selection changes, matching the TS
// app's selectedItem-change effect in App.tsx.
type tryItState struct {
	// Endpoint is captured once by enterTryIt — the component's own copy
	// of which endpoint this session is for, so Update (handleTryItKey)
	// doesn't need to re-derive it from the root's left-panel selection on
	// every keystroke. Safe: left-panel navigation is unreachable while
	// Mode == ModeTryIt, so the selection can't change mid-session.
	Endpoint *openapi.ParsedEndpoint

	ParamValues    map[string]string
	DisabledParams map[string]bool
	OverridePath   string
	OverrideMethod string

	ParamCursor  int
	ParamEditing bool

	// CustomParams backs the PARAMETERS table's always-present "[ + ]" row
	// — matching ParametersSection.tsx, which appends an `addNew` row
	// unconditionally in try-it-out mode regardless of whether the
	// endpoint declares any spec parameters, so the section (and its
	// hints) never disappears and custom query/path params can always be
	// added.
	CustomParams []storage.CustomParameter
	NewParamIn   string // in-progress type ("query"/"path") for the add-new row

	// HeaderTable backs a second, independent table above PARAMETERS for
	// CustomParams entries with In=="header" — matches HeadersSection.tsx.
	// Also supplies the ParamField/NameInput/ValueInput widgets used by
	// PARAMETERS' own custom/add-new row (and spec-param) editing, since
	// only one of the two tables can ever be mid-edit at once.
	HeaderTable headerTableState

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
	// BodyScrollFloor is RightScroll's value at the moment BODY was
	// focused — 'k'/'up' while focused scrolls back up as long as
	// RightScroll is still above this floor (undoing 'j' presses that
	// scrolled past the box), and only unfocuses back to PARAMETERS once
	// it's reached, so scrolling up mirrors scrolling down instead of
	// jumping straight back to PARAMETERS from wherever the user scrolled
	// to.
	BodyScrollFloor int

	ShowResetConfirm bool
}

// httpMethods is the cycle order for the 'm' key in try-it-out mode,
// matching useRightPanelKeyboard.ts's HTTP_METHODS.
var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

// enterTryIt switches to try-it-out mode for the selected endpoint, loading
// any previously saved override — matches App.tsx's selectedItem-change
// effect (override load) plus useAppKeyboard.ts's 't' handler (mode switch).
func (m Model) enterTryIt() Model {
	item := m.selectedItem()
	if item == nil || item.Type != ItemEndpoint {
		return m
	}
	ep := item.Endpoint

	state := tryItState{
		Endpoint:       ep,
		ParamValues:    map[string]string{},
		DisabledParams: map[string]bool{},
	}
	if m.Store != nil {
		if override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); override != nil {
			state.ParamValues = maps.Clone(override.Params)
			for _, d := range override.DisabledParams {
				state.DisabledParams[d] = true
			}
			state.OverridePath = override.OverridePath
			state.OverrideMethod = override.OverrideMethod
			state.Body = override.Body
			state.CustomParams = slices.Clone(override.CustomParams)
		}
	}
	// Matches useAppKeyboard.ts's 't' handler: auto-fill the body with
	// realistic fake data (scaffoldBody, not the placeholder-style
	// scaffoldPlaceholder) the moment try-it-out opens, but only if there's
	// nothing there already (a saved override's body always wins).
	if state.Body == "" && ep.Operation.RequestBody != nil {
		if schema := applicationJSONSchema(ep.Operation.RequestBody.Content); schema != nil {
			if scaffolded := openapi.ScaffoldFakeBody(schema); scaffolded != nil {
				state.Body = jsonPretty(scaffolded)
			}
		}
	}
	state.HeaderTable.ValueInput = textinput.New()
	state.HeaderTable.NameInput = textinput.New()
	state.NewParamIn = "query"
	state.PathInput = textinput.New()
	state.BodyInput = newBodyTextarea()

	m.Mode = ModeTryIt
	m.ActivePanel = PanelRight
	m.TryIt = state
	// Matches useAppKeyboard.ts's 't' handler: panelNav.setRightScroll(0).
	// Without this, whatever scroll offset was left over from browsing the
	// endpoint's docs (or a previous response) carries into try-it-out,
	// which can land the view mid-way through the (now longer, since the
	// body auto-scaffolds) content instead of at the top.
	m.RightScroll = 0
	return m
}

func newBodyTextarea() textarea.Model {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.SetHeight(10)
	// bubbles/textarea's default FocusedStyle.CursorLine paints a
	// full-width background behind whichever line the cursor sits on —
	// since most of a JSON body's lines are short, that background
	// extends across mostly blank padding, reading as a stray highlighted
	// block rather than a cursor indicator (found via a user report: "why
	// body have extra white line"). The terminal's own cursor already
	// shows position; this box doesn't need a second, wider indicator.
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	// bubbles/textarea also prints a "┃ " prompt at the start of every
	// line by default — redundant here, since the BODY box's own rounded
	// border already delineates it, and it read as a second, unwanted
	// vertical line running down the left edge (found via a user report
	// clarifying the "extra line" was this, not the cursor-line
	// background above).
	ta.Prompt = ""
	return ta
}

// setBodyValue sets a body textarea's content, sizes it to match, and
// positions the cursor at the very top.
//
// Height: the textarea's own SetHeight is a fixed row count (the
// constructor's SetHeight(10) padded shorter content with blank rows,
// same bug class the removed Prompt/CursorLine styling was), but the
// read-only preview box sizes itself to exactly its content's line count
// with no padding — so switching into edit mode visibly grew or shrank
// the box instead of it staying put. Sized here to match exactly (found
// via a user report: "textarea in edit mode should be the same as
// preview height").
//
// Cursor: SetValue alone leaves the cursor at the end of the inserted
// text — the bottom of a multi-line JSON body — so entering edit mode
// always opened scrolled to the bottom instead of the top (found via a
// user report: "move cursor top").
func setBodyValue(ta textarea.Model, value string) textarea.Model {
	ta.SetValue(value)
	ta.SetHeight(len(strings.Split(value, "\n")))
	for range ta.LineCount() {
		ta.CursorUp()
	}
	ta.CursorStart()
	return ta
}

// applicationJSONSchema looks up the "application/json" media type
// specifically, matching useAppKeyboard.ts's body-scaffold trigger — unlike
// firstSchema (used for read-only docs display, where any declared content
// type is a reasonable thing to show), scaffolding a body a user will
// actually send shouldn't depend on Go's nondeterministic map iteration
// order picking, say, "multipart/form-data" instead.
func applicationJSONSchema(content map[string]openapi.MediaType) *openapi.Schema {
	if mt, ok := content["application/json"]; ok {
		return mt.Schema
	}
	return nil
}

// exitTryIt persists the in-progress edit (params, disabled set, body,
// path/method overrides) before returning to browse mode, matching
// App.tsx's Esc handler — TS saves on Esc exit, not just on execute, so
// scaffolding or hand-editing a body and then backing out without pressing
// 'e' doesn't lose the work.
//
// Deliberate divergence from TS, not a port of it: TS's Esc handler calls
// saveOverride() unconditionally, even when every field is empty — which
// resurrects an empty-but-present override (and its "*saved params"/"~"
// indicators) right after 'r' (reset) clears everything and the user exits
// normally afterward, making a reset look like it didn't take. Deleting any
// existing override instead of writing an empty one when there's nothing
// left to save closes that gap without changing the "save on exit" behavior
// for every other case (an untouched write-method endpoint's auto-scaffolded
// Body is never empty, so it still gets saved as before).
func (m Model) exitTryIt() Model {
	if ep := m.TryIt.Endpoint; ep != nil && m.Store != nil {
		override := storage.EndpointOverride{
			Params:         m.TryIt.ParamValues,
			CustomParams:   m.TryIt.CustomParams,
			DisabledParams: disabledSlice(m.TryIt.DisabledParams),
			Body:           m.TryIt.Body,
			OverridePath:   m.TryIt.OverridePath,
			OverrideMethod: m.TryIt.OverrideMethod,
		}
		if isEmptyOverride(override) {
			m.Store.DeleteEndpointOverride(string(ep.Method), ep.Path)
		} else {
			m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, override)
		}
	}
	m.Mode = ModeBrowse
	return m
}

// isEmptyOverride reports whether an override has nothing worth persisting
// — every field at its zero value. See exitTryIt's doc comment for why this
// matters (a reset followed by a normal exit must not resurrect an
// empty-but-present override).
func isEmptyOverride(o storage.EndpointOverride) bool {
	return len(o.Params) == 0 && len(o.CustomParams) == 0 && len(o.DisabledParams) == 0 &&
		o.Body == "" && o.OverridePath == "" && o.OverrideMethod == ""
}

// tryItTotalRows matches ParametersSection.tsx's rows array: required specs,
// then optional specs (already what sortedParameters returns), then custom
// params, then one always-present "addNew" row — present even with zero
// spec parameters, which is why the section (and its hints) must never be
// skipped just because an endpoint like a POST with only a body has no
// query/path parameters of its own.
func tryItTotalRows(params []openapi.Parameter, custom []storage.CustomParameter) int {
	return len(params) + len(custom) + 1
}

// splitCustomParams separates HeadersSection.tsx's header-typed entries from
// everything ParametersSection.tsx shows (query/path) — same underlying
// list, filtered into two independent views, matching TS's
// nonHeaderParams/headerParams split.
func splitCustomParams(all []storage.CustomParameter) (headers, others []storage.CustomParameter) {
	for _, p := range all {
		if p.In == "header" {
			headers = append(headers, p)
		} else {
			others = append(others, p)
		}
	}
	return headers, others
}

// mergeCustomParams recombines a modified headers or non-header slice with
// the untouched other group, matching TS's
// `[...nonHeaderParams, ...updated]` / `[...updated, ...headerParams]`
// recombination on every HeadersSection/ParametersSection change.
func mergeCustomParams(headers, others []storage.CustomParameter) []storage.CustomParameter {
	merged := make([]storage.CustomParameter, 0, len(headers)+len(others))
	merged = append(merged, headers...)
	merged = append(merged, others...)
	return merged
}

func (m Model) resetOverride(ep *openapi.ParsedEndpoint) Model {
	if m.Store != nil {
		m.Store.DeleteEndpointOverride(string(ep.Method), ep.Path)
	}
	m.TryIt.ParamValues = map[string]string{}
	m.TryIt.DisabledParams = map[string]bool{}
	m.TryIt.CustomParams = nil
	m.TryIt.ParamCursor = 0
	m.TryIt.OverridePath = ""
	m.TryIt.OverrideMethod = ""
	m.TryIt.Body = ""
	m.TryIt.BodyFocused = false
	m.TryIt.ShowResetConfirm = false
	return m
}

func enumValues(p openapi.Parameter) []string {
	if p.Schema == nil {
		return nil
	}
	out := make([]string, 0, len(p.Schema.Enum))
	for _, e := range p.Schema.Enum {
		out = append(out, toStr(e))
	}
	return out
}

// toStr matches JS's `.toString()` coercion used throughout the TS app for
// enum/example/default values: strings pass through, other JSON scalars are
// stringified, nil becomes "".
func toStr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH":
		return true
	}
	return false
}

func cycleQueryPath(in string) string {
	if in == "query" {
		return "path"
	}
	return "query"
}

func jsonPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func disabledSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
