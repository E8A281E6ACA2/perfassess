#!/usr/bin/env bash
set -euo pipefail

binary="${PERFASSESS_BINARY:-./build/perfassess}"
output_dir="${PERFASSESS_ACCEPTANCE_DIR:-/tmp/perfassess-acceptance}"
iperf3_server="${PERFASSESS_IPERF3_SERVER:-}"
iperf3_server_file="${PERFASSESS_IPERF3_SERVER_FILE:-}"
run_optional="${PERFASSESS_ACCEPTANCE_OPTIONAL:-auto}"
acceptance_matrix="${PERFASSESS_ACCEPTANCE_MATRIX:-smoke}"
strict_matrix="${PERFASSESS_ACCEPTANCE_STRICT:-0}"
strict_min_available_memory_mb="${PERFASSESS_ACCEPTANCE_STRICT_MIN_MEM_MB:-768}"
generated_reports=()
optional_reports=()
skipped_reports=()
optional_reports_file="$output_dir/optional-reports.txt"
skipped_reports_file="$output_dir/skipped-reports.txt"

if [[ ! -x "$binary" ]]; then
  echo "VPS acceptance failed: binary is not executable: $binary" >&2
  echo "Build it first, for example: go build -o build/perfassess cmd/main.go" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "VPS acceptance failed: python3 is required for JSON validation" >&2
  exit 1
fi

mkdir -p "$output_dir"
rm -f "$output_dir"/*.json "$output_dir"/*.txt "$output_dir"/summary.md
: >"$optional_reports_file"
: >"$skipped_reports_file"

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

run_json_report() {
  local name="$1"
  shift
  echo "running: $name"
  "$binary" "$@" --output-format json -o "$output_dir/$name.json" >"$output_dir/$name.stdout.txt"
  python3 -m json.tool "$output_dir/$name.json" >/dev/null
  generated_reports+=("$name.json")
}

matrix_selected() {
  local target="$1"
  local item
  IFS=',' read -r -a items <<<"$acceptance_matrix"
  for item in "${items[@]}"; do
    item="${item//[[:space:]]/}"
    case "$item" in
      all) return 0 ;;
      smoke)
        [[ "$target" == "smoke" ]] && return 0
        ;;
      low|standard|full)
        [[ "$target" == "$item" ]] && return 0
        ;;
      "")
        ;;
      *)
        echo "VPS acceptance failed: unsupported PERFASSESS_ACCEPTANCE_MATRIX item: $item" >&2
        echo "Valid values: smoke, low, standard, full, all" >&2
        exit 1
        ;;
    esac
  done
  return 1
}

strict_enabled() {
  case "$strict_matrix" in
    1|true|yes) return 0 ;;
    *) return 1 ;;
  esac
}

matrix_requires_core_complete() {
  matrix_selected low || matrix_selected standard || matrix_selected full
}

available_memory_mb() {
  python3 - <<'PY'
import re
from pathlib import Path

for line in Path("/proc/meminfo").read_text(encoding="utf-8").splitlines():
    if line.startswith("MemAvailable:"):
        match = re.search(r"(\d+)", line)
        if match:
            print(int(match.group(1)) // 1024)
            raise SystemExit(0)
print(0)
PY
}

strict_resource_preflight() {
  local context="${1:-strict matrix}"
  strict_enabled || return 0
  matrix_requires_core_complete || return 0
  local available_mb
  available_mb="$(available_memory_mb)"
  if [[ "$available_mb" =~ ^[0-9]+$ && "$available_mb" -lt "$strict_min_available_memory_mb" ]]; then
    echo "VPS acceptance failed: $context requires at least ${strict_min_available_memory_mb} MB available memory; current MemAvailable is ${available_mb} MB." >&2
    echo "Free memory, add swap, stop other workloads, or run without PERFASSESS_ACCEPTANCE_STRICT=1 if you want a degraded-but-explained report." >&2
    exit 1
  fi
}

skip_or_fail() {
  local name="$1"
  local reason="$2"
  if strict_enabled; then
    echo "VPS acceptance failed: $name skipped in strict mode: $reason" >&2
    exit 1
  fi
  echo "skipped: $name ($reason)"
  skipped_reports+=("$name: $reason")
  printf '%s: %s\n' "$name" "$reason" >>"$skipped_reports_file"
}

require_commands_for_report() {
  local name="$1"
  shift
  local missing=()
  local cmd
  for cmd in "$@"; do
    if ! have_cmd "$cmd"; then
      missing+=("$cmd")
    fi
  done
  if [[ "${#missing[@]}" -gt 0 ]]; then
    skip_or_fail "$name" "missing commands: ${missing[*]}"
    return 1
  fi
  return 0
}

assert_report_contract() {
  local report_path="$1"
  local expected_min_success="$2"
  local require_complete="${3:-0}"
  python3 - "$report_path" "$expected_min_success" "$require_complete" <<'PY'
import json
import sys

path = sys.argv[1]
expected_min_success = int(sys.argv[2])
require_complete = sys.argv[3] in {"1", "true", "yes"}
with open(path, encoding="utf-8") as f:
    report = json.load(f)

summary = report.get("summary", {})
required_summary = [
    "benchmark_profile",
    "confidence_level",
    "score_calibration",
    "score_breakdown",
    "vps_benchmark_summary",
    "assessment_conclusion",
    "module_assessments",
    "share_templates",
]
missing = [key for key in required_summary if key not in summary]
if missing:
    raise SystemExit(f"{path}: missing summary fields: {missing}")

if summary.get("tests_success", 0) < expected_min_success:
    raise SystemExit(f"{path}: tests_success below expected minimum")
if require_complete and summary.get("tests_success", 0) < 4:
    raise SystemExit(f"{path}: strict matrix requires all four core tests to succeed")
if summary.get("tests_success", 0) < 4:
    confidence = summary.get("confidence_level")
    if not isinstance(confidence, dict) or not confidence.get("reasons"):
        raise SystemExit(f"{path}: incomplete core tests must include confidence reasons")
    if not summary.get("performance_note"):
        raise SystemExit(f"{path}: incomplete core tests must include performance_note")

calibration = summary["score_calibration"]
if calibration.get("version") != "2026-06-v1":
    raise SystemExit(f"{path}: unexpected score calibration version")
profiles = calibration.get("profiles", {})
for name in ["vps", "server", "workstation"]:
    if name not in profiles:
        raise SystemExit(f"{path}: missing calibration profile {name}")

share = summary["share_templates"]
if not share.get("plain_text") or not share.get("markdown"):
    raise SystemExit(f"{path}: missing share templates")

conclusion = summary["assessment_conclusion"]
if not conclusion.get("headline") or not conclusion.get("scenario"):
    raise SystemExit(f"{path}: missing assessment conclusion text")
if not conclusion.get("evidence"):
    raise SystemExit(f"{path}: missing assessment conclusion evidence")

modules = summary["module_assessments"]
for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
    if key not in modules:
        raise SystemExit(f"{path}: missing module assessment {key}")
for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
    module = modules.get(key)
    if not isinstance(module, dict):
        raise SystemExit(f"{path}: missing {key} module assessment")
    if not module.get("confidence"):
        raise SystemExit(f"{path}: missing {key} module confidence")
    if module.get("status") != "skipped" and not module.get("evidence"):
        raise SystemExit(f"{path}: missing {key} module evidence")
PY
}

assert_optional_report_contract() {
  local name="$1"
  local report_path="$2"
  local optional_status
  set +e
  python3 - "$report_path" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    report = json.load(f)
summary = report.get("summary", {})
if summary.get("tests_success", 0) > 0:
    raise SystemExit(0)
if (
    (summary.get("tests_skipped", 0) > 0 or summary.get("tests_failed", 0) > 0)
    and summary.get("confidence_level", {}).get("reasons")
    and summary.get("assessment_conclusion", {}).get("evidence")
):
    raise SystemExit(2)
raise SystemExit(1)
PY
  optional_status=$?
  set -e

  if [[ "$optional_status" == "0" ]]; then
    assert_report_contract "$report_path" 1
    optional_reports+=("$name")
    printf '%s\n' "$name" >>"$optional_reports_file"
    return 0
  fi

  if [[ "$optional_status" == "2" ]]; then
    assert_report_contract "$report_path" 0
    skip_or_fail "$name" "optional backend did not complete but report explains why; see $report_path"
    return 0
  fi

  assert_report_contract "$report_path" 1
}

"$binary" version >"$output_dir/version.txt"
grep -q '^perfassess ' "$output_dir/version.txt"

"$binary" check-deps >"$output_dir/check-deps.txt"
grep -q '外部依赖检查' "$output_dir/check-deps.txt"

strict_resource_preflight

run_json_report "quick" --quick
assert_report_contract "$output_dir/quick.json" 1

run_json_report "network" -b network
assert_report_contract "$output_dir/network.json" 1

run_json_report "cpu" -b cpu
assert_report_contract "$output_dir/cpu.json" 1

if matrix_selected low; then
  strict_resource_preflight "low-basic strict matrix"
  run_json_report "low-basic" -b all --network-profile quick
  assert_report_contract "$output_dir/low-basic.json" 3 "$strict_matrix"
fi

if matrix_selected standard; then
  strict_resource_preflight "standard-builtin strict matrix"
  run_json_report "standard-builtin" -b all --network-profile standard --route-trace --streaming --streaming-profile standard --ai-services --ip-quality --security
  assert_report_contract "$output_dir/standard-builtin.json" 3 "$strict_matrix"
fi

if matrix_selected full; then
  if require_commands_for_report "full-mainstream" sysbench fio; then
    strict_resource_preflight "full-mainstream strict matrix"
    run_json_report "full-mainstream" --full
    assert_report_contract "$output_dir/full-mainstream.json" 4 "$strict_matrix"
  fi
fi

"$binary" compare "$output_dir/quick.json" "$output_dir/cpu.json" --format json >"$output_dir/compare.json"
python3 -m json.tool "$output_dir/compare.json" >/dev/null
generated_reports+=("compare.json")
"$binary" compare-dir "$output_dir" --sort-by total --format json >"$output_dir/compare-dir.json"
python3 -m json.tool "$output_dir/compare-dir.json" >/dev/null
generated_reports+=("compare-dir.json")

if [[ "$run_optional" != "never" ]] && have_cmd sysbench; then
  run_json_report "cpu-sysbench" -b cpu --cpu-backend sysbench
  assert_optional_report_contract "cpu-sysbench" "$output_dir/cpu-sysbench.json"
  strict_resource_preflight "memory-sysbench strict optional report"
  run_json_report "memory-sysbench" -b memory --memory-backend sysbench
  assert_optional_report_contract "memory-sysbench" "$output_dir/memory-sysbench.json"
fi

if [[ "$run_optional" != "never" ]] && have_cmd fio; then
  run_json_report "disk-fio" -b disk --disk-backend fio
  assert_optional_report_contract "disk-fio" "$output_dir/disk-fio.json"
fi

if [[ "$run_optional" != "never" ]] && have_cmd speedtest; then
  run_json_report "network-speedtest" -b network --network-backend speedtest
  assert_optional_report_contract "network-speedtest" "$output_dir/network-speedtest.json"
fi

if [[ "$run_optional" != "never" ]] && have_cmd iperf3; then
  if [[ -n "$iperf3_server" ]]; then
    run_json_report "network-iperf3" -b network --network-backend iperf3 --iperf3-server "$iperf3_server"
    assert_optional_report_contract "network-iperf3" "$output_dir/network-iperf3.json"
  elif [[ -n "$iperf3_server_file" ]]; then
    run_json_report "network-iperf3-file" -b network --network-backend iperf3 --iperf3-server-file "$iperf3_server_file"
    assert_optional_report_contract "network-iperf3-file" "$output_dir/network-iperf3-file.json"
  fi
fi

python3 - "$output_dir" "$acceptance_matrix" "${generated_reports[*]}" "$optional_reports_file" "$skipped_reports_file" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
acceptance_matrix = sys.argv[2]
generated_reports = [item for item in sys.argv[3].split() if item]
optional_reports_file = pathlib.Path(sys.argv[4])
skipped_reports_file = pathlib.Path(sys.argv[5])
optional_reports = [line.strip() for line in optional_reports_file.read_text(encoding="utf-8").splitlines() if line.strip()]
skipped_reports = [line.strip() for line in skipped_reports_file.read_text(encoding="utf-8").splitlines() if line.strip()]

def load(name):
    with (out / f"{name}.json").open(encoding="utf-8") as f:
        return json.load(f)

def report_line(filename):
    path = out / filename
    if not path.exists() or not filename.endswith(".json"):
        return None
    with path.open(encoding="utf-8") as f:
        report = json.load(f)
    if not isinstance(report, dict) or not isinstance(report.get("summary"), dict):
        return None
    summary = report.get("summary", {})
    confidence = summary.get("confidence_level", {})
    reasons = confidence.get("reasons") or []
    reason_text = "; ".join(str(item) for item in reasons) if reasons else "none"
    return (
        f"- {filename}: score={summary.get('total_score')} | grade={summary.get('grade')} | "
        f"success={summary.get('tests_success')} | failed={summary.get('tests_failed')} | "
        f"skipped={summary.get('tests_skipped')} | confidence={confidence.get('level')} | "
        f"reasons={reason_text}"
    )

quick = load("quick")
network = load("network")
cpu = load("cpu")
report_lines = [line for name in generated_reports for line in [report_line(name)] if line]
report_coverage = []
for filename in generated_reports:
    path = out / filename
    if not path.exists() or not filename.endswith(".json"):
        continue
    with path.open(encoding="utf-8") as f:
        report = json.load(f)
    if not isinstance(report, dict) or not isinstance(report.get("summary"), dict):
        continue
    summary = report.get("summary", {})
    confidence = summary.get("confidence_level", {})
    report_coverage.append({
        "file": filename,
        "total_score": summary.get("total_score"),
        "grade": summary.get("grade"),
        "tests_success": summary.get("tests_success"),
        "tests_failed": summary.get("tests_failed"),
        "tests_skipped": summary.get("tests_skipped"),
        "confidence": confidence.get("level"),
        "confidence_reasons": confidence.get("reasons") or [],
    })

summary_json = {
    "binary_version": (out / "version.txt").read_text(encoding="utf-8").strip(),
    "output_dir": str(out),
    "acceptance_matrix": acceptance_matrix,
    "quick_total_score": quick["summary"].get("total_score"),
    "quick_grade": quick["summary"].get("grade"),
    "quick_confidence": quick["summary"]["confidence_level"].get("level"),
    "quick_score_profile": quick["summary"].get("score_profile"),
    "quick_calibration_version": quick["summary"]["score_calibration"].get("version"),
    "network_confidence": network["summary"]["confidence_level"].get("level"),
    "cpu_confidence": cpu["summary"]["confidence_level"].get("level"),
    "generated_reports": generated_reports,
    "optional_reports": optional_reports,
    "skipped_reports": skipped_reports,
    "required_reports": ["quick.json", "network.json", "cpu.json", "compare.json", "compare-dir.json"],
    "report_coverage": report_coverage,
}
(out / "summary.json").write_text(json.dumps(summary_json, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

lines = [
    "# VPS Acceptance Summary",
    "",
    f"- binary_version: {summary_json['binary_version']}",
    f"- output_dir: {out}",
    f"- quick_total_score: {quick['summary'].get('total_score')}",
    f"- quick_grade: {quick['summary'].get('grade')}",
    f"- quick_confidence: {quick['summary']['confidence_level'].get('level')}",
    f"- quick_score_profile: {quick['summary'].get('score_profile')}",
    f"- quick_calibration_version: {quick['summary']['score_calibration'].get('version')}",
    f"- network_confidence: {network['summary']['confidence_level'].get('level')}",
    f"- cpu_confidence: {cpu['summary']['confidence_level'].get('level')}",
    f"- acceptance_matrix: {acceptance_matrix}",
    f"- generated_reports: {', '.join(generated_reports) if generated_reports else 'none'}",
    f"- optional_reports: {', '.join(optional_reports) if optional_reports else 'none'}",
    f"- skipped_reports: {'; '.join(skipped_reports) if skipped_reports else 'none'}",
    "",
    "## Required Reports",
    "",
    "- quick.json",
    "- network.json",
    "- cpu.json",
    "- compare.json",
    "- compare-dir.json",
    "",
    "## Report Coverage",
    "",
    *report_lines,
]
(out / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

for required_summary_fragment in \
  "## Report Coverage" \
  "quick.json:" \
  "network.json:" \
  "cpu.json:" \
  "success=" \
  "failed=" \
  "skipped=" \
  "confidence=" \
  "reasons="
do
  grep -q "$required_summary_fragment" "$output_dir/summary.md" || {
    echo "VPS acceptance failed: summary.md missing fragment: $required_summary_fragment" >&2
    exit 1
  }
done

python3 -m json.tool "$output_dir/summary.json" >/dev/null
for required_summary_json_fragment in \
  '"acceptance_matrix"' \
  '"generated_reports"' \
  '"optional_reports"' \
  '"skipped_reports"' \
  '"report_coverage"' \
  '"confidence_reasons"'
do
  grep -q "$required_summary_json_fragment" "$output_dir/summary.json" || {
    echo "VPS acceptance failed: summary.json missing fragment: $required_summary_json_fragment" >&2
    exit 1
  }
done

echo "VPS acceptance passed. Output: $output_dir"
echo "Summary: $output_dir/summary.md"
