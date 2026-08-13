package request

import (
	"strings"
	"testing"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

func TestBuildSetsDefaultAcceptHeader(t *testing.T) {
	built, err := Build(Spec{Method: "get", BaseURL: "http://api.test", Path: "/x"})
	if err != nil {
		t.Fatal(err)
	}
	v, ok := headerValue(built.Headers, "Accept")
	if !ok || v != "application/json" {
		t.Errorf("expected default Accept header, got %+v", built.Headers)
	}
}

func TestBuildInjectsBearerAuth(t *testing.T) {
	built, err := Build(Spec{
		Method: "get", BaseURL: "http://api.test", Path: "/x",
		OperationSecurity: []openapi.SecurityRequirement{{"bearerAuth": {}}},
		SecuritySchemes:   map[string]openapi.SecurityScheme{"bearerAuth": {Type: "http", Scheme: "bearer"}},
		AuthCredentials:   map[string]string{"bearerAuth": "tok123"},
	})
	if err != nil {
		t.Fatal(err)
	}
	v, ok := headerValue(built.Headers, "Authorization")
	if !ok || v != "Bearer tok123" {
		t.Errorf("expected Bearer auth header, got %+v", built.Headers)
	}
}

func TestBuildInjectsBasicAuthBase64Encoded(t *testing.T) {
	built, err := Build(Spec{
		Method: "get", BaseURL: "http://api.test", Path: "/x",
		OperationSecurity: []openapi.SecurityRequirement{{"basicAuth": {}}},
		SecuritySchemes:   map[string]openapi.SecurityScheme{"basicAuth": {Type: "http", Scheme: "Basic"}},
		AuthCredentials:   map[string]string{"basicAuth": "user:pass"},
	})
	if err != nil {
		t.Fatal(err)
	}
	v, ok := headerValue(built.Headers, "Authorization")
	if !ok || v != "Basic dXNlcjpwYXNz" {
		t.Errorf("expected base64-encoded Basic auth, got %+v", built.Headers)
	}
}

func TestBuildInjectsAPIKeyInQuery(t *testing.T) {
	built, err := Build(Spec{
		Method: "get", BaseURL: "http://api.test", Path: "/x",
		OperationSecurity: []openapi.SecurityRequirement{{"apiKeyAuth": {}}},
		SecuritySchemes:   map[string]openapi.SecurityScheme{"apiKeyAuth": {Type: "apiKey", In: "query", Name: "api_key"}},
		AuthCredentials:   map[string]string{"apiKeyAuth": "secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(built.URL, "api_key=secret") {
		t.Errorf("expected api_key in query, got %q", built.URL)
	}
}

func TestBuildSkipsAuthWhenCredentialMissing(t *testing.T) {
	built, err := Build(Spec{
		Method: "get", BaseURL: "http://api.test", Path: "/x",
		OperationSecurity: []openapi.SecurityRequirement{{"bearerAuth": {}}},
		SecuritySchemes:   map[string]openapi.SecurityScheme{"bearerAuth": {Type: "http", Scheme: "bearer"}},
		AuthCredentials:   map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := headerValue(built.Headers, "Authorization"); ok {
		t.Errorf("expected no Authorization header without credential, got %+v", built.Headers)
	}
}

func TestBuildHeaderParamsOverwriteInPlace(t *testing.T) {
	built, err := Build(Spec{
		Method: "get", BaseURL: "http://api.test", Path: "/x",
		HeaderParams: []storage.KeyValuePair{{Key: "Accept", Value: "text/plain", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Headers) != 1 {
		t.Fatalf("expected overwrite in place (1 header), got %+v", built.Headers)
	}
	if v, _ := headerValue(built.Headers, "Accept"); v != "text/plain" {
		t.Errorf("expected overwritten Accept, got %q", v)
	}
}

func TestBuildSetsContentTypeOnlyWhenBodyPresentAndNotSet(t *testing.T) {
	built, err := Build(Spec{Method: "post", BaseURL: "http://api.test", Path: "/x", Body: `{"a":1}`})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := headerValue(built.Headers, "Content-Type"); !ok || v != "application/json" {
		t.Errorf("expected default Content-Type, got %+v", built.Headers)
	}
}

func TestBuildOnlyAttachesBodyForWriteMethods(t *testing.T) {
	get, err := Build(Spec{Method: "get", BaseURL: "http://api.test", Path: "/x", Body: `{"a":1}`})
	if err != nil {
		t.Fatal(err)
	}
	if get.Body != "" {
		t.Errorf("expected GET to drop body, got %q", get.Body)
	}

	post, err := Build(Spec{Method: "post", BaseURL: "http://api.test", Path: "/x", Body: `{"a":1}`})
	if err != nil {
		t.Fatal(err)
	}
	if post.Body != `{"a":1}` {
		t.Errorf("expected POST to keep body, got %q", post.Body)
	}
}

func TestBuildProducesCurl(t *testing.T) {
	built, err := Build(Spec{Method: "get", BaseURL: "http://api.test", Path: "/x"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(built.Curl, "curl -X 'GET'") {
		t.Errorf("expected curl command, got %q", built.Curl)
	}
}

func headerValue(headers []HeaderPair, name string) (string, bool) {
	for _, h := range headers {
		if h.Name == name {
			return h.Value, true
		}
	}
	return "", false
}
