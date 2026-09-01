#!/usr/bin/env bash
set -euo pipefail
doc="docs/TEACHER_VETTING.md"
test -f "$doc" || { echo "missing $doc"; exit 1; }
grep -q "Daniella" "$doc" || { echo "Daniella not in doc"; exit 1; }
for h in "Pipeline" "Rubric" "Recording prompt" "Certificate checklist" "video-call" "Expert roster"; do
  grep -qi "$h" "$doc" || { echo "heading missing: $h"; exit 1; }
done
echo "doc ok"
go vet ./...
go test ./internal/services -run TestTeacherVetting -count=1
echo "harness green"
