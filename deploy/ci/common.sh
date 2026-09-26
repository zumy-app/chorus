#!/usr/bin/env bash
# Shared helpers for the Chorus CI/CD quality gates (NFR-26).
# Usage: source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
set -uo pipefail

log()  { printf '\033[1;34m[chore]\033[0m       %s\n' "$*"; }
ok()   { printf '\033[1;32m[GATE PASS]\033[0m  %s\n' "$*"; }
warn() { printf '\033[1;33m[GATE WARN]\033[0m  %s\n' "$*"; }
fail() { printf '\033[1;31m[GATE FAIL]\033[0m  %s\n' "$*" >&2; exit 1; }

# Truthy helper for boolean envs (1/true/TRUE/yes/on).
is_true() { case "${1:-}" in 1|true|TRUE|yes|on) return 0;; *) return 1;; esac; }

# Require an environment variable to be non-empty.
req() {
  local var="$1"
  if [ -z "${!var:-}" ]; then
    fail "required environment variable '$var' is not set"
  fi
}

# Numeric comparisons via awk (float-safe, portable).
float_ge() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a+0 >= b+0)}'; }
float_lt() { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a+0 < b+0)}'; }

# Log a non-zero exit code as a gate failure.
exit_on_nonzero() {
  local step="$1" code="$2"
  if [ "$code" -ne 0 ]; then
    fail "$step exited with code $code"
  fi
}
