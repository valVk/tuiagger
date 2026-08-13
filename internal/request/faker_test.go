package request

import "testing"

func TestExpandFakerCoversEveryMappedPath(t *testing.T) {
	for module, methods := range fakerFuncs {
		for method := range methods {
			expr := "{{faker." + module + "." + method + "()}}"
			got := expandFaker(expr)
			if got == expr || got == "" {
				t.Errorf("faker.%s.%s(): expected a generated value, got %q", module, method, got)
			}
		}
	}
}

func TestExpandFakerLeavesUnknownModuleIntact(t *testing.T) {
	expr := "{{faker.nope.nope()}}"
	if got := expandFaker(expr); got != expr {
		t.Errorf("expected unknown module untouched, got %q", got)
	}
}

func TestExpandFakerLeavesUnknownMethodIntact(t *testing.T) {
	expr := "{{faker.person.bogusMethod()}}"
	if got := expandFaker(expr); got != expr {
		t.Errorf("expected unknown method untouched, got %q", got)
	}
}

func TestExpandFakerLeavesNonFakerTextAlone(t *testing.T) {
	if got := expandFaker("plain text, no faker here"); got != "plain text, no faker here" {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestExpandFakerExpandsMultipleOccurrences(t *testing.T) {
	got := expandFaker("{{faker.lorem.word()}} and {{faker.lorem.word()}}")
	if got == "{{faker.lorem.word()}} and {{faker.lorem.word()}}" {
		t.Errorf("expected both occurrences expanded, got %q", got)
	}
}
