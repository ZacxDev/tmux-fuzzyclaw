# tmux-fuzzyclaw

Go-based TUI dashboard for managing Claude Code sessions across tmux windows.

## Project Structure

```
cmd/
  root.go              # Cobra root command + config init
  dashboard.go         # Alt+c dashboard (Bubble Tea TUI)
  hook.go              # Claude Code hook handlers (stop/resume/session-start/session-end/notification)
  statusline.go        # Statusline JSON receiver → telemetry files + formatted display
  pipe.go              # Pipe-pane activity receiver
  search.go            # Conversation search command
  export.go            # Export session data
  idle.go              # Idle color updater
  status.go            # Status bar output
  version.go           # Version command

internal/
  claude/
    index.go           # sessions-index.json parser (fast metadata: firstPrompt, summary, branch)
    index_test.go      # Index tests + testdata fixture
    statusline.go      # Statusline JSON parser (model, cost, context %, rate limits)
    jsonl.go           # JSONL conversation parser (keywords, summary, prompts)
    jsonl_test.go      # JSONL tests
    search.go          # Ripgrep-based conversation search
    search_test.go     # Search tests
    session.go         # Session file discovery (project dir mapping, latest/all/by-ID)
  config/
    config.go          # Config struct + YAML loading + defaults
    keys.go            # Key bindings
    theme.go           # Theme/color config
  state/
    task.go            # Task state JSON persistence (~/.tmux/tasks/<wid>.json)
    task_test.go       # Task tests
    telemetry.go       # Statusline telemetry reader (reads .statusline.json files)
    activity.go        # Activity timestamp files
    watcher.go         # Filesystem watcher
  tmux/
    tmux.go            # Raw tmux commands (display-message, rename-window, etc.)
    window.go          # Window struct + list-windows
    hooks.go           # Tmux hook management
    pipe.go            # Pipe-pane management
  tui/
    app.go             # Bubble Tea app launcher
    messages.go        # Message types, WindowEntry, TaskSnapshot
    messages_test.go   # Message tests
    dashboard/
      model.go         # Main dashboard model (fetch, filter, deep search, preview loading)
      preview.go       # Right-side preview panel (task state, session metadata, prompts, search results)
      table.go         # Table rendering
    styles/            # Lipgloss style definitions
```

## Architecture

### Data Sources (priority order)
1. **sessions-index.json** — per-project session metadata (5-50KB, used in 2s poll loop)
2. **Statusline telemetry** — `~/.tmux/activity/<id>.statusline.json` (model, cost, context %)
3. **Task state files** — `~/.tmux/tasks/<wid>.json` (written by hooks)
4. **JSONL conversation files** — fallback for keywords/summary when index unavailable; on-demand for preview/search

### Dashboard Data Flow
```
fetchWindows() [2s poll]
  ├─ tmux.ListAllWindows()
  ├─ state.ReadAllTasks()
  ├─ state.ReadActivity()
  ├─ state.ReadTelemetryByCwd()        # Statusline telemetry (model, cost, ctx%)
  └─ Per unique cwd:
      ├─ claude.LatestIndexEntry()     # Fast path: sessions-index.json
      └─ claude.ExtractKeywords()      # Fallback: JSONL parsing
```

### Hook Lifecycle
```
SessionStart hook  → status "running", window "🔄 taskname"
PreToolUse hook    → flips ⏸→🔄 (idempotent)
Stop hook          → status "paused", window "⏸ taskname", writes summary
SessionEnd hook    → status "done", window "✅ taskname"
Notification hook  → permission_prompt → status "waiting"
```

### Hook Input (enriched JSON from Claude Code)
Hooks receive JSON on stdin with: `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `stop_hook_active`, `last_assistant_message`, `notification_type`

### Statusline Integration
Claude Code sends statusline JSON to `fuzzyclaw statusline` on each assistant message. The command:
1. Writes telemetry to `~/.tmux/activity/<session_id>.statusline.json`
2. Outputs formatted status: `Model | ctx:████░░░░░░ 40% | $0.42 | +156/-23`

### State Files
- `~/.tmux/tasks/<window_id>.json` — task state (name, status, cwd, session ID, summary, transcript_path, timestamps)
- `~/.tmux/activity/<window_id>` — unix timestamp of last pane output
- `~/.tmux/activity/<session_id>.statusline.json` — statusline telemetry (model, cost, context %, lines)
- `~/.claude/projects/<encoded-cwd>/sessions-index.json` — Claude-managed session metadata (read-only)

## Key Technical Details

- **Emoji lifecycle**: auto-rename shows `●` → SessionStart sets `🔄` → Stop sets `⏸` → SessionEnd sets `✅`
- **Multi-byte UTF-8**: emoji prefix stripping uses `strings.HasPrefix` with full emoji+space strings
- **Performance**: sessions-index.json replaces JSONL parsing in the hot poll loop (<5ms vs 50-200ms)
- **Deep search**: single ripgrep call across all project dirs for async conversation search (~66ms for 1.2GB)
- **Transcript path**: hooks propagate `transcript_path` from Claude Code for direct JSONL file lookup
- **Status filter**: keys 1-4 filter by running/paused/waiting/done; 0 clears filter
- **Telemetry columns**: CTX and COST columns in table; color-coded context bar (green→yellow→orange→red at 40/60/80%)

## Dependencies

- Go 1.25+ (Bubble Tea, Lipgloss, Cobra)
- tmux >= 3.0
- ripgrep (conversation search)

## Rules

- All paths use `os.UserHomeDir()`, never hardcoded
- Hooks must exit fast: PreToolUse <10ms, Stop <50ms
- Dashboard pre-caches per unique cwd, not per window
- JSONL parsing only as fallback when sessions-index.json unavailable
- Task state backward-compatible: new fields use `omitempty`

## TUI / Search Rendering Footguns

- **Ripgrep `-F`**: deep search must pass `-F` for literal matching — without it, queries with
  regex metacharacters (`+ . *`) are treated as regex and match nothing or the wrong thing.
- **Ripgrep exit code 2**: if ANY search path is missing, rg exits 2 and discards ALL stdout
  (even valid matches from other dirs). `BatchCwdSearch` filters non-existent dirs with `os.Stat`
  first and uses `CombinedOutput()` (not `Output()`) to preserve stdout on non-zero exit.
- **Deep search keys on CWD, not index**: matched results stored as `map[string]bool` of CWDs,
  not entry indices — indices go stale on every 2s refresh when entries are rebuilt.
- **Lipgloss `Height()` is a MINIMUM**: it pads short content but does NOT clip overflow. Content
  exceeding the allocated height must be manually truncated (split/slice/rejoin) before `Render()`,
  or long preview content pushes the table off-screen.
- **Lipgloss `RoundedBorder()` adds 2 lines** (top+bottom). Subtract 2 from available space when
  allocating height for bordered content (preview pane uses `contentHeight - 2`).
- **`JoinHorizontal` pads the shorter side** to the taller side's height — if the preview overflows
  its allocation, the table side gets extra padding and visible rows shift down.
- **Edge-based scrolling**: `scrollOffset` viewport moves only when the cursor hits top/bottom edge.
  Center-tracking shifted the visible list on every cursor move. Cursor + `scrollOffset` reset to 0
  on search-query change and on Esc, to prevent stale offsets from hiding top results.
