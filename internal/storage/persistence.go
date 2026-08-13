package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	storageDirName    = ".tuiagger"
	savedRequestsFile = "saved-requests.json"
	overridesFile     = "overrides.json"
	authFile          = "auth.json"
	environmentsFile  = "environments.json"
)

// Store resolves the on-disk paths for one running instance of tuiagger.
// collectionDir is the directory holding a named collection's spec (e.g.
// ~/.tuiagger/PetStore); it's empty when the spec was loaded from a bare
// local file path or a URL. This replaces the TS version's package-level
// mutable currentCollectionPath with an explicit value threaded by the
// caller — same behavior, no global state to reset between tests.
type Store struct {
	collectionDir string
	cwd           string
}

// NewStore builds a Store rooted at cwd, optionally scoped to a collection
// directory. Passing "" for cwd resolves the real working directory.
func NewStore(collectionDir, cwd string) (*Store, error) {
	if cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		cwd = wd
	}
	return &Store{collectionDir: collectionDir, cwd: cwd}, nil
}

// savedRequestsPath always lives under cwd/.tuiagger — saved requests are
// not collection-scoped, matching the TS version's getSavedRequestsPath.
func (s *Store) savedRequestsPath() string {
	return filepath.Join(s.cwd, storageDirName, savedRequestsFile)
}

func (s *Store) scopedPath(filename string) string {
	if s.collectionDir != "" {
		return filepath.Join(s.collectionDir, filename)
	}
	return filepath.Join(s.cwd, storageDirName, filename)
}

func (s *Store) overridesPath() string    { return s.scopedPath(overridesFile) }
func (s *Store) authPath() string         { return s.scopedPath(authFile) }
func (s *Store) environmentsPath() string { return s.scopedPath(environmentsFile) }

// atomicWriteJSON writes data to path via a tmp-file-then-rename so a crash
// mid-write can never leave a truncated/corrupt store on disk.
func atomicWriteJSON(path string, data any) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// loadJSON reads and decodes path into a T, returning fallback for any
// error (missing file, unreadable, malformed JSON) — matching the TS
// load*'s blanket try/catch.
func loadJSON[T any](path string, fallback T) T {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return fallback
	}
	return v
}

// ============ Saved Requests ============

func (s *Store) LoadSavedRequests() SavedRequestsStore {
	return loadJSON(s.savedRequestsPath(), defaultSavedRequestsStore())
}

func (s *Store) SaveSavedRequests(store SavedRequestsStore) error {
	return atomicWriteJSON(s.savedRequestsPath(), store)
}

func (s *Store) AddSavedRequest(req SavedRequest) (SavedRequest, error) {
	store := s.LoadSavedRequests()
	now := time.Now().UTC().Format(time.RFC3339)
	req.ID = uuid.NewString()
	req.CreatedAt = now
	req.UpdatedAt = now
	store.Requests = append(store.Requests, req)
	if err := s.SaveSavedRequests(store); err != nil {
		return SavedRequest{}, err
	}
	return req, nil
}

func (s *Store) UpdateSavedRequest(id string, updates func(*SavedRequest)) (*SavedRequest, error) {
	store := s.LoadSavedRequests()
	for i := range store.Requests {
		if store.Requests[i].ID == id {
			updates(&store.Requests[i])
			store.Requests[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := s.SaveSavedRequests(store); err != nil {
				return nil, err
			}
			return &store.Requests[i], nil
		}
	}
	return nil, nil
}

func (s *Store) DeleteSavedRequest(id string) (bool, error) {
	store := s.LoadSavedRequests()
	initialLen := len(store.Requests)
	filtered := store.Requests[:0]
	for _, r := range store.Requests {
		if r.ID != id {
			filtered = append(filtered, r)
		}
	}
	store.Requests = filtered
	if len(store.Requests) == initialLen {
		return false, nil
	}
	return true, s.SaveSavedRequests(store)
}

func (s *Store) AddCustomTag(tag CustomTag) error {
	store := s.LoadSavedRequests()
	for _, t := range store.CustomTags {
		if t.Name == tag.Name {
			return nil
		}
	}
	store.CustomTags = append(store.CustomTags, tag)
	return s.SaveSavedRequests(store)
}

// RenameCustomTag renames a custom tag and every saved request tagged with
// it, matching useSavedRequests.ts's renameTag: refuses empty/unchanged
// names and names already in use by another tag or request.
func (s *Store) RenameCustomTag(oldName, newName string) (bool, error) {
	if newName == "" || oldName == newName {
		return false, nil
	}
	store := s.LoadSavedRequests()
	for _, t := range store.CustomTags {
		if t.Name == newName {
			return false, nil
		}
	}
	for _, r := range store.Requests {
		if r.Tag == newName {
			return false, nil
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range store.CustomTags {
		if store.CustomTags[i].Name == oldName {
			store.CustomTags[i].Name = newName
		}
	}
	for i := range store.Requests {
		if store.Requests[i].Tag == oldName {
			store.Requests[i].Tag = newName
			store.Requests[i].UpdatedAt = now
		}
	}
	return true, s.SaveSavedRequests(store)
}

func (s *Store) DeleteCustomTag(tagName string) (bool, error) {
	store := s.LoadSavedRequests()
	initialLen := len(store.CustomTags)
	filtered := store.CustomTags[:0]
	for _, t := range store.CustomTags {
		if t.Name != tagName {
			filtered = append(filtered, t)
		}
	}
	store.CustomTags = filtered
	if len(store.CustomTags) == initialLen {
		return false, nil
	}
	return true, s.SaveSavedRequests(store)
}

// ============ Overrides ============

// GetEndpointID builds the key overrides are stored under, e.g. "GET /pet/{petId}".
func GetEndpointID(method, path string) string {
	return strings.ToUpper(method) + " " + path
}

func (s *Store) LoadOverrides() OverridesStore {
	return loadJSON(s.overridesPath(), defaultOverridesStore())
}

func (s *Store) SaveOverridesStore(store OverridesStore) error {
	return atomicWriteJSON(s.overridesPath(), store)
}

func (s *Store) GetEndpointOverride(method, path string) *EndpointOverride {
	store := s.LoadOverrides()
	if o, ok := store.Endpoints[GetEndpointID(method, path)]; ok {
		return &o
	}
	return nil
}

func (s *Store) SaveEndpointOverride(method, path string, override EndpointOverride) error {
	store := s.LoadOverrides()
	override.LastUsed = time.Now().UTC().Format(time.RFC3339)
	store.Endpoints[GetEndpointID(method, path)] = override
	return s.SaveOverridesStore(store)
}

func (s *Store) DeleteEndpointOverride(method, path string) (bool, error) {
	store := s.LoadOverrides()
	id := GetEndpointID(method, path)
	if _, ok := store.Endpoints[id]; !ok {
		return false, nil
	}
	delete(store.Endpoints, id)
	return true, s.SaveOverridesStore(store)
}

// ============ Auth ============

func (s *Store) LoadAuth() AuthStore {
	return loadJSON(s.authPath(), defaultAuthStore())
}

func (s *Store) SaveAuth(store AuthStore) error {
	return atomicWriteJSON(s.authPath(), store)
}

// ============ Environments ============

func (s *Store) LoadEnvironments() EnvironmentsStore {
	return loadJSON(s.environmentsPath(), defaultEnvironmentsStore())
}

func (s *Store) SaveEnvironments(store EnvironmentsStore) error {
	return atomicWriteJSON(s.environmentsPath(), store)
}
