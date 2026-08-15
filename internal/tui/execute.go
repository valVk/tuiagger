package tui

import (
	"github.com/valVK/tuiagger/internal/bodyformat"
	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/request"
	"github.com/valVK/tuiagger/internal/storage"
)

// buildRequestSpec assembles a request.Spec from spec/override context —
// baseURL selection, env/auth loading, path/query/header collection via
// collector — shared by try-it-out's executeWithOverride
// (tryit_execute.go) and the manual builder's runRequestCmd
// (manual_execute.go). body is interpolated against the loaded
// environment here; callers needing a body-scaffold fallback (try-it-out)
// or override persistence (also try-it-out) do that themselves before
// calling in, since only one caller needs either.
func buildRequestSpec(servers []openapi.Server, selectedServer int, store *storage.Store, collector *request.ParameterCollector, method, path, body, contentType string, security []openapi.SecurityRequirement, securitySchemes map[string]openapi.SecurityScheme) request.Spec {
	baseURL := "http://localhost"
	if len(servers) > 0 {
		idx := selectedServer
		if idx < 0 || idx >= len(servers) {
			idx = 0
		}
		baseURL = servers[idx].URL
	}

	envVars, authCreds := loadEnvAndAuth(store)
	collector.EnvVars = envVars

	// bodyformat.WireEncode runs after {{env}} interpolation, not before —
	// interpolation must see the human-editable text a form-urlencoded
	// body is still in at this point, not an already percent-escaped
	// string (where a literal "{{" would already be "%7B%7B" and never
	// match). It's a no-op for JSON/XML, so every caller runs it
	// unconditionally rather than special-casing one content type here.
	interpolatedBody := bodyformat.WireEncode(contentType, request.Interpolate(body, envVars))

	return request.Spec{
		Method:            method,
		BaseURL:           baseURL,
		Path:              collector.ApplyPathParams(path),
		QueryParams:       collector.QueryParams(),
		HeaderParams:      collector.HeaderParams(),
		Body:              interpolatedBody,
		ContentType:       contentType,
		OperationSecurity: security,
		SecuritySchemes:   securitySchemes,
		AuthCredentials:   authCreds,
	}
}

// loadEnvAndAuth reads the active environment's variables and stored auth
// credentials at execute time — shared by tryit_execute.go and
// manual_execute.go so both request paths get {{envVar}} interpolation and
// auth injection now that Phase 5 wired credential/environment editing
// into the info popup.
func loadEnvAndAuth(store *storage.Store) (envVars, authCreds map[string]string) {
	if store == nil {
		return nil, nil
	}
	envStore := store.LoadEnvironments()
	if envStore.ActiveIndex >= 0 && envStore.ActiveIndex < len(envStore.Environments) {
		envVars = envStore.Environments[envStore.ActiveIndex].Variables
	}
	authCreds = store.LoadAuth().Credentials
	return envVars, authCreds
}
