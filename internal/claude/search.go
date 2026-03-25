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
// Uses ripgrep for speed.
func CwdContains(claudeProjectsDir, cwd, query string) bool {
	if query == "" {
		return false
	}
	pdir := ProjectDir(claudeProjectsDir, cwd)
	cmd := exec.Command("rg", "-i", "-l", "--max-count=1", "--glob=*.jsonl", query, pdir)
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

// BatchCwdSearch runs a single ripgrep across all given cwd project dirs.
// Returns the set of cwds that contain matches. Single rg call = ~66ms for 1.2GB.
func BatchCwdSearch(claudeProjectsDir, query string, cwds []string) map[string]bool {
	result := make(map[string]bool)
	if query == "" || len(cwds) == 0 {
		return result
	}

	// Build list of existing project dirs and map back to cwds
	var dirs []string
	dirToCwd := make(map[string]string)
	for _, cwd := range cwds {
		pdir := ProjectDir(claudeProjectsDir, cwd)
		dirToCwd[pdir] = cwd
		dirs = append(dirs, pdir)
	}

	// Single rg call across all dirs: returns matching file paths
	args := []string{"-i", "-l", "--max-count=1", "--glob=*.jsonl", query}
	args = append(args, dirs...)
	cmd := exec.Command("rg", args...)
	out, err := cmd.Output()
	if err != nil {
		return result // no matches or rg error
	}

	// Map matching file paths back to cwds
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// Find which project dir this file belongs to
		for pdir, cwd := range dirToCwd {
			if strings.HasPrefix(line, pdir) {
				result[cwd] = true
				break
			}
		}
	}
	return result
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
