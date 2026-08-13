package request

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteSuccessJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	resp, curl := Execute(srv.Client(), Spec{Method: "get", BaseURL: srv.URL, Path: "/"})
	if resp.Status != 200 {
		t.Fatalf("expected 200, got %d (err=%q)", resp.Status, resp.Error)
	}
	if resp.Body != "{\n  \"id\": 1\n}" {
		t.Errorf("expected pretty-printed JSON body, got %q", resp.Body)
	}
	if curl == "" {
		t.Errorf("expected a curl command")
	}
}

func TestExecuteNonJSONResponseStripsHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<p>hi</p>"))
	}))
	defer srv.Close()

	resp, _ := Execute(srv.Client(), Spec{Method: "get", BaseURL: srv.URL, Path: "/"})
	if resp.Body != "hi" {
		t.Errorf("expected HTML stripped, got %q", resp.Body)
	}
}

func TestExecuteNetworkErrorPopulatesErrorField(t *testing.T) {
	resp, _ := Execute(http.DefaultClient, Spec{Method: "get", BaseURL: "http://127.0.0.1:1", Path: "/"})
	if resp.Error == "" {
		t.Errorf("expected an error message for unreachable host")
	}
	if resp.Status != 0 {
		t.Errorf("expected status 0 on network failure, got %d", resp.Status)
	}
}

func TestExecuteSendsRequestHeadersAndBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	resp, _ := Execute(srv.Client(), Spec{
		Method: "post", BaseURL: srv.URL, Path: "/", Body: `{"a":1}`,
	})
	if resp.Status != 201 {
		t.Fatalf("expected 201, got %d", resp.Status)
	}
	if gotBody != `{"a":1}` {
		t.Errorf("expected body sent to server, got %q", gotBody)
	}
}
