package request

import "testing"

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

func TestInterpolateLeavesFakerSyntaxUntouched(t *testing.T) {
	// Faker expansion is Phase 6 scope; the pattern must survive today.
	got := Interpolate("{{faker.internet.email()}}", map[string]string{})
	if got != "{{faker.internet.email()}}" {
		t.Errorf("expected faker syntax untouched, got %q", got)
	}
}
