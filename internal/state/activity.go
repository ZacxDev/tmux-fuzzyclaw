package state

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ReadActivity reads the activity timestamp for a window.
func ReadActivity(activityDir, windowID string) (time.Time, error) {
	cleanID := cleanWindowID(windowID)
	path := filepath.Join(activityDir, cleanID)
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

// WriteActivity writes the current timestamp to the activity file.
func WriteActivity(activityDir, windowID string) error {
	if err := os.MkdirAll(activityDir, 0755); err != nil {
		return err
	}
	cleanID := cleanWindowID(windowID)
	path := filepath.Join(activityDir, cleanID)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	return os.WriteFile(path, []byte(ts), 0644)
}

// CleanOrphans removes activity files for windows that no longer exist.
func CleanOrphans(activityDir string, validIDs map[string]bool) error {
	entries, err := os.ReadDir(activityDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		if !validIDs[name] {
			os.Remove(filepath.Join(activityDir, name))
		}
	}
	return nil
}
