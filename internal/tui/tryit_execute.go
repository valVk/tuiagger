package tui

import (
	"maps"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/bodyformat"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

// executeCmd saves the current overrides and runs the request in the
// background, matching App.tsx's executeCurrentEndpoint — including the
// in-progress (possibly hand-edited) body, scaffolded once on entering
// try-it-out (see enterTryIt) rather than regenerated on every execute.
//
// Shared by try-it-out's 'e' (uses the in-progress edit state) and browse
// mode's quick-execute 'e' (uses whatever was last saved to disk) — see
// quickExecuteCmd.
func (m Model) executeCmd(ep *openapi.ParsedEndpoint) tea.Cmd {
	tryIt := m.TryIt
	contentType := rawContentType(ep, tryIt.ContentTypeTab)
	return m.executeWithOverride(ep, tryIt.ParamValues, tryIt.DisabledParams, tryIt.CustomParams, tryIt.OverridePath, tryIt.OverrideMethod, tryIt.Body, contentType)
}

// quickExecuteCmd runs a request from browse mode using the endpoint's
// saved override (if any), without entering try-it-out — matches
// CLAUDE.md's "e - Quick execute (reuses saved overrides)".
func (m Model) quickExecuteCmd(ep *openapi.ParsedEndpoint) tea.Cmd {
	paramValues := map[string]string{}
	disabled := map[string]bool{}
	var customParams []storage.CustomParameter
	overridePath, overrideMethod, body, contentType := "", "", "", ""
	if m.Store != nil {
		if override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); override != nil {
			paramValues = override.Params
			for _, d := range override.DisabledParams {
				disabled[d] = true
			}
			customParams = override.CustomParams
			overridePath = override.OverridePath
			overrideMethod = override.OverrideMethod
			body = override.Body
			contentType = override.ContentType
		}
	}
	return m.executeWithOverride(ep, paramValues, disabled, customParams, overridePath, overrideMethod, body, contentType)
}

func (m Model) executeWithOverride(ep *openapi.ParsedEndpoint, values map[string]string, disabledSet map[string]bool, customParams []storage.CustomParameter, overridePath, overrideMethod, body, contentType string) tea.Cmd {
	method := ep.Method
	paramValues := maps.Clone(values)
	disabled := disabledSlice(disabledSet)
	specParams := ep.Operation.Parameters
	requestBody := ep.Operation.RequestBody
	security := ep.Operation.Security
	if len(security) == 0 {
		security = m.Spec.Spec.Security
	}
	var securitySchemes map[string]openapi.SecurityScheme
	if m.Spec.Spec.Components != nil {
		securitySchemes = m.Spec.Spec.Components.SecuritySchemes
	}
	servers := m.Spec.Spec.Servers
	selectedServer := m.SelectedServer
	client := m.HTTPClient
	store := m.Store

	return func() tea.Msg {
		if store != nil {
			override := storage.EndpointOverride{
				Params:         paramValues,
				CustomParams:   customParams,
				DisabledParams: disabled,
				Body:           body,
				ContentType:    contentType,
				OverridePath:   overridePath,
				OverrideMethod: overrideMethod,
			}
			// Matches exitTryIt's isEmptyOverride check — without it, every
			// execute unconditionally persisted an override (matching
			// App.tsx's own unconditional saveOverride() call before
			// executing), so even a browse-mode quick-execute with nothing
			// ever touched (no try-it-out session, no saved override to
			// begin with) would mark the endpoint "~"/"*saved params" from
			// the request alone. Found via a user report ("if I execute
			// request even if I did not change anything... it marked as
			// overridden"). Deliberate divergence from TS, not a port of
			// it, same reasoning as exitTryIt's fix.
			if isEmptyOverride(override) {
				store.DeleteEndpointOverride(string(method), ep.Path)
			} else {
				store.SaveEndpointOverride(string(method), ep.Path, override)
			}
		}

		collector := &request.ParameterCollector{
			SpecParams:      specParams,
			CustomParams:    customParams,
			DisabledParams:  disabled,
			ParameterValues: paramValues,
		}

		path := ep.Path
		if overridePath != "" {
			path = overridePath
		}
		effectiveMethod := string(method)
		if overrideMethod != "" {
			effectiveMethod = overrideMethod
		}

		// effectiveContentType resolves contentType's raw, possibly-empty
		// value (see rawContentType's doc comment) to an actual encodable
		// type for scaffolding/sending, without changing what got
		// persisted into the override above.
		effectiveContentType := contentType
		if effectiveContentType == "" {
			effectiveContentType = selectedContentType(ep, 0)
		}

		// Fallback for callers that never went through enterTryIt (browse
		// mode's quick-execute 'e' on an endpoint with no saved override
		// yet) — same realistic-data scaffold, just generated here instead
		// of once up front.
		if body == "" && requestBody != nil && isWriteMethod(effectiveMethod) {
			if schema := selectedSchema(requestBody.Content, effectiveContentType); schema != nil {
				body = bodyformat.Encode(effectiveContentType, openapi.ScaffoldFakeBody(schema))
			}
		}

		spec := buildRequestSpec(servers, selectedServer, store, collector, effectiveMethod, path, body, effectiveContentType, security, securitySchemes)

		if client == nil {
			return responseMsg{response: &request.Response{Error: "no HTTP client configured"}}
		}
		resp, curl := request.Execute(client, spec)
		return responseMsg{response: resp, curl: curl}
	}
}
