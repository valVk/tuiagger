package request

import "net/http"

// HTTPClient is the DI seam for HTTP execution — this is the Go analogue
// of types/services.ts's HttpClient interface. *http.Client satisfies it
// directly, so production code needs no adapter, and tests can swap in any
// http.RoundTripper-backed fake or a plain function type.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
