package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTask(t *testing.T) {
	task, err := ReadTask(filepath.Join("..", "..", "testdata", "tasks"), "@42")
	if err != nil {
		t.Fatal(err)
	}
	if task.Task != "auth-implementation" {
		t.Errorf("expected task=auth-implementation, got %s", task.Task)
	}
	if task.WindowIndex != 5 {
		t.Errorf("expected window_index=5, got %d", task.WindowIndex)
	}
	if task.Status != "paused" {
		t.Errorf("expected status=paused, got %s", task.Status)
	}
	if task.ClaudeSession != "abc123def456" {
		t.Errorf("expected claude_session=abc123def456, got %s", task.ClaudeSession)
	}
}

func TestWriteAndReadTask(t *testing.T) {
	dir := t.TempDir()
	task := &TaskState{
		Task:          "test-task",
		WindowID:      "@99",
		TmuxSession:   "dev",
		WindowIndex:   3,
		Status:        "paused",
		Cwd:           "/tmp/project",
		ClaudeSession: "session123",
		Started:       "2026-03-24T10:00:00Z",
		LastActivity:  "2026-03-24T11:00:00Z",
		Summary:       "test summary",
	}

	if err := WriteTask(dir, task); err != nil {
		t.Fatal(err)
	}

	// Read it back
	read, err := ReadTask(dir, "@99")
	if err != nil {
		t.Fatal(err)
	}
	if read.Task != task.Task {
		t.Errorf("expected task=%s, got %s", task.Task, read.Task)
	}
	if read.WindowIndex != task.WindowIndex {
		t.Errorf("expected window_index=%d, got %d", task.WindowIndex, read.WindowIndex)
	}
}

func TestReadAllTasks(t *testing.T) {
	tasks, err := ReadAllTasks(filepath.Join("..", "..", "testdata", "tasks"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if _, ok := tasks["42"]; !ok {
		t.Error("expected task key '42'")
	}
}

func TestReadAllTasksMissing(t *testing.T) {
	tasks, err := ReadAllTasks("/nonexistent/dir")
	if err != nil {
		t.Fatal("expected no error for missing directory")
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestCleanOrphans(t *testing.T) {
	dir := t.TempDir()
	// Create some activity files
	os.WriteFile(filepath.Join(dir, "42"), []byte("1234"), 0644)
	os.WriteFile(filepath.Join(dir, "99"), []byte("5678"), 0644)
	os.WriteFile(filepath.Join(dir, ".prev_main"), []byte("@42"), 0644) // dotfile

	valid := map[string]bool{"42": true}
	if err := CleanOrphans(dir, valid); err != nil {
		t.Fatal(err)
	}

	// 42 should still exist
	if _, err := os.Stat(filepath.Join(dir, "42")); err != nil {
		t.Error("expected file 42 to exist")
	}
	// 99 should be removed
	if _, err := os.Stat(filepath.Join(dir, "99")); !os.IsNotExist(err) {
		t.Error("expected file 99 to be removed")
	}
	// dotfile should remain
	if _, err := os.Stat(filepath.Join(dir, ".prev_main")); err != nil {
		t.Error("expected dotfile .prev_main to exist")
	}
}
