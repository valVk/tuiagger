package request

import (
	"encoding/base64"
	"slices"
	"strings"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// Spec is everything needed to build one HTTP request, matching
// types/request.ts's RequestSpec.
type Spec struct {
	Method            string
	BaseURL           string
	Path              string // path params already substituted
	QueryParams       []storage.KeyValuePair
	HeaderParams      []storage.KeyValuePair
	Body              string
	OperationSecurity []openapi.SecurityRequirement
	SecuritySchemes   map[string]openapi.SecurityScheme
	AuthCredentials   map[string]string
}

// BuiltRequest is a fully assembled, ready-to-send request plus its curl
// equivalent, matching requestBuilder.ts's BuiltRequest.
type BuiltRequest struct {
	URL     string
	Method  string
	Headers []HeaderPair
	Body    string
	Curl    string
}

// Build assembles URL, headers (with auth injected from the first
// satisfied security requirement), and a curl command, matching
// requestBuilder.ts's RequestBuilder.build().
func Build(spec Spec) (BuiltRequest, error) {
	var headers []HeaderPair
	setHeader := func(name, value string) {
		for i := range headers {
			if headers[i].Name == name {
				headers[i].Value = value
				return
			}
		}
		headers = append(headers, HeaderPair{Name: name, Value: value})
	}
	getHeader := func(name string) (string, bool) {
		for _, h := range headers {
			if h.Name == name {
				return h.Value, true
			}
		}
		return "", false
	}

	setHeader("Accept", "application/json")
	effectiveQueryParams := slices.Clone(spec.QueryParams)

	// Inject auth from the first satisfied security requirement.
outer:
	for _, requirement := range spec.OperationSecurity {
		for schemeName := range requirement {
			scheme, hasScheme := spec.SecuritySchemes[schemeName]
			credential, hasCred := spec.AuthCredentials[schemeName]
			if !hasScheme || !hasCred || credential == "" {
				continue
			}

			switch {
			case scheme.Type == "http":
				isBasic := strings.EqualFold(scheme.Scheme, "basic")
				value := credential
				if isBasic {
					value = base64.StdEncoding.EncodeToString([]byte(credential))
				}
				kind := "Bearer"
				if isBasic {
					kind = "Basic"
				}
				setHeader("Authorization", kind+" "+value)
			case scheme.Type == "apiKey" && scheme.In == "header" && scheme.Name != "":
				setHeader(scheme.Name, credential)
			case scheme.Type == "apiKey" && scheme.In == "query" && scheme.Name != "":
				effectiveQueryParams = append(effectiveQueryParams, storage.KeyValuePair{
					ID: scheme.Name, Key: scheme.Name, Value: credential, Enabled: true,
				})
			}
			break outer
		}
	}

	for _, h := range spec.HeaderParams {
		if h.Enabled && h.Key != "" {
			setHeader(h.Key, h.Value)
		}
	}

	if spec.Body != "" {
		if _, ok := getHeader("Content-Type"); !ok {
			setHeader("Content-Type", "application/json")
		}
	}

	url, err := BuildRequestURL(spec.BaseURL, spec.Path, effectiveQueryParams)
	if err != nil {
		return BuiltRequest{}, err
	}

	body := ""
	method := strings.ToUpper(spec.Method)
	if spec.Body != "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		body = spec.Body
	}

	return BuiltRequest{
		URL:     url,
		Method:  method,
		Headers: headers,
		Body:    body,
		Curl:    GenerateCurl(method, url, headers, body),
	}, nil
}
