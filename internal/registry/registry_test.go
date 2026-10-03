package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorktreeEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "nil expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: new(time.Now().Add(-time.Hour)),
			want:      true,
		},
		{
			name:      "not expired",
			expiresAt: new(time.Now().Add(time.Hour)),
			want:      false,
		},
		{
			name:      "just expired",
			expiresAt: new(time.Now().Add(-time.Second)),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &WorktreeEntry{
				ExpiresAt: tt.expiresAt,
			}
			if got := e.IsExpired(); got != tt.want {
				t.Errorf("WorktreeEntry.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegistry_ListExpired(t *testing.T) {
	// Create a temporary registry
	tmpDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	registryPath := filepath.Join(tmpDir, "registry.json")

	pastTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	entries := []*WorktreeEntry{
		{
			Path:         "/path/to/expired1",
			Branch:       "expired-branch-1",
			RegisteredAt: time.Now(),
			ExpiresAt:    &pastTime,
		},
		{
			Path:         "/path/to/not-expired",
			Branch:       "not-expired-branch",
			RegisteredAt: time.Now(),
			ExpiresAt:    &futureTime,
		},
		{
			Path:         "/path/to/expired2",
			Branch:       "expired-branch-2",
			RegisteredAt: time.Now(),
			ExpiresAt:    &pastTime,
		},
		{
			Path:         "/path/to/no-expiration",
			Branch:       "no-expiration-branch",
			RegisteredAt: time.Now(),
			ExpiresAt:    nil,
		},
	}

	// Write entries to registry file
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal entries: %v", err)
	}
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("Failed to write registry: %v", err)
	}

	// Create registry and load
	r := &Registry{
		entries: make(map[string]*WorktreeEntry),
		path:    registryPath,
	}
	if err := r.load(); err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Test ListExpired
	expired := r.ListExpired()
	if len(expired) != 2 {
		t.Errorf("ListExpired() returned %d entries, want 2", len(expired))
	}

	// Check that only expired entries are returned
	expiredPaths := make(map[string]bool)
	for _, e := range expired {
		expiredPaths[e.Path] = true
	}

	if !expiredPaths["/path/to/expired1"] {
		t.Error("ListExpired() missing /path/to/expired1")
	}
	if !expiredPaths["/path/to/expired2"] {
		t.Error("ListExpired() missing /path/to/expired2")
	}
	if expiredPaths["/path/to/not-expired"] {
		t.Error("ListExpired() should not include /path/to/not-expired")
	}
	if expiredPaths["/path/to/no-expiration"] {
		t.Error("ListExpired() should not include /path/to/no-expiration")
	}
}

func TestWorktreeEntry_ExpiresAt_JSONMarshal(t *testing.T) {
	// Test that ExpiresAt is omitted when nil (backwards compatibility)
	entry := &WorktreeEntry{
		Repository:   "https://github.com/test/repo",
		Branch:       "main",
		Path:         "/path/to/worktree",
		RegisteredAt: time.Now(),
		ExpiresAt:    nil,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	// Check that expires_at is not present in JSON
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := m["expires_at"]; ok {
		t.Error("expires_at should be omitted when nil")
	}

	// Test that ExpiresAt is included when set
	expiresAt := time.Now().Add(time.Hour)
	entry.ExpiresAt = &expiresAt

	data, err = json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal entry with expiration: %v", err)
	}

	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := m["expires_at"]; !ok {
		t.Error("expires_at should be present when set")
	}
}

func TestRegistry_UpdatePathAndBranch_PreservesExpiresAt(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	expiresAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	registeredAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	r := &Registry{
		entries: map[string]*WorktreeEntry{
			"/old/path": {
				Repository:   "https://github.com/test/repo",
				Branch:       "feature/old",
				Path:         "/old/path",
				RegisteredAt: registeredAt,
				ExpiresAt:    &expiresAt,
			},
		},
		path: registryPath,
	}

	if err := r.UpdatePathAndBranch("/old/path", "/new/path", "feature/new"); err != nil {
		t.Fatalf("UpdatePathAndBranch() error = %v", err)
	}

	if _, ok := r.Get("/old/path"); ok {
		t.Error("old path should be removed from registry")
	}

	got, ok := r.Get("/new/path")
	if !ok {
		t.Fatal("new path should be registered")
	}
	if got.Branch != "feature/new" {
		t.Errorf("Branch = %s, want feature/new", got.Branch)
	}
	if got.Path != "/new/path" {
		t.Errorf("Path = %s, want /new/path", got.Path)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, expiresAt)
	}
	if !got.RegisteredAt.Equal(registeredAt) {
		t.Errorf("RegisteredAt = %v, want %v", got.RegisteredAt, registeredAt)
	}

	// Reload from disk to confirm persistence.
	reloaded := &Registry{entries: make(map[string]*WorktreeEntry), path: registryPath}
	if err := reloaded.load(); err != nil {
		t.Fatalf("load() error = %v", err)
	}
	got, ok = reloaded.Get("/new/path")
	if !ok {
		t.Fatal("reloaded registry missing new path")
	}
	if got.Branch != "feature/new" {
		t.Errorf("reloaded Branch = %s, want feature/new", got.Branch)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("reloaded ExpiresAt = %v, want %v", got.ExpiresAt, expiresAt)
	}
}

func TestRegistry_UpdatePathAndBranch_MissingIsNoop(t *testing.T) {
	r := &Registry{
		entries: make(map[string]*WorktreeEntry),
		path:    filepath.Join(t.TempDir(), "registry.json"),
	}

	if err := r.UpdatePathAndBranch("/missing", "/new", "branch"); err != nil {
		t.Fatalf("UpdatePathAndBranch() error = %v", err)
	}
	if len(r.entries) != 0 {
		t.Errorf("entries = %d, want 0", len(r.entries))
	}
}

func TestRegistry_UpdatePathAndBranch_SamePathUpdatesBranch(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	r := &Registry{
		entries: map[string]*WorktreeEntry{
			"/same/path": {
				Branch:    "old",
				Path:      "/same/path",
				ExpiresAt: &expiresAt,
			},
		},
		path: filepath.Join(t.TempDir(), "registry.json"),
	}

	if err := r.UpdatePathAndBranch("/same/path", "/same/path", "new"); err != nil {
		t.Fatalf("UpdatePathAndBranch() error = %v", err)
	}
	got, ok := r.Get("/same/path")
	if !ok {
		t.Fatal("entry missing after same-path update")
	}
	if got.Branch != "new" {
		t.Errorf("Branch = %s, want new", got.Branch)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, expiresAt)
	}
}
