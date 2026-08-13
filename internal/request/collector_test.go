package request

import (
	"testing"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

func TestQueryParamsSkipsDisabledAndEmpty(t *testing.T) {
	c := &ParameterCollector{
		SpecParams: []openapi.Parameter{
			{Name: "status", In: "query"},
			{Name: "limit", In: "query"},
			{Name: "id", In: "path"},
		},
		DisabledParams:  []string{"limit"},
		ParameterValues: map[string]string{"status": "available", "limit": "10"},
	}
	got := c.QueryParams()
	if len(got) != 1 || got[0].Key != "status" || got[0].Value != "available" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestQueryParamsIncludesEnabledCustomParams(t *testing.T) {
	c := &ParameterCollector{
		CustomParams: []storage.CustomParameter{
			{ID: "1", Name: "extra", Value: "v", In: "query", Enabled: true},
			{ID: "2", Name: "off", Value: "v", In: "query", Enabled: false},
			{ID: "3", Name: "", Value: "v", In: "query", Enabled: true}, // no name, skipped
		},
	}
	got := c.QueryParams()
	if len(got) != 1 || got[0].Key != "extra" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestHeaderParamsInterpolatesEnvVars(t *testing.T) {
	c := &ParameterCollector{
		SpecParams:      []openapi.Parameter{{Name: "X-Trace", In: "header"}},
		ParameterValues: map[string]string{"X-Trace": "{{traceId}}"},
		EnvVars:         map[string]string{"traceId": "abc-123"},
	}
	got := c.HeaderParams()
	if len(got) != 1 || got[0].Value != "abc-123" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestApplyPathParamsEscapesValues(t *testing.T) {
	c := &ParameterCollector{
		SpecParams:      []openapi.Parameter{{Name: "id", In: "path"}},
		ParameterValues: map[string]string{"id": "a b/c"},
	}
	got := c.ApplyPathParams("/widgets/{id}")
	if got != "/widgets/a%20b%2Fc" {
		t.Errorf("unexpected escaped path: %q", got)
	}
}

func TestApplyPathParamsSkipsDisabled(t *testing.T) {
	c := &ParameterCollector{
		SpecParams:      []openapi.Parameter{{Name: "id", In: "path"}},
		DisabledParams:  []string{"id"},
		ParameterValues: map[string]string{"id": "42"},
	}
	got := c.ApplyPathParams("/widgets/{id}")
	if got != "/widgets/{id}" {
		t.Errorf("expected placeholder left intact when disabled, got %q", got)
	}
}
