# tmux-fuzzyclaw

Fuzzy tmux task dashboard for Claude Code. Track, search, and switch between AI coding sessions.

## Project Structure

```
scripts/
  task-dashboard.sh    # Main fzf dashboard (Alt+c popup)
  task-hook.sh         # Claude Code Stop hook — sets ⏸, writes task state JSON
  task-resume.sh       # Claude Code PreToolUse hook — flips ⏸ → 🔄
  idle-update.sh       # Batch window color updater (Gruvbox 8-color idle scale)
  pipe-activity.sh     # Manages pipe-pane for background activity tracking
  activity-receiver.sh # Receives piped pane output, updates activity timestamps
```

## Architecture

Three independent subsystems:

1. **Task Management** (task-dashboard, task-hook, task-resume) — Claude Code integration for session tracking
2. **Idle Tracking** (idle-update, pipe-activity, activity-receiver) — real-time activity timestamps and window color coding
3. **UI Layer** (fzf in task-dashboard) — interactive window selection with live preview and conversation search

## Key Technical Details

- **State files**: `~/.tmux/tasks/<window_id>.json` — task state per window
- **Activity files**: `~/.tmux/activity/<window_id>` — unix timestamp per window
- **Session files**: `~/.claude/projects/<cwd-path>/<uuid>.jsonl` — Claude conversation history (read-only)
- **Emoji lifecycle**: auto-rename shows `●` when claude running → Stop hook sets `⏸` → PreToolUse hook sets `🔄`
- **Multi-byte UTF-8**: emoji stripping uses `sed -E 's/^(🔄|⏸|✅) //'` (alternation, not char classes)
- **fzf search**: keywords appended in bg-color text (`\033[38;2;40;40;40m`) + `--no-hscroll` so fzf searches but doesn't display them
- **Performance target**: dashboard generation <300ms for 40+ windows via per-cwd pre-caching

## Dependencies

- tmux >= 3.0
- bash >= 4.0 (associative arrays)
- fzf (fuzzy finder)
- jq (JSON processor)
- rg / ripgrep (fast text search)

## Rules

- Shebang: `#!/usr/bin/env bash`
- All paths use `$HOME` expansion, never hardcoded user paths
- Scripts must be self-contained with no cross-script imports
- `FUZZYCLAW_DIR` env var overrides default install path (`~/.config/tmux`)
- Hooks must exit fast: PreToolUse <10ms, Stop <50ms
- Dashboard pre-caches per unique cwd, not per window
- Never use `sed` character classes for multi-byte emoji — use alternation
