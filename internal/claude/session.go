package claude

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectDir returns the Claude projects directory for a given cwd.
// Claude encodes the path by replacing / with -.
func ProjectDir(claudeProjectsDir, cwd string) string {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	return filepath.Join(claudeProjectsDir, encoded)
}

// LatestSessionFile returns the most recently modified JSONL file in a project dir.
func LatestSessionFile(claudeProjectsDir, cwd string) (string, error) {
	pdir := ProjectDir(claudeProjectsDir, cwd)
	entries, err := os.ReadDir(pdir)
	if err != nil {
		return "", err
	}

	type fileInfo struct {
		path    string
		modTime int64
	}
	var jsonls []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		jsonls = append(jsonls, fileInfo{
			path:    filepath.Join(pdir, e.Name()),
			modTime: info.ModTime().UnixNano(),
		})
	}
	if len(jsonls) == 0 {
		return "", os.ErrNotExist
	}
	sort.Slice(jsonls, func(i, j int) bool {
		return jsonls[i].modTime > jsonls[j].modTime
	})
	return jsonls[0].path, nil
}

// AllSessionFiles returns all JSONL files in a project dir, sorted newest first.
func AllSessionFiles(claudeProjectsDir, cwd string) ([]string, error) {
	pdir := ProjectDir(claudeProjectsDir, cwd)
	entries, err := os.ReadDir(pdir)
	if err != nil {
		return nil, err
	}

	type fileInfo struct {
		path    string
		modTime int64
	}
	var jsonls []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		jsonls = append(jsonls, fileInfo{
			path:    filepath.Join(pdir, e.Name()),
			modTime: info.ModTime().UnixNano(),
		})
	}
	sort.Slice(jsonls, func(i, j int) bool {
		return jsonls[i].modTime > jsonls[j].modTime
	})
	paths := make([]string, len(jsonls))
	for i, f := range jsonls {
		paths[i] = f.path
	}
	return paths, nil
}

// SessionFile returns the path to a specific session's JSONL file.
func SessionFile(claudeProjectsDir, cwd, sessionID string) (string, error) {
	pdir := ProjectDir(claudeProjectsDir, cwd)
	path := filepath.Join(pdir, sessionID+".jsonl")
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}
