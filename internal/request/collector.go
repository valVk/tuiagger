package request

import (
	"net/url"
	"slices"
	"strings"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// ParameterCollector merges spec-defined parameters with user-added custom
// ones into query/header/path values, matching parameterCollector.ts.
type ParameterCollector struct {
	SpecParams      []openapi.Parameter
	CustomParams    []storage.CustomParameter
	DisabledParams  []string
	ParameterValues map[string]string
	EnvVars         map[string]string
}

func (c *ParameterCollector) disabled(name string) bool {
	return slices.Contains(c.DisabledParams, name)
}

func (c *ParameterCollector) collect(in string) []storage.KeyValuePair {
	var params []storage.KeyValuePair

	for _, p := range c.SpecParams {
		if p.In != in || c.disabled(p.Name) {
			continue
		}
		value := Interpolate(c.ParameterValues[p.Name], c.EnvVars)
		if value != "" {
			params = append(params, storage.KeyValuePair{ID: p.Name, Key: p.Name, Value: value, Enabled: true})
		}
	}

	for _, p := range c.CustomParams {
		if p.In != in || !p.Enabled || p.Name == "" {
			continue
		}
		params = append(params, storage.KeyValuePair{ID: p.ID, Key: p.Name, Value: Interpolate(p.Value, c.EnvVars), Enabled: true})
	}

	return params
}

// QueryParams returns enabled query-string parameters (spec + custom).
func (c *ParameterCollector) QueryParams() []storage.KeyValuePair { return c.collect("query") }

// HeaderParams returns enabled header parameters (spec + custom).
func (c *ParameterCollector) HeaderParams() []storage.KeyValuePair { return c.collect("header") }

// ApplyPathParams substitutes {name} placeholders in path with interpolated,
// URL-escaped parameter values (spec + custom).
func (c *ParameterCollector) ApplyPathParams(path string) string {
	result := path

	for _, p := range c.SpecParams {
		if p.In != "path" || c.disabled(p.Name) {
			continue
		}
		value := Interpolate(c.ParameterValues[p.Name], c.EnvVars)
		result = strings.ReplaceAll(result, "{"+p.Name+"}", url.PathEscape(value))
	}

	for _, p := range c.CustomParams {
		if p.In != "path" || !p.Enabled || p.Name == "" {
			continue
		}
		result = strings.ReplaceAll(result, "{"+p.Name+"}", url.PathEscape(Interpolate(p.Value, c.EnvVars)))
	}

	return result
}
