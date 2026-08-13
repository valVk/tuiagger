// Package openapi holds a plain, resolved representation of an OpenAPI 3.x
// spec (no $ref, no ordered-map indirection) so later phases can render and
// walk it without touching libopenapi's model types directly.
package openapi

// HTTPMethod is a lowercase OpenAPI operation verb.
type HTTPMethod string

const (
	MethodGet     HTTPMethod = "get"
	MethodPost    HTTPMethod = "post"
	MethodPut     HTTPMethod = "put"
	MethodDelete  HTTPMethod = "delete"
	MethodPatch   HTTPMethod = "patch"
	MethodOptions HTTPMethod = "options"
	MethodHead    HTTPMethod = "head"
	MethodTrace   HTTPMethod = "trace"
)

// HTTPMethods lists verbs in the fixed order operations are extracted in.
var HTTPMethods = []HTTPMethod{
	MethodGet, MethodPost, MethodPut, MethodDelete,
	MethodPatch, MethodOptions, MethodHead, MethodTrace,
}

type Spec struct {
	OpenAPI    string
	Info       Info
	Servers    []Server
	Paths      []PathEntry
	Components *Components
	Tags       []Tag
}

// PathEntry preserves spec declaration order (a Go map would not).
type PathEntry struct {
	Path string
	Item PathItem
}

type Info struct {
	Title       string
	Version     string
	Description string
}

type Server struct {
	URL         string
	Description string
}

type PathItem struct {
	Summary     string
	Description string
	Operations  map[HTTPMethod]*Operation
}

type Operation struct {
	Tags        []string
	Summary     string
	Description string
	OperationID string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   []ResponseEntry
	Deprecated  bool
}

type ResponseEntry struct {
	Status   string
	Response Response
}

type Parameter struct {
	Name        string
	In          string // path | query | header | cookie
	Required    bool
	Description string
	Deprecated  bool
	Schema      *Schema
}

type RequestBody struct {
	Description string
	Required    bool
	Content     map[string]MediaType
}

type MediaType struct {
	Schema  *Schema
	Example any
}

type Response struct {
	Description string
	Content     map[string]MediaType
}

// Property preserves schema property declaration order.
type Property struct {
	Name   string
	Schema *Schema
}

type Schema struct {
	// Type holds one or more JSON Schema types. OpenAPI 3.0 has exactly one
	// (or none); 3.1 allows an array (e.g. ["string", "null"]).
	Type        []string
	Format      string
	Title       string
	Description string
	Default     any
	Enum        []any
	Items       *Schema
	Properties  []Property
	Required    []string
	Nullable    bool
	Example     any
}

type Components struct {
	Schemas         map[string]*Schema
	SecuritySchemes map[string]SecurityScheme
}

type SecurityScheme struct {
	Type         string
	Description  string
	Name         string
	In           string
	Scheme       string
	BearerFormat string
}

type Tag struct {
	Name        string
	Description string
}
