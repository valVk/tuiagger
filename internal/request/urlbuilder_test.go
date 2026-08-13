package request

import (
	"testing"

	"github.com/valVK/tuiagger/internal/storage"
)

func TestBuildRequestURLJoinsBaseAndPath(t *testing.T) {
	got, err := BuildRequestURL("http://api.test", "/widgets", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://api.test/widgets" {
		t.Errorf("got %q", got)
	}
}

func TestBuildRequestURLHandlesMissingTrailingSlash(t *testing.T) {
	got, err := BuildRequestURL("http://api.test/v1", "widgets/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://api.test/v1/widgets/1" {
		t.Errorf("got %q", got)
	}
}

func TestBuildRequestURLAppendsEnabledQueryParamsOnly(t *testing.T) {
	got, err := BuildRequestURL("http://api.test", "/widgets", []storage.KeyValuePair{
		{Key: "status", Value: "available", Enabled: true},
		{Key: "off", Value: "x", Enabled: false},
		{Key: "", Value: "x", Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://api.test/widgets?status=available" {
		t.Errorf("got %q", got)
	}
}
