#!/usr/bin/env bash
set -euo pipefail

binary="${PERFASSESS_BINARY:-./build/perfassess}"
output_dir="${PERFASSESS_ACCEPTANCE_DIR:-/tmp/perfassess-acceptance}"
iperf3_server="${PERFASSESS_IPERF3_SERVER:-}"
iperf3_server_file="${PERFASSESS_IPERF3_SERVER_FILE:-}"
run_optional="${PERFASSESS_ACCEPTANCE_OPTIONAL:-auto}"

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
rm -f "$output_dir"/*.json "$output_dir"/*.txt "$output_dir"/history.jsonl "$output_dir"/summary.md

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

run_json_report() {
  local name="$1"
  shift
  echo "running: $name"
  "$binary" "$@" --output-format json -o "$output_dir/$name.json" >"$output_dir/$name.stdout.txt"
  python3 -m json.tool "$output_dir/$name.json" >/dev/null
}

assert_report_contract() {
  local report_path="$1"
  local expected_min_success="$2"
  python3 - "$report_path" "$expected_min_success" <<'PY'
import json
import sys

path = sys.argv[1]
expected_min_success = int(sys.argv[2])
with open(path, encoding="utf-8") as f:
    report = json.load(f)

summary = report.get("summary", {})
required_summary = [
    "benchmark_profile",
    "confidence_level",
    "score_calibration",
    "score_breakdown",
    "vps_benchmark_summary",
    "share_templates",
]
missing = [key for key in required_summary if key not in summary]
if missing:
    raise SystemExit(f"{path}: missing summary fields: {missing}")

if summary.get("tests_success", 0) < expected_min_success:
    raise SystemExit(f"{path}: tests_success below expected minimum")

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
PY
}

"$binary" version >"$output_dir/version.txt"
grep -q '^perfassess ' "$output_dir/version.txt"

"$binary" check-deps >"$output_dir/check-deps.txt"
grep -q '外部依赖检查' "$output_dir/check-deps.txt"

run_json_report "quick" --quick
assert_report_contract "$output_dir/quick.json" 1

run_json_report "network" -b network
assert_report_contract "$output_dir/network.json" 1

run_json_report "cpu" -b cpu
assert_report_contract "$output_dir/cpu.json" 1

"$binary" compare "$output_dir/quick.json" "$output_dir/cpu.json" --format json >"$output_dir/compare.json"
python3 -m json.tool "$output_dir/compare.json" >/dev/null

"$binary" history add "$output_dir/quick.json" --store "$output_dir/history.jsonl" >"$output_dir/history-add-quick.txt"
"$binary" history add "$output_dir/cpu.json" --store "$output_dir/history.jsonl" >"$output_dir/history-add-cpu.txt"
"$binary" history list --store "$output_dir/history.jsonl" --format json >"$output_dir/history-list.json"
python3 -m json.tool "$output_dir/history-list.json" >/dev/null
"$binary" history trend --store "$output_dir/history.jsonl" --format json >"$output_dir/history-trend.json"
python3 -m json.tool "$output_dir/history-trend.json" >/dev/null

optional_reports=()
if [[ "$run_optional" != "never" ]] && have_cmd sysbench; then
  run_json_report "cpu-sysbench" -b cpu --cpu-backend sysbench
  assert_report_contract "$output_dir/cpu-sysbench.json" 1
  run_json_report "memory-sysbench" -b memory --memory-backend sysbench
  assert_report_contract "$output_dir/memory-sysbench.json" 1
  optional_reports+=("cpu-sysbench" "memory-sysbench")
fi

if [[ "$run_optional" != "never" ]] && have_cmd fio; then
  run_json_report "disk-fio" -b disk --disk-backend fio
  assert_report_contract "$output_dir/disk-fio.json" 1
  optional_reports+=("disk-fio")
fi

if [[ "$run_optional" != "never" ]] && have_cmd speedtest; then
  run_json_report "network-speedtest" -b network --network-backend speedtest
  assert_report_contract "$output_dir/network-speedtest.json" 1
  optional_reports+=("network-speedtest")
fi

if [[ "$run_optional" != "never" ]] && have_cmd iperf3; then
  if [[ -n "$iperf3_server" ]]; then
    run_json_report "network-iperf3" -b network --network-backend iperf3 --iperf3-server "$iperf3_server"
    assert_report_contract "$output_dir/network-iperf3.json" 1
    optional_reports+=("network-iperf3")
  elif [[ -n "$iperf3_server_file" ]]; then
    run_json_report "network-iperf3-file" -b network --network-backend iperf3 --iperf3-server-file "$iperf3_server_file"
    assert_report_contract "$output_dir/network-iperf3-file.json" 1
    optional_reports+=("network-iperf3-file")
  fi
fi

python3 - "$output_dir" "${optional_reports[@]}" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
optional_reports = sys.argv[2:]

def load(name):
    with (out / f"{name}.json").open(encoding="utf-8") as f:
        return json.load(f)

quick = load("quick")
network = load("network")
cpu = load("cpu")

lines = [
    "# VPS Acceptance Summary",
    "",
    f"- binary_version: {(out / 'version.txt').read_text(encoding='utf-8').strip()}",
    f"- output_dir: {out}",
    f"- quick_total_score: {quick['summary'].get('total_score')}",
    f"- quick_grade: {quick['summary'].get('grade')}",
    f"- quick_confidence: {quick['summary']['confidence_level'].get('level')}",
    f"- quick_score_profile: {quick['summary'].get('score_profile')}",
    f"- quick_calibration_version: {quick['summary']['score_calibration'].get('version')}",
    f"- network_confidence: {network['summary']['confidence_level'].get('level')}",
    f"- cpu_confidence: {cpu['summary']['confidence_level'].get('level')}",
    f"- optional_reports: {', '.join(optional_reports) if optional_reports else 'none'}",
    "",
    "## Required Reports",
    "",
    "- quick.json",
    "- network.json",
    "- cpu.json",
    "- compare.json",
    "- history-list.json",
    "- history-trend.json",
]
(out / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

echo "VPS acceptance passed. Output: $output_dir"
echo "Summary: $output_dir/summary.md"
