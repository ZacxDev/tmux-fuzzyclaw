#!/usr/bin/env bash
# tmux-task-hook.sh - Claude Code Stop hook (thin wrapper for fuzzyclaw)
# Called by Claude Code on every Stop event
exec fuzzyclaw hook stop
