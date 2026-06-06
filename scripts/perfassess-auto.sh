#!/usr/bin/env bash
set -euo pipefail

output_dir="${PERFASSESS_AUTO_DIR:-/tmp/perfassess-auto}"
binary="${PERFASSESS_BINARY:-./build/perfassess}"
skip_build="${PERFASSESS_SKIP_BUILD:-0}"
optional_mode="${PERFASSESS_AUTO_OPTIONAL:-never}"
show_progress="${PERFASSESS_AUTO_PROGRESS:-1}"
progress_file="${PERFASSESS_PROGRESS_FILE:-$output_dir/progress.json}"
auto_profile="${PERFASSESS_AUTO_PROFILE:-standard}"
quality_profile="${PERFASSESS_QUALITY_PROFILE:-builtin}"
extra_args="${PERFASSESS_AUTO_ARGS:-}"
stress_enabled="${PERFASSESS_AUTO_STRESS:-0}"
iperf3_server="${PERFASSESS_IPERF3_SERVER:-}"
iperf3_servers="${PERFASSESS_IPERF3_SERVERS:-}"
iperf3_server_file="${PERFASSESS_IPERF3_SERVER_FILE:-}"
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

case "$quality_profile" in
  auto) quality_profile="builtin" ;;
  builtin|mainstream) ;;
  *)
    echo "perfassess auto failed: PERFASSESS_QUALITY_PROFILE must be auto, builtin, or mainstream" >&2
    exit 1
    ;;
esac

if [[ "$auto_profile" == "full" ]]; then
  stress_enabled="1"
fi

default_args=(--output-format json -o "$output_dir/default.json")
default_text_args=(-o "$output_dir/default.txt")
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
    ("终端彩色报告", "console.ansi"),
    ("终端纯文本报告", "console.txt"),
    ("报告压缩包", "perfassess-report.zip"),
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
python3 - "$output_dir" "$auto_profile" "$quality_profile" <<'PY'
import json
import pathlib
import sys
import unicodedata
import zipfile

out = pathlib.Path(sys.argv[1])
auto_profile = sys.argv[2]
quality_profile = sys.argv[3]

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

def status_text(value):
    if value in {"success", True}:
        return "完成"
    if value in {"failed", False}:
        return "失败"
    if value in {"skipped", None}:
        return "未执行"
    return text(value)

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
    if lowered in {"完成", "success", "ok", "可用", "通过", "低", "low"}:
        return "green"
    if lowered in {"未执行", "skipped", "medium", "中", "中等"}:
        return "yellow"
    if lowered in {"失败", "failed", "不可用", "high", "高"}:
        return "red"
    return "cyan"

def write_report_archive():
    archive_path = out / "perfassess-report.zip"
    include = [
        "console.ansi",
        "console.txt",
        "summary.md",
        "default.json",
        "default.txt",
        "quick.json",
        "check-deps.txt",
        "version.txt",
        "progress.json",
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

route_ok, route_total = route_count(default_summary.get("route_trace_results"))
stream_ok, stream_total = availability_count(default_summary.get("streaming_results"))
ai_ok, ai_total = availability_count(default_summary.get("ai_results"))
security_total, security_severity = security_count(default_summary.get("security_report"))
ip_report = default_summary.get("ip_quality_report") if isinstance(default_summary.get("ip_quality_report"), dict) else {}
stress_report = default_summary.get("stress_report") if isinstance(default_summary.get("stress_report"), dict) else {}
version = (out / "version.txt").read_text(encoding="utf-8").strip()
confidence = default_summary.get("confidence_level", {}).get("level")
calibration = default_summary.get("score_calibration", {}).get("version")
vps = default_summary.get("vps_benchmark_summary") if isinstance(default_summary.get("vps_benchmark_summary"), dict) else {}
system = vps.get("system", {}) if isinstance(vps.get("system"), dict) else {}
cpu = vps.get("cpu", {}) if isinstance(vps.get("cpu"), dict) else {}
memory = vps.get("memory", {}) if isinstance(vps.get("memory"), dict) else {}
disk = vps.get("disk", {}) if isinstance(vps.get("disk"), dict) else {}
network = vps.get("network", {}) if isinstance(vps.get("network"), dict) else {}
network_metrics = default_report.get("test_results", {}).get("network_result", {}).get("metrics", {})
if not isinstance(network_metrics, dict):
    network_metrics = {}

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
    mail_ok = sum(1 for item in mail_checks if isinstance(item, dict) and item.get("reachable"))

    rows = [
        c("bold", "Perfassess 自动测评报告"),
        line("═"),
        f"{kv('版本', version)}    {kv('档位', auto_profile, 'yellow')}    {kv('质量', quality_profile, 'yellow')}",
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

    rows += section("IP 质量")
    if ip_report:
        rows.extend([
            kv("IP", f"{text(ip_report.get('public_ip'))} | {text(ip_report.get('country'))} {text(ip_report.get('city'))}"),
            kv("运营商", f"{text(ip_report.get('isp'))} / {text(ip_report.get('organization'))}"),
            kv("类型", f"{text(ip_report.get('ip_version'))} | {text(ip_report.get('ip_type'))}"),
            kv("风险", f"{text(ip_report.get('risk_level'))} | {text(ip_report.get('risk_score'))}/100", style_for_status(ip_report.get("risk_level"))),
            kv("黑名单", f"干净 {text(blacklists.get('clean'))}/{text(blacklists.get('total'))} | 命中 {text(blacklists.get('listed'))}"),
            kv("邮件端口", f"连通 {mail_ok}/{len(mail_checks)}"),
        ])
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

    streaming = default_summary.get("streaming_results")
    if isinstance(streaming, dict) and streaming:
        rows += section("流媒体解锁")
        rows.extend(table(
            ["平台", "状态", "区域", "说明"],
            [[text(item.get("platform"), name), yes_no(item.get("available")), text(item.get("region")), text(item.get("message"))] for name, item in streaming.items() if isinstance(item, dict)],
            [18, 10, 12, 30],
            status_col=1,
        ))

    ai = default_summary.get("ai_results")
    if isinstance(ai, dict) and ai:
        rows += section("AI 服务")
        rows.extend(table(
            ["服务", "状态", "说明"],
            [[text(item.get("service"), name), yes_no(item.get("available")), text(item.get("message"))] for name, item in ai.items() if isinstance(item, dict)],
            [18, 10, 44],
            status_col=1,
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
    f"- 终端彩色报告: {out / 'console.ansi'}",
    f"- 终端纯文本报告: {out / 'console.txt'}",
    f"- 报告压缩包: {out / 'perfassess-report.zip'}",
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
(out / "console.txt").write_text(console_report(color=False) + "\n", encoding="utf-8")
(out / "console.ansi").write_text(console_report(color=True) + "\n", encoding="utf-8")
write_report_archive()
PY
progress_update "summary" "success" "汇总已生成"
trap - EXIT

echo "perfassess auto passed. Output: $output_dir"
echo "Console: $output_dir/console.txt"
echo "Summary: $output_dir/summary.md"
echo "Archive: $output_dir/perfassess-report.zip"
