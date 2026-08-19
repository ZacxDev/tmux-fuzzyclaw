package claude

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestIndex copies testdata/sessions-index.json into a temp project dir structure.
func setupTestIndex(t *testing.T) (claudeProjectsDir, cwd string) {
	t.Helper()
	cwd = "/tmp/test-project"

	// Create dir structure: claudeProjectsDir / ProjectDir-encoded-cwd /
	claudeProjectsDir = t.TempDir()
	encoded := ProjectDir(claudeProjectsDir, cwd)
	if err := os.MkdirAll(encoded, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy fixture
	src := filepath.Join("testdata", "sessions-index.json")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(encoded, "sessions-index.json")
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}
	return claudeProjectsDir, cwd
}

func TestReadIndex(t *testing.T) {
	dir, cwd := setupTestIndex(t)
	idx, err := ReadIndex(dir, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Version != 1 {
		t.Errorf("expected version 1, got %d", idx.Version)
	}
	if len(idx.Entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(idx.Entries))
	}
}

func TestLatestIndexEntry(t *testing.T) {
	dir, cwd := setupTestIndex(t)
	entry, err := LatestIndexEntry(dir, cwd)
	if err != nil {
		t.Fatal(err)
	}
	// aaa-111 has Modified "2026-01-03T12:00:00.000Z" which is latest non-sidechain
	if entry.SessionID != "aaa-111" {
		t.Errorf("expected aaa-111, got %s", entry.SessionID)
	}
	if entry.Summary != "Built login form with validation" {
		t.Errorf("unexpected summary: %s", entry.Summary)
	}
	if entry.GitBranch != "feature/auth" {
		t.Errorf("unexpected branch: %s", entry.GitBranch)
	}
}

func TestLatestIndexEntry_SkipsSidechain(t *testing.T) {
	dir, cwd := setupTestIndex(t)
	entry, err := LatestIndexEntry(dir, cwd)
	if err != nil {
		t.Fatal(err)
	}
	// ccc-333 is newer but sidechain — should be skipped
	if entry.IsSidechain {
		t.Error("should not return sidechain entry")
	}
}

func TestIndexEntryBySessionID(t *testing.T) {
	dir, cwd := setupTestIndex(t)

	entry, err := IndexEntryBySessionID(dir, cwd, "bbb-222")
	if err != nil {
		t.Fatal(err)
	}
	if entry.FirstPrompt != "fix bug in parser" {
		t.Errorf("unexpected prompt: %s", entry.FirstPrompt)
	}
	if entry.MessageCount != 5 {
		t.Errorf("expected 5 messages, got %d", entry.MessageCount)
	}
}

func TestIndexEntryBySessionID_NotFound(t *testing.T) {
	dir, cwd := setupTestIndex(t)
	_, err := IndexEntryBySessionID(dir, cwd, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestReadIndex_NoFile(t *testing.T) {
	_, err := ReadIndex(t.TempDir(), "/no/such/path")
	if err == nil {
		t.Error("expected error when index file doesn't exist")
	}
}

func TestLatestIndexEntry_NoEntries(t *testing.T) {
	// Create empty index
	dir := t.TempDir()
	cwd := "/empty"
	encoded := ProjectDir(dir, cwd)
	if err := os.MkdirAll(encoded, 0755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"version":1,"entries":[]}`)
	if err := os.WriteFile(filepath.Join(encoded, "sessions-index.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LatestIndexEntry(dir, cwd)
	if err == nil {
		t.Error("expected error for empty entries")
	}
}
