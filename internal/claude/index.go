package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SessionIndex represents the sessions-index.json file structure.
type SessionIndex struct {
	Version int          `json:"version"`
	Entries []IndexEntry `json:"entries"`
}

// IndexEntry represents a single session entry in the index.
type IndexEntry struct {
	SessionID    string `json:"sessionId"`
	FullPath     string `json:"fullPath"`
	FileMtime    int64  `json:"fileMtime"`
	FirstPrompt  string `json:"firstPrompt"`
	Summary      string `json:"summary,omitempty"`
	MessageCount int    `json:"messageCount"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	GitBranch    string `json:"gitBranch"`
	ProjectPath  string `json:"projectPath"`
	IsSidechain  bool   `json:"isSidechain"`
}

// ReadIndex parses sessions-index.json for a given cwd.
func ReadIndex(claudeProjectsDir, cwd string) (*SessionIndex, error) {
	pdir := ProjectDir(claudeProjectsDir, cwd)
	path := filepath.Join(pdir, "sessions-index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx SessionIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// LatestIndexEntry returns the most recently modified non-sidechain entry.
func LatestIndexEntry(claudeProjectsDir, cwd string) (*IndexEntry, error) {
	idx, err := ReadIndex(claudeProjectsDir, cwd)
	if err != nil {
		return nil, err
	}
	var best *IndexEntry
	for i := range idx.Entries {
		e := &idx.Entries[i]
		if e.IsSidechain {
			continue
		}
		if best == nil || e.Modified > best.Modified {
			best = e
		}
	}
	if best == nil {
		return nil, os.ErrNotExist
	}
	return best, nil
}

// IndexEntryBySessionID looks up an entry by session ID.
func IndexEntryBySessionID(claudeProjectsDir, cwd, sessionID string) (*IndexEntry, error) {
	idx, err := ReadIndex(claudeProjectsDir, cwd)
	if err != nil {
		return nil, err
	}
	for i := range idx.Entries {
		if idx.Entries[i].SessionID == sessionID {
			return &idx.Entries[i], nil
		}
	}
	return nil, os.ErrNotExist
}
