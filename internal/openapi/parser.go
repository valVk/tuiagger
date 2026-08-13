package openapi

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml "go.yaml.in/yaml/v4"
)

// ParsedEndpoint is one operation extracted from a spec's paths.
type ParsedEndpoint struct {
	Path      string
	Method    HTTPMethod
	Operation *Operation
	Tags      []string
}

type ParsedSpec struct {
	Spec      *Spec
	Endpoints []ParsedEndpoint
	Tags      []string
}

// ParseOpenAPISpec loads a spec from a local file path or an http(s) URL and
// converts it into the plain, resolved Spec representation.
func ParseOpenAPISpec(source string) (*ParsedSpec, error) {
	data, err := loadSource(source)
	if err != nil {
		return nil, err
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", source, err)
	}

	model, buildErr := doc.BuildV3Model()
	if buildErr != nil {
		return nil, fmt.Errorf("build model for %s: %w", source, buildErr)
	}

	spec := convertDocument(&model.Model)
	endpoints := ExtractEndpoints(spec)
	tags := ExtractTags(spec, endpoints)

	return &ParsedSpec{Spec: spec, Endpoints: endpoints, Tags: tags}, nil
}

func loadSource(source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", source, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch %s: unexpected status %s", source, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	return os.ReadFile(source)
}

// ExtractEndpoints walks Paths in declaration order and, within each path,
// HTTPMethods in a fixed order, matching parser.ts's extractEndpoints.
func ExtractEndpoints(spec *Spec) []ParsedEndpoint {
	var endpoints []ParsedEndpoint
	for _, entry := range spec.Paths {
		for _, method := range HTTPMethods {
			op, ok := entry.Item.Operations[method]
			if !ok || op == nil {
				continue
			}
			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{"default"}
			}
			endpoints = append(endpoints, ParsedEndpoint{
				Path:      entry.Path,
				Method:    method,
				Operation: op,
				Tags:      tags,
			})
		}
	}
	return endpoints
}

// ExtractTags merges spec-declared tags with tags only referenced by
// endpoints, preserving spec declaration order first.
func ExtractTags(spec *Spec, endpoints []ParsedEndpoint) []string {
	seen := make(map[string]bool)
	var allTags []string
	for _, t := range spec.Tags {
		if !seen[t.Name] {
			seen[t.Name] = true
			allTags = append(allTags, t.Name)
		}
	}
	for _, ep := range endpoints {
		for _, tag := range ep.Tags {
			if !seen[tag] {
				seen[tag] = true
				allTags = append(allTags, tag)
			}
		}
	}
	return allTags
}

// GetEndpointsByTag groups endpoints by tag, preserving each tag's endpoint
// order (a single endpoint with multiple tags appears in each tag's slice).
func GetEndpointsByTag(endpoints []ParsedEndpoint) map[string][]ParsedEndpoint {
	byTag := make(map[string][]ParsedEndpoint)
	for _, ep := range endpoints {
		for _, tag := range ep.Tags {
			byTag[tag] = append(byTag[tag], ep)
		}
	}
	return byTag
}

func convertDocument(doc *v3.Document) *Spec {
	spec := &Spec{
		OpenAPI: doc.Version,
		Info: Info{
			Title:       doc.Info.Title,
			Version:     doc.Info.Version,
			Description: doc.Info.Description,
		},
	}

	for _, s := range doc.Servers {
		spec.Servers = append(spec.Servers, Server{URL: s.URL, Description: s.Description})
	}

	for _, t := range doc.Tags {
		spec.Tags = append(spec.Tags, Tag{Name: t.Name, Description: t.Description})
	}

	if doc.Paths != nil {
		for path, item := range doc.Paths.PathItems.FromOldest() {
			spec.Paths = append(spec.Paths, PathEntry{Path: path, Item: convertPathItem(item)})
		}
	}

	if doc.Components != nil {
		spec.Components = convertComponents(doc.Components)
	}

	return spec
}

func convertPathItem(item *v3.PathItem) PathItem {
	pi := PathItem{
		Summary:     item.Summary,
		Description: item.Description,
		Operations:  make(map[HTTPMethod]*Operation),
	}
	add := func(method HTTPMethod, op *v3.Operation) {
		if op != nil {
			pi.Operations[method] = convertOperation(op)
		}
	}
	add(MethodGet, item.Get)
	add(MethodPost, item.Post)
	add(MethodPut, item.Put)
	add(MethodDelete, item.Delete)
	add(MethodPatch, item.Patch)
	add(MethodOptions, item.Options)
	add(MethodHead, item.Head)
	add(MethodTrace, item.Trace)
	return pi
}

func convertOperation(op *v3.Operation) *Operation {
	out := &Operation{
		Tags:        op.Tags,
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationId,
		Deprecated:  op.Deprecated != nil && *op.Deprecated,
	}

	for _, p := range op.Parameters {
		out.Parameters = append(out.Parameters, convertParameter(p))
	}

	if op.RequestBody != nil {
		out.RequestBody = &RequestBody{
			Description: op.RequestBody.Description,
			Required:    op.RequestBody.Required != nil && *op.RequestBody.Required,
			Content:     convertMediaTypeMap(op.RequestBody.Content),
		}
	}

	if op.Responses != nil {
		for status, resp := range op.Responses.Codes.FromOldest() {
			out.Responses = append(out.Responses, ResponseEntry{
				Status: status,
				Response: Response{
					Description: resp.Description,
					Content:     convertMediaTypeMap(resp.Content),
				},
			})
		}
		if op.Responses.Default != nil {
			out.Responses = append(out.Responses, ResponseEntry{
				Status: "default",
				Response: Response{
					Description: op.Responses.Default.Description,
					Content:     convertMediaTypeMap(op.Responses.Default.Content),
				},
			})
		}
	}

	return out
}

func convertParameter(p *v3.Parameter) Parameter {
	return Parameter{
		Name:        p.Name,
		In:          p.In,
		Required:    p.Required != nil && *p.Required,
		Description: p.Description,
		Deprecated:  p.Deprecated,
		Schema:      convertSchemaProxy(p.Schema, nil),
	}
}

func convertMediaTypeMap(content *orderedmap.Map[string, *v3.MediaType]) map[string]MediaType {
	if content == nil {
		return nil
	}
	out := make(map[string]MediaType)
	for k, mt := range content.FromOldest() {
		out[k] = MediaType{
			Schema:  convertSchemaProxy(mt.Schema, nil),
			Example: nodeToAny(mt.Example),
		}
	}
	return out
}

func convertComponents(c *v3.Components) *Components {
	out := &Components{
		Schemas:         make(map[string]*Schema),
		SecuritySchemes: make(map[string]SecurityScheme),
	}
	if c.Schemas != nil {
		for name, proxy := range c.Schemas.FromOldest() {
			out.Schemas[name] = convertSchemaProxy(proxy, nil)
		}
	}
	if c.SecuritySchemes != nil {
		for name, ss := range c.SecuritySchemes.FromOldest() {
			out.SecuritySchemes[name] = SecurityScheme{
				Type:         ss.Type,
				Description:  ss.Description,
				Name:         ss.Name,
				In:           ss.In,
				Scheme:       ss.Scheme,
				BearerFormat: ss.BearerFormat,
			}
		}
	}
	return out
}

// convertSchemaProxy resolves a schema proxy into a plain Schema, breaking
// cycles via the $ref name of any proxy currently being expanded (mirrors
// parser.ts's resolveSchema "seen" set, which relies on JS object identity —
// Go rebuilds a schema per call, so identity isn't available, but the
// $ref string is a stable substitute).
func convertSchemaProxy(proxy *base.SchemaProxy, visiting map[string]bool) *Schema {
	if proxy == nil {
		return nil
	}

	if proxy.IsReference() {
		ref := proxy.GetReference()
		if visiting[ref] {
			return &Schema{Description: "(circular reference)"}
		}
		next := make(map[string]bool, len(visiting)+1)
		for k := range visiting {
			next[k] = true
		}
		next[ref] = true
		visiting = next
	}

	s := proxy.Schema()
	if s == nil {
		return nil
	}
	return convertSchema(s, visiting)
}

func convertSchema(s *base.Schema, visiting map[string]bool) *Schema {
	out := &Schema{
		Type:        s.Type,
		Format:      s.Format,
		Title:       s.Title,
		Description: s.Description,
		Required:    s.Required,
		Nullable:    s.Nullable != nil && *s.Nullable,
		Default:     nodeToAny(s.Default),
		Example:     nodeToAny(s.Example),
	}

	for _, n := range s.Enum {
		out.Enum = append(out.Enum, nodeToAny(n))
	}

	if s.Items != nil && s.Items.IsA() {
		out.Items = convertSchemaProxy(s.Items.A, visiting)
	}

	if s.Properties != nil {
		for name, propProxy := range s.Properties.FromOldest() {
			out.Properties = append(out.Properties, Property{
				Name:   name,
				Schema: convertSchemaProxy(propProxy, visiting),
			})
		}
	}

	return out
}

func nodeToAny(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	var v any
	if err := node.Decode(&v); err != nil {
		return nil
	}
	return v
}
