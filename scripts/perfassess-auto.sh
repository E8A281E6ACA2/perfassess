#!/usr/bin/env bash
set -euo pipefail

output_dir="${PERFASSESS_AUTO_DIR:-/tmp/perfassess-auto}"
binary="${PERFASSESS_BINARY:-./build/perfassess}"
skip_build="${PERFASSESS_SKIP_BUILD:-0}"
optional_mode="${PERFASSESS_AUTO_OPTIONAL:-never}"
show_progress="${PERFASSESS_AUTO_PROGRESS:-1}"
progress_file="${PERFASSESS_PROGRESS_FILE:-$output_dir/progress.json}"
auto_profile="${PERFASSESS_AUTO_PROFILE:-standard}"
extra_args="${PERFASSESS_AUTO_ARGS:-}"
stress_enabled="${PERFASSESS_AUTO_STRESS:-0}"
current_progress_step="prepare"

case "$auto_profile" in
  auto) auto_profile="standard" ;;
  stress) auto_profile="full" ;;
  basic|standard|full) ;;
  *)
    echo "perfassess auto failed: PERFASSESS_AUTO_PROFILE must be auto, basic, standard, or full" >&2
    exit 1
    ;;
esac

if [[ "$auto_profile" == "full" ]]; then
  stress_enabled="1"
fi

default_args=(--output-format json -o "$output_dir/default.json")
default_text_args=(-o "$output_dir/default.txt")
case "$auto_profile" in
  basic)
    ;;
  standard|full)
    default_args+=(--route-trace --streaming --ai-services --ip-quality --security)
    default_text_args+=(--route-trace --streaming --ai-services --ip-quality --security)
    ;;
esac
if [[ "$stress_enabled" == "1" || "$stress_enabled" == "true" ]]; then
  default_args+=(--stress)
  default_text_args+=(--stress)
fi
if [[ -n "$extra_args" ]]; then
  # shellcheck disable=SC2206
  user_args=($extra_args)
  default_args+=("${user_args[@]}")
  default_text_args+=("${user_args[@]}")
fi

progress_update() {
  local step_id="$1"
  local step_status="$2"
  local message="${3:-}"

  [[ -n "$progress_file" ]] || return 0
  mkdir -p "$(dirname "$progress_file")"

  python3 - "$progress_file" "$output_dir" "$step_id" "$step_status" "$message" <<'PY'
import json
import pathlib
import sys
from datetime import datetime, timezone

progress_path = pathlib.Path(sys.argv[1])
output_dir = pathlib.Path(sys.argv[2])
step_id = sys.argv[3]
step_status = sys.argv[4]
message = sys.argv[5]
now = datetime.now(timezone.utc).isoformat()

step_defs = [
    ("prepare", "准备输出目录"),
    ("build", "构建二进制"),
    ("deps", "检查版本和依赖"),
    ("default_json", "自动测评 JSON 报告"),
    ("default_text", "自动测评文本报告"),
    ("quick", "快速测评"),
    ("acceptance", "验收流程"),
    ("summary", "生成汇总"),
]

data = {
    "title": "Perfassess 实时测评",
    "status": "running",
    "message": message,
    "output_dir": str(output_dir),
    "started_at": now,
    "updated_at": now,
    "current_step": step_id,
    "steps": [
        {"id": sid, "label": label, "status": "pending", "message": "", "updated_at": ""}
        for sid, label in step_defs
    ],
}

if progress_path.exists():
    try:
        old = json.loads(progress_path.read_text(encoding="utf-8"))
        if isinstance(old, dict):
            data.update({k: v for k, v in old.items() if k not in {"steps", "updated_at", "current_step", "message", "status"}})
            old_steps = {step.get("id"): step for step in old.get("steps", []) if isinstance(step, dict)}
            for step in data["steps"]:
                if step["id"] in old_steps:
                    step.update(old_steps[step["id"]])
    except json.JSONDecodeError:
        pass

for step in data["steps"]:
    if step["id"] == step_id:
        step["status"] = step_status
        step["message"] = message
        step["updated_at"] = now
        break

statuses = [step["status"] for step in data["steps"]]
if any(status == "failed" for status in statuses):
    overall = "failed"
elif statuses and all(status == "success" for status in statuses):
    overall = "success"
else:
    overall = "running"

artifacts = [
    ("Markdown 摘要", "summary.md"),
    ("默认 JSON 报告", "default.json"),
    ("默认文本报告", "default.txt"),
    ("快速 JSON 报告", "quick.json"),
    ("依赖检查", "check-deps.txt"),
    ("验收摘要", "acceptance/summary.md"),
]

data["status"] = overall
data["message"] = message
data["updated_at"] = now
data["current_step"] = step_id
data["artifacts"] = [
    {"label": label, "path": path, "available": (output_dir / path).exists()}
    for label, path in artifacts
]

tmp = progress_path.with_suffix(progress_path.suffix + ".tmp")
tmp.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
tmp.replace(progress_path)
PY
}

mark_failed() {
  local code="$?"
  if [[ "$code" -ne 0 ]]; then
    progress_update "$current_progress_step" "failed" "步骤失败，退出码: $code"
  fi
  exit "$code"
}

trap mark_failed EXIT

step() {
  echo
  echo "==> $*"
}

run_capture() {
  local step_id="$1"
  local label="$2"
  local stdout_file="$3"
  shift 3

  step "$label"
  current_progress_step="$step_id"
  progress_update "$step_id" "running" "$label"
  if [[ "$show_progress" == "1" ]]; then
    "$@" \
      > >(tee "$stdout_file" | awk '/^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*[[:space:]](DEBUG|INFO|WARN|ERROR)[[:space:]]/ { print; fflush() }' >&2) \
      2> >(tee "$output_dir/${stdout_file##*/}.stderr.log" >&2)
  else
    "$@" >"$stdout_file" 2>"$output_dir/${stdout_file##*/}.stderr.log"
  fi
  progress_update "$step_id" "success" "$label 完成"
}

if ! command -v python3 >/dev/null 2>&1; then
  echo "perfassess auto failed: python3 is required for JSON validation" >&2
  exit 1
fi

mkdir -p "$output_dir"
rm -f "$output_dir"/*.json "$output_dir"/*.txt "$output_dir"/*.md "$output_dir"/*.log
progress_update "prepare" "success" "输出目录已准备: $output_dir"

if [[ "$skip_build" != "1" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "perfassess auto failed: go is required to build from source" >&2
    echo "Set PERFASSESS_SKIP_BUILD=1 and PERFASSESS_BINARY=/path/to/perfassess to use an existing binary." >&2
    exit 1
  fi
  mkdir -p "$(dirname "$binary")"
  step "building: $binary"
  progress_update "build" "running" "构建二进制: $binary"
  go build -o "$binary" cmd/main.go
  progress_update "build" "success" "二进制构建完成"
else
  progress_update "build" "success" "跳过构建，使用已有二进制"
fi

if [[ ! -x "$binary" ]]; then
  progress_update "build" "failed" "二进制不可执行: $binary"
  echo "perfassess auto failed: binary is not executable: $binary" >&2
  exit 1
fi

step "checking version and dependencies"
current_progress_step="deps"
progress_update "deps" "running" "检查版本和依赖"
"$binary" version >"$output_dir/version.txt"
"$binary" check-deps >"$output_dir/check-deps.txt"
progress_update "deps" "success" "版本和依赖检查完成"

run_capture "default_json" "running auto benchmark profile: $auto_profile (json report)" "$output_dir/default.stdout.txt" \
  "$binary" "${default_args[@]}"
python3 -m json.tool "$output_dir/default.json" >/dev/null

run_capture "default_text" "running auto benchmark profile: $auto_profile (text report)" "$output_dir/default-text.stdout.txt" \
  "$binary" "${default_text_args[@]}"

run_capture "quick" "running quick benchmark" "$output_dir/quick.stdout.txt" \
  "$binary" --quick --output-format json -o "$output_dir/quick.json"
python3 -m json.tool "$output_dir/quick.json" >/dev/null

step "running acceptance"
current_progress_step="acceptance"
progress_update "acceptance" "running" "running acceptance"
if [[ "$show_progress" == "1" ]]; then
  PERFASSESS_BINARY="$binary" \
  PERFASSESS_ACCEPTANCE_DIR="$output_dir/acceptance" \
  PERFASSESS_ACCEPTANCE_OPTIONAL="$optional_mode" \
  scripts/vps-acceptance.sh > >(tee "$output_dir/acceptance.stdout.txt") 2> >(tee "$output_dir/acceptance.stderr.log" >&2)
else
  PERFASSESS_BINARY="$binary" \
  PERFASSESS_ACCEPTANCE_DIR="$output_dir/acceptance" \
  PERFASSESS_ACCEPTANCE_OPTIONAL="$optional_mode" \
  scripts/vps-acceptance.sh >"$output_dir/acceptance.stdout.txt" 2>"$output_dir/acceptance.stderr.log"
fi
progress_update "acceptance" "success" "验收流程完成"

current_progress_step="summary"
progress_update "summary" "running" "生成汇总"
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
    f"- auto_profile: {default_summary['benchmark_profile'].get('name')}",
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
progress_update "summary" "success" "汇总已生成"
trap - EXIT

echo "perfassess auto passed. Output: $output_dir"
echo "Summary: $output_dir/summary.md"
