#!/usr/bin/env bash
set -euo pipefail

binary="${1:-./build/perfassess}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ ! -x "$binary" ]]; then
  echo "release smoke failed: binary is not executable: $binary" >&2
  exit 1
fi
binary="$(cd "$(dirname "$binary")" && pwd)/$(basename "$binary")"

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
    "score_calibration",
    "score_breakdown",
    "vps_benchmark_summary",
    "assessment_conclusion",
    "module_assessments",
    "share_templates",
]
missing = [key for key in required if key not in summary]
if missing:
    raise SystemExit(f"missing summary fields: {missing}")

share = summary["share_templates"]
if not share.get("plain_text") or not share.get("markdown"):
    raise SystemExit("missing share templates")

conclusion = summary["assessment_conclusion"]
if not conclusion.get("headline") or not conclusion.get("scenario"):
    raise SystemExit("missing assessment conclusion text")
if not conclusion.get("evidence"):
    raise SystemExit("missing assessment conclusion evidence")

modules = summary["module_assessments"]
for key in ["network", "route", "ip_quality", "streaming", "ai_services"]:
    if key not in modules:
        raise SystemExit(f"missing module assessment: {key}")
network_module = modules["network"]
if not network_module.get("confidence"):
    raise SystemExit("missing network module confidence")
if network_module.get("status") != "skipped" and not network_module.get("evidence"):
    raise SystemExit("missing network module evidence")
PY

python3 - "$repo_root" "$tmpdir/quick.json" "$tmpdir" <<'PY'
import importlib.util
import json
import pathlib
import sys

repo_root = pathlib.Path(sys.argv[1])
report_path = pathlib.Path(sys.argv[2])
output_dir = pathlib.Path(sys.argv[3])
module_path = repo_root / "scripts" / "perfassess-progress-server.py"
spec = importlib.util.spec_from_file_location("perfassess_progress_server", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

with report_path.open(encoding="utf-8") as f:
    report = json.load(f)

html = module.render_report(report, output_dir)
required = [
    "decision-panel",
    "budget-strip",
    "结论优先摘要",
    "测评预算",
    "适用判断",
    "优先建议",
    "module-health-card",
    "evidence-map",
    "证据地图",
    "核心性能证据",
    "限制与建议",
    "网络质量",
    "报告目录",
]
missing = [item for item in required if item not in html]
if missing:
    raise SystemExit(f"missing web report fragments: {missing}")
PY

"$binary" -b cpu --output-format json -o "$tmpdir/cpu.json" >"$tmpdir/cpu.stdout"
python3 -m json.tool "$tmpdir/cpu.json" >/dev/null

"$binary" compare "$tmpdir/quick.json" "$tmpdir/cpu.json" --format json >"$tmpdir/compare.json"
python3 -m json.tool "$tmpdir/compare.json" >/dev/null
"$binary" compare-dir "$tmpdir" --sort-by total --format json >"$tmpdir/compare-dir.json"
python3 -m json.tool "$tmpdir/compare-dir.json" >/dev/null

PERFASSESS_AUTO_DIR="$tmpdir/auto" \
PERFASSESS_AUTO_PROFILE=basic \
PERFASSESS_AUTO_ACCEPTANCE=0 \
PERFASSESS_AUTO_PROGRESS=0 \
PERFASSESS_SKIP_BUILD=1 \
PERFASSESS_BINARY="$binary" \
  "$repo_root/scripts/perfassess-auto.sh" >/dev/null
python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto" >/dev/null
python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto/perfassess-report.zip" >/dev/null

echo "release smoke passed: $binary"
