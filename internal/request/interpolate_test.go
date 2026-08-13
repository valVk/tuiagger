package request

import (
	"strings"
	"testing"
)

func TestInterpolateSubstitutesEnvVars(t *testing.T) {
	got := Interpolate("hello {{name}}", map[string]string{"name": "world"})
	if got != "hello world" {
		t.Errorf("got %q", got)
	}
}

func TestInterpolateLeavesUnknownRefsIntact(t *testing.T) {
	got := Interpolate("{{missing}}", map[string]string{})
	if got != "{{missing}}" {
		t.Errorf("got %q", got)
	}
}

func TestInterpolateResolvesChainedRefs(t *testing.T) {
	got := Interpolate("{{a}}", map[string]string{"a": "{{b}}", "b": "final"})
	if got != "final" {
		t.Errorf("expected chained resolution, got %q", got)
	}
}

func TestInterpolateNilEnvVarsIsNoop(t *testing.T) {
	got := Interpolate("{{x}}", nil)
	if got != "{{x}}" {
		t.Errorf("got %q", got)
	}
}

func TestInterpolateExpandsKnownFakerPath(t *testing.T) {
	got := Interpolate("{{faker.internet.email()}}", map[string]string{})
	if got == "{{faker.internet.email()}}" || !strings.Contains(got, "@") {
		t.Errorf("expected an expanded email address, got %q", got)
	}
}

func TestInterpolateLeavesUnknownFakerPathIntact(t *testing.T) {
	got := Interpolate("{{faker.bogus.nope()}}", map[string]string{})
	if got != "{{faker.bogus.nope()}}" {
		t.Errorf("expected unknown faker path left untouched, got %q", got)
	}
}

func TestInterpolateExpandsFakerBeforeEnvVars(t *testing.T) {
	// A faker-generated value must never be re-interpreted as an env-var
	// placeholder — expandFaker runs first, so this only breaks if the
	// pass order regresses.
	got := Interpolate("{{faker.internet.email()}} {{missing}}", map[string]string{})
	if strings.Contains(got, "faker.internet.email") {
		t.Errorf("expected faker expression expanded, got %q", got)
	}
	if !strings.Contains(got, "{{missing}}") {
		t.Errorf("expected unrelated env-var ref left intact, got %q", got)
	}
}
