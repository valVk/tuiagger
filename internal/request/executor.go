package request

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/valVK/tuiagger/internal/openapi"
)

// Response mirrors types/request.ts's ResponseState.
type Response struct {
	Status         int
	StatusText     string
	Headers        map[string]string
	Body           string
	TimeMs         int64
	Error          string
	RequestMethod  string
	RequestURL     string
	RequestHeaders map[string]string
	RequestBody    string
}

// Execute builds and sends spec's request, matching useRequest.ts's
// execute — build/network/decode failures are captured into Response.Error
// rather than returned as a Go error, since (like the TS version) the
// caller always wants a Response to render, error or not. Curl is returned
// separately so it's available even when the build itself fails.
func Execute(client HTTPClient, spec Spec) (*Response, string) {
	built, err := Build(spec)
	if err != nil {
		return &Response{Status: 0, StatusText: "Error", Error: err.Error(), RequestMethod: strings.ToUpper(spec.Method)}, ""
	}

	headerMap := make(map[string]string, len(built.Headers))
	for _, h := range built.Headers {
		headerMap[h.Name] = h.Value
	}

	start := time.Now()

	httpReq, err := http.NewRequest(built.Method, built.URL, strings.NewReader(built.Body))
	if err != nil {
		return &Response{
			Status: 0, StatusText: "Error", Error: err.Error(),
			TimeMs:        time.Since(start).Milliseconds(),
			RequestMethod: built.Method, RequestURL: built.URL,
			RequestHeaders: headerMap, RequestBody: built.Body,
		}, built.Curl
	}
	for _, h := range built.Headers {
		httpReq.Header.Set(h.Name, h.Value)
	}

	res, err := client.Do(httpReq)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return &Response{
			Status: 0, StatusText: "Error", Error: err.Error(), TimeMs: elapsed,
			RequestMethod: built.Method, RequestURL: built.URL,
			RequestHeaders: headerMap, RequestBody: built.Body,
		}, built.Curl
	}
	defer res.Body.Close()

	responseHeaders := make(map[string]string, len(res.Header))
	for k := range res.Header {
		responseHeaders[k] = res.Header.Get(k)
	}

	bodyBytes, readErr := io.ReadAll(res.Body)
	var bodyStr string
	if readErr != nil {
		bodyStr = ""
	} else if strings.Contains(res.Header.Get("Content-Type"), "application/json") {
		var v any
		if json.Unmarshal(bodyBytes, &v) == nil {
			pretty, _ := json.MarshalIndent(v, "", "  ")
			bodyStr = string(pretty)
		} else {
			bodyStr = string(bodyBytes)
		}
	} else {
		bodyStr = openapi.HTMLToPlainText(string(bodyBytes))
	}

	return &Response{
		Status:         res.StatusCode,
		StatusText:     http.StatusText(res.StatusCode),
		Headers:        responseHeaders,
		Body:           bodyStr,
		TimeMs:         elapsed,
		RequestMethod:  built.Method,
		RequestURL:     built.URL,
		RequestHeaders: headerMap,
		RequestBody:    built.Body,
	}, built.Curl
}
