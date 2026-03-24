#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${FUZZYCLAW_DIR:-$HOME/.config/tmux}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)/scripts"

echo "Installing fuzzyclaw to $INSTALL_DIR"

# Check dependencies
missing=()
command -v tmux >/dev/null || missing+=(tmux)
command -v fzf >/dev/null || missing+=(fzf)
command -v jq >/dev/null || missing+=(jq)
command -v rg >/dev/null || missing+=(ripgrep)

if (( ${#missing[@]} > 0 )); then
    echo "Missing dependencies: ${missing[*]}"
    echo "Install them first, then re-run this script."
    exit 1
fi

# Check bash version (need 4+ for associative arrays)
if (( BASH_VERSINFO[0] < 4 )); then
    echo "bash >= 4.0 required (found $BASH_VERSION)"
    exit 1
fi

mkdir -p "$INSTALL_DIR"

# Copy scripts
for script in "$SCRIPT_DIR"/*.sh; do
    name=$(basename "$script")
    cp "$script" "$INSTALL_DIR/$name"
    chmod +x "$INSTALL_DIR/$name"
    echo "  installed $name"
done

# Create state directories
mkdir -p "$HOME/.tmux/tasks"
mkdir -p "$HOME/.tmux/activity"

echo ""
echo "Scripts installed to $INSTALL_DIR"
echo ""
echo "Add the following to your ~/.tmux.conf:"
echo ""
cat <<'TMUX'
# fuzzyclaw: task dashboard
bind-key -n M-c display-popup -E -w 90% -h 70% -T ' tasks ' \
    '~/.config/tmux/task-dashboard.sh'

# fuzzyclaw: window auto-rename with Claude indicator
set -g automatic-rename-format '#{b:pane_current_path}#{?#{m:claude*,#{pane_current_command}}, ●,}'

# fuzzyclaw: idle tracking (optional, enables color-coded window tabs)
set-hook -g after-select-window 'run-shell -b "~/.config/tmux/pipe-activity.sh switch"'
set-hook -g window-linked 'run-shell -b "~/.config/tmux/pipe-activity.sh linked"'
set-hook -g session-created 'run-shell -b "~/.config/tmux/pipe-activity.sh init"'
TMUX
echo ""
echo "Then reload: tmux source-file ~/.tmux.conf"
echo ""
echo "For Claude Code integration, see README.md"
