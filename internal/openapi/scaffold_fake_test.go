package openapi

import (
	"strings"
	"testing"
)

func TestScaffoldFakeBodyObjectGeneratesTypedValues(t *testing.T) {
	s := &Schema{
		Type: []string{"object"},
		Properties: []Property{
			{Name: "email", Schema: &Schema{Type: []string{"string"}, Format: "email"}},
			{Name: "age", Schema: &Schema{Type: []string{"integer"}}},
			{Name: "active", Schema: &Schema{Type: []string{"boolean"}}},
		},
	}
	out := ScaffoldFakeBody(s).(map[string]any)

	email, ok := out["email"].(string)
	if !ok || !strings.Contains(email, "@") {
		t.Errorf("expected a generated email address, got %v", out["email"])
	}
	if _, ok := out["age"].(int); !ok {
		t.Errorf("expected an int for age, got %T %v", out["age"], out["age"])
	}
	if _, ok := out["active"].(bool); !ok {
		t.Errorf("expected a bool for active, got %T %v", out["active"], out["active"])
	}
}

func TestScaffoldFakeBodyPrefersExampleOverGeneration(t *testing.T) {
	s := &Schema{Type: []string{"string"}, Example: "literal-example"}
	if got := ScaffoldFakeBody(s); got != "literal-example" {
		t.Errorf("expected the schema's example to win, got %v", got)
	}
}

func TestScaffoldFakeBodyPicksFromEnum(t *testing.T) {
	s := &Schema{Type: []string{"string"}, Enum: []any{"available", "pending", "sold"}}
	got := ScaffoldFakeBody(s)
	found := false
	for _, v := range s.Enum {
		if got == v {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a value from the enum, got %v", got)
	}
}

func TestScaffoldFakeBodyArrayWrapsOneGeneratedItem(t *testing.T) {
	s := &Schema{Type: []string{"array"}, Items: &Schema{Type: []string{"string"}}}
	out, ok := ScaffoldFakeBody(s).([]any)
	if !ok || len(out) != 1 {
		t.Fatalf("expected a one-element slice, got %v", ScaffoldFakeBody(s))
	}
	if _, ok := out[0].(string); !ok {
		t.Errorf("expected a generated string item, got %T", out[0])
	}
}

func TestScaffoldFakeBodyArrayWithoutItemsIsEmpty(t *testing.T) {
	s := &Schema{Type: []string{"array"}}
	out, ok := ScaffoldFakeBody(s).([]any)
	if !ok || len(out) != 0 {
		t.Errorf("expected an empty slice, got %v", ScaffoldFakeBody(s))
	}
}

func TestScaffoldFakeBodyStringFormats(t *testing.T) {
	cases := map[string]func(string) bool{
		"uuid":      func(v string) bool { return strings.Count(v, "-") == 4 },
		"email":     func(v string) bool { return strings.Contains(v, "@") },
		"date":      func(v string) bool { return len(v) == len("2006-01-02") },
		"date-time": func(v string) bool { return strings.Contains(v, "T") },
		"ipv4":      func(v string) bool { return strings.Count(v, ".") == 3 },
	}
	for format, check := range cases {
		got, ok := ScaffoldFakeBody(&Schema{Type: []string{"string"}, Format: format}).(string)
		if !ok || !check(got) {
			t.Errorf("format %q: unexpected value %q", format, got)
		}
	}
}

func TestScaffoldFakeBodyNilSchema(t *testing.T) {
	if ScaffoldFakeBody(nil) != nil {
		t.Errorf("expected nil for a nil schema")
	}
}
