package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsFallbackWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	store := s.LoadOverrides()
	if store.Version != "1.0" || len(store.Endpoints) != 0 {
		t.Errorf("expected default store, got %+v", store)
	}
}

func TestLoadReturnsFallbackOnMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	path := s.overridesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := s.LoadOverrides()
	if store.Version != "1.0" {
		t.Errorf("expected fallback on malformed JSON, got %+v", store)
	}
}

func TestSaveEndpointOverrideRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	override := EndpointOverride{
		Params:         map[string]string{"id": "42"},
		CustomParams:   []CustomParameter{},
		DisabledParams: []string{},
	}
	if err := s.SaveEndpointOverride("get", "/pet/{id}", override); err != nil {
		t.Fatal(err)
	}
	got := s.GetEndpointOverride("GET", "/pet/{id}")
	if got == nil {
		t.Fatalf("expected override to round-trip")
	}
	if got.Params["id"] != "42" {
		t.Errorf("expected param id=42, got %+v", got.Params)
	}
	if got.LastUsed == "" {
		t.Errorf("expected LastUsed to be stamped")
	}
}

func TestGetEndpointIDUppercasesMethod(t *testing.T) {
	if got := GetEndpointID("get", "/pet/{id}"); got != "GET /pet/{id}" {
		t.Errorf("unexpected id: %q", got)
	}
}

func TestOverridesAreAtomicSurvivesInterruptedWrite(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	// Write a valid store first.
	if err := s.SaveOverridesStore(OverridesStore{Version: "1.0", Endpoints: map[string]EndpointOverride{
		"GET /x": {Params: map[string]string{}, CustomParams: []CustomParameter{}, DisabledParams: []string{}},
	}}); err != nil {
		t.Fatal(err)
	}

	// Simulate a crash mid-write: a stray .tmp file exists, but the real
	// file was never renamed over. The real file must still load cleanly.
	if err := os.WriteFile(s.overridesPath()+".tmp", []byte("{truncated"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := s.LoadOverrides()
	if len(store.Endpoints) != 1 {
		t.Fatalf("expected the real file untouched by the stray .tmp, got %+v", store)
	}
}

func TestSavedRequestsAreAtomicNotJustPlainWrite(t *testing.T) {
	// Regression guard for the bug found in review: TS's saveSavedRequests
	// used a plain writeFile, not atomicWrite, unlike the other 3 stores.
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddSavedRequest(SavedRequest{Name: "test", Tag: "default"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.savedRequestsPath() + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("expected no leftover .tmp after a clean save")
	}
	// A stray .tmp from an interrupted write must not corrupt the real file.
	if err := os.WriteFile(s.savedRequestsPath()+".tmp", []byte("{truncated"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := s.LoadSavedRequests()
	if len(store.Requests) != 1 {
		t.Fatalf("expected saved request to survive a stray .tmp, got %+v", store)
	}
}

func TestSavedRequestsCRUD(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}

	saved, err := s.AddSavedRequest(SavedRequest{Name: "list pets", Tag: "default", ManualRequestState: ManualRequestState{Method: "GET", Path: "/pets"}})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" {
		t.Fatalf("expected generated ID")
	}

	updated, err := s.UpdateSavedRequest(saved.ID, func(r *SavedRequest) { r.Name = "renamed" })
	if err != nil || updated == nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("expected renamed, got %q", updated.Name)
	}

	deleted, err := s.DeleteSavedRequest(saved.ID)
	if err != nil || !deleted {
		t.Fatalf("expected delete to succeed: %v", err)
	}
	if len(s.LoadSavedRequests().Requests) != 0 {
		t.Errorf("expected no requests left")
	}
}

func TestCustomTagAddIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustomTag(CustomTag{Name: "billing"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustomTag(CustomTag{Name: "billing"}); err != nil {
		t.Fatal(err)
	}
	if len(s.LoadSavedRequests().CustomTags) != 1 {
		t.Errorf("expected duplicate tag add to be a no-op")
	}
}

func TestRenameCustomTagUpdatesTagAndItsRequests(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustomTag(CustomTag{Name: "billing"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddSavedRequest(SavedRequest{Tag: "billing", Name: "Invoice"}); err != nil {
		t.Fatal(err)
	}

	renamed, err := s.RenameCustomTag("billing", "finance")
	if err != nil || !renamed {
		t.Fatalf("expected rename to succeed: %v", err)
	}

	store := s.LoadSavedRequests()
	if len(store.CustomTags) != 1 || store.CustomTags[0].Name != "finance" {
		t.Errorf("expected tag renamed, got %+v", store.CustomTags)
	}
	if store.Requests[0].Tag != "finance" {
		t.Errorf("expected request's tag updated too, got %q", store.Requests[0].Tag)
	}
}

func TestRenameCustomTagRejectsConflictsAndNoop(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustomTag(CustomTag{Name: "billing"}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustomTag(CustomTag{Name: "finance"}); err != nil {
		t.Fatal(err)
	}

	if ok, err := s.RenameCustomTag("billing", "finance"); err != nil || ok {
		t.Errorf("expected rename to a name already in use to be rejected")
	}
	if ok, err := s.RenameCustomTag("billing", "billing"); err != nil || ok {
		t.Errorf("expected unchanged rename to be a no-op")
	}
	if ok, err := s.RenameCustomTag("billing", ""); err != nil || ok {
		t.Errorf("expected empty new name to be rejected")
	}
}

func TestSetCredential(t *testing.T) {
	s, err := NewStore("", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCredential("bearerAuth", "token-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCredential("apiKeyAuth", "key-1"); err != nil {
		t.Fatal(err)
	}
	creds := s.LoadAuth().Credentials
	if creds["bearerAuth"] != "token-1" || creds["apiKeyAuth"] != "key-1" {
		t.Errorf("expected both credentials persisted, got %+v", creds)
	}

	if err := s.SetCredential("bearerAuth", "token-2"); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadAuth().Credentials["bearerAuth"]; got != "token-2" {
		t.Errorf("expected overwrite, got %q", got)
	}
}

func TestEnvironmentCRUD(t *testing.T) {
	s, err := NewStore("", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := s.AddEnvironment("dev"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddEnvironment("prod"); err != nil {
		t.Fatal(err)
	}
	envs := s.LoadEnvironments().Environments
	if len(envs) != 2 || envs[0].Name != "dev" || envs[1].Name != "prod" {
		t.Fatalf("expected two environments in insertion order, got %+v", envs)
	}

	if err := s.SetActiveEnvironment(1); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadEnvironments().ActiveIndex; got != 1 {
		t.Errorf("expected active index 1, got %d", got)
	}

	if err := s.SetEnvironmentVariable(1, "BASE_URL", "https://prod.example.com"); err != nil {
		t.Fatal(err)
	}
	if got := s.LoadEnvironments().Environments[1].Variables["BASE_URL"]; got != "https://prod.example.com" {
		t.Errorf("expected variable set, got %q", got)
	}

	if err := s.DeleteEnvironmentVariable(1, "BASE_URL"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.LoadEnvironments().Environments[1].Variables["BASE_URL"]; ok {
		t.Errorf("expected variable deleted")
	}

	// Deleting the active environment (index 1, the last one) must clamp
	// ActiveIndex back onto the remaining list, matching
	// useEnvironments.ts's deleteEnvironment.
	if err := s.DeleteEnvironment(1); err != nil {
		t.Fatal(err)
	}
	store := s.LoadEnvironments()
	if len(store.Environments) != 1 || store.Environments[0].Name != "dev" {
		t.Fatalf("expected only 'dev' left, got %+v", store.Environments)
	}
	if store.ActiveIndex != 0 {
		t.Errorf("expected active index clamped to 0, got %d", store.ActiveIndex)
	}
}

func TestCollectionScopedVsCwdScopedPaths(t *testing.T) {
	cwd := t.TempDir()
	collectionDir := t.TempDir()

	scoped, err := NewStore(collectionDir, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if got := scoped.overridesPath(); filepath.Dir(got) != collectionDir {
		t.Errorf("expected overrides scoped to collection dir, got %q", got)
	}
	// Saved requests are never collection-scoped, even with a collection active.
	if got := scoped.savedRequestsPath(); got != filepath.Join(cwd, ".tuiagger", "saved-requests.json") {
		t.Errorf("expected saved-requests to stay cwd-scoped, got %q", got)
	}

	unscoped, err := NewStore("", cwd)
	if err != nil {
		t.Fatal(err)
	}
	if got := unscoped.overridesPath(); got != filepath.Join(cwd, ".tuiagger", "overrides.json") {
		t.Errorf("expected overrides to fall back to cwd/.tuiagger, got %q", got)
	}
}
