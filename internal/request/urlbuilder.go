package request

import (
	"net/url"
	"strings"

	"github.com/valVK/tuiagger/internal/storage"
)

// BuildRequestURL joins baseURL and path (both may or may not have
// slashes) and appends enabled, non-empty query parameters, matching
// urlBuilder.ts's buildManualRequestUrl (the only one of its two exported
// builders actually used by the TS app — buildRequestUrl and
// extractPathParams are unreferenced dead code there and aren't ported).
func BuildRequestURL(baseURL, path string, queryParams []storage.KeyValuePair) (string, error) {
	baseWithSlash := baseURL
	if !strings.HasSuffix(baseWithSlash, "/") {
		baseWithSlash += "/"
	}
	relativePath := strings.TrimPrefix(path, "/")

	base, err := url.Parse(baseWithSlash)
	if err != nil {
		return "", err
	}
	full, err := base.Parse(relativePath)
	if err != nil {
		return "", err
	}

	q := full.Query()
	for _, p := range queryParams {
		if p.Enabled && p.Key != "" {
			q.Add(p.Key, p.Value)
		}
	}
	full.RawQuery = q.Encode()

	return full.String(), nil
}
