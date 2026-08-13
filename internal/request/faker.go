package request

import (
	"fmt"
	"regexp"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// fakerPattern is a verbatim port of interpolate.ts's INTERPOLATION_RE,
// letters only — a real faker-js path like `internet.ipv4` never matches
// in the TS source either, since digits fall outside `[a-zA-Z]+`. Kept
// intentionally identical rather than "fixed" here.
var fakerPattern = regexp.MustCompile(`\{\{faker\.([a-zA-Z]+)\.([a-zA-Z]+)\(\)\}\}`)

// fakerFuncs maps a curated subset of faker-js's module.method namespace —
// the paths CLAUDE.md documents plus the ones most useful for API testing —
// to gofakeit. This is deliberately not the full faker-js surface: the TS
// version calls `faker[module][method]()` via dynamic property access, so
// it works for literally any valid faker-js path with zero enumeration；
// gofakeit's API doesn't mirror faker-js's module/method namespacing (it's
// a flat set of top-level functions), so there's no equivalent reflection
// trick in Go. An unmapped module/method pair is left intact rather than
// erroring, matching interpolate.ts's own try/catch-and-fallback for an
// unknown faker path.
var fakerFuncs = map[string]map[string]func() string{
	"person": {
		"fullName":  gofakeit.Name,
		"firstName": gofakeit.FirstName,
		"lastName":  gofakeit.LastName,
		"jobTitle":  gofakeit.JobTitle,
	},
	"internet": {
		"email":      gofakeit.Email,
		"url":        gofakeit.URL,
		"ip":         gofakeit.IPv4Address,
		"userName":   gofakeit.Username,
		"username":   gofakeit.Username,
		"domainName": gofakeit.DomainName,
		"password":   func() string { return gofakeit.Password(true, true, true, true, false, 12) },
	},
	"phone": {
		"number": gofakeit.Phone,
	},
	"location": {
		"city":          gofakeit.City,
		"country":       gofakeit.Country,
		"state":         gofakeit.State,
		"zipCode":       gofakeit.Zip,
		"streetAddress": func() string { return gofakeit.Address().Street },
	},
	"company": {
		"name": gofakeit.Company,
	},
	"lorem": {
		"word":      gofakeit.Word,
		"sentence":  func() string { return gofakeit.Sentence(10) },
		"paragraph": func() string { return gofakeit.Paragraph(3, 5, 10, " ") },
	},
	"finance": {
		"creditCardNumber": func() string { return gofakeit.CreditCardNumber(nil) },
		"amount":           func() string { return fmt.Sprintf("%.2f", gofakeit.Price(1, 1000)) },
	},
	"commerce": {
		"productName": gofakeit.ProductName,
		"price":       func() string { return fmt.Sprintf("%.2f", gofakeit.Price(1, 1000)) },
	},
	"string": {
		"uuid": gofakeit.UUID,
	},
	"datatype": {
		"uuid": gofakeit.UUID,
	},
	"number": {
		"int": func() string { return fmt.Sprintf("%d", gofakeit.Number(1, 1000)) },
	},
	"date": {
		"recent": func() string { return gofakeit.Date().Format(time.RFC3339) },
		"past":   func() string { return gofakeit.Date().Format(time.RFC3339) },
		"future": func() string { return gofakeit.Date().Format(time.RFC3339) },
	},
	"color": {
		"human":  gofakeit.Color,
		"rgbHex": gofakeit.HexColor,
	},
}

// expandFaker replaces {{faker.module.method()}} expressions, matching
// interpolate.ts's faker pass — run before env-var substitution so an env
// var's own value can't accidentally look like a faker call.
func expandFaker(value string) string {
	return fakerPattern.ReplaceAllStringFunc(value, func(match string) string {
		sub := fakerPattern.FindStringSubmatch(match)
		module, method := sub[1], sub[2]
		if fns, ok := fakerFuncs[module]; ok {
			if fn, ok := fns[method]; ok {
				return fn()
			}
		}
		return match
	})
}
