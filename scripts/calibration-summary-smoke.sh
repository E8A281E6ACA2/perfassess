#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
binary="${1:-${PERFASSESS_BINARY:-$repo_root/build/perfassess}}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

PERFASSESS_AUTO_DIR="$tmpdir/auto-a" \
PERFASSESS_AUTO_PROFILE=basic \
PERFASSESS_AUTO_ACCEPTANCE=0 \
PERFASSESS_AUTO_PROGRESS=0 \
PERFASSESS_SKIP_BUILD=1 \
PERFASSESS_BINARY="$binary" \
"$repo_root/scripts/perfassess-auto.sh" >/dev/null

mkdir -p "$tmpdir/sample-a" "$tmpdir/sample-b"
cp "$tmpdir/auto-a/calibration_sample.json" "$tmpdir/sample-a/calibration_sample.json"
cp "$tmpdir/auto-a/calibration_sample.json" "$tmpdir/sample-b/calibration_sample.json"

cp "$tmpdir/auto-a/calibration_sample.json" "$tmpdir/unredacted.json"
python3 - "$tmpdir/unredacted.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
sample = json.loads(path.read_text(encoding="utf-8"))
sample["redacted"] = False
path.write_text(json.dumps(sample, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
if "$repo_root/scripts/calibration-summary.py" "$tmpdir/unredacted.json" --format json -o "$tmpdir/unredacted-summary.json" 2>"$tmpdir/unredacted.err"; then
  echo "expected calibration summary to reject unredacted samples" >&2
  exit 1
fi
grep -q "not marked redacted" "$tmpdir/unredacted.err"

cp "$tmpdir/auto-a/calibration_sample.json" "$tmpdir/sensitive.json"
python3 - "$tmpdir/sensitive.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
sample = json.loads(path.read_text(encoding="utf-8"))
sample.setdefault("environment", {})["public_ip"] = "8.8.8.8"
path.write_text(json.dumps(sample, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
if "$repo_root/scripts/calibration-summary.py" "$tmpdir/sensitive.json" --format json -o "$tmpdir/sensitive-summary.json" 2>"$tmpdir/sensitive.err"; then
  echo "expected calibration summary to reject sensitive samples" >&2
  exit 1
fi
grep -Eq "forbidden key|unexpected keys|public IPv4" "$tmpdir/sensitive.err"

"$repo_root/scripts/calibration-summary.py" "$tmpdir" --format markdown -o "$tmpdir/summary.md"
"$repo_root/scripts/calibration-summary.py" "$tmpdir" --format json -o "$tmpdir/summary.json"
"$repo_root/scripts/calibration-summary.py" "$tmpdir" --format jsonl -o "$tmpdir/samples.jsonl"
"$repo_root/scripts/calibration-summary.py" "$tmpdir" --score-profile server --min-confidence low --format json -o "$tmpdir/filtered.json"
if "$repo_root/scripts/calibration-summary.py" "$tmpdir" --require-policy candidate --format json -o "$tmpdir/policy-candidate.json" 2>"$tmpdir/policy-candidate.err"; then
  echo "expected candidate policy gate to reject tiny builtin smoke samples" >&2
  exit 1
fi
grep -q "does not satisfy required level candidate" "$tmpdir/policy-candidate.err"
if "$repo_root/scripts/calibration-summary.py" "$tmpdir" --require-mainstream --format json -o "$tmpdir/mainstream.json" 2>"$tmpdir/mainstream.err"; then
  echo "expected require-mainstream to reject builtin smoke samples" >&2
  exit 1
fi
grep -q "no calibration samples matched filters" "$tmpdir/mainstream.err"

grep -q "Perfassess Calibration Summary" "$tmpdir/summary.md"
grep -q "overall.total_score" "$tmpdir/summary.md"
grep -q "Dataset Policy" "$tmpdir/summary.md"
grep -q "Calibration Recommendations" "$tmpdir/summary.md"
grep -q "继续收集" "$tmpdir/summary.md"
python3 -m json.tool "$tmpdir/summary.json" >/dev/null
python3 -m json.tool "$tmpdir/filtered.json" >/dev/null
python3 - "$tmpdir/summary.json" "$tmpdir/samples.jsonl" "$tmpdir/filtered.json" <<'PY'
import json
import pathlib
import sys

summary = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
filtered = json.loads(pathlib.Path(sys.argv[3]).read_text(encoding="utf-8"))
if summary.get("schema_version") != "perfassess-calibration-summary-v1":
    raise SystemExit("unexpected calibration summary schema")
if summary.get("sample_count", 0) < 2:
    raise SystemExit("expected at least two calibration samples")
policy = summary.get("policy")
if not isinstance(policy, dict) or policy.get("level") != "exploratory":
    raise SystemExit("expected exploratory dataset policy for smoke samples")
if not policy.get("blockers"):
    raise SystemExit("expected policy blockers for tiny builtin smoke dataset")
if filtered.get("filter_info", {}).get("filters", {}).get("score_profile") != "server":
    raise SystemExit("filtered summary did not record score profile filter")
if filtered.get("filter_info", {}).get("kept_count", 0) < 2:
    raise SystemExit("filtered summary unexpectedly dropped server samples")
overall = summary.get("overall", {}).get("metrics", {}).get("overall", {}).get("total_score", {})
if overall.get("count", 0) < 2 or "p50" not in overall or "p90" not in overall:
    raise SystemExit("missing total score percentiles")
recommendations = summary.get("recommendations")
if not isinstance(recommendations, dict) or not recommendations:
    raise SystemExit("missing calibration recommendations")
server = recommendations.get("server")
if not isinstance(server, dict):
    raise SystemExit("missing server calibration recommendation")
if "baseline_suggestions" not in server:
    raise SystemExit("missing baseline suggestions")
if "memory_read_base_mbps" not in server["baseline_suggestions"]:
    raise SystemExit("missing memory read baseline suggestion")

jsonl_lines = [line for line in pathlib.Path(sys.argv[2]).read_text(encoding="utf-8").splitlines() if line.strip()]
if len(jsonl_lines) != summary["sample_count"]:
    raise SystemExit("jsonl sample count mismatch")
for line in jsonl_lines:
    json.loads(line)
PY

echo "calibration summary smoke passed"
