#!/usr/bin/env bash
set -euo pipefail

output_dir="${PERFASSESS_AUTO_DIR:-/tmp/perfassess-auto}"
binary="${PERFASSESS_BINARY:-./build/perfassess}"
skip_build="${PERFASSESS_SKIP_BUILD:-0}"
optional_mode="${PERFASSESS_AUTO_OPTIONAL:-never}"
show_progress="${PERFASSESS_AUTO_PROGRESS:-1}"
heartbeat_interval="${PERFASSESS_AUTO_HEARTBEAT:-10}"
progress_file="${PERFASSESS_PROGRESS_FILE:-$output_dir/progress.json}"
auto_profile="${PERFASSESS_AUTO_PROFILE:-standard}"
quality_profile="${PERFASSESS_QUALITY_PROFILE:-builtin}"
network_profile="${PERFASSESS_NETWORK_PROFILE:-auto}"
streaming_profile="${PERFASSESS_STREAMING_PROFILE:-auto}"
extra_args="${PERFASSESS_AUTO_ARGS:-}"
stress_enabled="${PERFASSESS_AUTO_STRESS:-0}"
acceptance_enabled="${PERFASSESS_AUTO_ACCEPTANCE:-1}"
iperf3_server="${PERFASSESS_IPERF3_SERVER:-}"
iperf3_servers="${PERFASSESS_IPERF3_SERVERS:-}"
iperf3_server_file="${PERFASSESS_IPERF3_SERVER_FILE:-}"
current_progress_step="prepare"
heartbeat_pid=""
failure_summary_written="0"

profile_budget() {
  local profile="$1"
  local quality="$2"
  local network="$3"

  case "$profile" in
    basic)
      if [[ "$quality" == "mainstream" ]]; then
        echo "预计耗时 2-5 分钟；资源占用低到中等；网络流量约 100-500 MB；适合低配机器或首次摸底。"
      else
        echo "预计耗时 1-3 分钟；资源占用低；网络流量约 50-200 MB；适合 512MB/低配机器和快速验收。"
      fi
      ;;
    full)
      if [[ "$quality" == "mainstream" ]]; then
        echo "预计耗时 10-25 分钟；CPU/内存/磁盘占用高；网络流量可能超过 2 GB；适合发布前深度测评。"
      else
        echo "预计耗时 6-15 分钟；CPU/内存/磁盘占用中到高；网络流量约 500 MB-2 GB；包含压力测试。"
      fi
      ;;
    standard|*)
      if [[ "$quality" == "mainstream" ]]; then
        echo "预计耗时 5-12 分钟；资源占用中等；网络流量约 500 MB-1.5 GB；适合主流口径对比。"
      else
        echo "预计耗时 3-8 分钟；资源占用中等；网络流量约 200-800 MB；适合常规 VPS 完整报告。"
      fi
      ;;
  esac

  case "$network" in
    full)
      echo "网络档位 full 会增加更多目标，可能额外增加 1-3 分钟。"
      ;;
    standard)
      echo "网络档位 standard 会增加国内方向参考，耗时适中。"
      ;;
    quick)
      echo "网络档位 quick 只做轻量出站质量参考。"
      ;;
  esac
}

profile_budget_inline() {
  local profile="$1"
  local quality="$2"
  local network="$3"
  local line combined=""
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    if [[ -z "$combined" ]]; then
      combined="$line"
    else
      combined="$combined $line"
    fi
  done < <(profile_budget "$profile" "$quality" "$network")
  echo "$combined"
}

case "$auto_profile" in
  auto) auto_profile="standard" ;;
  stress) auto_profile="full" ;;
  basic|standard|full) ;;
  *)
    echo "perfassess auto failed: PERFASSESS_AUTO_PROFILE must be auto, basic, standard, or full" >&2
    exit 1
    ;;
esac

case "$quality_profile" in
  auto) quality_profile="builtin" ;;
  builtin|mainstream) ;;
  *)
    echo "perfassess auto failed: PERFASSESS_QUALITY_PROFILE must be auto, builtin, or mainstream" >&2
    exit 1
    ;;
esac

case "$network_profile" in
  auto|quick|standard|full) ;;
  *)
    echo "perfassess auto failed: PERFASSESS_NETWORK_PROFILE must be auto, quick, standard, or full" >&2
    exit 1
    ;;
esac

case "$streaming_profile" in
  auto|quick|standard|full) ;;
  *)
    echo "perfassess auto failed: PERFASSESS_STREAMING_PROFILE must be auto, quick, standard, or full" >&2
    exit 1
    ;;
esac

if [[ "$network_profile" == "auto" ]]; then
  case "$auto_profile" in
    full) network_profile="full" ;;
    standard) network_profile="standard" ;;
    *) network_profile="quick" ;;
  esac
fi

if [[ "$streaming_profile" == "auto" ]]; then
  case "$auto_profile" in
    full) streaming_profile="full" ;;
    standard) streaming_profile="standard" ;;
    *) streaming_profile="quick" ;;
  esac
fi

if [[ "$auto_profile" == "full" ]]; then
  stress_enabled="1"
fi

budget_summary="$(profile_budget_inline "$auto_profile" "$quality_profile" "$network_profile")"

default_args=(--output-format json -o "$output_dir/default.json")
default_text_args=(-o "$output_dir/default.txt")
default_args+=(--network-profile "$network_profile")
default_text_args+=(--network-profile "$network_profile")
if [[ "$quality_profile" == "mainstream" ]]; then
  default_args+=(--cpu-backend sysbench --memory-backend sysbench --disk-backend fio)
  default_text_args+=(--cpu-backend sysbench --memory-backend sysbench --disk-backend fio)
  if [[ -n "$iperf3_server" || -n "$iperf3_servers" || -n "$iperf3_server_file" ]]; then
    default_args+=(--network-backend iperf3)
    default_text_args+=(--network-backend iperf3)
    if [[ -n "$iperf3_server" ]]; then
      default_args+=(--iperf3-server "$iperf3_server")
      default_text_args+=(--iperf3-server "$iperf3_server")
    fi
    if [[ -n "$iperf3_servers" ]]; then
      default_args+=(--iperf3-servers "$iperf3_servers")
      default_text_args+=(--iperf3-servers "$iperf3_servers")
    fi
    if [[ -n "$iperf3_server_file" ]]; then
      default_args+=(--iperf3-server-file "$iperf3_server_file")
      default_text_args+=(--iperf3-server-file "$iperf3_server_file")
    fi
  fi
fi
case "$auto_profile" in
  basic)
    ;;
  standard|full)
    default_args+=(--route-trace --streaming --streaming-profile "$streaming_profile" --ai-services --ip-quality --security)
    default_text_args+=(--route-trace --streaming --streaming-profile "$streaming_profile" --ai-services --ip-quality --security)
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
  local activity="${4:-}"

  [[ -n "$progress_file" ]] || return 0
  mkdir -p "$(dirname "$progress_file")"

  python3 - "$progress_file" "$output_dir" "$step_id" "$step_status" "$message" "$activity" <<'PY'
import json
import pathlib
import sys
from datetime import datetime, timezone

progress_path = pathlib.Path(sys.argv[1])
output_dir = pathlib.Path(sys.argv[2])
step_id = sys.argv[3]
step_status = sys.argv[4]
message = sys.argv[5]
activity = sys.argv[6]
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
    "activity": activity,
    "activity_updated_at": now if activity else "",
    "output_dir": str(output_dir),
    "started_at": now,
    "updated_at": now,
    "current_step": step_id,
    "steps": [
        {"id": sid, "label": label, "status": "pending", "message": "", "activity": "", "started_at": "", "updated_at": "", "finished_at": "", "duration_seconds": None}
        for sid, label in step_defs
    ],
}

if progress_path.exists():
    try:
        old = json.loads(progress_path.read_text(encoding="utf-8"))
        if isinstance(old, dict):
            data.update({k: v for k, v in old.items() if k not in {"steps", "updated_at", "current_step", "message", "status", "activity", "activity_updated_at"}})
            old_steps = {step.get("id"): step for step in old.get("steps", []) if isinstance(step, dict)}
            for step in data["steps"]:
                if step["id"] in old_steps:
                    step.update(old_steps[step["id"]])
    except json.JSONDecodeError:
        pass

for step in data["steps"]:
    if step["id"] == step_id:
        if step_status == "running" and not step.get("started_at"):
            step["started_at"] = now
        if step_status in {"success", "failed"} and not step.get("started_at"):
            step["started_at"] = now
        step["status"] = step_status
        step["message"] = message
        if activity:
            step["activity"] = activity
        step["updated_at"] = now
        if step_status in {"success", "failed"}:
            step["finished_at"] = now
            started_at = step.get("started_at")
            if started_at:
                try:
                    started = datetime.fromisoformat(started_at.replace("Z", "+00:00"))
                    finished = datetime.fromisoformat(now.replace("Z", "+00:00"))
                    step["duration_seconds"] = round(max(0, (finished - started).total_seconds()), 3)
                except ValueError:
                    pass
        break

statuses = [step["status"] for step in data["steps"]]
if any(status == "failed" for status in statuses):
    overall = "failed"
elif statuses and all(status == "success" for status in statuses):
    overall = "success"
else:
    overall = "running"

artifacts = [
    ("终端彩色报告", "console.ansi"),
    ("终端纯文本报告", "console.txt"),
    ("报告压缩包", "perfassess-report.zip"),
    ("报告产物清单", "artifact_manifest.json"),
    ("Markdown 摘要", "summary.md"),
    ("默认 JSON 报告", "default.json"),
    ("默认文本报告", "default.txt"),
    ("硬件质量报告", "hardware_quality.json"),
    ("网络质量报告", "net_quality.json"),
    ("脱敏校准样本", "calibration_sample.json"),
    ("路由追踪报告", "route_trace.json"),
    ("国内方向参考报告", "backroute_trace.json"),
    ("IP 质量报告", "ip_quality.json"),
    ("快速 JSON 报告", "quick.json"),
    ("依赖检查", "check-deps.txt"),
    ("验收摘要", "acceptance/summary.md"),
]

data["status"] = overall
data["message"] = message
if activity:
    data["activity"] = activity
    data["activity_updated_at"] = now
elif not data.get("activity"):
    data["activity"] = ""
    data["activity_updated_at"] = ""
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
  stop_heartbeat
  if [[ "$code" -ne 0 ]]; then
    progress_update "$current_progress_step" "failed" "步骤失败，退出码: $code"
    if [[ "$failure_summary_written" != "1" ]]; then
      write_failure_summary "$code" "$current_progress_step" "自动测评在 $current_progress_step 阶段失败。"
    fi
  fi
  exit "$code"
}

trap mark_failed EXIT

step() {
  [[ "$show_progress" == "1" ]] || return 0
  echo
  echo "==> $*"
}

format_duration() {
  local total="${1:-0}"
  local minutes=$((total / 60))
  local seconds=$((total % 60))
  if (( minutes > 0 )); then
    printf '%dm%02ds' "$minutes" "$seconds"
  else
    printf '%ds' "$seconds"
  fi
}

time_bar() {
  local elapsed="${1:-0}"
  local width=18
  local filled=$(((elapsed / heartbeat_interval) % (width + 1)))
  local bar=""
  local i
  for ((i = 0; i < width; i++)); do
    if (( i < filled )); then
      bar+="#"
    else
      bar+="-"
    fi
  done
  printf '[%s]' "$bar"
}

current_activity() {
  local file="$1"
  [[ -f "$file" ]] || return 0
  awk '
    /开始测试:|测试 .* 完成|可选步骤:|开始路由追踪|路由追踪完成|路由追踪超时|检测流媒体平台:|检测 .* 完成|检测 AI 服务:|AI服务检测完成|IP 质量检测完成|CPU 压测|内存压测|磁盘压测|压力测试完成|安全体检/ {
      line=$0
    }
    END {
      if (line != "") {
        sub(/^[[:space:]]*[0-9T:+.-]+[[:space:]]+/, "", line)
        sub(/^[[:space:]]*(INFO|WARN|ERROR)[[:space:]]+/, "", line)
        gsub(/[[:space:]]+/, " ", line)
        print line
      }
    }
  ' "$file" 2>/dev/null | tail -n 1
}

start_heartbeat() {
  local label="$1"
  local start_seconds="$2"
  local activity_file="${3:-}"
  local step_id="${4:-$current_progress_step}"
  [[ "$show_progress" == "1" ]] || return 0
  [[ "$heartbeat_interval" =~ ^[0-9]+$ && "$heartbeat_interval" -gt 0 ]] || return 0
  (
    while true; do
      sleep "$heartbeat_interval"
      local elapsed=$((SECONDS - start_seconds))
      local activity=""
      if [[ -n "$activity_file" ]]; then
        activity="$(current_activity "$activity_file")"
      fi
      if [[ -n "$activity" ]]; then
        progress_update "$step_id" "running" "$label" "$activity"
      fi
      write_running_summary "running" "$step_id" "$label" "$activity"
      if [[ -n "$activity" ]]; then
        echo "[..] $(time_bar "$elapsed") $label 运行中，已耗时 $(format_duration "$elapsed")，当前: $activity"
      else
        echo "[..] $(time_bar "$elapsed") $label 运行中，已耗时 $(format_duration "$elapsed")"
      fi
    done
  ) &
  heartbeat_pid="$!"
}

stop_heartbeat() {
  if [[ -n "$heartbeat_pid" ]] && kill -0 "$heartbeat_pid" >/dev/null 2>&1; then
    kill "$heartbeat_pid" >/dev/null 2>&1 || true
    wait "$heartbeat_pid" >/dev/null 2>&1 || true
  fi
  heartbeat_pid=""
}

finish_step() {
  local label="$1"
  local duration="${2:-}"
  [[ "$show_progress" == "1" ]] || return 0
  if [[ -n "$duration" ]]; then
    echo "[OK] $label 完成，用时 $duration"
  else
    echo "[OK] $label 完成"
  fi
}

print_log_tail() {
  local title="$1"
  local file="$2"
  [[ -f "$file" ]] || return 0
  echo
  echo "---- $title: $file ----" >&2
  tail -n 80 "$file" >&2 || true
}

append_log_tail_markdown() {
  local title="$1"
  local file="$2"
  [[ -f "$file" ]] || return 0
  {
    echo
    echo "### $title"
    echo
    echo '```text'
    tail -n 120 "$file" || true
    echo '```'
  } >>"$output_dir/summary.md"
}

write_running_summary() {
  local status="${1:-running}"
  local step_id="${2:-$current_progress_step}"
  local message="${3:-自动测评运行中}"
  local activity="${4:-}"
  mkdir -p "$output_dir"

  {
    echo "# Perfassess 自动测评进度"
    echo
    echo "| 项目 | 值 |"
    echo "|------|----|"
    echo "| 状态 | $status |"
    echo "| 当前步骤 | $step_id |"
    echo "| 当前说明 | $message |"
    if [[ -n "$activity" ]]; then
      echo "| 当前活动 | $activity |"
    fi
    echo "| 自动档位 | $auto_profile |"
    echo "| 质量档位 | $quality_profile |"
    echo "| 网络档位 | $network_profile |"
    echo "| 流媒体档位 | $streaming_profile |"
    echo "| 输出目录 | $output_dir |"
    echo "| 预算说明 | $budget_summary |"
    echo
    echo "## 步骤进度"
    echo
    echo "| 步骤 | 状态 | 耗时 | 说明 |"
    echo "|------|------|------|------|"
  } >"$output_dir/summary.md"

  if [[ -f "$progress_file" ]]; then
    python3 - "$progress_file" >>"$output_dir/summary.md" <<'PY' || true
import json
import pathlib
import sys

def text(value, default="-"):
    if value is None:
        return default
    value = str(value).strip()
    return value if value else default

def duration(value):
    try:
        seconds = int(round(float(value)))
    except (TypeError, ValueError):
        return "-"
    minutes, remain = divmod(seconds, 60)
    return f"{minutes}m{remain:02d}s" if minutes else f"{remain}s"

labels = {
    "success": "完成",
    "failed": "失败",
    "running": "运行中",
    "pending": "等待",
    "skipped": "跳过",
}

try:
    data = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
except Exception:
    data = {}

for step in data.get("steps", []):
    if not isinstance(step, dict):
        continue
    status = labels.get(step.get("status"), text(step.get("status")))
    print(f"| {text(step.get('label'))} | {status} | {duration(step.get('duration_seconds'))} | {text(step.get('message'))} |")
PY
  else
    echo "| 准备输出目录 | 运行中 | - | 进度文件尚未生成 |" >>"$output_dir/summary.md"
  fi

  {
    echo
    echo "## 已产出文件"
    echo
  } >>"$output_dir/summary.md"

  local artifact label path
  while IFS='|' read -r label path; do
    if [[ -f "$output_dir/$path" ]]; then
      echo "- $label: $output_dir/$path" >>"$output_dir/summary.md"
    fi
  done <<'EOF'
进度状态|progress.json
版本信息|version.txt
依赖检查|check-deps.txt
默认 JSON 报告|default.json
快速 JSON 报告|quick.json
验收摘要|acceptance/summary.md
报告压缩包|perfassess-report.zip
完整文本报告|default.txt
EOF

  {
    echo
    echo "## 排查文件"
    echo
    echo "- 默认 JSON 标准输出: $output_dir/default.stdout.txt"
    echo "- 默认 JSON 错误输出: $output_dir/default.stdout.txt.stderr.log"
    echo "- 快速测评标准输出: $output_dir/quick.stdout.txt"
    echo "- 快速测评错误输出: $output_dir/quick.stdout.txt.stderr.log"
    echo "- 验收标准输出: $output_dir/acceptance.stdout.txt"
    echo "- 验收错误输出: $output_dir/acceptance.stdout.txt.stderr.log"
    echo
    echo "完整测评成功后，本文件会被最终结构化 Markdown 报告覆盖。"
  } >>"$output_dir/summary.md"

  cp "$output_dir/summary.md" "$output_dir/console.txt" 2>/dev/null || true
}

write_failure_summary() {
  local code="$1"
  local step_id="${2:-$current_progress_step}"
  local message="${3:-自动测评失败}"
  mkdir -p "$output_dir"
  failure_summary_written="1"
  {
    echo "# Perfassess 自动测评失败"
    echo
    echo "| 项目 | 值 |"
    echo "|------|----|"
    echo "| 失败步骤 | $step_id |"
    echo "| 退出码 | $code |"
    echo "| 自动档位 | $auto_profile |"
    echo "| 质量档位 | $quality_profile |"
    echo "| 网络档位 | $network_profile |"
    echo "| 流媒体档位 | $streaming_profile |"
    echo "| 输出目录 | $output_dir |"
    echo
    echo "## 说明"
    echo
    echo "$message"
    echo
    echo "请优先查看下面的日志尾部，或把整个目录打包排查。"
    echo
    echo "## 排查文件"
    echo
    echo "- 进度状态: $progress_file"
    echo "- 构建标准输出: $output_dir/build.stdout.txt"
    echo "- 构建错误输出: $output_dir/build.stderr.log"
    echo "- 默认 JSON 标准输出: $output_dir/default.stdout.txt"
    echo "- 默认 JSON 错误输出: $output_dir/default.stdout.txt.stderr.log"
    echo "- 文本报告标准输出: $output_dir/default-text.stdout.txt"
    echo "- 文本报告错误输出: $output_dir/default-text.stdout.txt.stderr.log"
    echo "- 快速测评标准输出: $output_dir/quick.stdout.txt"
    echo "- 快速测评错误输出: $output_dir/quick.stdout.txt.stderr.log"
    echo "- 依赖检查: $output_dir/check-deps.txt"
    echo
    echo "## 已产出文件"
    echo
  } >"$output_dir/summary.md"

  local label path
  while IFS='|' read -r label path; do
    if [[ -f "$output_dir/$path" ]]; then
      echo "- $label: $output_dir/$path" >>"$output_dir/summary.md"
    fi
  done <<'EOF'
进度状态|progress.json
构建标准输出|build.stdout.txt
构建错误输出|build.stderr.log
版本信息|version.txt
依赖检查|check-deps.txt
默认 JSON 报告|default.json
快速 JSON 报告|quick.json
验收摘要|acceptance/summary.md
报告压缩包|perfassess-report.zip
完整文本报告|default.txt
EOF

  append_log_tail_markdown "构建标准输出" "$output_dir/build.stdout.txt"
  append_log_tail_markdown "构建错误输出" "$output_dir/build.stderr.log"
  append_log_tail_markdown "当前步骤标准输出" "$output_dir/default.stdout.txt"
  append_log_tail_markdown "当前步骤错误输出" "$output_dir/default.stdout.txt.stderr.log"
  append_log_tail_markdown "文本报告标准输出" "$output_dir/default-text.stdout.txt"
  append_log_tail_markdown "文本报告错误输出" "$output_dir/default-text.stdout.txt.stderr.log"
  append_log_tail_markdown "快速测评标准输出" "$output_dir/quick.stdout.txt"
  append_log_tail_markdown "快速测评错误输出" "$output_dir/quick.stdout.txt.stderr.log"
  append_log_tail_markdown "依赖检查" "$output_dir/check-deps.txt"

  cp "$output_dir/summary.md" "$output_dir/console.txt" 2>/dev/null || true
}

run_capture() {
  local step_id="$1"
  local label="$2"
  local stdout_file="$3"
  shift 3

  step "$label"
  current_progress_step="$step_id"
  progress_update "$step_id" "running" "$label"
  write_running_summary "running" "$step_id" "$label"
  local start_seconds="$SECONDS"
  start_heartbeat "$label" "$start_seconds" "$stdout_file" "$step_id"
  set +e
  "$@" >"$stdout_file" 2>"$output_dir/${stdout_file##*/}.stderr.log"
  local code="$?"
  set -e
  stop_heartbeat
  local duration
  duration="$(format_duration "$((SECONDS - start_seconds))")"
  if [[ "$code" -ne 0 ]]; then
    local stderr_file="$output_dir/${stdout_file##*/}.stderr.log"
    local message="$label 失败，用时 $duration，退出码: $code"
    progress_update "$step_id" "failed" "$message" "$(current_activity "$stdout_file")"
    write_running_summary "failed" "$step_id" "$message" "$(current_activity "$stdout_file")"
    echo "[FAIL] $message" >&2
    print_log_tail "$label 标准输出尾部" "$stdout_file"
    print_log_tail "$label 错误输出尾部" "$stderr_file"
    write_failure_summary "$code" "$step_id" "$message"
    return "$code"
  fi
  progress_update "$step_id" "success" "$label 完成，用时 $duration" "$(current_activity "$stdout_file")"
  write_running_summary "running" "$step_id" "$label 完成，用时 $duration" "$(current_activity "$stdout_file")"
  finish_step "$label" "$duration"
}

if ! command -v python3 >/dev/null 2>&1; then
  echo "perfassess auto failed: python3 is required for JSON validation" >&2
  exit 1
fi

mkdir -p "$output_dir"
rm -f "$output_dir"/*.json "$output_dir"/*.txt "$output_dir"/*.md "$output_dir"/*.log
progress_update "prepare" "success" "输出目录已准备: $output_dir"
write_running_summary "running" "prepare" "输出目录已准备: $output_dir"
if [[ "$show_progress" == "1" ]]; then
  echo
  echo "==> 测评预算"
  echo "自动档位: $auto_profile"
  echo "质量档位: $quality_profile"
  echo "网络档位: $network_profile"
  echo "预算说明: $budget_summary"
fi

if [[ "$skip_build" != "1" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "perfassess auto failed: go is required to build from source" >&2
    echo "Set PERFASSESS_SKIP_BUILD=1 and PERFASSESS_BINARY=/path/to/perfassess to use an existing binary." >&2
    exit 1
  fi
  mkdir -p "$(dirname "$binary")"
  step "building: $binary"
  current_progress_step="build"
  progress_update "build" "running" "构建二进制: $binary"
  write_running_summary "running" "build" "构建二进制: $binary"
  build_start="$SECONDS"
  set +e
  go build -o "$binary" cmd/main.go >"$output_dir/build.stdout.txt" 2>"$output_dir/build.stderr.log"
  build_code="$?"
  set -e
  build_duration="$(format_duration "$((SECONDS - build_start))")"
  if [[ "$build_code" -ne 0 ]]; then
    progress_update "build" "failed" "二进制构建失败，用时 $build_duration，退出码: $build_code"
    write_running_summary "failed" "build" "二进制构建失败，用时 $build_duration，退出码: $build_code"
    echo "[FAIL] building: $binary 失败，用时 $build_duration，退出码: $build_code" >&2
    print_log_tail "构建标准输出尾部" "$output_dir/build.stdout.txt"
    print_log_tail "构建错误输出尾部" "$output_dir/build.stderr.log"
    write_failure_summary "$build_code" "build" "二进制构建失败，用时 $build_duration，退出码: $build_code"
    exit "$build_code"
  fi
  progress_update "build" "success" "二进制构建完成，用时 $build_duration"
  write_running_summary "running" "build" "二进制构建完成，用时 $build_duration"
  finish_step "building: $binary" "$build_duration"
else
  printf 'build skipped; using existing binary: %s\n' "$binary" >"$output_dir/build.stdout.txt"
  : >"$output_dir/build.stderr.log"
  progress_update "build" "success" "跳过构建，使用已有二进制"
  write_running_summary "running" "build" "跳过构建，使用已有二进制"
fi

if [[ ! -x "$binary" ]]; then
  progress_update "build" "failed" "二进制不可执行: $binary"
  write_running_summary "failed" "build" "二进制不可执行: $binary"
  echo "perfassess auto failed: binary is not executable: $binary" >&2
  exit 1
fi

step "checking version and dependencies"
current_progress_step="deps"
progress_update "deps" "running" "检查版本和依赖"
write_running_summary "running" "deps" "检查版本和依赖"
deps_start="$SECONDS"
"$binary" version >"$output_dir/version.txt"
"$binary" check-deps >"$output_dir/check-deps.txt"
deps_duration="$(format_duration "$((SECONDS - deps_start))")"
progress_update "deps" "success" "版本和依赖检查完成，用时 $deps_duration"
write_running_summary "running" "deps" "版本和依赖检查完成，用时 $deps_duration"
finish_step "checking version and dependencies" "$deps_duration"

run_capture "default_json" "运行自动测评档位: $auto_profile (JSON 报告)" "$output_dir/default.stdout.txt" \
  "$binary" "${default_args[@]}"
python3 -m json.tool "$output_dir/default.json" >/dev/null

step "生成自动测评文本报告"
current_progress_step="default_text"
progress_update "default_text" "running" "等待汇总阶段生成文本报告"
write_running_summary "running" "default_text" "等待汇总阶段生成文本报告"
touch "$output_dir/default-text.stdout.txt" "$output_dir/default-text.stdout.txt.stderr.log"
progress_update "default_text" "success" "文本报告将在汇总阶段由 JSON 结果生成"
write_running_summary "running" "default_text" "文本报告将在汇总阶段由 JSON 结果生成"
finish_step "生成自动测评文本报告" "0s"

run_capture "quick" "运行快速测评" "$output_dir/quick.stdout.txt" \
  "$binary" --quick --output-format json -o "$output_dir/quick.json"
python3 -m json.tool "$output_dir/quick.json" >/dev/null

case "$acceptance_enabled" in
  0|false|no)
    step "跳过验收流程"
    progress_update "acceptance" "success" "已按配置跳过验收流程"
    mkdir -p "$output_dir/acceptance"
    printf '# VPS Acceptance Summary\n\n- status: skipped\n' >"$output_dir/acceptance/summary.md"
    write_running_summary "running" "acceptance" "已按配置跳过验收流程"
    finish_step "跳过验收流程" "0s"
    ;;
  *)
    PERFASSESS_BINARY="$binary" \
    PERFASSESS_ACCEPTANCE_DIR="$output_dir/acceptance" \
    PERFASSESS_ACCEPTANCE_OPTIONAL="$optional_mode" \
      run_capture "acceptance" "运行验收流程" "$output_dir/acceptance.stdout.txt" \
      scripts/vps-acceptance.sh
    ;;
esac

current_progress_step="summary"
progress_update "summary" "running" "生成汇总"
write_running_summary "running" "summary" "生成最终结构化汇总"
python3 - "$output_dir" "$auto_profile" "$quality_profile" "$network_profile" "$streaming_profile" "$budget_summary" <<'PY'
import json
import pathlib
import sys
import hashlib
import unicodedata
import zipfile
from datetime import datetime, timezone

out = pathlib.Path(sys.argv[1])
auto_profile = sys.argv[2]
quality_profile = sys.argv[3]
network_profile = sys.argv[4]
streaming_profile = sys.argv[5]
budget_summary = sys.argv[6]

def load(name):
    with (out / name).open(encoding="utf-8") as f:
        return json.load(f)

default_report = load("default.json")
quick_report = load("quick.json")
default_summary = default_report["summary"]
default_summary["budget_summary"] = budget_summary
quick_summary = quick_report["summary"]
acceptance_summary_path = out / "acceptance" / "summary.md"
progress_report = load("progress.json") if (out / "progress.json").exists() else {}
if isinstance(progress_report, dict):
    for step in progress_report.get("steps", []):
        if isinstance(step, dict) and step.get("id") == "summary" and step.get("status") == "running":
            step["status"] = "success"
            step["message"] = "汇总生成中"
            step["duration_seconds"] = step.get("duration_seconds") or 0
            break

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

def duration_text(value):
    if isinstance(value, bool) or value is None:
        return "-"
    try:
        seconds = int(round(float(value)))
    except (TypeError, ValueError):
        return "-"
    minutes, remain = divmod(seconds, 60)
    if minutes:
        return f"{minutes}m{remain:02d}s"
    return f"{remain}s"

def as_dict(value):
    return value if isinstance(value, dict) else {}

def metric(result_name, key, digits=2):
    result = as_dict(as_dict(default_report.get("test_results")).get(result_name))
    metrics = as_dict(result.get("metrics"))
    return num(metrics.get(key), digits)

def summary_metric(section_name, key, result_name=None, metric_key=None, digits=2):
    section = as_dict(vps.get(section_name))
    value = section.get(key)
    if value is None and result_name:
        value = metric(result_name, metric_key or key, digits)
        return value
    return num(value, digits)

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
    verdict = value.get("verdict") if isinstance(value.get("verdict"), dict) else {}
    if verdict.get("summary"):
        return text(verdict.get("summary"))
    return f"{text(value.get('public_ip'))} / 风险 {text(value.get('risk_level'))} / 评分 {text(value.get('risk_score'))}/100"

def ip_verdict(value):
    if not isinstance(value, dict):
        return {}
    verdict = value.get("verdict")
    return verdict if isinstance(verdict, dict) else {}

def ip_evidence_rows(value):
    if not isinstance(value, dict):
        return []
    evidence = value.get("evidence") if isinstance(value.get("evidence"), list) else []
    return [
        [text(item.get("name")), text(item.get("value")), text(item.get("status")), text(item.get("detail"))]
        for item in evidence
        if isinstance(item, dict)
    ]

def ip_recommendations(value):
    if not isinstance(value, dict):
        return []
    items = value.get("recommendations") if isinstance(value.get("recommendations"), list) else []
    return [text(item) for item in items if text(item, "")]

def security_summary(value, total, severity):
    if not isinstance(value, dict):
        return "未执行"
    return f"{total} 条提示，高危 {severity['high']}，中危 {severity['medium']}"

def stress_summary(value):
    if not isinstance(value, dict) or not value:
        return "未执行"
    return f"{num(value.get('total_duration_seconds'), 0)} 秒"

def mb(value):
    return f"{num(value, 0)} MB"

def gb(value):
    return f"{num(value, 2)} GB"

def yes_no(value):
    if value is True:
        return "可用"
    if value is False:
        return "不可用"
    return "-"

def streaming_category(value):
    return {
        "global": "全球",
        "us": "美国",
        "jp": "日本",
        "cn": "中国",
        "hk": "香港",
        "kr": "韩国",
        "eu": "欧洲",
        "asia": "亚洲",
        "music": "音乐",
        "sports": "体育",
    }.get(text(value, ""), text(value))

def streaming_unlock_type(value):
    return {
        "full": "完整解锁",
        "partial": "部分解锁",
        "limited": "受限",
        "blocked": "不可用",
        "login_required": "需要登录",
        "available": "可访问",
    }.get(text(value, ""), text(value))

def streaming_rows(results):
    if not isinstance(results, dict):
        return []
    rows = []
    for name, item in results.items():
        if not isinstance(item, dict):
            continue
        rows.append([
            streaming_category(item.get("category")),
            text(item.get("platform"), name),
            yes_no(item.get("available")),
            text(item.get("region")),
            streaming_unlock_type(item.get("unlock_type")),
            text(item.get("message")),
        ])
    rows.sort(key=lambda row: (row[0], row[1]))
    return rows

def ai_category(value):
    return {
        "chatbot": "对话",
        "assistant": "助手",
        "search": "搜索",
        "coding": "编程",
    }.get(text(value, ""), text(value))

def ai_access_type(value):
    return {
        "full": "可访问",
        "login_required": "需要登录",
        "verification_required": "需验证",
        "rate_limited": "限流",
        "restricted": "受限",
        "available": "可用",
    }.get(text(value, ""), text(value))

def ai_rows(results):
    if not isinstance(results, dict):
        return []
    rows = []
    for name, item in results.items():
        if not isinstance(item, dict):
            continue
        rows.append([
            ai_category(item.get("category")),
            text(item.get("service"), name),
            yes_no(item.get("available")),
            ai_access_type(item.get("access_type")),
            text(item.get("region_hint")),
            text(item.get("message")),
        ])
    rows.sort(key=lambda row: (row[0], row[1]))
    return rows

def progress_step_rows():
    rows = []
    for item in as_dict(progress_report).get("steps", []):
        if not isinstance(item, dict):
            continue
        rows.append([
            text(item.get("label")),
            status_text(item.get("status")),
            duration_text(item.get("duration_seconds")),
            text(item.get("message")),
        ])
    return rows

def status_text(value):
    if value in {"success", True}:
        return "完成"
    if value in {"failed", False}:
        return "失败"
    if value in {"skipped", None}:
        return "未执行"
    return text(value)

def bottleneck_label(value):
    value = text(value)
    return {
        "good": "表现正常",
        "watch": "需要关注",
        "weak": "明显短板",
    }.get(value, value)

def evidence_label(value):
    value = text(value)
    return {
        "success": "通过",
        "partial": "部分",
        "warning": "注意",
        "failed": "风险",
    }.get(value, value)

def module_status_label(value):
    value = text(value)
    return {
        "success": "通过",
        "warning": "注意",
        "failed": "风险",
        "skipped": "未执行",
    }.get(value, value)

def module_assessment_rows(modules):
    if not isinstance(modules, dict):
        return []
    rows = []
    for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
        item = modules.get(key)
        if not isinstance(item, dict) or item.get("status") == "skipped":
            continue
        evidence = item.get("evidence") if isinstance(item.get("evidence"), list) else []
        first_evidence = "-"
        if evidence and isinstance(evidence[0], dict):
            first_evidence = f"{text(evidence[0].get('label'))}: {text(evidence[0].get('value'))}"
        rows.append([
            text(item.get("title")),
            module_status_label(item.get("status")),
            text(item.get("confidence")),
            text(item.get("summary")),
            first_evidence,
        ])
    return rows

def number_value(value):
    if isinstance(value, bool) or value is None:
        return None
    try:
        return float(value)
    except (TypeError, ValueError):
        return None

def int_value(value, default=0):
    number = number_value(value)
    if number is None:
        return default
    return int(number)

def json_dump(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

def sha256_file(path):
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()

def artifact_manifest_entry(rel, label, include_in_archive=True):
    path = out / rel
    exists = path.is_file()
    entry = {
        "path": rel,
        "label": label,
        "exists": exists,
        "included_in_archive": bool(include_in_archive and exists),
    }
    if exists:
        entry["size_bytes"] = path.stat().st_size
        entry["sha256"] = sha256_file(path)
    return entry

def artifact_manifest_defs():
    return [
        ("console.ansi", "终端彩色报告"),
        ("console.txt", "终端纯文本报告"),
        ("summary.md", "Markdown 摘要"),
        ("default.json", "完整 JSON 报告"),
        ("default.txt", "完整文本报告"),
        ("hardware_quality.json", "硬件质量模块"),
        ("hardware_quality.txt", "硬件质量文本"),
        ("net_quality.json", "网络质量模块"),
        ("net_quality.txt", "网络质量文本"),
        ("calibration_sample.json", "脱敏校准样本"),
        ("route_trace.json", "路由追踪模块"),
        ("route_trace.txt", "路由追踪文本"),
        ("backroute_trace.json", "国内方向参考模块"),
        ("backroute_trace.txt", "国内方向参考文本"),
        ("ip_quality.json", "IP 质量模块"),
        ("ip_quality.txt", "IP 质量文本"),
        ("streaming_unlock.json", "流媒体模块"),
        ("ai_services.json", "AI 服务模块"),
        ("security_scan.json", "安全体检模块"),
        ("stress_test.json", "压力测试模块"),
        ("quick.json", "快速测评 JSON"),
        ("build.stdout.txt", "构建标准输出"),
        ("build.stderr.log", "构建错误输出"),
        ("check-deps.txt", "依赖检查"),
        ("version.txt", "版本信息"),
        ("default.stdout.txt", "完整测评标准输出"),
        ("default.stdout.txt.stderr.log", "完整测评错误输出"),
        ("default-text.stdout.txt", "文本报告标准输出"),
        ("default-text.stdout.txt.stderr.log", "文本报告错误输出"),
        ("quick.stdout.txt", "快速测评标准输出"),
        ("quick.stdout.txt.stderr.log", "快速测评错误输出"),
        ("acceptance/summary.md", "验收摘要"),
        ("acceptance.stdout.txt", "验收标准输出"),
        ("acceptance.stderr.log", "验收错误输出"),
    ]

def build_artifact_manifest(include_archive=False):
    entries = [artifact_manifest_entry(rel, label) for rel, label in artifact_manifest_defs()]
    if include_archive:
        entries.append(artifact_manifest_entry("perfassess-report.zip", "报告压缩包", include_in_archive=False))
    existing = [entry for entry in entries if entry.get("exists")]
    return {
        "schema_version": "perfassess-artifact-manifest-v1",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "tool": {
            "name": "perfassess",
            "version": version,
        },
        "run": {
            "output_dir": str(out),
            "auto_profile": auto_profile,
            "quality_profile": quality_profile,
            "network_profile": network_profile,
            "streaming_profile": streaming_profile,
            "budget_summary": budget_summary,
            "session_id": default_report.get("session_id"),
        },
        "summary": {
            "total_artifacts": len(entries),
            "existing_artifacts": len(existing),
            "archive": "perfassess-report.zip" if include_archive else None,
        },
        "artifacts": entries,
    }

def write_artifact_manifest(include_archive=False):
    manifest = build_artifact_manifest(include_archive=include_archive)
    json_dump(out / "artifact_manifest.json", manifest)
    return manifest

def metric_value(metrics, key, fallback=None):
    if not isinstance(metrics, dict):
        return fallback
    value = metrics.get(key)
    return fallback if value is None else value

def result_metrics(test_results, key):
    if not isinstance(test_results, dict):
        return {}
    result = test_results.get(key)
    if not isinstance(result, dict):
        return {}
    metrics = result.get("metrics")
    return metrics if isinstance(metrics, dict) else {}

def redacted_system_sample(system):
    if not isinstance(system, dict):
        return {}
    return {
        "cpu_model": text(system.get("cpu_model")),
        "cpu_cores": system.get("cpu_cores"),
        "cpu_threads": system.get("cpu_threads"),
        "memory_total_mb": system.get("memory_total_mb"),
        "disk_total_gb": system.get("disk_total_gb"),
        "os": text(system.get("os")),
        "architecture": text(system.get("architecture")),
        "virtualization": text(system.get("virtualization")),
    }

def module_confidence_sample(modules):
    if not isinstance(modules, dict):
        return {}
    sample = {}
    for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
        item = modules.get(key)
        if not isinstance(item, dict):
            continue
        sample[key] = {
            "status": item.get("status"),
            "confidence": item.get("confidence"),
            "evidence_count": len(item.get("evidence")) if isinstance(item.get("evidence"), list) else 0,
        }
    return sample

def calibration_component_samples(default_report, default_summary, vps, modules):
    test_results = default_report.get("test_results") if isinstance(default_report.get("test_results"), dict) else {}
    cpu_metrics = result_metrics(test_results, "cpu_result")
    memory_metrics = result_metrics(test_results, "memory_result")
    disk_metrics = result_metrics(test_results, "disk_result")
    network_metrics = result_metrics(test_results, "network_result")
    score_breakdown = default_summary.get("score_breakdown") if isinstance(default_summary.get("score_breakdown"), dict) else {}
    vps_cpu = vps.get("cpu", {}) if isinstance(vps.get("cpu"), dict) else {}
    vps_memory = vps.get("memory", {}) if isinstance(vps.get("memory"), dict) else {}
    vps_disk = vps.get("disk", {}) if isinstance(vps.get("disk"), dict) else {}
    vps_network = vps.get("network", {}) if isinstance(vps.get("network"), dict) else {}
    return {
        "cpu": {
            "backend": metric_value(cpu_metrics, "backend", vps_cpu.get("backend")),
            "score": default_summary.get("cpu_score"),
            "single_core_score": metric_value(cpu_metrics, "single_core_score", vps_cpu.get("single_core_score")),
            "multi_core_score": metric_value(cpu_metrics, "multi_core_score", vps_cpu.get("multi_core_score")),
            "total_score": metric_value(cpu_metrics, "total_score", vps_cpu.get("total_score")),
            "events_single": metric_value(cpu_metrics, "single_core_events_per_sec", vps_cpu.get("single_core_events_per_sec")),
            "events_multi": metric_value(cpu_metrics, "multi_core_events_per_sec", vps_cpu.get("multi_core_events_per_sec")),
            "breakdown": score_breakdown.get("cpu", {}),
            "module": modules.get("cpu", {}) if isinstance(modules, dict) else {},
        },
        "memory": {
            "backend": metric_value(memory_metrics, "backend", vps_memory.get("backend")),
            "score": default_summary.get("memory_score"),
            "read_mbps": metric_value(memory_metrics, "read_speed_mbps", vps_memory.get("read_mbps")),
            "write_mbps": metric_value(memory_metrics, "write_speed_mbps", vps_memory.get("write_mbps")),
            "breakdown": score_breakdown.get("memory", {}),
            "module": modules.get("memory", {}) if isinstance(modules, dict) else {},
        },
        "disk": {
            "backend": metric_value(disk_metrics, "backend", vps_disk.get("backend")),
            "score": default_summary.get("disk_score"),
            "sequential_read_mbps": metric_value(disk_metrics, "sequential_read_mbps", vps_disk.get("sequential_read_mbps")),
            "sequential_write_mbps": metric_value(disk_metrics, "sequential_write_mbps", vps_disk.get("sequential_write_mbps")),
            "random_iops": metric_value(disk_metrics, "random_iops", vps_disk.get("random_iops")),
            "breakdown": score_breakdown.get("disk", {}),
            "module": modules.get("disk", {}) if isinstance(modules, dict) else {},
        },
        "network": {
            "backend": metric_value(network_metrics, "backend", vps_network.get("backend")),
            "score": default_summary.get("network_score"),
            "latency_ms": metric_value(network_metrics, "latency_ms", vps_network.get("latency_ms")),
            "download_mbps": metric_value(network_metrics, "download_speed_mbps", vps_network.get("download_mbps")),
            "upload_mbps": metric_value(network_metrics, "upload_speed_mbps", vps_network.get("upload_mbps")),
            "upload_estimated": metric_value(network_metrics, "upload_speed_estimated", vps_network.get("upload_estimated")),
            "quality_failure_rate": metric_value(network_metrics, "network_quality_failure_rate", vps_network.get("quality_failure_rate")),
            "quality_jitter_ms": metric_value(network_metrics, "network_quality_jitter_ms", vps_network.get("quality_jitter_ms")),
            "breakdown": score_breakdown.get("network", {}),
            "module": modules.get("network", {}) if isinstance(modules, dict) else {},
        },
    }

def strip_module_details(component_samples):
    stripped = {}
    for key, value in component_samples.items():
        if not isinstance(value, dict):
            continue
        item = dict(value)
        module = item.get("module")
        if isinstance(module, dict):
            item["module"] = {
                "status": module.get("status"),
                "confidence": module.get("confidence"),
                "evidence_count": len(module.get("evidence")) if isinstance(module.get("evidence"), list) else 0,
            }
        stripped[key] = item
    return stripped

def calibration_sample(default_report, default_summary, vps, modules):
    calibration_info = default_summary.get("score_calibration") if isinstance(default_summary.get("score_calibration"), dict) else {}
    score_breakdown = default_summary.get("score_breakdown") if isinstance(default_summary.get("score_breakdown"), dict) else {}
    confidence_info = default_summary.get("confidence_level") if isinstance(default_summary.get("confidence_level"), dict) else {}
    system_info = vps.get("system", {}) if isinstance(vps.get("system"), dict) else {}
    component_samples = strip_module_details(calibration_component_samples(default_report, default_summary, vps, modules))
    return {
        "schema_version": "perfassess-calibration-sample-v1",
        "redacted": True,
        "privacy_note": "该样本用于评分阈值回测，已排除公网 IP、ISP、ASN、精确地理位置、会话 ID、路由 hop、原始日志和完整原始报告。",
        "generated_at": default_report.get("timestamp"),
        "tool": {
            "name": "perfassess",
            "version": version,
            "auto_profile": auto_profile,
            "quality_profile": quality_profile,
            "network_profile": network_profile,
            "streaming_profile": streaming_profile,
        },
        "environment": redacted_system_sample(system_info),
        "scores": {
            "total_score": default_summary.get("total_score"),
            "grade": default_summary.get("grade"),
            "score_profile": default_summary.get("score_profile"),
            "calibration_version": calibration_info.get("version"),
            "confidence_level": confidence_info.get("level"),
            "tests_success": default_summary.get("tests_success"),
            "tests_failed": default_summary.get("tests_failed"),
            "tests_skipped": default_summary.get("tests_skipped"),
            "tests_degraded": default_summary.get("tests_degraded"),
        },
        "benchmark_profile": default_summary.get("benchmark_profile", {}),
        "score_calibration": calibration_info,
        "score_breakdown": score_breakdown,
        "component_samples": component_samples,
        "module_confidence": module_confidence_sample(modules),
        "quality_notes": default_summary.get("quality_notes", []),
    }

def md_cell(value):
    return text(value).replace("|", "\\|")

def md_table(headers, rows):
    lines = [
        "| " + " | ".join(md_cell(item) for item in headers) + " |",
        "| " + " | ".join("---" for _ in headers) + " |",
    ]
    for row in rows:
        lines.append("| " + " | ".join(md_cell(item) for item in row) + " |")
    return lines

def detected_ip_factors(value, limit=8):
    if not isinstance(value, dict):
        return []
    factors = value.get("risk_factors") if isinstance(value.get("risk_factors"), list) else []
    rows = []
    for item in factors:
        if not isinstance(item, dict) or not item.get("detected"):
            continue
        rows.append([text(item.get("name")), text(item.get("confidence")), text(item.get("detail"))])
        if len(rows) >= limit:
            break
    return rows

def route_group(target):
    lowered = text(target, "").lower()
    china_markers = ("189.cn", "10086.cn", "chinaunicom", "ctyun", "qq.com")
    if any(marker in lowered for marker in china_markers):
        return "国内方向参考"
    return "公共网络"

def route_direction(item):
    if not isinstance(item, dict):
        return "公共网络"
    group = text(item.get("direction_group"), "")
    if group == "china_reference":
        return "国内方向参考"
    if group == "public":
        return "公共网络"
    return route_group(item.get("target"))

def route_quality(item):
    if not isinstance(item, dict):
        return "-"
    quality = item.get("quality") if isinstance(item.get("quality"), dict) else {}
    grade = text(quality.get("grade"), "")
    status = route_status_label(quality.get("status"))
    if grade and status:
        return f"{grade}/{status}"
    if grade:
        return grade
    return "完成" if item.get("success") is True else "失败"

def route_status_label(value):
    lowered = text(value, "").lower()
    if lowered == "success":
        return "良好"
    if lowered == "warning":
        return "需关注"
    if lowered == "failed":
        return "异常"
    return text(value, "")

def route_avg_latency(item):
    if not isinstance(item, dict):
        return "-"
    value = number_value(item.get("average_latency_ms"))
    return f"{value:.2f} ms" if value is not None and value > 0 else "-"

def latency_ms(value):
    if isinstance(value, bool) or value is None:
        return "-"
    try:
        # Go time.Duration is encoded as nanoseconds in JSON.
        return f"{float(value) / 1000000:.2f} ms"
    except (TypeError, ValueError):
        return "-"

def route_last_hop(result):
    if isinstance(result, dict) and text(result.get("last_visible_hop"), ""):
        return text(result.get("last_visible_hop"))
    hops = result.get("hops", []) if isinstance(result, dict) else []
    if not isinstance(hops, list):
        return "-"
    for hop in reversed(hops):
        if not isinstance(hop, dict):
            continue
        ip = text(hop.get("ip"), "")
        if not ip or ip == "*":
            continue
        hostname = text(hop.get("hostname"), "")
        label = ip if not hostname or hostname == "-" else f"{ip} {hostname}"
        return f"{label} / {latency_ms(hop.get('latency'))}"
    return "无有效末跳"

def route_rows(results):
    if not isinstance(results, list):
        return []
    rows = []
    for item in results:
        if not isinstance(item, dict):
            continue
        success = item.get("success") is True
        rows.append([
            route_direction(item),
            text(item.get("target")),
            route_quality(item),
            text(item.get("total_hops"), "0"),
            text(item.get("timeout_hops"), "0"),
            route_avg_latency(item),
            route_last_hop(item) if success else text(item.get("error_message"), "无错误信息"),
        ])
    return rows

def has_china_route_reference(rows):
    return any(row and row[0] == "国内方向参考" for row in rows)

def split_route_results(results):
    grouped = {"public": [], "china_reference": []}
    if not isinstance(results, list):
        return grouped
    for item in results:
        if not isinstance(item, dict):
            continue
        key = "china_reference" if route_direction(item) == "国内方向参考" else "public"
        grouped[key].append(item)
    return grouped

def route_direction_summary(results):
    grouped = split_route_results(results)
    summary = {}
    for key, items in grouped.items():
        ok = sum(1 for item in items if isinstance(item, dict) and item.get("success"))
        summary[key] = {
            "total": len(items),
            "success": ok,
            "failed": len(items) - ok,
        }
    return summary

def iperf3_matrix_nodes(metrics):
    if not isinstance(metrics, dict):
        return []
    count = int_value(metrics.get("iperf3_matrix_server_count"))
    nodes = []
    for index in range(1, count + 1):
        prefix = f"iperf3_matrix_{index}"
        download = number_value(metrics.get(f"{prefix}_download_mbps"))
        upload = number_value(metrics.get(f"{prefix}_upload_mbps"))
        latency = number_value(metrics.get(f"{prefix}_latency_ms"))
        error = text(metrics.get(f"{prefix}_error"), "")
        has_result = (download or 0) > 0 or (upload or 0) > 0
        if error and has_result:
            status = "部分"
        elif error:
            status = "失败"
        else:
            status = "完成"
        nodes.append({
            "index": index,
            "server": text(metrics.get(f"{prefix}_server")),
            "name": text(metrics.get(f"{prefix}_name"), ""),
            "region": text(metrics.get(f"{prefix}_region"), ""),
            "provider": text(metrics.get(f"{prefix}_provider"), ""),
            "host": text(metrics.get(f"{prefix}_host")),
            "port": text(metrics.get(f"{prefix}_port"), ""),
            "protocol": text(metrics.get(f"{prefix}_protocol")),
            "download_mbps": download,
            "upload_mbps": upload,
            "latency_ms": latency,
            "status": status,
            "error": error,
        })
    return nodes

def iperf3_matrix_summary(metrics):
    nodes = iperf3_matrix_nodes(metrics)
    if not nodes:
        return {}
    return {
        "profile": text(metrics.get("iperf3_matrix_profile"), "multi_server"),
        "server_count": int_value(metrics.get("iperf3_matrix_server_count"), len(nodes)),
        "success_count": int_value(metrics.get("iperf3_matrix_success_count")),
        "avg_download_mbps": number_value(metrics.get("iperf3_matrix_avg_download_mbps")),
        "avg_upload_mbps": number_value(metrics.get("iperf3_matrix_avg_upload_mbps")),
        "best_download_mbps": number_value(metrics.get("iperf3_matrix_best_download_mbps")),
        "best_upload_mbps": number_value(metrics.get("iperf3_matrix_best_upload_mbps")),
        "nodes": nodes,
    }

def iperf3_console_rows(metrics):
    rows = []
    for node in iperf3_matrix_nodes(metrics):
        status = node["status"] if not node["error"] else f"{node['status']}: {node['error']}"
        label = node["server"]
        if node["name"]:
            label = f"{node['name']} ({node['server']})"
        rows.append([
            label,
            node["region"] or "-",
            node["provider"] or "-",
            node["protocol"],
            f"{num(node['download_mbps'])} Mbps" if node["download_mbps"] is not None else "-",
            f"{num(node['upload_mbps'])} Mbps" if node["upload_mbps"] is not None else "-",
            f"{num(node['latency_ms'])} ms" if node["latency_ms"] is not None else "-",
            status,
        ])
    return rows

def display_width(value):
    width = 0
    for ch in str(value):
        if unicodedata.combining(ch):
            continue
        width += 2 if unicodedata.east_asian_width(ch) in {"F", "W"} else 1
    return width

def fit(value, width):
    value = str(value)
    current = 0
    result = []
    for ch in value:
        char_width = 0 if unicodedata.combining(ch) else (2 if unicodedata.east_asian_width(ch) in {"F", "W"} else 1)
        if current + char_width > width:
            break
        result.append(ch)
        current += char_width
    if current < display_width(value) and width >= 1:
        while result and current + 1 > width:
            removed = result.pop()
            current -= 0 if unicodedata.combining(removed) else (2 if unicodedata.east_asian_width(removed) in {"F", "W"} else 1)
        result.append("…")
        current += 1
    return "".join(result) + " " * max(0, width - current)

def pad(value, width):
    return str(value) + " " * max(0, width - display_width(value))

def style_for_status(value):
    lowered = text(value).lower()
    if "需关注" in lowered:
        return "yellow"
    if "异常" in lowered:
        return "red"
    if lowered in {"完成", "success", "ok", "可用", "通过", "低", "low"}:
        return "green"
    if lowered in {"未执行", "skipped", "medium", "中", "中等", "需关注"}:
        return "yellow"
    if lowered in {"失败", "failed", "不可用", "high", "高", "异常"}:
        return "red"
    return "cyan"

def write_report_archive():
    archive_path = out / "perfassess-report.zip"
    include = [
        "console.ansi",
        "console.txt",
        "summary.md",
        "artifact_manifest.json",
        "default.json",
        "default.txt",
        "hardware_quality.json",
        "hardware_quality.txt",
        "net_quality.json",
        "net_quality.txt",
        "calibration_sample.json",
        "route_trace.json",
        "route_trace.txt",
        "backroute_trace.json",
        "backroute_trace.txt",
        "ip_quality.json",
        "ip_quality.txt",
        "streaming_unlock.json",
        "ai_services.json",
        "security_scan.json",
        "stress_test.json",
        "quick.json",
        "build.stdout.txt",
        "build.stderr.log",
        "check-deps.txt",
        "version.txt",
        "default.stdout.txt",
        "default.stdout.txt.stderr.log",
        "default-text.stdout.txt",
        "default-text.stdout.txt.stderr.log",
        "quick.stdout.txt",
        "quick.stdout.txt.stderr.log",
        "acceptance/summary.md",
        "acceptance.stdout.txt",
        "acceptance.stderr.log",
    ]
    with zipfile.ZipFile(archive_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for rel in include:
            path = out / rel
            if path.is_file():
                archive.write(path, rel)
    return archive_path

def write_module_artifacts():
    test_results = default_report.get("test_results", {})
    if not isinstance(test_results, dict):
        test_results = {}
    module_common = {
        "session_id": default_report.get("session_id"),
        "timestamp": default_report.get("timestamp"),
        "auto_profile": auto_profile,
        "quality_profile": quality_profile,
        "network_profile": network_profile,
        "streaming_profile": streaming_profile,
        "budget_summary": budget_summary,
    }
    hardware_payload = {
        **module_common,
        "system": system,
        "cpu": cpu,
        "memory": memory,
        "disk": disk,
        "score": {
            "total_score": default_summary.get("total_score"),
            "grade": default_summary.get("grade"),
            "confidence": confidence,
            "score_profile": default_summary.get("score_profile"),
            "calibration_version": calibration,
        },
        "test_results": {
            "cpu_result": test_results.get("cpu_result"),
            "memory_result": test_results.get("memory_result"),
            "disk_result": test_results.get("disk_result"),
        },
    }
    net_payload = {
        **module_common,
        "assessment": module_assessments.get("network", {}),
        "network": network,
        "network_metrics": network_metrics,
        "iperf3_matrix": iperf3_matrix_summary(network_metrics),
        "route_trace_note": route_note,
        "route_trace_results": default_summary.get("route_trace_results", []),
        "streaming_results": default_summary.get("streaming_results", {}),
        "ai_results": default_summary.get("ai_results", {}),
        "test_results": {"network_result": test_results.get("network_result")},
    }
    route_payload = {
        **module_common,
        "assessment": module_assessments.get("route", {}),
        "note": route_note,
        "direction_summary": route_direction_summary(default_summary.get("route_trace_results")),
        "groups": split_route_results(default_summary.get("route_trace_results")),
        "results": default_summary.get("route_trace_results", []),
    }
    backroute_payload = {
        **module_common,
        "type": "outbound_china_direction_reference",
        "is_real_return_route": False,
        "note": "该文件为本机到国内目标的出站方向参考，不是真实回程。真实回程需要远端探针或第三方平台配合。",
        "summary": route_direction_summary(default_summary.get("route_trace_results")),
        "results": split_route_results(default_summary.get("route_trace_results")).get("china_reference", []),
    }
    ip_payload = {
        **module_common,
        "assessment": module_assessments.get("ip_quality", {}),
        "verdict": ip_verdict(ip_report),
        "evidence": ip_report.get("evidence", []) if isinstance(ip_report, dict) else [],
        "recommendations": ip_recommendations(ip_report),
        "risk_sources": ip_report.get("risk_sources", []) if isinstance(ip_report, dict) else [],
        "mail_summary": ip_report.get("mail_summary", {}) if isinstance(ip_report, dict) else {},
        "network_stack": ip_report.get("network_stack", {}) if isinstance(ip_report, dict) else {},
        "report": ip_report,
    }
    streaming_payload = {
        **module_common,
        "assessment": module_assessments.get("streaming", {}),
        "profile": text(default_summary.get("streaming_profile"), streaming_profile),
        "results": default_summary.get("streaming_results", {}),
        "rows": streaming_rows(default_summary.get("streaming_results")),
    }
    ai_payload = {**module_common, "assessment": module_assessments.get("ai_services", {}), "results": default_summary.get("ai_results", {}), "rows": ai_rows(default_summary.get("ai_results"))}
    security_payload = {**module_common, "report": default_summary.get("security_report", {})}
    stress_payload = {**module_common, "report": stress_report}

    json_dump(out / "hardware_quality.json", hardware_payload)
    json_dump(out / "net_quality.json", net_payload)
    json_dump(out / "calibration_sample.json", calibration_sample(default_report, default_summary, vps, module_assessments))
    json_dump(out / "route_trace.json", route_payload)
    json_dump(out / "backroute_trace.json", backroute_payload)
    json_dump(out / "ip_quality.json", ip_payload)
    json_dump(out / "streaming_unlock.json", streaming_payload)
    json_dump(out / "ai_services.json", ai_payload)
    json_dump(out / "security_scan.json", security_payload)
    json_dump(out / "stress_test.json", stress_payload)

    (out / "hardware_quality.txt").write_text(
        "\n".join([
            "Perfassess 硬件质量摘要",
            f"CPU: 单核 {num(cpu.get('single_core_score'))} | 多核 {num(cpu.get('multi_core_score'))} | 后端 {text(cpu.get('backend'))}",
            f"内存: 读 {num(memory.get('read_mbps'))} MB/s | 写 {num(memory.get('write_mbps'))} MB/s",
            f"磁盘: 读 {num(disk.get('sequential_read_mbps'))} MB/s | 写 {num(disk.get('sequential_write_mbps'))} MB/s | 随机 {num(disk.get('random_iops'), 0)} IOPS",
            f"评分: {num(default_summary.get('total_score'))} / 100 | {text(default_summary.get('grade'))} | 置信 {text(confidence)}",
            "",
        ]),
        encoding="utf-8",
    )
    (out / "net_quality.txt").write_text(
        "\n".join([
            "Perfassess 网络质量摘要",
            f"网络档位: {network_profile}",
            f"吞吐: 延迟 {num(network.get('latency_ms'))} ms | 下载 {num(network.get('download_mbps'))} Mbps | 上传 {num(network.get('upload_mbps'))} Mbps",
            f"质量: IPv4 {yes_no(network.get('ipv4_available'))} | IPv6 {yes_no(network.get('ipv6_available'))} | 抖动 {num(network.get('quality_jitter_ms'))} ms | 失败率 {num(network.get('quality_failure_rate'))}%",
            f"iperf3: {ratio_or_skipped(int_value(network_metrics.get('iperf3_matrix_success_count')), int_value(network_metrics.get('iperf3_matrix_server_count')), '节点成功')}",
            f"路由: {ratio_or_skipped(route_ok, route_total, '成功')}",
            f"说明: {route_note}",
            "",
        ]),
        encoding="utf-8",
    )
    route_lines = ["Perfassess 路由追踪摘要", f"说明: {route_note}", ""]
    for row in route_rows(default_summary.get("route_trace_results")):
        route_lines.append(f"{row[0]} | {row[1]} | 评级 {row[2]} | {row[3]} 跳 | 超时 {row[4]} | 平均 {row[5]} | {row[6]}")
    if len(route_lines) == 3:
        route_lines.append("未执行")
    route_lines.append("")
    (out / "route_trace.txt").write_text("\n".join(route_lines), encoding="utf-8")
    backroute_lines = [
        "Perfassess 国内方向参考摘要",
        "说明: 本文件不是第三方真实回程，只是本机到国内目标的出站路径参考。",
        "",
    ]
    china_rows = [row for row in route_rows(default_summary.get("route_trace_results")) if row[0] == "国内方向参考"]
    for row in china_rows:
        backroute_lines.append(f"{row[1]} | 评级 {row[2]} | {row[3]} 跳 | 超时 {row[4]} | 平均 {row[5]} | {row[6]}")
    if not china_rows:
        backroute_lines.append("未执行或当前网络档位未包含国内方向目标。")
    backroute_lines.append("")
    (out / "backroute_trace.txt").write_text("\n".join(backroute_lines), encoding="utf-8")
    (out / "ip_quality.txt").write_text(
        "\n".join([
            "Perfassess IP 质量摘要",
            ip_quality_summary(ip_report),
            f"ASN: {text(ip_report.get('asn'))} | 组织: {text(ip_report.get('organization'))}" if ip_report else "未执行",
            f"网络栈: {text(ip_report.get('network_stack', {}).get('detected_version'))} | 双栈 {yes_no(ip_report.get('network_stack', {}).get('dual_stack'))}" if ip_report else "网络栈: 未执行",
            f"邮件汇总: 服务商 {text(ip_report.get('mail_summary', {}).get('providers'), '0')} | 可连服务商 {text(ip_report.get('mail_summary', {}).get('provider_open'), '0')} | 可连端口 {text(ip_report.get('mail_summary', {}).get('reachable'), '0')}/{text(ip_report.get('mail_summary', {}).get('total'), '0')}" if ip_report else "邮件汇总: 未执行",
            "风险来源:",
            *[f"- {text(item.get('name'))}: {text(item.get('status'))} | {text(item.get('signal'))} | {text(item.get('detail'))}" for item in (ip_report.get("risk_sources", []) if isinstance(ip_report.get("risk_sources"), list) else []) if isinstance(item, dict)],
            "证据:",
            *[f"- {row[0]}: {row[1]} | {row[2]} | {row[3]}" for row in ip_evidence_rows(ip_report)],
            "建议:",
            *[f"- {item}" for item in ip_recommendations(ip_report)],
            "",
        ]),
        encoding="utf-8",
    )

route_ok, route_total = route_count(default_summary.get("route_trace_results"))
stream_ok, stream_total = availability_count(default_summary.get("streaming_results"))
ai_ok, ai_total = availability_count(default_summary.get("ai_results"))
security_total, security_severity = security_count(default_summary.get("security_report"))
ip_report = default_summary.get("ip_quality_report") if isinstance(default_summary.get("ip_quality_report"), dict) else {}
stress_report = default_summary.get("stress_report") if isinstance(default_summary.get("stress_report"), dict) else {}
module_assessments = default_summary.get("module_assessments") if isinstance(default_summary.get("module_assessments"), dict) else {}
version = (out / "version.txt").read_text(encoding="utf-8").strip()
confidence = default_summary.get("confidence_level", {}).get("level")
calibration = default_summary.get("score_calibration", {}).get("version")
route_note = text(default_summary.get("route_trace_note"), "路由追踪为本机出站路径参考，不等同于真实回程。")
vps = default_summary.get("vps_benchmark_summary") if isinstance(default_summary.get("vps_benchmark_summary"), dict) else {}
system = vps.get("system", {}) if isinstance(vps.get("system"), dict) else {}
cpu = vps.get("cpu", {}) if isinstance(vps.get("cpu"), dict) else {}
memory = vps.get("memory", {}) if isinstance(vps.get("memory"), dict) else {}
disk = vps.get("disk", {}) if isinstance(vps.get("disk"), dict) else {}
network = vps.get("network", {}) if isinstance(vps.get("network"), dict) else {}
network_metrics = default_report.get("test_results", {}).get("network_result", {}).get("metrics", {})
if not isinstance(network_metrics, dict):
    network_metrics = {}
conclusion = default_summary.get("assessment_conclusion") if isinstance(default_summary.get("assessment_conclusion"), dict) else {}
write_module_artifacts()

def console_report(color=False):
    colors = {
        "reset": "\033[0m",
        "bold": "\033[1m",
        "green": "\033[32m",
        "yellow": "\033[33m",
        "red": "\033[31m",
        "cyan": "\033[36m",
        "blue": "\033[34m",
        "muted": "\033[90m",
    }

    def c(name, value):
        if not color:
            return value
        return f"{colors.get(name, '')}{value}{colors['reset']}"

    def line(char="─"):
        return c("green", char * 78)

    def section(title):
        return ["", c("green", f"▶ {title}"), line()]

    def kv(label, value, value_style="cyan"):
        return f"{c('muted', pad(label, 14))}: {c(value_style, text(value))}"

    def table(headers, rows, widths, status_col=None):
        rendered = []
        header = "  ".join(c("bold", fit(head, widths[idx])) for idx, head in enumerate(headers))
        rendered.append(header)
        rendered.append(c("muted", "  ".join("─" * width for width in widths)))
        for row in rows:
            parts = []
            for idx, cell in enumerate(row):
                cell_text = fit(cell, widths[idx])
                cell_style = None
                if status_col is not None and idx == status_col:
                    cell_style = style_for_status(cell)
                parts.append(c(cell_style, cell_text) if cell_style else cell_text)
            rendered.append("  ".join(parts))
        return rendered

    total_score = default_summary.get("total_score")
    score_line = f"{num(total_score)} / 100"
    upload_suffix = " (估算)" if network.get("upload_estimated") else ""
    ip_location = "，".join(part for part in [text(system.get("location"), ""), text(system.get("isp"), "")] if part)
    blacklists = ip_report.get("blacklist_summary", {}) if isinstance(ip_report.get("blacklist_summary"), dict) else {}
    mail_checks = ip_report.get("mail_checks", []) if isinstance(ip_report.get("mail_checks"), list) else []
    mail_summary = ip_report.get("mail_summary", {}) if isinstance(ip_report.get("mail_summary"), dict) else {}
    network_stack = ip_report.get("network_stack", {}) if isinstance(ip_report.get("network_stack"), dict) else {}
    mail_ok = sum(1 for item in mail_checks if isinstance(item, dict) and item.get("reachable"))

    rows = [
        c("bold", "Perfassess 自动测评报告"),
        line("═"),
        f"{kv('版本', version)}    {kv('档位', auto_profile, 'yellow')}    {kv('质量', quality_profile, 'yellow')}",
        kv("网络档位", network_profile, "yellow"),
        kv("流媒体档位", streaming_profile, "yellow"),
        kv("预算说明", budget_summary, "yellow"),
        kv("输出目录", out),
    ]

    rows += section("系统信息")
    rows.extend([
        kv("处理器", text(system.get("cpu_model"))),
        kv("规格", f"{text(system.get('cpu_cores'))}C/{text(system.get('cpu_threads'))}T | 内存 {mb(system.get('memory_total_mb'))} | 磁盘 {gb(system.get('disk_total_gb'))}"),
        kv("系统", f"{text(system.get('os'))} {text(system.get('os_version'))} | {text(system.get('architecture'))}"),
        kv("虚拟化", f"{text(system.get('virtualization'))} / {text(system.get('virtualization_vendor'))}"),
        kv("公网", f"{text(system.get('public_ip'))} | {ip_location or '-'}"),
    ])

    rows += section("综合评分")
    rows.extend(table(
        ["项目", "结果", "说明"],
        [
            ["总分", score_line, text(default_summary.get("grade"))],
            ["置信度", text(confidence), f"基准 {text(default_summary.get('score_profile'))}"],
            ["校准", text(calibration), "服务器评分模型"],
        ],
        [14, 22, 36],
        status_col=2,
    ))
    if conclusion:
        bottlenecks = conclusion.get("bottlenecks", []) if isinstance(conclusion.get("bottlenecks"), list) else []
        evidence = conclusion.get("evidence", []) if isinstance(conclusion.get("evidence"), list) else []
        limitations = conclusion.get("limitations", []) if isinstance(conclusion.get("limitations"), list) else []
        recommendations = conclusion.get("recommendations", []) if isinstance(conclusion.get("recommendations"), list) else []
        suitability = conclusion.get("suitability", []) if isinstance(conclusion.get("suitability"), list) else []
        rows += section("测评结论")
        rows.extend([
            kv("结论", text(conclusion.get("headline"))),
            kv("适用判断", text(conclusion.get("scenario"))),
            kv("适合场景", "、".join(text(item) for item in suitability) if suitability else "-"),
        ])
        if bottlenecks:
            rows.extend(table(
                ["模块", "评分", "状态"],
                [[text(item.get("label")), num(item.get("score")), bottleneck_label(item.get("severity"))] for item in bottlenecks if isinstance(item, dict)],
                [12, 12, 18],
                status_col=2,
            ))
        if evidence:
            rows.extend(table(
                ["证据", "结果", "状态", "说明"],
                [[text(item.get("label")), text(item.get("value")), evidence_label(item.get("status")), text(item.get("detail"))] for item in evidence[:5] if isinstance(item, dict)],
                [16, 24, 10, 28],
                status_col=2,
            ))
        if limitations:
            rows.append(kv("主要限制", text(limitations[0]), "yellow"))
        if recommendations:
            rows.append(kv("优先建议", text(recommendations[0]), "yellow"))

    module_rows = module_assessment_rows(module_assessments)
    if module_rows:
        rows += section("模块可信度")
        rows.extend(table(
            ["模块", "状态", "置信度", "结论", "首要证据"],
            module_rows,
            [12, 8, 8, 28, 22],
            status_col=1,
        ))

    rows += section("核心性能")
    rows.extend(table(
        ["模块", "关键指标", "评分/状态"],
        [
            ["CPU", f"单核 {num(cpu.get('single_core_score'))} | 多核 {num(cpu.get('multi_core_score'))} | {text(cpu.get('backend'))}", f"{num(cpu.get('total_score'))}"],
            ["内存", f"读 {num(memory.get('read_mbps'))} MB/s | 写 {num(memory.get('write_mbps'))} MB/s", f"{num(memory.get('score'))}"],
            ["磁盘", f"读 {num(disk.get('sequential_read_mbps'))} MB/s | 写 {num(disk.get('sequential_write_mbps'))} MB/s | {num(disk.get('random_iops'), 0)} IOPS", f"{num(disk.get('score'))}"],
            ["网络", f"延迟 {num(network.get('latency_ms'))} ms | 下 {num(network.get('download_mbps'))} Mbps | 上 {num(network.get('upload_mbps'))} Mbps{upload_suffix}", f"{num(network.get('score'))}"],
        ],
        [10, 52, 12],
    ))

    timing_rows = progress_step_rows()
    if timing_rows:
        rows += section("耗时统计")
        rows.extend(table(
            ["步骤", "状态", "耗时", "说明"],
            timing_rows,
            [20, 10, 10, 34],
            status_col=1,
        ))

    rows += section("网络质量")
    rows.extend(table(
        ["项目", "结果", "说明"],
        [
            ["IPv4", yes_no(network.get("ipv4_available")), f"失败率 {num(network_metrics.get('network_quality_ipv4_failure_rate'))}%"],
            ["IPv6", yes_no(network.get("ipv6_available")), f"失败率 {num(network_metrics.get('network_quality_ipv6_failure_rate'))}%"],
            ["平均延迟", f"{num(network.get('quality_avg_latency_ms'))} ms", f"抖动 {num(network.get('quality_jitter_ms'))} ms"],
            ["总体失败率", f"{num(network.get('quality_failure_rate'))}%", f"目标 {text(network_metrics.get('network_quality_target_count'))} 个"],
        ],
        [14, 24, 34],
        status_col=1,
    ))
    iperf3_rows = iperf3_console_rows(network_metrics)
    if iperf3_rows:
        rows += section("iperf3 多节点矩阵")
        rows.extend(table(
            ["节点", "区域", "提供方", "协议", "下载", "上传", "延迟", "状态/错误"],
            iperf3_rows,
            [24, 8, 12, 8, 12, 12, 10, 12],
            status_col=7,
        ))

    rows += section("IP 质量")
    if ip_report:
        verdict = ip_verdict(ip_report)
        rows.extend([
            kv("结论", text(verdict.get("summary"), ip_quality_summary(ip_report))),
            kv("IP", f"{text(ip_report.get('public_ip'))} | {text(ip_report.get('country'))} {text(ip_report.get('city'))}"),
            kv("运营商", f"{text(ip_report.get('isp'))} / {text(ip_report.get('organization'))}"),
            kv("类型", f"{text(ip_report.get('ip_version'))} | {text(verdict.get('ip_type_label'), text(ip_report.get('ip_type')))}"),
            kv("网络栈", f"{text(network_stack.get('detected_version'))} | 双栈 {yes_no(network_stack.get('dual_stack'))}"),
            kv("风险", f"{text(ip_report.get('risk_level'))} | {text(ip_report.get('risk_score'))}/100", style_for_status(ip_report.get("risk_level"))),
            kv("黑名单", f"干净 {text(blacklists.get('clean'))}/{text(blacklists.get('total'))} | 命中 {text(blacklists.get('listed'))}"),
            kv("邮件端口", f"连通 {mail_ok}/{len(mail_checks)} | 服务商 {text(mail_summary.get('provider_open'), '0')}/{text(mail_summary.get('providers'), '0')}"),
        ])
        risk_sources = ip_report.get("risk_sources") if isinstance(ip_report.get("risk_sources"), list) else []
        if risk_sources:
            rows.extend(table(
                ["来源", "类型", "状态", "信号"],
                [[text(item.get("name")), text(item.get("type")), text(item.get("status")), text(item.get("signal"))] for item in risk_sources if isinstance(item, dict)],
                [22, 12, 10, 30],
                status_col=2,
            ))
        evidence_rows = ip_evidence_rows(ip_report)
        if evidence_rows:
            rows.extend(table(
                ["证据", "结果", "状态", "说明"],
                evidence_rows[:6],
                [12, 18, 10, 40],
                status_col=2,
            ))
        recommendations = ip_recommendations(ip_report)
        if recommendations:
            rows.append(kv("建议", recommendations[0], "yellow"))
        factors = [item for item in ip_report.get("risk_factors", []) if isinstance(item, dict) and item.get("detected")]
        if factors:
            rows.extend(table(
                ["风险项", "置信度", "说明"],
                [[text(item.get("name")), text(item.get("confidence")), text(item.get("detail"))] for item in factors[:5]],
                [14, 12, 46],
                status_col=1,
            ))
    else:
        rows.append(kv("状态", "未执行", "yellow"))

    rows += section("扩展检测")
    rows.extend(table(
        ["模块", "结果", "说明"],
        [
            ["路由追踪", ratio_or_skipped(route_ok, route_total, "成功"), "目标连通路径"],
            ["流媒体", ratio_or_skipped(stream_ok, stream_total, "可用"), "解锁区域检测"],
            ["AI 服务", ratio_or_skipped(ai_ok, ai_total, "可用"), "主流 AI 访问性"],
            ["安全体检", security_summary(default_summary.get("security_report"), security_total, security_severity), "端口/SSH/系统策略"],
            ["压力测试", stress_summary(stress_report), "full 档位或显式开启"],
        ],
        [14, 28, 30],
        status_col=1,
    ))
    rows.append(kv("路由说明", route_note, "yellow"))

    route_detail_rows = route_rows(default_summary.get("route_trace_results"))
    if route_detail_rows:
        rows += section("路由追踪明细")
        rows.extend(table(
            ["分组", "目标", "评级", "跳数", "超时", "平均延迟", "末跳/错误"],
            route_detail_rows,
            [14, 20, 8, 5, 5, 11, 19],
            status_col=2,
        ))
        if has_china_route_reference(route_detail_rows):
            rows.append(kv("注意", "国内方向参考是本机到国内目标的出站路径，不是真实回程。", "yellow"))

    streaming = default_summary.get("streaming_results")
    if isinstance(streaming, dict) and streaming:
        rows += section("流媒体解锁")
        rows.extend(table(
            ["分组", "平台", "状态", "区域", "解锁", "说明"],
            streaming_rows(streaming),
            [8, 16, 8, 10, 10, 24],
            status_col=2,
        ))

    ai = default_summary.get("ai_results")
    if isinstance(ai, dict) and ai:
        rows += section("AI 服务")
        rows.extend(table(
            ["分组", "服务", "状态", "访问", "区域", "说明"],
            ai_rows(ai),
            [8, 16, 8, 10, 10, 26],
            status_col=2,
        ))

    security = default_summary.get("security_report")
    findings = security.get("findings", []) if isinstance(security, dict) else []
    if findings:
        rows += section("安全体检")
        rows.append(kv("统计", f"共 {security_total} 条 | 高危 {security_severity['high']} | 中危 {security_severity['medium']} | 低危 {security_severity['low']} | 信息 {security_severity['info']}"))
        rows.extend(table(
            ["级别", "类别", "问题"],
            [[text(item.get("severity")), text(item.get("category")), text(item.get("title"))] for item in findings[:8] if isinstance(item, dict)],
            [10, 14, 48],
            status_col=0,
        ))

    if stress_report:
        rows += section("压力测试")
        rows.append(kv("总耗时", stress_summary(stress_report)))
        components = stress_report.get("components", []) if isinstance(stress_report.get("components"), list) else []
        if components:
            rows.extend(table(
                ["组件", "耗时", "状态", "说明"],
                [[text(item.get("name")), f"{num(item.get('duration_seconds'), 0)} 秒", status_text(item.get("status")), text(item.get("notes"))] for item in components if isinstance(item, dict)],
                [12, 10, 10, 36],
                status_col=2,
            ))

    rows += section("输出文件")
    rows.extend([
        kv("终端彩色", out / "console.ansi"),
        kv("终端纯文本", out / "console.txt"),
        kv("报告压缩包", out / "perfassess-report.zip"),
        kv("Markdown", out / "summary.md"),
        kv("JSON", out / "default.json"),
        kv("文本", out / "default.txt"),
        kv("硬件模块", out / "hardware_quality.json"),
        kv("网络模块", out / "net_quality.json"),
        kv("校准样本", out / "calibration_sample.json"),
        kv("路由模块", out / "route_trace.json"),
        kv("国内方向", out / "backroute_trace.json"),
        kv("IP 模块", out / "ip_quality.json"),
        kv("验收摘要", acceptance_summary_path),
    ])
    rows.append("")
    return "\n".join(rows)

lines = [
    "# Perfassess 自动测评报告",
    "",
    "## 总览",
    "",
    "| 项目 | 值 |",
    "|------|----|",
    f"| 版本 | {version} |",
    f"| 自动档位 | {auto_profile} |",
    f"| 质量档位 | {quality_profile} |",
    f"| 网络档位 | {network_profile} |",
    f"| 流媒体档位 | {streaming_profile} |",
    f"| 预算说明 | {budget_summary} |",
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
    f"| CPU | 单核 {summary_metric('cpu', 'single_core_score', 'cpu_result')} / 多核 {summary_metric('cpu', 'multi_core_score', 'cpu_result')} / 总分 {summary_metric('cpu', 'total_score', 'cpu_result')} |",
    f"| 内存 | 读 {summary_metric('memory', 'read_mbps', 'memory_result', 'read_speed_mbps')} MB/s / 写 {summary_metric('memory', 'write_mbps', 'memory_result', 'write_speed_mbps')} MB/s |",
    f"| 磁盘 | 读 {summary_metric('disk', 'sequential_read_mbps', 'disk_result')} MB/s / 写 {summary_metric('disk', 'sequential_write_mbps', 'disk_result')} MB/s / 随机 {summary_metric('disk', 'random_iops', 'disk_result', 'random_iops', 0)} IOPS |",
    f"| 网络 | 延迟 {summary_metric('network', 'latency_ms', 'network_result')} ms / 下载 {summary_metric('network', 'download_mbps', 'network_result', 'download_speed_mbps')} Mbps / 上传 {summary_metric('network', 'upload_mbps', 'network_result', 'upload_speed_mbps')} Mbps |",
    "",
]

if conclusion:
    bottlenecks = conclusion.get("bottlenecks", []) if isinstance(conclusion.get("bottlenecks"), list) else []
    evidence = conclusion.get("evidence", []) if isinstance(conclusion.get("evidence"), list) else []
    limitations = conclusion.get("limitations", []) if isinstance(conclusion.get("limitations"), list) else []
    recommendations = conclusion.get("recommendations", []) if isinstance(conclusion.get("recommendations"), list) else []
    suitability = conclusion.get("suitability", []) if isinstance(conclusion.get("suitability"), list) else []
    lines.extend([
        "## 测评结论",
        "",
        f"- 结论: {text(conclusion.get('headline'))}",
        f"- 适用判断: {text(conclusion.get('scenario'))}",
        f"- 适合场景: {', '.join(text(item) for item in suitability) if suitability else '-'}",
        "",
        "| 短板排序 | 评分 | 状态 |",
        "|----------|------|------|",
    ])
    for item in bottlenecks:
        if isinstance(item, dict):
            lines.append(f"| {md_cell(text(item.get('label')))} | {num(item.get('score'))} | {md_cell(bottleneck_label(item.get('severity')))} |")
    if evidence:
        lines.extend([
            "",
            "### 关键证据",
            "",
            "| 证据 | 结果 | 状态 | 说明 |",
            "|------|------|------|------|",
        ])
        for item in evidence[:5]:
            if isinstance(item, dict):
                lines.append(f"| {md_cell(text(item.get('label')))} | {md_cell(text(item.get('value')))} | {md_cell(evidence_label(item.get('status')))} | {md_cell(text(item.get('detail')))} |")
    lines.extend([
        "",
        "### 主要限制",
        "",
    ])
    lines.extend(f"- {text(item)}" for item in limitations[:5])
    lines.extend([
        "",
        "### 建议",
        "",
    ])
    lines.extend(f"- {text(item)}" for item in recommendations[:5])
    lines.append("")

module_rows = module_assessment_rows(module_assessments)
if module_rows:
    lines.extend([
        "## 模块可信度",
        "",
        "| 模块 | 状态 | 置信度 | 结论 | 首要证据 |",
        "|------|------|--------|------|----------|",
    ])
    for row in module_rows:
        lines.append(f"| {md_cell(row[0])} | {md_cell(row[1])} | {md_cell(row[2])} | {md_cell(row[3])} | {md_cell(row[4])} |")
    lines.append("")

timing_rows = progress_step_rows()
if timing_rows:
    lines.extend([
        "## 耗时统计",
        "",
        "| 步骤 | 状态 | 耗时 | 说明 |",
        "|------|------|------|------|",
    ])
    for row in timing_rows:
        lines.append(f"| {md_cell(row[0])} | {md_cell(row[1])} | {md_cell(row[2])} | {md_cell(row[3])} |")
    lines.append("")

iperf3_rows = iperf3_console_rows(network_metrics)
if iperf3_rows:
    lines.extend([
        "## iperf3 多节点矩阵",
        "",
        "| 节点 | 区域 | 提供方 | 协议 | 下载 | 上传 | 延迟 | 状态/错误 |",
        "|------|------|--------|------|------|------|------|-----------|",
    ])
    for row in iperf3_rows:
        lines.append(f"| {md_cell(row[0])} | {md_cell(row[1])} | {md_cell(row[2])} | {md_cell(row[3])} | {md_cell(row[4])} | {md_cell(row[5])} | {md_cell(row[6])} | {md_cell(row[7])} |")
    lines.append("")

lines.extend([
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
    f"| 路由说明 | {route_note} |",
    "",
])

if ip_report:
    verdict = ip_verdict(ip_report)
    blacklists = ip_report.get("blacklist_summary", {}) if isinstance(ip_report.get("blacklist_summary"), dict) else {}
    mail_summary = ip_report.get("mail_summary", {}) if isinstance(ip_report.get("mail_summary"), dict) else {}
    network_stack = ip_report.get("network_stack", {}) if isinstance(ip_report.get("network_stack"), dict) else {}
    ip_identity = f"{text(ip_report.get('public_ip'))} / {text(ip_report.get('ip_version'))}"
    ip_location = f"{text(ip_report.get('country'))} {text(ip_report.get('city'))}"
    ip_operator = f"{text(ip_report.get('isp'))} / {text(ip_report.get('organization'))}"
    ip_risk = f"{text(ip_report.get('risk_level'))} / {text(ip_report.get('risk_score'))}/100"
    ip_blacklist = f"干净 {text(blacklists.get('clean'))}/{text(blacklists.get('total'))}，命中 {text(blacklists.get('listed'))}"
    ip_mail = f"可连 {text(mail_summary.get('provider_open'), '0')}/{text(mail_summary.get('providers'), '0')}，端口 {text(mail_summary.get('reachable'), '0')}/{text(mail_summary.get('total'), '0')}"
    ip_stack = f"{text(network_stack.get('detected_version'))}，双栈 {yes_no(network_stack.get('dual_stack'))}"
    lines.extend([
        "## IP 质量",
        "",
        "| 项目 | 值 |",
        "|------|----|",
        f"| 结论 | {md_cell(text(verdict.get('summary'), ip_quality_summary(ip_report)))} |",
        f"| IP | {md_cell(ip_identity)} |",
        f"| 位置 | {md_cell(ip_location)} |",
        f"| 运营商 | {md_cell(ip_operator)} |",
        f"| 类型 | {md_cell(text(verdict.get('ip_type_label'), text(ip_report.get('ip_type'))))} |",
        f"| 风险 | {md_cell(ip_risk)} |",
        f"| 黑名单 | {md_cell(ip_blacklist)} |",
        f"| 邮件服务商 | {md_cell(ip_mail)} |",
        f"| 网络栈 | {md_cell(ip_stack)} |",
        "",
    ])
    risk_sources = ip_report.get("risk_sources") if isinstance(ip_report.get("risk_sources"), list) else []
    if risk_sources:
        lines.extend([
            "### IP 风险来源",
            "",
            *md_table(
                ["来源", "类型", "状态", "信号", "说明"],
                [[text(item.get("name")), text(item.get("type")), text(item.get("status")), text(item.get("signal")), text(item.get("detail"))] for item in risk_sources if isinstance(item, dict)],
            ),
            "",
        ])
    evidence_rows = ip_evidence_rows(ip_report)
    if evidence_rows:
        lines.extend(["### IP 证据", "", *md_table(["证据", "结果", "状态", "说明"], evidence_rows[:8]), ""])
    factor_rows = detected_ip_factors(ip_report)
    if factor_rows:
        lines.extend(["### 命中风险因子", "", *md_table(["风险项", "置信度", "说明"], factor_rows), ""])
    recommendations = ip_recommendations(ip_report)
    if recommendations:
        lines.extend(["### IP 建议", ""])
        lines.extend(f"- {item}" for item in recommendations[:5])
        lines.append("")

streaming_detail_rows = streaming_rows(default_summary.get("streaming_results"))
if streaming_detail_rows:
    lines.extend([
        "## 流媒体解锁明细",
        "",
        *md_table(["分组", "平台", "状态", "区域", "解锁", "说明"], streaming_detail_rows),
        "",
    ])

ai_detail_rows = ai_rows(default_summary.get("ai_results"))
if ai_detail_rows:
    lines.extend([
        "## AI 服务明细",
        "",
        *md_table(["分组", "服务", "状态", "访问", "区域", "说明"], ai_detail_rows),
        "",
    ])

route_detail_rows = route_rows(default_summary.get("route_trace_results"))
if route_detail_rows:
    lines.extend([
        "## 路由追踪明细",
        "",
        "| 分组 | 目标 | 评级 | 跳数 | 超时 | 平均延迟 | 末跳/错误 |",
        "|------|------|------|------|------|----------|-----------|",
    ])
    for row in route_detail_rows:
        lines.append(f"| {row[0]} | {row[1]} | {row[2]} | {row[3]} | {row[4]} | {row[5]} | {row[6]} |")
    lines.append("")
    if has_china_route_reference(route_detail_rows):
        lines.extend([
            "注意：国内方向参考是本机到国内目标的出站路径，不是真实回程。",
            "",
        ])

security = default_summary.get("security_report")
security_findings = security.get("findings", []) if isinstance(security, dict) and isinstance(security.get("findings"), list) else []
if security_findings:
    lines.extend([
        "## 安全体检明细",
        "",
        *md_table(
            ["级别", "类别", "问题", "建议"],
            [[text(item.get("severity")), text(item.get("category")), text(item.get("title")), text(item.get("advice"))] for item in security_findings[:10] if isinstance(item, dict)],
        ),
        "",
    ])

stress_components = stress_report.get("components", []) if isinstance(stress_report.get("components"), list) else []
if stress_components:
    lines.extend([
        "## 压力测试组件",
        "",
        *md_table(
            ["组件", "状态", "耗时", "备注"],
            [[text(item.get("name")), status_text(item.get("status")), duration_text(item.get("duration_seconds")), text(item.get("notes"))] for item in stress_components if isinstance(item, dict)],
        ),
        "",
    ])

lines.extend([
    "## 输出文件",
    "",
    f"- 终端彩色报告: {out / 'console.ansi'}",
    f"- 终端纯文本报告: {out / 'console.txt'}",
    f"- 报告压缩包: {out / 'perfassess-report.zip'}",
    f"- 报告产物清单: {out / 'artifact_manifest.json'}",
    f"- Markdown 摘要: {out / 'summary.md'}",
    f"- JSON 完整报告: {out / 'default.json'}",
    f"- 文本完整报告: {out / 'default.txt'}",
    f"- 硬件质量模块: {out / 'hardware_quality.json'}",
    f"- 网络质量模块: {out / 'net_quality.json'}",
    f"- 脱敏校准样本: {out / 'calibration_sample.json'}",
    f"- 路由追踪模块: {out / 'route_trace.json'}",
    f"- 国内方向参考模块: {out / 'backroute_trace.json'}",
    f"- IP 质量模块: {out / 'ip_quality.json'}",
    f"- 快速测评 JSON: {out / 'quick.json'}",
    f"- 依赖检查: {out / 'check-deps.txt'}",
    f"- 验收摘要: {acceptance_summary_path}",
])

share = default_summary.get("share_templates", {}).get("plain_text")
if share:
    lines.extend(["", "## 分享模板", "", "```text", share, "```"])

(out / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
json_dump(out / "default.json", default_report)
(out / "console.txt").write_text(console_report(color=False) + "\n", encoding="utf-8")
(out / "console.ansi").write_text(console_report(color=True) + "\n", encoding="utf-8")
(out / "default.txt").write_text(console_report(color=False) + "\n", encoding="utf-8")
write_artifact_manifest(include_archive=False)
write_report_archive()
write_artifact_manifest(include_archive=True)
PY
progress_update "summary" "success" "汇总已生成"
trap - EXIT

echo "perfassess auto passed. Output: $output_dir"
echo "Console: $output_dir/console.txt"
echo "Summary: $output_dir/summary.md"
echo "Archive: $output_dir/perfassess-report.zip"
