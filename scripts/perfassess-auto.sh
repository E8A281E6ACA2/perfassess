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
  [[ "$show_progress" == "1" ]] || return 0
  echo
  echo "==> $*"
}

finish_step() {
  local label="$1"
  [[ "$show_progress" == "1" ]] || return 0
  echo "[OK] $label 完成"
}

run_capture() {
  local step_id="$1"
  local label="$2"
  local stdout_file="$3"
  shift 3

  step "$label"
  current_progress_step="$step_id"
  progress_update "$step_id" "running" "$label"
  "$@" >"$stdout_file" 2>"$output_dir/${stdout_file##*/}.stderr.log"
  progress_update "$step_id" "success" "$label 完成"
  finish_step "$label"
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
  finish_step "building: $binary"
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
finish_step "checking version and dependencies"

run_capture "default_json" "运行自动测评档位: $auto_profile (JSON 报告)" "$output_dir/default.stdout.txt" \
  "$binary" "${default_args[@]}"
python3 -m json.tool "$output_dir/default.json" >/dev/null

run_capture "default_text" "运行自动测评档位: $auto_profile (文本报告)" "$output_dir/default-text.stdout.txt" \
  "$binary" "${default_text_args[@]}"

run_capture "quick" "运行快速测评" "$output_dir/quick.stdout.txt" \
  "$binary" --quick --output-format json -o "$output_dir/quick.json"
python3 -m json.tool "$output_dir/quick.json" >/dev/null

step "运行验收流程"
current_progress_step="acceptance"
progress_update "acceptance" "running" "运行验收流程"
PERFASSESS_BINARY="$binary" \
PERFASSESS_ACCEPTANCE_DIR="$output_dir/acceptance" \
PERFASSESS_ACCEPTANCE_OPTIONAL="$optional_mode" \
scripts/vps-acceptance.sh >"$output_dir/acceptance.stdout.txt" 2>"$output_dir/acceptance.stderr.log"
progress_update "acceptance" "success" "验收流程完成"
finish_step "运行验收流程"

current_progress_step="summary"
progress_update "summary" "running" "生成汇总"
python3 - "$output_dir" "$auto_profile" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
auto_profile = sys.argv[2]

def load(name):
    with (out / name).open(encoding="utf-8") as f:
        return json.load(f)

default_report = load("default.json")
quick_report = load("quick.json")
default_summary = default_report["summary"]
quick_summary = quick_report["summary"]
acceptance_summary_path = out / "acceptance" / "summary.md"

def text(value, default="-"):
    if value is None:
        return default
    if isinstance(value, str):
        value = value.strip()
        return value if value else default
    return str(value)

def num(value, digits=2, default="-"):
    if isinstance(value, bool) or value is None:
        return default
    try:
        return f"{float(value):.{digits}f}"
    except (TypeError, ValueError):
        return default

def metric(result_name, key, digits=2):
    result = default_report.get("test_results", {}).get(result_name, {})
    metrics = result.get("metrics", {}) if isinstance(result, dict) else {}
    return num(metrics.get(key), digits)

def availability_count(value):
    if not isinstance(value, dict):
        return (0, 0)
    items = [item for item in value.values() if isinstance(item, dict)]
    return (sum(1 for item in items if item.get("available")), len(items))

def route_count(value):
    if not isinstance(value, list):
        return (0, 0)
    return (sum(1 for item in value if isinstance(item, dict) and item.get("success")), len(value))

def security_count(value):
    findings = value.get("findings", []) if isinstance(value, dict) else []
    severity = {"high": 0, "medium": 0, "low": 0, "info": 0}
    for item in findings:
        if not isinstance(item, dict):
            continue
        key = text(item.get("severity"), "info").lower()
        severity[key if key in severity else "info"] += 1
    return len(findings), severity

def ratio_or_skipped(ok, total, suffix):
    if total <= 0:
        return "未执行"
    return f"{ok}/{total} {suffix}"

def ip_quality_summary(value):
    if not isinstance(value, dict) or not value:
        return "未执行"
    return f"{text(value.get('public_ip'))} / 风险 {text(value.get('risk_level'))} / 评分 {text(value.get('risk_score'))}/100"

def security_summary(value, total, severity):
    if not isinstance(value, dict):
        return "未执行"
    return f"{total} 条提示，高危 {severity['high']}，中危 {severity['medium']}"

def stress_summary(value):
    if not isinstance(value, dict):
        return "未执行"
    return f"{num(value.get('total_duration_seconds'), 0)} 秒"

route_ok, route_total = route_count(default_summary.get("route_trace_results"))
stream_ok, stream_total = availability_count(default_summary.get("streaming_results"))
ai_ok, ai_total = availability_count(default_summary.get("ai_results"))
security_total, security_severity = security_count(default_summary.get("security_report"))
ip_report = default_summary.get("ip_quality_report") if isinstance(default_summary.get("ip_quality_report"), dict) else {}
stress_report = default_summary.get("stress_report") if isinstance(default_summary.get("stress_report"), dict) else {}
version = (out / "version.txt").read_text(encoding="utf-8").strip()
confidence = default_summary.get("confidence_level", {}).get("level")
calibration = default_summary.get("score_calibration", {}).get("version")

lines = [
    "# Perfassess 自动测评报告",
    "",
    "## 总览",
    "",
    "| 项目 | 值 |",
    "|------|----|",
    f"| 版本 | {version} |",
    f"| 自动档位 | {auto_profile} |",
    f"| 综合评分 | {num(default_summary.get('total_score'))} / 100 |",
    f"| 等级 | {text(default_summary.get('grade'))} |",
    f"| 置信度 | {text(confidence)} |",
    f"| 评分基准 | {text(default_summary.get('score_profile'))} |",
    f"| 校准版本 | {text(calibration)} |",
    f"| 输出目录 | {out} |",
    "",
    "## 核心性能",
    "",
    "| 模块 | 关键结果 |",
    "|------|----------|",
    f"| CPU | 单核 {metric('cpu_result', 'single_core_score')} / 多核 {metric('cpu_result', 'multi_core_score')} / 总分 {metric('cpu_result', 'total_score')} |",
    f"| 内存 | 读 {metric('memory_result', 'read_speed_mbps')} MB/s / 写 {metric('memory_result', 'write_speed_mbps')} MB/s |",
    f"| 磁盘 | 读 {metric('disk_result', 'sequential_read_mbps')} MB/s / 写 {metric('disk_result', 'sequential_write_mbps')} MB/s / 随机 {metric('disk_result', 'random_iops', 0)} IOPS |",
    f"| 网络 | 延迟 {metric('network_result', 'latency_ms')} ms / 下载 {metric('network_result', 'download_speed_mbps')} Mbps / 上传 {metric('network_result', 'upload_speed_mbps')} Mbps |",
    "",
    "## 扩展检测",
    "",
    "| 模块 | 结果 |",
    "|------|------|",
    f"| 路由追踪 | {ratio_or_skipped(route_ok, route_total, '成功')} |",
    f"| 流媒体解锁 | {ratio_or_skipped(stream_ok, stream_total, '可用')} |",
    f"| AI 服务 | {ratio_or_skipped(ai_ok, ai_total, '可用')} |",
    f"| IP 质量 | {ip_quality_summary(ip_report)} |",
    f"| 安全体检 | {security_summary(default_summary.get('security_report'), security_total, security_severity)} |",
    f"| 压力测试 | {stress_summary(stress_report)} |",
    "",
    "## 输出文件",
    "",
    f"- Markdown 摘要: {out / 'summary.md'}",
    f"- JSON 完整报告: {out / 'default.json'}",
    f"- 文本完整报告: {out / 'default.txt'}",
    f"- 快速测评 JSON: {out / 'quick.json'}",
    f"- 依赖检查: {out / 'check-deps.txt'}",
    f"- 验收摘要: {acceptance_summary_path}",
]

share = default_summary.get("share_templates", {}).get("plain_text")
if share:
    lines.extend(["", "## 分享模板", "", "```text", share, "```"])

(out / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
PY
progress_update "summary" "success" "汇总已生成"
trap - EXIT

echo "perfassess auto passed. Output: $output_dir"
echo "Summary: $output_dir/summary.md"
