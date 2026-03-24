---
name: fuzzyclaw
description: "Develop, debug, and maintain the fuzzyclaw tmux task dashboard system"
---

# /fuzzyclaw - Development & Maintenance

## Triggers
- Bug reports with the dashboard, hooks, or idle tracking
- Feature requests for the task management system
- Performance issues with dashboard generation or hook execution
- Questions about fuzzyclaw architecture or configuration

## Usage
```
/fuzzyclaw [scope] [action]

Scopes:
  all              Full system (default)
  dashboard        task-dashboard.sh — fzf popup, search, preview
  hooks            task-hook.sh + task-resume.sh — Claude Code integration
  idle             idle-update.sh + pipe-activity.sh + activity-receiver.sh
  install          install.sh, README, configuration

Actions:
  debug            Investigate and fix a reported issue
  improve          Analyze and implement improvements
  verify           Run verification checks on installed system
  test             Test all components end-to-end
```

## Architecture

### Three Subsystems

**1. Task Management** (Claude Code integration)
- `task-hook.sh` — Stop hook: reads JSON from stdin, sets ⏸ prefix, writes `~/.tmux/tasks/<wid>.json`
- `task-resume.sh` — PreToolUse hook: flips ⏸→🔄 (idempotent, <10ms)
- State files: `~/.tmux/tasks/<window_id>.json` with task, status, cwd, claude_session, summary, timestamps

**2. Idle Tracking** (independent of Claude)
- `pipe-activity.sh` — manages tmux pipe-pane for background windows
- `activity-receiver.sh` — receives piped output, writes unix timestamp (throttled 1/sec)
- `idle-update.sh` — batch color updater, 8-color Gruvbox scale, runs once per status-interval

**3. Dashboard UI** (fzf)
- `task-dashboard.sh` — generates tab-delimited lines, pipes to fzf
- Pre-caches keywords/summaries per unique cwd (not per window) for performance
- Keywords appended in bg-color text for invisible fzf search
- Preview: task state + recent prompts (default) or search matches with context (when query typed)
- `change:refresh-preview` for live preview updates

### Emoji Lifecycle
```
tmux auto-rename: "dirname ●"     (claude process running)
Stop hook:        "⏸ dirname"     (claude finished, waiting)
PreToolUse hook:  "🔄 dirname"    (claude resumed working)
```

### Key Gotchas
- `tmux rename-window` disables automatic-rename for that window
- Multi-byte emoji: use `sed -E 's/^(🔄|⏸|✅) //'` not char classes
- fzf `{q}` in preview must be single-quoted: `QUERY='{q}'`
- fzf `--with-nth` makes fields unsearchable in fzf 0.70+ (use bg-color text instead)
- PreToolUse fires on every tool call — must be fast (<10ms)
- Per-cwd caching avoids duplicate JSONL extraction (13 cwds vs 41 windows)

### Performance Targets
- Dashboard generation: <300ms for 40+ windows
- PreToolUse hook: <10ms
- Stop hook: <50ms
- ripgrep JSONL scan: ~14ms per file for full conversation history

## Verification Checks

```bash
# Scripts installed
ls -la ~/.config/tmux/{task-dashboard,task-hook,task-resume,idle-update,pipe-activity,activity-receiver}.sh

# Dashboard binding
tmux list-keys | grep task-dashboard

# Hook config
jq '.hooks' ~/.claude/settings.json

# Task state
ls ~/.tmux/tasks/*.json 2>/dev/null | head -5
jq . ~/.tmux/tasks/*.json 2>/dev/null | head -20

# Activity tracking
ls ~/.tmux/activity/ | head -5

# Dashboard performance
time bash ~/.config/tmux/task-dashboard.sh --lines 2>/dev/null | wc -l

# Pipe hooks
tmux show-hooks -g | grep -E 'select-window|window-linked|session-created'
```

## Dependencies
- tmux >= 3.0, bash >= 4.0
- fzf, jq, ripgrep
- POSIX coreutils (date, sed, mktemp, sort, tail, head, tr)

## File Locations
| File | Purpose |
|------|---------|
| `scripts/task-dashboard.sh` | Main fzf dashboard |
| `scripts/task-hook.sh` | Claude Code Stop hook |
| `scripts/task-resume.sh` | Claude Code PreToolUse hook |
| `scripts/idle-update.sh` | Batch window color updater |
| `scripts/pipe-activity.sh` | Pipe-pane manager |
| `scripts/activity-receiver.sh` | Activity timestamp writer |
| `install.sh` | Installation script |
| `CLAUDE.md` | Project instructions for Claude Code |
| `README.md` | User-facing documentation |
