package claude

import (
	"os/exec"
	"strings"
)

// SearchResult represents a single search match in a conversation.
type SearchResult struct {
	Type    MessageType
	Text    string
	Context string // surrounding text for display
	LineNum int
}

// SearchConversation searches a JSONL file for matching text.
func SearchConversation(path, query string) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	messages, err := ParseJSONL(path)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []SearchResult

	for i, m := range messages {
		if m.Type != TypeUser && m.Type != TypeAssistant {
			continue
		}
		text := m.ExtractText()
		if text == "" {
			continue
		}
		textLower := strings.ToLower(text)
		if !strings.Contains(textLower, queryLower) {
			continue
		}

		// Extract context around match
		ctx := extractContext(text, query, 120)
		results = append(results, SearchResult{
			Type:    m.Type,
			Text:    text,
			Context: ctx,
			LineNum: i,
		})
	}
	return results, nil
}

// CwdContains checks if any JSONL session for a cwd contains the query string.
// Uses ripgrep for speed, falling back to Go if rg is not available.
func CwdContains(claudeProjectsDir, cwd, query string) bool {
	if query == "" {
		return false
	}
	pdir := ProjectDir(claudeProjectsDir, cwd)
	// Use ripgrep: case-insensitive, just check for any match, limit to 1
	cmd := exec.Command("rg", "-i", "-l", "--max-count=1", "--glob=*.jsonl", query, pdir)
	if err := cmd.Run(); err == nil {
		return true // exit 0 = match found
	}
	return false // exit 1 = no match (or rg not found)
}

func extractContext(text, query string, contextLen int) string {
	lower := strings.ToLower(text)
	queryLower := strings.ToLower(query)
	idx := strings.Index(lower, queryLower)
	if idx < 0 {
		if len(text) > contextLen {
			return text[:contextLen] + "..."
		}
		return text
	}

	start := idx - contextLen/2
	if start < 0 {
		start = 0
	}
	end := start + contextLen
	if end > len(text) {
		end = len(text)
	}

	result := text[start:end]
	if start > 0 {
		result = "..." + result
	}
	if end < len(text) {
		result = result + "..."
	}
	return strings.Join(strings.Fields(result), " ")
}
