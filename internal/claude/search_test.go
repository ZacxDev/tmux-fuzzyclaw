package claude

import (
	"path/filepath"
	"testing"
)

func TestSearchConversation(t *testing.T) {
	results, err := SearchConversation(filepath.Join(testDataDir, "test.jsonl"), "token")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected search results for 'token'")
	}

	// Check that results contain matching text
	found := false
	for _, r := range results {
		if r.Context != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one result with context")
	}
}

func TestSearchConversationCaseInsensitive(t *testing.T) {
	results, err := SearchConversation(filepath.Join(testDataDir, "test.jsonl"), "JWT")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected case-insensitive match for 'JWT'")
	}
}

func TestSearchConversationNoMatch(t *testing.T) {
	results, err := SearchConversation(filepath.Join(testDataDir, "test.jsonl"), "zzzznonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchConversationEmpty(t *testing.T) {
	results, err := SearchConversation(filepath.Join(testDataDir, "test.jsonl"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}
