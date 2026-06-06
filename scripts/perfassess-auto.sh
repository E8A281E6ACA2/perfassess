#!/usr/bin/env bash
set -euo pipefail

output_dir="${PERFASSESS_AUTO_DIR:-/tmp/perfassess-auto}"
binary="${PERFASSESS_BINARY:-./build/perfassess}"
skip_build="${PERFASSESS_SKIP_BUILD:-0}"
optional_mode="${PERFASSESS_AUTO_OPTIONAL:-never}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "perfassess auto failed: python3 is required for JSON validation" >&2
  exit 1
fi

if [[ "$skip_build" != "1" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "perfassess auto failed: go is required to build from source" >&2
    echo "Set PERFASSESS_SKIP_BUILD=1 and PERFASSESS_BINARY=/path/to/perfassess to use an existing binary." >&2
    exit 1
  fi
  mkdir -p "$(dirname "$binary")"
  echo "building: $binary"
  go build -o "$binary" cmd/main.go
fi

if [[ ! -x "$binary" ]]; then
  echo "perfassess auto failed: binary is not executable: $binary" >&2
  exit 1
fi

mkdir -p "$output_dir"
rm -f "$output_dir"/*.json "$output_dir"/*.txt "$output_dir"/*.md "$output_dir"/*.log

echo "checking version and dependencies"
"$binary" version >"$output_dir/version.txt"
"$binary" check-deps >"$output_dir/check-deps.txt"

echo "running default benchmark"
"$binary" --output-format json -o "$output_dir/default.json" >"$output_dir/default.stdout.txt"
python3 -m json.tool "$output_dir/default.json" >/dev/null

"$binary" -o "$output_dir/default.txt" >"$output_dir/default-text.stdout.txt"

echo "running quick benchmark"
"$binary" --quick --output-format json -o "$output_dir/quick.json" >"$output_dir/quick.stdout.txt"
python3 -m json.tool "$output_dir/quick.json" >/dev/null

echo "running acceptance"
PERFASSESS_BINARY="$binary" \
PERFASSESS_ACCEPTANCE_DIR="$output_dir/acceptance" \
PERFASSESS_ACCEPTANCE_OPTIONAL="$optional_mode" \
scripts/vps-acceptance.sh >"$output_dir/acceptance.stdout.txt"

python3 - "$output_dir" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])

def load(name):
    with (out / name).open(encoding="utf-8") as f:
        return json.load(f)

default_report = load("default.json")
quick_report = load("quick.json")
default_summary = default_report["summary"]
quick_summary = quick_report["summary"]
acceptance_summary_path = out / "acceptance" / "summary.md"

lines = [
    "# Perfassess Auto Summary",
    "",
    f"- binary_version: {(out / 'version.txt').read_text(encoding='utf-8').strip()}",
    f"- output_dir: {out}",
    f"- default_total_score: {default_summary.get('total_score')}",
    f"- default_grade: {default_summary.get('grade')}",
    f"- default_confidence: {default_summary['confidence_level'].get('level')}",
    f"- default_benchmark_profile: {default_summary['benchmark_profile'].get('name')}",
    f"- default_score_profile: {default_summary.get('score_profile')}",
    f"- calibration_version: {default_summary['score_calibration'].get('version')}",
    f"- quick_total_score: {quick_summary.get('total_score')}",
    f"- quick_grade: {quick_summary.get('grade')}",
    f"- acceptance_summary: {acceptance_summary_path}",
    "",
    "## Output Files",
    "",
    "- default.json",
    "- default.txt",
    "- quick.json",
    "- check-deps.txt",
    "- acceptance/summary.md",
]

share = default_summary.get("share_templates", {}).get("plain_text")
if share:
    lines.extend(["", "## Share Template", "", "```text", share, "```"])

(out / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

echo "perfassess auto passed. Output: $output_dir"
echo "Summary: $output_dir/summary.md"
