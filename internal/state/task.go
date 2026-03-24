package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TaskState matches the existing JSON format written by task-hook.sh.
type TaskState struct {
	Task          string `json:"task"`
	WindowID      string `json:"window_id"`
	TmuxSession   string `json:"tmux_session"`
	WindowIndex   int    `json:"window_index"`
	Status        string `json:"status"`
	Cwd           string `json:"cwd"`
	ClaudeSession string `json:"claude_session"`
	Started       string `json:"started"`
	LastActivity  string `json:"last_activity"`
	Summary       string `json:"summary"`
}

// ReadTask reads a task state file for a given window ID.
func ReadTask(stateDir, windowID string) (*TaskState, error) {
	cleanID := cleanWindowID(windowID)
	path := filepath.Join(stateDir, cleanID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t TaskState
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// WriteTask writes a task state file.
func WriteTask(stateDir string, task *TaskState) error {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}
	cleanID := cleanWindowID(task.WindowID)
	path := filepath.Join(stateDir, cleanID+".json")
	data, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadAllTasks reads all task state files from the state directory.
func ReadAllTasks(stateDir string) (map[string]*TaskState, error) {
	tasks := make(map[string]*TaskState)
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return tasks, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(stateDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var t TaskState
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		key := strings.TrimSuffix(e.Name(), ".json")
		tasks[key] = &t
	}
	return tasks, nil
}

func cleanWindowID(id string) string {
	r := strings.NewReplacer("@", "", "%", "")
	return r.Replace(id)
}
