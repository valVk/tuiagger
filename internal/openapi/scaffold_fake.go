package openapi

import (
	"encoding/base64"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// ScaffoldFakeBody builds a realistic-looking request body from a schema
// using randomly generated data, matching scaffoldBody.ts. This is
// distinct from ScaffoldPlaceholder, which produces human-readable
// placeholders like "<string>" for the read-only docs view — this one
// generates actual values a write-method request can be auto-filled with
// (try-it-out's "no body editor yet" auto-fill, see HANDOFF.md).
//
// One simplification vs the TS source: OpenAPI's schema.minimum/maximum
// aren't carried on this rewrite's Schema type (nothing else needed them),
// so numeric bounds from the spec aren't honored — generated numbers are
// unbounded/reasonable defaults rather than clamped to the schema's declared
// range.
func ScaffoldFakeBody(s *Schema) any {
	if s == nil {
		return nil
	}

	if s.Example != nil {
		return s.Example
	}

	if len(s.Enum) > 0 {
		return s.Enum[gofakeit.Number(0, len(s.Enum)-1)]
	}

	primaryType := ""
	if len(s.Type) > 0 {
		primaryType = s.Type[0]
	}

	switch primaryType {
	case "string":
		return scaffoldFakeString(s.Format)
	case "integer":
		return gofakeit.Number(0, 1000)
	case "number":
		if s.Format == "float" {
			return round(float64(gofakeit.Float32()), 2)
		}
		return round(gofakeit.Float64(), 4)
	case "boolean":
		return gofakeit.Bool()
	case "array":
		if s.Items == nil {
			return []any{}
		}
		return []any{ScaffoldFakeBody(s.Items)}
	}

	if primaryType == "object" || len(s.Properties) > 0 {
		result := make(map[string]any, len(s.Properties))
		for _, prop := range s.Properties {
			result[prop.Name] = ScaffoldFakeBody(prop.Schema)
		}
		return result
	}

	return nil
}

func scaffoldFakeString(format string) string {
	switch format {
	case "uuid":
		return gofakeit.UUID()
	case "email":
		return gofakeit.Email()
	case "date":
		return gofakeit.Date().Format("2006-01-02")
	case "date-time":
		return gofakeit.Date().Format(time.RFC3339)
	case "uri", "url":
		return gofakeit.URL()
	case "hostname":
		return gofakeit.DomainName()
	case "ipv4":
		return gofakeit.IPv4Address()
	case "ipv6":
		return gofakeit.IPv6Address()
	case "byte":
		return base64.StdEncoding.EncodeToString([]byte(gofakeit.Word()))
	case "password":
		return gofakeit.Password(true, true, true, true, false, 12)
	default:
		return gofakeit.Word()
	}
}

func round(v float64, decimals int) float64 {
	mult := 1.0
	for range decimals {
		mult *= 10
	}
	return float64(int(v*mult+0.5)) / mult
}
