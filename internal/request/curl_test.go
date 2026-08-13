package request

import "testing"

func TestGenerateCurlEscapesSingleQuotesInBody(t *testing.T) {
	got := GenerateCurl("post", "http://x", nil, `{"name":"O'Brien"}`)
	if got != `curl -X 'POST' \
  'http://x' \
  -d '{"name":"O'\''Brien"}'` {
		t.Errorf("unexpected curl: %q", got)
	}
}

func TestGenerateCurlIncludesHeadersInOrder(t *testing.T) {
	got := GenerateCurl("get", "http://x", []HeaderPair{{Name: "Accept", Value: "application/json"}, {Name: "Authorization", Value: "Bearer t"}}, "")
	want := "curl -X 'GET' \\\n  'http://x' \\\n  -H 'Accept: application/json' \\\n  -H 'Authorization: Bearer t'"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
