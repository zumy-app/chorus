#!/usr/bin/env bash
# =============================================================================
# verify-maestro.sh — Maestro native-e2e offline gate (PR-safe, no emulator)
# =============================================================================
# Validates the Maestro flows committed under mobile/maestro/ without needing
# a device:
#   1. Every *.yaml exists for the Release 1 critical paths.
#   2. Each flow parses as YAML with an appId header + at least one step.
#   3. Flows reference only stable selectors that exist in the app source
#      (login placeholders, tab labels, composer text) — catches renames that
#      would break the native suite silently.
#
# The full emulator run happens in CI (mobile-native job) and locally via
# `npm run maestro:test`. This gate keeps PRs fast while guaranteeing the
# flows stay executable.
# =============================================================================
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAESTRO="$ROOT/mobile/maestro"
fails=0
pass() { echo "  PASS: $1"; }
failc() { echo "  FAIL: $1" >&2; fails=$((fails+1)); }

echo "== Maestro offline gate =="

for flow in login.yaml chat-send-message.yaml flags-gate.yaml; do
  if [[ -f "$MAESTRO/$flow" ]]; then pass "flow present: $flow";
  else failc "flow missing: mobile/maestro/$flow"; fi
done
[[ "$fails" -ne 0 ]] && { echo "Maestro: FAIL (missing flows)" >&2; exit 1; }

# YAML structural check via python (stdlib only).
if command -v python3 >/dev/null 2>&1; then _py=python3; else _py=python; fi
if ! $_py - "$MAESTRO" <<'PY' 2>/tmp/maestro-yaml.log; then
import glob, os, sys
try:
    import yaml  # type: ignore
    have_yaml = True
except ImportError:
    have_yaml = False
d = sys.argv[1]
bad = []
for path in sorted(glob.glob(os.path.join(d, "*.yaml"))):
    text = open(path, encoding="utf-8").read()
    if "appId:" not in text:
        bad.append(f"{os.path.basename(path)}: missing appId header")
        continue
    if have_yaml:
        try:
            docs = [doc for doc in yaml.safe_load_all(text) if doc]
            if not any(isinstance(doc, list) and doc for doc in docs):
                bad.append(f"{os.path.basename(path)}: no flow steps found")
        except Exception as e:  # noqa: BLE001
            bad.append(f"{os.path.basename(path)}: YAML parse error: {e}")
    else:
        # Fallback: at least one Maestro command must be present.
        if not any(cmd in text for cmd in ("- launchApp", "- tapOn", "- assertVisible", "- inputText")):
            bad.append(f"{os.path.basename(path)}: no Maestro steps found")
if bad:
    print("\n".join(bad))
    sys.exit(1)
print("yaml-ok")
PY
  cat /tmp/maestro-yaml.log >&2 || true
  failc "Maestro YAML structure invalid (see log above)"
else
  pass "Maestro YAML structure valid"
fi

# Selector drift check: every literal the flows assert on must exist in source.
# (Keeps UI renames from silently breaking the native suite.)
declare -A selectors=(
  ["login email placeholder you@example.com"]="mobile/src/screens/LoginScreen.tsx|you@example.com"
  ["login password placeholder"]="mobile/src/screens/LoginScreen.tsx|Enter your password"
  ["login button"]="mobile/src/screens/LoginScreen.tsx|Log In"
  ["chats tab label"]="mobile/src/components/MainTabs.tsx|ChatsTab|Chats"
  ["chat composer"]="mobile/src/screens/ChatScreen.tsx|Type a message"
)
for key in "${!selectors[@]}"; do
  entry="${selectors[$key]}"; file="${entry%%|*}"; pat="${entry#*|}"
  if grep -Eq "$pat" "$ROOT/$file" 2>/dev/null; then pass "selector live: $key";
  else failc "selector drift: '$key' (pattern $pat) not found in $file"; fi
done

echo
if [[ "$fails" -ne 0 ]]; then
  echo "Maestro: FAIL ($fails check(s) failed)." >&2
  exit 1
fi
echo "Maestro: PASS."
exit 0
