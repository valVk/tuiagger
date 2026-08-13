// Package request builds and executes HTTP requests from a resolved
// OpenAPI operation plus user-supplied parameter values, mirroring
// requestBuilder.ts / urlBuilder.ts / parameterCollector.ts / curlGenerator.ts
// / useRequest.ts's execute logic.
package request

import "regexp"

var envVarPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}`)

// Interpolate expands {{faker.module.method()}} expressions and then
// substitutes {{envVarName}} references, matching interpolate.ts's
// two-pass order exactly (faker first, so a faker-generated value can never
// accidentally look like an unresolved env-var placeholder). Env-var
// resolution chases chained references (an env var whose value is itself
// {{another}}) up to 10 passes, same as the TS version. Unresolved
// references — of either kind — are left intact rather than blanked out.
func Interpolate(value string, envVars map[string]string) string {
	result := expandFaker(value)
	if envVars == nil {
		return result
	}
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
