#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
binary="${1:-${PERFASSESS_BINARY:-$repo_root/build/perfassess}}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

if [[ -x "$binary" ]]; then
  "$binary" --quick --output-format json -o "$tmpdir/quick.json" >"$tmpdir/quick.stdout"
else
  (cd "$repo_root" && go run ./cmd --quick --output-format json -o "$tmpdir/quick.json" >"$tmpdir/quick.stdout")
fi

python3 -m json.tool "$tmpdir/quick.json" >/dev/null

python3 - "$tmpdir/quick.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    report = json.load(f)

summary = report.get("summary")
if not isinstance(summary, dict):
    raise SystemExit("missing summary object")

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
    raise SystemExit(f"missing summary fields: {missing}")

conclusion = summary.get("assessment_conclusion")
if not isinstance(conclusion, dict):
    raise SystemExit("assessment_conclusion must be an object")
for key in ["headline", "scenario", "suitability", "confidence", "recommendations", "evidence"]:
    if key not in conclusion:
        raise SystemExit(f"missing assessment_conclusion.{key}")
if not isinstance(conclusion.get("evidence"), list) or not conclusion["evidence"]:
    raise SystemExit("assessment_conclusion.evidence must be a non-empty list")
for index, item in enumerate(conclusion["evidence"], start=1):
    if not isinstance(item, dict):
        raise SystemExit(f"assessment_conclusion.evidence[{index}] must be an object")
    for key in ["label", "value", "status"]:
        if not item.get(key):
            raise SystemExit(f"assessment_conclusion.evidence[{index}] missing {key}")

share = summary.get("share_templates")
if not isinstance(share, dict) or not share.get("plain_text") or not share.get("markdown"):
    raise SystemExit("missing share_templates plain_text or markdown")

modules = summary.get("module_assessments")
if not isinstance(modules, dict):
    raise SystemExit("module_assessments must be an object")

for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
    item = modules.get(key)
    if not isinstance(item, dict):
        raise SystemExit(f"missing module assessment: {key}")
    for field in ["id", "title", "status", "confidence", "summary", "evidence", "limitations", "recommendations"]:
        if field not in item:
            raise SystemExit(f"missing module_assessments.{key}.{field}")
    if item.get("id") != key:
        raise SystemExit(f"module_assessments.{key}.id mismatch")
    if item.get("status") != "skipped" and not item.get("evidence"):
        raise SystemExit(f"module_assessments.{key}.evidence must not be empty when module is not skipped")

confidence = summary.get("confidence_level")
if not isinstance(confidence, dict) or not confidence.get("level"):
    raise SystemExit("missing confidence_level.level")

calibration = summary.get("score_calibration")
if not isinstance(calibration, dict) or not calibration.get("version"):
    raise SystemExit("missing score_calibration.version")
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
required_fragments = [
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
    "模块结论",
    "证据审计",
    "网络质量",
    "报告目录",
]
missing = [item for item in required_fragments if item not in html]
if missing:
    raise SystemExit(f"missing web report fragments: {missing}")

diagnostic_report = json.loads(json.dumps(report))
cpu_result = diagnostic_report.setdefault("test_results", {}).setdefault("cpu_result", {})
cpu_result["status"] = "failed"
cpu_result["error_message"] = "sysbench not found"
metrics = cpu_result.setdefault("metrics", {})
metrics["backend"] = "sysbench"
metrics["error_category"] = "missing_dependency"
metrics["error_stage"] = "cpu_sysbench_lookup"
metrics["error_hint"] = "安装 sysbench 后重试。"
diagnostic_html = module.render_report(diagnostic_report, output_dir)
for item in ["诊断提示", "错误诊断", "依赖缺失", "cpu_sysbench_lookup"]:
    if item not in diagnostic_html:
        raise SystemExit(f"missing diagnostic web fragment: {item}")

extended_report = json.loads(json.dumps(report))
summary = extended_report.setdefault("summary", {})
modules = summary.setdefault("module_assessments", {})
modules["ip_quality"] = {
    "id": "ip_quality",
    "title": "IP 质量",
    "status": "warning",
    "confidence": "high",
    "summary": "IP 风险中等，存在邮件连通限制。",
    "evidence": [{"label": "DNSBL", "value": "命中 1 / 总计 12", "status": "warning", "detail": "存在黑名单命中。"}],
    "limitations": ["IP 风控源为启发式判断。"],
    "recommendations": ["保留 ip_quality.json 复查风险来源。"],
}
modules["streaming"] = {
    "id": "streaming",
    "title": "流媒体解锁",
    "status": "warning",
    "confidence": "high",
    "summary": "部分平台可用。",
    "evidence": [{"label": "可用平台", "value": "2 / 3", "status": "warning", "detail": "部分平台区域受限。"}],
    "limitations": ["平台策略可能变化。"],
    "recommendations": ["使用 full 档位复测。"],
}
modules["ai_services"] = {
    "id": "ai_services",
    "title": "AI 服务",
    "status": "success",
    "confidence": "high",
    "summary": "主流 AI 服务可访问。",
    "evidence": [{"label": "可访问服务", "value": "4 / 4", "status": "success", "detail": "访问性检测通过。"}],
    "limitations": ["不替代账号风控验证。"],
    "recommendations": ["用真实账号路径复测。"],
}
summary["ip_quality_report"] = {
    "public_ip": "203.0.113.10",
    "risk_level": "medium",
    "risk_score": 42,
    "country": "US",
    "city": "Example",
    "blacklist_summary": {"listed": 1, "total": 12},
    "mail_summary": {"provider_open": 1, "providers": 2, "reachable": 1, "total": 4},
    "verdict": {"summary": "IP 风险中等。", "grade": "C", "risk_label": "中风险", "mail_usable": False},
}
summary["streaming_results"] = {
    "netflix": {"category": "global", "platform": "Netflix", "available": True, "region": "US", "unlock_type": "full", "protocol": "https", "message": "OK"}
}
summary["ai_results"] = {
    "openai": {"category": "chatbot", "service": "OpenAI", "available": True, "access_type": "full", "region_hint": "US", "message": "OK"}
}
extended_html = module.render_report(extended_report, output_dir)
for item in ["IP 风险中等", "DNSBL", "部分平台可用", "可用平台", "主流 AI 服务可访问", "可访问服务"]:
    if item not in extended_html:
        raise SystemExit(f"missing extended module assessment fragment: {item}")
PY

auto_dir="$tmpdir/auto"
PERFASSESS_AUTO_DIR="$auto_dir" \
PERFASSESS_AUTO_PROFILE=basic \
PERFASSESS_AUTO_ACCEPTANCE=0 \
PERFASSESS_AUTO_PROGRESS=0 \
PERFASSESS_SKIP_BUILD=1 \
PERFASSESS_BINARY="$binary" \
"$repo_root/scripts/perfassess-auto.sh" >/dev/null

python3 - "$auto_dir/calibration_sample.json" <<'PY'
import ipaddress
import json
import re
import sys

path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    sample = json.load(f)

if sample.get("schema_version") != "perfassess-calibration-sample-v1":
    raise SystemExit("unexpected calibration sample schema version")
if sample.get("redacted") is not True:
    raise SystemExit("calibration sample must be marked redacted")
privacy_note = sample.get("privacy_note")
if not isinstance(privacy_note, str) or "已排除" not in privacy_note or "原始日志" not in privacy_note:
    raise SystemExit("calibration sample privacy_note must document redaction boundary")
for forbidden_top_level in ["system_info", "test_results", "summary", "formatted_content", "session_id"]:
    if forbidden_top_level in sample:
        raise SystemExit(f"calibration sample must not embed raw report field: {forbidden_top_level}")
for key in ["tool", "environment", "scores", "score_calibration", "score_breakdown", "component_samples", "module_confidence"]:
    if key not in sample:
        raise SystemExit(f"missing calibration sample field: {key}")

environment = sample.get("environment")
if not isinstance(environment, dict):
    raise SystemExit("calibration sample environment must be an object")
allowed_environment_keys = {
    "cpu_model",
    "cpu_cores",
    "cpu_threads",
    "memory_total_mb",
    "disk_total_gb",
    "os",
    "architecture",
    "virtualization",
}
unexpected_environment_keys = sorted(set(environment) - allowed_environment_keys)
if unexpected_environment_keys:
    raise SystemExit(f"calibration sample environment has unexpected keys: {unexpected_environment_keys}")

modules = sample.get("module_confidence")
if not isinstance(modules, dict) or "route" not in modules:
    raise SystemExit("calibration sample missing route module confidence")
route_module = modules["route"]
if not isinstance(route_module, dict) or not route_module.get("status") or not route_module.get("confidence"):
    raise SystemExit("calibration sample route module confidence is incomplete")

ipv4_pattern = re.compile(r"\b(?:\d{1,3}\.){3}\d{1,3}\b")

def walk(value, path="$"):
    if isinstance(value, dict):
        for key, child in value.items():
            key_lower = str(key).lower()
            for forbidden in ["public_ip", "isp", "asn", "organization", "reverse_dns", "route_trace_results", "hops", "session_id"]:
                if forbidden in key_lower:
                    raise SystemExit(f"calibration sample leaks forbidden key: {path}.{key}")
            walk(child, f"{path}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            walk(child, f"{path}[{index}]")
    elif isinstance(value, str):
        for candidate in ipv4_pattern.findall(value):
            try:
                address = ipaddress.ip_address(candidate)
            except ValueError:
                continue
            if address.is_global:
                raise SystemExit(f"calibration sample leaks public IPv4 value at {path}: {candidate}")

walk(sample)
PY

python3 "$repo_root/scripts/calibration-summary.py" "$auto_dir/calibration_sample.json" --format json -o "$tmpdir/calibration-summary.json"
python3 - "$tmpdir/calibration-summary.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    summary = json.load(f)

if summary.get("schema_version") != "perfassess-calibration-summary-v1":
    raise SystemExit("calibration summary schema mismatch")
if summary.get("sample_count") != 1:
    raise SystemExit(f"expected one calibration sample, got {summary.get('sample_count')}")
policy = summary.get("policy")
if not isinstance(policy, dict) or not policy.get("level"):
    raise SystemExit("calibration summary missing dataset policy")
if not isinstance(policy.get("collection_plan"), list) or not policy["collection_plan"]:
    raise SystemExit("calibration summary missing collection plan")
profiles = summary.get("profile_policies")
if not isinstance(profiles, dict) or not profiles:
    raise SystemExit("calibration summary missing score profile policies")
rows = summary.get("samples")
if not isinstance(rows, list) or len(rows) != 1:
    raise SystemExit("calibration summary missing sample row")
if not rows[0].get("score_profile"):
    raise SystemExit("calibration sample row missing score profile")
PY

python3 - "$auto_dir/route_trace.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    route = json.load(f)

assessment = route.get("assessment")
if not isinstance(assessment, dict):
    raise SystemExit("route_trace.json missing assessment object")
if assessment.get("id") != "route":
    raise SystemExit("route_trace.json assessment id must be route")
for key in ["status", "confidence", "summary", "limitations", "recommendations"]:
    if key not in assessment:
        raise SystemExit(f"route_trace.json assessment missing {key}")
PY

python3 "$repo_root/scripts/verify-artifacts.py" "$auto_dir" >/dev/null

echo "report contract smoke passed"
