#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
binary="${1:-${PERFASSESS_BINARY:-$repo_root/build/perfassess}}"

if [[ ! -x "$binary" ]]; then
  echo "acceptance summary smoke failed: binary is not executable: $binary" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

PERFASSESS_BINARY="$binary" \
PERFASSESS_ACCEPTANCE_DIR="$tmpdir/acceptance" \
PERFASSESS_ACCEPTANCE_MATRIX=smoke \
PERFASSESS_ACCEPTANCE_OPTIONAL=never \
"$repo_root/scripts/vps-acceptance.sh" >"$tmpdir/stdout.txt"

summary_md="$tmpdir/acceptance/summary.md"
summary_json="$tmpdir/acceptance/summary.json"

grep -q "## Calibration Readiness" "$summary_md"
grep -q "sample_policy=" "$summary_md"
grep -q "exploratory_only" "$summary_md"

python3 - "$summary_json" <<'PY'
import json
import sys
from pathlib import Path

summary = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
readiness = summary.get("calibration_readiness")
if not isinstance(readiness, list) or not readiness:
    raise SystemExit("calibration_readiness missing or empty")
counts = summary.get("calibration_readiness_counts")
if not isinstance(counts, dict) or not counts:
    raise SystemExit("calibration_readiness_counts missing or empty")
quick = next((item for item in readiness if item.get("file") == "quick.json"), None)
if not quick:
    raise SystemExit("quick.json readiness missing")
if quick.get("level") not in {"exploratory_only", "candidate_eligible", "formal_eligible"}:
    raise SystemExit(f"unexpected quick readiness level: {quick.get('level')}")
if "blockers" not in quick or "warnings" not in quick:
    raise SystemExit("readiness item must include blockers and warnings")
coverage = summary.get("report_coverage")
if not isinstance(coverage, list) or not coverage:
    raise SystemExit("report_coverage missing")
if not isinstance(coverage[0].get("calibration_readiness"), dict):
    raise SystemExit("report coverage missing calibration_readiness")
PY

echo "acceptance summary smoke passed"
