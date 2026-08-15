package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

// manualExecuteCmd builds and runs the in-progress manual draft, matching
// App.tsx's handleManualExecuteFromState. No {{env}} interpolation yet —
// consistent with executeWithOverride in tryit_execute.go, environments
// aren't wired into the TUI until Phase 5.
func (m Model) manualExecuteCmd() tea.Cmd {
	manual := m.Manual
	return m.runRequestCmd(manual.Method, manual.Path, manual.Params, manual.Body)
}

// savedRequestExecuteCmd runs a persisted saved request directly from
// browse mode, matching App.tsx's executeCurrentEndpoint's savedRequest
// branch (browse-mode 'e' quick-execute).
func (m Model) savedRequestExecuteCmd(sr *storage.SavedRequest) tea.Cmd {
	var params []storage.CustomParameter
	for _, p := range sr.QueryParams {
		params = append(params, storage.CustomParameter{ID: p.ID, Name: p.Key, Value: p.Value, In: "query", Enabled: p.Enabled})
	}
	for _, h := range sr.Headers {
		params = append(params, storage.CustomParameter{ID: h.ID, Name: h.Key, Value: h.Value, In: "header", Enabled: h.Enabled})
	}
	return m.runRequestCmd(sr.Method, sr.Path, params, sr.Body)
}

func (m Model) runRequestCmd(method, path string, params []storage.CustomParameter, body string) tea.Cmd {
	specServers := m.Spec.Spec.Servers
	selectedServer := m.SelectedServer
	client := m.HTTPClient
	store := m.Store
	security := m.Spec.Spec.Security
	var securitySchemes map[string]openapi.SecurityScheme
	if m.Spec.Spec.Components != nil {
		securitySchemes = m.Spec.Spec.Components.SecuritySchemes
	}

	if method == "" {
		method = "GET"
	}

	return func() tea.Msg {
		collector := &request.ParameterCollector{CustomParams: params}
		spec := buildRequestSpec(specServers, selectedServer, store, collector, method, path, body, security, securitySchemes)

		if client == nil {
			return responseMsg{response: &request.Response{Error: "no HTTP client configured"}}
		}
		resp, curl := request.Execute(client, spec)
		return responseMsg{response: resp, curl: curl}
	}
}
