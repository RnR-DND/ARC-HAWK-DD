#!/usr/bin/env bash
# claude-yolo.sh — launch Claude Code with --dangerously-skip-permissions
#
# WARNING: This flag bypasses ALL permission prompts. Claude will run any
# shell command, edit any file, and call any tool without asking. Use only:
#   - In a disposable worktree or container
#   - On a branch (never directly on main)
#   - When you're actively watching the session
#
# Never use this flag against a production checkout with credentials.
set -euo pipefail

cat <<'BANNER'
================================================================================
  ⚠️  YOLO MODE  ⚠️
  Running: claude --dangerously-skip-permissions
  All permission prompts are DISABLED. Claude can run any command, touch any
  file. You are responsible for watching the session and stopping it if it
  goes off the rails.
================================================================================
BANNER

# 10-second escape hatch
for s in 10 9 8 7 6 5 4 3 2 1; do
  printf "\rStarting in %02d seconds... (Ctrl-C to abort) " "$s"
  sleep 1
done
echo

exec claude --dangerously-skip-permissions "$@"
