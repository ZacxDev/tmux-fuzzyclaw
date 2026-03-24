package tmux

import (
	"strconv"
	"strings"
)

// Window represents a parsed tmux window from list-windows.
type Window struct {
	SessionName string
	Target      string // session:index
	WindowID    string // @42
	WindowName  string
	Dir         string // basename of pane_current_path
	FullCwd     string // full pane_current_path
	Command     string // pane_current_command
	WindowFlags string
	Activity    int64
	BellFlag    bool
	Active      bool
}

// CleanID returns the window ID with @ and % stripped (for filenames).
func (w *Window) CleanID() string {
	r := strings.NewReplacer("@", "", "%", "")
	return r.Replace(w.WindowID)
}

// ListAllWindows returns all windows across all tmux sessions.
func ListAllWindows() ([]Window, error) {
	format := strings.Join([]string{
		"#{session_name}",
		"#{session_name}:#{window_index}",
		"#{window_id}",
		"#{window_name}",
		"#{b:pane_current_path}",
		"#{window_activity}",
		"#{pane_current_path}",
		"#{pane_current_command}",
		"#{window_bell_flag}",
		"#{window_active}",
		"#{window_flags}",
	}, "\t")

	out, err := Run("list-windows", "-a", "-F", format)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var windows []Window
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 11 {
			continue
		}
		activity, _ := strconv.ParseInt(fields[5], 10, 64)
		w := Window{
			SessionName: fields[0],
			Target:      fields[1],
			WindowID:    fields[2],
			WindowName:  fields[3],
			Dir:         fields[4],
			Activity:    activity,
			FullCwd:     fields[6],
			Command:     fields[7],
			BellFlag:    fields[8] == "1",
			Active:      fields[9] == "1",
			WindowFlags: fields[10],
		}
		windows = append(windows, w)
	}
	return windows, nil
}

// ListSessionWindows returns windows for a specific session.
func ListSessionWindows(session string) ([]Window, error) {
	format := strings.Join([]string{
		"#{window_id}",
		"#{window_index}",
		"#{window_name}",
		"#{window_flags}",
		"#{window_bell_flag}",
	}, "|")

	out, err := Run("list-windows", "-t", session, "-F", format)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var windows []Window
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 5 {
			continue
		}
		w := Window{
			SessionName: session,
			WindowID:    fields[0],
			WindowName:  fields[2],
			WindowFlags: fields[3],
			BellFlag:    fields[4] == "1",
		}
		idx, _ := strconv.Atoi(fields[1])
		w.Target = session + ":" + strconv.Itoa(idx)
		windows = append(windows, w)
	}
	return windows, nil
}
