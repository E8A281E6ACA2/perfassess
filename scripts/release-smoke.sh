#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./build/perfassess}"

if [[ ! -x "$binary" ]]; then
  echo "release smoke failed: binary is not executable: $binary" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

"$binary" version >"$tmpdir/version.txt"
grep -q '^perfassess ' "$tmpdir/version.txt"

"$binary" --help >"$tmpdir/help.txt"
grep -q 'check-deps' "$tmpdir/help.txt"
grep -q 'version' "$tmpdir/help.txt"

"$binary" check-deps >"$tmpdir/check-deps.txt"
grep -q '外部依赖检查' "$tmpdir/check-deps.txt"

"$binary" --quick --output-format json -o "$tmpdir/quick.json" >"$tmpdir/quick.stdout"
python3 -m json.tool "$tmpdir/quick.json" >/dev/null
python3 - "$tmpdir/quick.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    report = json.load(f)

summary = report.get("summary", {})
required = [
    "benchmark_profile",
    "confidence_level",
    "score_breakdown",
    "vps_benchmark_summary",
    "share_templates",
]
missing = [key for key in required if key not in summary]
if missing:
    raise SystemExit(f"missing summary fields: {missing}")

share = summary["share_templates"]
if not share.get("plain_text") or not share.get("markdown"):
    raise SystemExit("missing share templates")
PY

"$binary" -b cpu --output-format json -o "$tmpdir/cpu.json" >"$tmpdir/cpu.stdout"
python3 -m json.tool "$tmpdir/cpu.json" >/dev/null

"$binary" compare "$tmpdir/quick.json" "$tmpdir/cpu.json" --format json >"$tmpdir/compare.json"
python3 -m json.tool "$tmpdir/compare.json" >/dev/null

"$binary" history add "$tmpdir/quick.json" --store "$tmpdir/history.jsonl" >"$tmpdir/history-add-quick.txt"
"$binary" history add "$tmpdir/cpu.json" --store "$tmpdir/history.jsonl" >"$tmpdir/history-add-cpu.txt"
"$binary" history list --store "$tmpdir/history.jsonl" --format json >"$tmpdir/history-list.json"
python3 -m json.tool "$tmpdir/history-list.json" >/dev/null
"$binary" history trend --store "$tmpdir/history.jsonl" --format json >"$tmpdir/history-trend.json"
python3 -m json.tool "$tmpdir/history-trend.json" >/dev/null

echo "release smoke passed: $binary"
