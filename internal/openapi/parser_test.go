package openapi

import (
	"slices"
	"testing"
)

func TestParsePetstoreFixture(t *testing.T) {
	parsed, err := ParseOpenAPISpec("testdata/petstore.json")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.Spec.OpenAPI == "" {
		t.Fatalf("expected openapi version to be set")
	}
	if len(parsed.Spec.Paths) == 0 {
		t.Fatalf("expected paths to be parsed")
	}
	if len(parsed.Endpoints) == 0 {
		t.Fatalf("expected endpoints to be extracted")
	}
	wantTags := []string{"pet", "store", "user"}
	for _, want := range wantTags {
		if !slices.Contains(parsed.Tags, want) {
			t.Errorf("expected tag %q in %v", want, parsed.Tags)
		}
	}

	byTag := GetEndpointsByTag(parsed.Endpoints)
	if len(byTag["pet"]) == 0 {
		t.Errorf("expected pet-tagged endpoints")
	}

	var findByStatus *ParsedEndpoint
	for i := range parsed.Endpoints {
		if parsed.Endpoints[i].Path == "/pet/findByStatus" && parsed.Endpoints[i].Method == MethodGet {
			findByStatus = &parsed.Endpoints[i]
		}
	}
	if findByStatus == nil {
		t.Fatalf("expected GET /pet/findByStatus endpoint")
	}
	if findByStatus.Operation.OperationID != "findPetsByStatus" {
		t.Errorf("expected operationId findPetsByStatus, got %q", findByStatus.Operation.OperationID)
	}
	if len(findByStatus.Operation.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(findByStatus.Operation.Parameters))
	}
}

func TestParse31FixtureWebhooksAndNullableType(t *testing.T) {
	parsed, err := ParseOpenAPISpec("testdata/mini31.json")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.Spec.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi 3.1.0, got %q", parsed.Spec.OpenAPI)
	}

	var getWidget *ParsedEndpoint
	for i := range parsed.Endpoints {
		if parsed.Endpoints[i].Operation.OperationID == "getWidget" {
			getWidget = &parsed.Endpoints[i]
		}
	}
	if getWidget == nil {
		t.Fatalf("expected getWidget endpoint")
	}

	resp := getWidget.Operation.Responses[0]
	schema := resp.Response.Content["application/json"].Schema
	var nickname *Schema
	for _, p := range schema.Properties {
		if p.Name == "nickname" {
			nickname = p.Schema
		}
	}
	if nickname == nil {
		t.Fatalf("expected nickname property")
	}
	if len(nickname.Type) != 2 || nickname.Type[0] != "string" || nickname.Type[1] != "null" {
		t.Errorf("expected nullable-via-type-array [string null], got %v", nickname.Type)
	}
}

func TestExtractTagsDefaultsUntaggedEndpoints(t *testing.T) {
	parsed, err := ParseOpenAPISpec("testdata/mini31.json")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	found := false
	for _, ep := range parsed.Endpoints {
		if ep.Operation.OperationID == "getUntagged" {
			found = true
			if len(ep.Tags) != 1 || ep.Tags[0] != "default" {
				t.Errorf("expected untagged endpoint to default to [\"default\"], got %v", ep.Tags)
			}
		}
	}
	if !found {
		t.Fatalf("expected getUntagged endpoint")
	}
}

func TestCircularSchemaDoesNotHang(t *testing.T) {
	parsed, err := ParseOpenAPISpec("testdata/circular.json")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	ep := parsed.Endpoints[0]
	schema := ep.Operation.Responses[0].Response.Content["application/json"].Schema
	if schema == nil {
		t.Fatalf("expected schema")
	}
	// The top-level $ref (TreeNode) resolves once, so its own properties
	// (name, children) are visible — the cycle only bites one level deeper.
	if len(schema.Properties) != 2 {
		t.Fatalf("expected TreeNode's own 2 properties to resolve, got %d", len(schema.Properties))
	}

	var childrenItems *Schema
	for _, p := range schema.Properties {
		if p.Name == "children" {
			childrenItems = p.Schema.Items
		}
	}
	if childrenItems == nil {
		t.Fatalf("expected children.items schema")
	}
	// children.items points back to the same TreeNode $ref already being
	// expanded — that second occurrence is where the cycle is cut.
	if len(childrenItems.Properties) != 0 {
		t.Errorf("expected the second occurrence of the TreeNode ref to be cut, got properties %v", childrenItems.Properties)
	}
	if childrenItems.Description != "(circular reference)" {
		t.Errorf("expected circular placeholder, got %+v", childrenItems)
	}
}
