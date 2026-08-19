# tmux-fuzzyclaw

Fuzzy tmux session dashboard for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Track, search, and switch between AI coding sessions across all tmux windows.

## Features

- **Session dashboard** (Alt+c) — Bubble Tea TUI listing all tmux windows with task name, directory, idle time, context %, cost, and AI session summary
- **Conversation search** — type to fuzzy-match against session metadata and full Claude conversation history; preview pane shows matching lines with context
- **Session lifecycle** — automatic status emoji: `●` (claude running) → `🔄` (session started) → `⏸` (paused) → `✅` (session ended)
- **Stale detection** — windows color-coded by idle time (8-color Gruvbox scale from green to gray)
- **Statusline telemetry** — live model, cost, context %, lines added/removed from Claude Code's statusline API
- **Status filters** — keys 1-4 to filter by running/paused/waiting/done; 0 to clear
- **Session metadata** — preview shows first prompt, git branch, message count from sessions-index.json
- **Live preview** — task state, telemetry, recent prompts, or search matches per highlighted window
- **Multi-select** — tab to select, ctrl+a to toggle all, ctrl+x to kill selected windows

## Requirements

- Go 1.25+
- tmux >= 3.0
- [ripgrep](https://github.com/BurntSushi/ripgrep)

## Install

### From source

```bash
git clone https://github.com/ZacxDev/tmux-fuzzyclaw.git
cd tmux-fuzzyclaw
go build -o fuzzyclaw .
# Move to somewhere on your PATH
cp fuzzyclaw ~/.local/bin/
```

### Nix

```nix
# In your packages list:
(pkgs.buildGoModule {
  pname = "tmux-fuzzyclaw";
  version = "2.1.0";
  src = pkgs.fetchFromGitHub {
    owner = "ZacxDev";
    repo = "tmux-fuzzyclaw";
    rev = "...";
    sha256 = "...";
  };
  vendorHash = null;
  subPackages = [ "." ];
  postInstall = ''mv $out/bin/tmux-fuzzyclaw $out/bin/fuzzyclaw'';
})
```

### tmux keybinding

Add to `~/.tmux.conf`:

```tmux
# fuzzyclaw dashboard
bind-key -n M-c display-popup -E -w 90% -h 70% -T ' tasks ' 'fuzzyclaw dashboard'

# Idle tracking (optional, enables color-coded window tabs)
set -g automatic-rename-format '#{b:pane_current_path}#{?#{m:claude*,#{pane_current_command}}, ●,}'
```

### Claude Code hooks

Add to `~/.claude/settings.json` to enable full session lifecycle tracking:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "hooks": [
          { "type": "command", "command": "fuzzyclaw hook resume" }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "fuzzyclaw hook stop" }
        ]
      }
    ],
    "SessionStart": [
      {
        "hooks": [
          { "type": "command", "command": "fuzzyclaw hook session-start" }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          { "type": "command", "command": "fuzzyclaw hook session-end" }
        ]
      }
    ],
    "Notification": [
      {
        "hooks": [
          { "type": "command", "command": "fuzzyclaw hook notification" }
        ]
      }
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "fuzzyclaw statusline"
  }
}
```

## Dashboard controls

| Key | Action |
|-----|--------|
| j/k, Up/Down | Navigate list |
| Enter | Jump to selected window |
| / | Start search |
| 1-4 | Filter by status (1=running, 2=paused, 3=waiting, 4=done) |
| 0 | Clear status filter |
| Tab | Toggle selection |
| Ctrl+A | Select/deselect all |
| Ctrl+X | Kill selected window(s) |
| g/G | Jump to top/bottom |
| q, Esc | Quit |

## How it works

```
Claude Code session starts   → SessionStart hook sets "🔄 taskname", status "running"
Claude Code uses a tool       → PreToolUse hook flips ⏸→🔄 if paused
Claude Code stops (waiting)   → Stop hook sets "⏸ taskname", writes summary to task state
Claude Code session ends      → SessionEnd hook sets "✅ taskname", status "done"
Permission prompt             → Notification hook sets status "waiting"
Alt+c                         → Opens dashboard TUI with all windows
```

### Data sources

The dashboard reads session metadata from **sessions-index.json** (5-50KB) instead of parsing full JSONL conversation files (1-10MB). JSONL parsing is used as a fallback when the index doesn't exist, and on-demand for preview panel content and deep search.

### Statusline telemetry

When configured, Claude Code sends statusline JSON to `fuzzyclaw statusline` on each assistant message. This provides live model info, cost tracking, and context window usage. The data is written to telemetry files and displayed in the dashboard's CTX and COST columns, plus the preview panel's Telemetry section.

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

- `~/.tmux/tasks/<window_id>.json` — task name, status, cwd, Claude session ID, transcript path, summary, timestamps
- `~/.tmux/activity/<window_id>` — unix timestamp of last pane output
- `~/.tmux/activity/<session_id>.statusline.json` — statusline telemetry (model, cost, context %, lines changed)
- `~/.claude/projects/<cwd-path>/sessions-index.json` — Claude-managed session metadata (read-only)

## License

MIT
