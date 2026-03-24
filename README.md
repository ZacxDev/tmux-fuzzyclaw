# tmux-fuzzyclaw

Fuzzy tmux session dashboard for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Track, search, and switch between AI coding sessions across all tmux windows.

## Features

- **Session dashboard** (Alt+c) — fzf popup listing all tmux windows with task name, directory, idle time, and AI session summary
- **Conversation search** — type to fuzzy-match against full Claude conversation history; preview pane shows matching lines with context
- **Session lifecycle** — automatic status emoji: `●` (claude running) → `⏸` (paused) → `🔄` (resumed)
- **Stale detection** — windows color-coded by idle time (8-color Gruvbox scale from green to gray)
- **"Waiting for input" section** — windows with terminal bell + claude running pinned at top
- **Live preview** — task state, recent prompts, or search matches per highlighted window

## Requirements

- tmux >= 3.0
- bash >= 4.0
- [fzf](https://github.com/junegunn/fzf)
- [jq](https://github.com/jqlang/jq)
- [ripgrep](https://github.com/BurntAnalern/ripgrep)

## Install

```bash
git clone https://github.com/ZacxDev/tmux-fuzzyclaw.git
cd tmux-fuzzyclaw
./install.sh
```

Or manually:

```bash
# Copy scripts
mkdir -p ~/.config/tmux
cp scripts/*.sh ~/.config/tmux/
chmod +x ~/.config/tmux/*.sh

# Add to ~/.tmux.conf
cat <<'EOF' >> ~/.tmux.conf

# fuzzyclaw: task dashboard
bind-key -n M-c display-popup -E -w 90% -h 70% -T ' tasks ' \
    '~/.config/tmux/task-dashboard.sh'

# fuzzyclaw: idle tracking (optional, enables color-coded window tabs)
set -g automatic-rename-format '#{b:pane_current_path}#{?#{m:claude*,#{pane_current_command}}, ●,}'
set-hook -g after-select-window 'run-shell -b "~/.config/tmux/pipe-activity.sh switch"'
set-hook -g window-linked 'run-shell -b "~/.config/tmux/pipe-activity.sh linked"'
set-hook -g session-created 'run-shell -b "~/.config/tmux/pipe-activity.sh init"'
EOF

tmux source-file ~/.tmux.conf
```

### Claude Code hooks (optional)

Add to `~/.claude/settings.json` to enable task tracking:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "~/.config/tmux/task-resume.sh"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "~/.config/tmux/task-hook.sh"
          }
        ]
      }
    ]
  }
}
```

## Dashboard controls

| Key | Action |
|-----|--------|
| Type | Fuzzy search across window names, directories, summaries, and conversation history |
| Enter | Jump to selected window |
| Ctrl+X / Ctrl+D | Kill selected window |
| Up/Down | Navigate list |

## How it works

```
Claude Code running        → auto-rename shows "dirname ●"
Claude Code stops (Stop)   → task-hook.sh renames to "⏸ dirname", writes state JSON
Claude Code resumes (Tool) → task-resume.sh renames to "🔄 dirname"
Alt+c                      → task-dashboard.sh opens fzf popup with all windows
```

### Idle color scale (Gruvbox)

| Idle Time | Color |
|-----------|-------|
| <10 min | bright green `#b8bb26` |
| 10-30 min | green `#98971a` |
| 30-60 min | aqua `#689d6a` |
| 1-2 hr | yellow `#d79921` |
| 2-4 hr | orange `#d65d0e` |
| 4-8 hr | red `#cc241d` |
| 8-24 hr | purple `#b16286` |
| >24 hr | gray `#665c54` |

## State files

- `~/.tmux/tasks/<window_id>.json` — task name, status, cwd, Claude session ID, summary, timestamps
- `~/.tmux/activity/<window_id>` — unix timestamp of last pane output

## License

MIT
