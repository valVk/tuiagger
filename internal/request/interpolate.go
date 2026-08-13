// Package request builds and executes HTTP requests from a resolved
// OpenAPI operation plus user-supplied parameter values, mirroring
// requestBuilder.ts / urlBuilder.ts / parameterCollector.ts / curlGenerator.ts
// / useRequest.ts's execute logic.
package request

import "regexp"

var envVarPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}`)

// Interpolate substitutes {{envVarName}} references, matching interpolate.ts.
// It resolves chained references (an env var whose value is itself
// {{another}}) up to 10 passes, same as the TS version. Unresolved
// references are left intact rather than blanked out.
//
// interpolate.ts also expands {{faker.module.method()}} expressions before
// this substitution; that's deferred to Phase 6 (Faker interpolation isn't
// scoped until then) — the faker pattern contains dots and parens so it
// doesn't match envVarPattern and passes through untouched today, meaning
// no rework is needed here when Phase 6 adds it as a preprocessing step.
func Interpolate(value string, envVars map[string]string) string {
	if envVars == nil {
		return value
	}
	result := value
	for range 10 {
		next := envVarPattern.ReplaceAllStringFunc(result, func(match string) string {
			key := envVarPattern.FindStringSubmatch(match)[1]
			if v, ok := envVars[key]; ok {
				return v
			}
			return match
		})
		if next == result {
			break
		}
		result = next
	}
	return result
}
