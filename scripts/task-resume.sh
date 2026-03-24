#!/usr/bin/env bash
# tmux-task-resume.sh - Claude Code PreToolUse hook (thin wrapper for fuzzyclaw)
# Flips ⏸ → 🔄 on the first tool call after Claude resumes work
exec fuzzyclaw hook resume
