package tui

import (
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
func buildRequestSpec(servers []openapi.Server, selectedServer int, store *storage.Store, collector *request.ParameterCollector, method, path, body string, security []openapi.SecurityRequirement, securitySchemes map[string]openapi.SecurityScheme) request.Spec {
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

	return request.Spec{
		Method:            method,
		BaseURL:           baseURL,
		Path:              collector.ApplyPathParams(path),
		QueryParams:       collector.QueryParams(),
		HeaderParams:      collector.HeaderParams(),
		Body:              request.Interpolate(body, envVars),
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
