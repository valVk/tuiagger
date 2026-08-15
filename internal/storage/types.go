// Package storage mirrors types/request.ts's persisted-store shapes and
// utils/persistence.ts + utils/collectionResolver.ts's load/save/resolve
// behavior, with one deliberate fix: every store (including saved requests)
// is now written atomically. The TS version's saveSavedRequests skipped the
// tmp-then-rename dance the other three stores used — a real crash-safety
// gap, not just style drift (see architecture review Candidate D).
package storage

type KeyValuePair struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type ManualRequestState struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	QueryParams []KeyValuePair `json:"queryParams"`
	Headers     []KeyValuePair `json:"headers"`
	Body        string         `json:"body"`
	BodyType    string         `json:"bodyType"`
}

type SavedRequest struct {
	ManualRequestState
	ID        string `json:"id"`
	Name      string `json:"name"`
	Tag       string `json:"tag"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type CustomTag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type SavedRequestsStore struct {
	Version    string         `json:"version"`
	Requests   []SavedRequest `json:"requests"`
	CustomTags []CustomTag    `json:"customTags"`
}

// CustomParameter is a user-added parameter not present in the spec.
type CustomParameter struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Value   string `json:"value"`
	In      string `json:"in"` // query | header | path
	Enabled bool   `json:"enabled"`
}

type EndpointOverride struct {
	Params         map[string]string `json:"params"`
	CustomParams   []CustomParameter `json:"customParams"`
	DisabledParams []string          `json:"disabledParams"`
	Body           string            `json:"body,omitempty"`
	// ContentType is the body's format, e.g. "application/json",
	// "application/x-www-form-urlencoded", "application/xml" — empty
	// (the zero value) means "not yet chosen, default to whichever the
	// endpoint's declared content types put first", not "application/json"
	// specifically; the try-it-out session picks the actual default.
	ContentType    string `json:"contentType,omitempty"`
	OverridePath   string `json:"overridePath,omitempty"`
	OverrideMethod string `json:"overrideMethod,omitempty"`
	LastUsed       string `json:"lastUsed"`
}

type OverridesStore struct {
	Version   string                      `json:"version"`
	Endpoints map[string]EndpointOverride `json:"endpoints"`
}

type AuthStore struct {
	Version     string            `json:"version"`
	Credentials map[string]string `json:"credentials"`
}

type Environment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
}

type EnvironmentsStore struct {
	Version      string        `json:"version"`
	Environments []Environment `json:"environments"`
	ActiveIndex  int           `json:"activeIndex"`
}

func defaultSavedRequestsStore() SavedRequestsStore {
	return SavedRequestsStore{Version: "1.0", Requests: []SavedRequest{}, CustomTags: []CustomTag{}}
}

func defaultOverridesStore() OverridesStore {
	return OverridesStore{Version: "1.0", Endpoints: map[string]EndpointOverride{}}
}

func defaultAuthStore() AuthStore {
	return AuthStore{Version: "1.0", Credentials: map[string]string{}}
}

func defaultEnvironmentsStore() EnvironmentsStore {
	return EnvironmentsStore{Version: "1.0", Environments: []Environment{}, ActiveIndex: -1}
}
