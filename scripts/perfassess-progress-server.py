#!/usr/bin/env python3
import argparse
import html
import json
import mimetypes
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any, Optional
from urllib.parse import unquote, urlparse


PROGRESS_HTML = """<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Perfassess 实时测评</title>
  <style>
    :root {
      --bg: #f8fafd;
      --surface: #ffffff;
      --surface-2: #edf2fa;
      --text: #1f1f1f;
      --muted: #5f6368;
      --primary: #0b57d0;
      --green: #137333;
      --amber: #b06000;
      --red: #b3261e;
      --outline: #d9dee7;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--text);
      background: var(--bg);
      font-family: Inter, Roboto, "Noto Sans SC", Arial, sans-serif;
    }
    header {
      position: sticky;
      top: 0;
      z-index: 3;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
      padding: 18px 28px;
      background: rgba(248, 250, 253, 0.94);
      border-bottom: 1px solid var(--outline);
      backdrop-filter: blur(10px);
    }
    h1 { margin: 0; font-size: 22px; font-weight: 750; }
    .sub { margin-top: 4px; color: var(--muted); font-size: 13px; }
    .badge {
      display: inline-flex;
      align-items: center;
      min-height: 30px;
      padding: 6px 12px;
      border-radius: 999px;
      font-size: 13px;
      font-weight: 700;
      background: var(--surface-2);
      color: var(--muted);
      white-space: nowrap;
    }
    .success { color: #0d3b1e; background: #c4e7c6; }
    .running { color: #003355; background: #c2e7ff; }
    .failed { color: #601410; background: #f9dedc; }
    .pending { color: #4a3000; background: #fdd663; }
    .top-actions {
      display: inline-flex;
      align-items: center;
      gap: 10px;
    }
    .top-actions a {
      color: var(--primary);
      font-size: 13px;
      font-weight: 700;
      text-decoration: none;
      white-space: nowrap;
    }
    main {
      display: grid;
      grid-template-columns: 320px minmax(0, 1fr);
      gap: 18px;
      width: min(1180px, calc(100% - 32px));
      margin: 18px auto 32px;
    }
    nav, section {
      background: var(--surface);
      border: 1px solid var(--outline);
      border-radius: 8px;
      box-shadow: 0 1px 2px rgba(60, 64, 67, 0.08);
    }
    nav { padding: 12px; align-self: start; position: sticky; top: 88px; }
    .step {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 11px 10px;
      border-radius: 8px;
      color: inherit;
      text-decoration: none;
    }
    .step + .step { margin-top: 4px; }
    .step.active { background: var(--surface-2); }
    .step-name { font-size: 14px; font-weight: 700; }
    .content { display: grid; gap: 18px; }
    section { padding: 22px; }
    h2 { margin: 0 0 12px; font-size: 18px; }
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
      margin-top: 16px;
    }
    .metric {
      padding: 16px;
      background: var(--surface-2);
      border-radius: 8px;
    }
    .metric-label { color: var(--muted); font-size: 12px; font-weight: 700; }
    .metric-value { margin-top: 8px; color: var(--primary); font-size: 24px; font-weight: 780; }
    .message {
      margin: 0;
      color: var(--muted);
      line-height: 1.6;
      overflow-wrap: anywhere;
    }
    .bar {
      height: 10px;
      overflow: hidden;
      background: var(--surface-2);
      border-radius: 999px;
      margin-top: 16px;
    }
    .bar > div {
      height: 100%;
      width: 0;
      background: var(--primary);
      transition: width 240ms ease;
    }
    .artifacts {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 10px;
    }
    .artifact {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      padding: 12px 14px;
      border: 1px solid var(--outline);
      border-radius: 8px;
      color: var(--text);
      text-decoration: none;
      font-weight: 700;
    }
    .artifact[aria-disabled="true"] {
      color: var(--muted);
      background: var(--surface-2);
      pointer-events: none;
    }
    pre {
      max-height: 360px;
      overflow: auto;
      padding: 14px;
      background: #0f172a;
      color: #e2e8f0;
      border-radius: 8px;
      font-size: 12px;
      line-height: 1.55;
    }
    @media (max-width: 820px) {
      header { align-items: flex-start; flex-direction: column; }
      main { grid-template-columns: 1fr; }
      nav { position: static; }
      .summary-grid, .artifacts { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <h1>Perfassess 实时测评</h1>
      <div class="sub" id="subtitle">等待测评状态...</div>
    </div>
    <div class="top-actions">
      <a href="/report">完整报告</a>
      <span class="badge pending" id="overall">等待中</span>
    </div>
  </header>
  <main>
    <nav id="steps"></nav>
    <div class="content">
      <section>
        <h2>当前进度</h2>
        <p class="message" id="message">正在等待进度文件生成。</p>
        <div class="bar"><div id="bar"></div></div>
        <div class="summary-grid">
          <div class="metric"><div class="metric-label">已完成</div><div class="metric-value" id="done">0</div></div>
          <div class="metric"><div class="metric-label">总步骤</div><div class="metric-value" id="total">0</div></div>
          <div class="metric"><div class="metric-label">报告总分</div><div class="metric-value" id="score">-</div></div>
        </div>
      </section>
      <section>
        <h2>报告文件</h2>
        <div class="artifacts" id="artifacts"></div>
      </section>
      <section>
        <h2>JSON 状态</h2>
        <pre id="raw">{}</pre>
      </section>
    </div>
  </main>
  <script>
    const statusText = {pending: "等待", running: "进行中", success: "完成", failed: "失败"};
    function cls(status) { return ["pending", "running", "success", "failed"].includes(status) ? status : "pending"; }
    async function loadReportScore() {
      try {
        const resp = await fetch("/artifacts/default.json", {cache: "no-store"});
        if (!resp.ok) return "-";
        const report = await resp.json();
        const score = report && report.summary ? report.summary.total_score : null;
        return typeof score === "number" ? score.toFixed(2) : "-";
      } catch (_) {
        return "-";
      }
    }
    async function refresh() {
      const resp = await fetch("/api/progress", {cache: "no-store"});
      const data = await resp.json();
      const steps = data.steps || [];
      const done = steps.filter(s => s.status === "success").length;
      const failed = steps.filter(s => s.status === "failed").length;
      const active = data.current_step || "";
      const percent = steps.length ? Math.round((done / steps.length) * 100) : 0;

      document.getElementById("subtitle").textContent = `${data.output_dir || ""} · ${data.updated_at || ""}`;
      const overall = document.getElementById("overall");
      overall.className = `badge ${cls(data.status)}`;
      overall.textContent = statusText[data.status] || data.status || "等待";
      document.getElementById("message").textContent = data.message || "暂无状态信息。";
      document.getElementById("done").textContent = String(done);
      document.getElementById("total").textContent = String(steps.length);
      document.getElementById("bar").style.width = `${percent}%`;
      document.getElementById("raw").textContent = JSON.stringify(data, null, 2);

      document.getElementById("steps").innerHTML = steps.map(step => `
        <a class="step ${step.id === active ? "active" : ""}" href="#${step.id}">
          <span class="step-name">${step.label}</span>
          <span class="badge ${cls(step.status)}">${statusText[step.status] || step.status}</span>
        </a>`).join("");

      document.getElementById("artifacts").innerHTML = (data.artifacts || []).map(item => {
        const href = `/artifacts/${encodeURIComponent(item.path)}`;
        const disabled = item.available ? "false" : "true";
        const state = item.available ? "可打开" : "未生成";
        return `<a class="artifact" aria-disabled="${disabled}" href="${href}" target="_blank" rel="noreferrer"><span>${item.label}</span><span>${state}</span></a>`;
      }).join("");

      if (data.status === "success") {
        document.getElementById("score").textContent = await loadReportScore();
        if (window.location.pathname === "/") {
          setTimeout(() => { window.location.href = "/report"; }, 800);
        }
      }
      if (failed === 0 && data.status !== "success") {
        setTimeout(refresh, 1200);
      }
    }
    refresh();
  </script>
</body>
</html>
"""


def text(value: Any, default: str = "-") -> str:
    if value is None:
        return default
    if isinstance(value, str):
        value = value.strip()
        return value if value else default
    return str(value)


def esc(value: Any, default: str = "-") -> str:
    return html.escape(text(value, default), quote=True)


def fmt_number(value: Any, digits: int = 2, default: str = "-") -> str:
    if isinstance(value, bool) or value is None:
        return default
    if isinstance(value, (int, float)):
        return f"{value:.{digits}f}"
    try:
        return f"{float(value):.{digits}f}"
    except (TypeError, ValueError):
        return default


def fmt_int(value: Any, default: str = "-") -> str:
    if isinstance(value, bool) or value is None:
        return default
    try:
        return str(int(round(float(value))))
    except (TypeError, ValueError):
        return default


def data_get(data: dict[str, Any], *keys: str, default: Any = None) -> Any:
    current: Any = data
    for key in keys:
        if not isinstance(current, dict):
            return default
        current = current.get(key)
        if current is None:
            return default
    return current


def status_text(status: str) -> str:
    return {
        "success": "成功",
        "failed": "失败",
        "skipped": "跳过",
        "degraded": "降级",
        "warning": "注意",
        "running": "进行中",
        "pending": "等待",
        "not_run": "未执行",
    }.get(status or "", text(status, "未知"))


def status_class(status: str) -> str:
    status = status or "unknown"
    if status == "not_run":
        return "skipped"
    return status if status in {"success", "failed", "skipped", "degraded", "warning", "running", "pending"} else "unknown"


def metric(label: str, value: Any, unit: str = "", tone: str = "primary") -> dict[str, str]:
    return {"label": label, "value": text(value), "unit": unit, "tone": tone}


def detail(label: str, value: Any) -> dict[str, str]:
    return {"label": label, "value": text(value)}


def table(title: str, headers: list[str], rows: list[list[Any]]) -> dict[str, Any]:
    return {"title": title, "headers": headers, "rows": [[text(cell) for cell in row] for row in rows]}


def render_metrics(metrics: list[dict[str, str]]) -> str:
    if not metrics:
        return ""
    cards = []
    for item in metrics:
        unit = f'<div class="metric-unit">{esc(item.get("unit"))}</div>' if item.get("unit") else ""
        cards.append(
            f'<div class="metric-card tone-{esc(item.get("tone", "primary"))}">'
            f'<h3>{esc(item.get("label"))}</h3>'
            f'<div class="metric-value">{esc(item.get("value"))}</div>'
            f"{unit}</div>"
        )
    return f'<div class="metric-grid">{"".join(cards)}</div>'


def render_details(details: list[dict[str, str]]) -> str:
    if not details:
        return ""
    rows = [
        f'<div class="detail-row"><span class="detail-label">{esc(item.get("label"))}</span>'
        f'<span class="detail-value">{esc(item.get("value"))}</span></div>'
        for item in details
    ]
    return f'<div class="detail-grid">{"".join(rows)}</div>'


def render_tables(tables: list[dict[str, Any]]) -> str:
    rendered = []
    for item in tables:
        headers = "".join(f"<th>{esc(header)}</th>" for header in item.get("headers", []))
        body_rows = []
        for row in item.get("rows", []):
            body_rows.append("<tr>" + "".join(f"<td>{esc(cell)}</td>" for cell in row) + "</tr>")
        rendered.append(
            '<div class="data-table-wrap">'
            f'<h3 class="table-title">{esc(item.get("title"))}</h3>'
            '<table class="data-table"><thead><tr>'
            f"{headers}</tr></thead><tbody>{''.join(body_rows)}</tbody></table></div>"
        )
    return "".join(rendered)


def result_section(
    section_id: str,
    title: str,
    subtitle: str,
    result: Optional[dict[str, Any]],
    metrics: list[dict[str, str]],
    extra_details: list[dict[str, str]],
    hint: str,
) -> dict[str, Any]:
    if not result:
        return {
            "id": section_id,
            "title": title,
            "subtitle": subtitle,
            "status": "skipped",
            "status_text": "未执行",
            "summary": hint,
            "metrics": [],
            "details": [],
            "tables": [],
            "hint": hint,
        }

    status = text(result.get("status"), "unknown")
    result_metrics = result.get("metrics") if isinstance(result.get("metrics"), dict) else {}
    details = [
        detail("测试状态", status_text(status)),
        detail("耗时", f'{fmt_number(result.get("duration_seconds"))} 秒'),
    ]
    if result.get("error_message"):
        details.append(detail("错误信息", result.get("error_message")))
    details.extend(extra_details)

    metric_rows = [[key, value] for key, value in sorted(result_metrics.items()) if not isinstance(value, (dict, list))]
    tables = [table("原始指标", ["指标", "值"], metric_rows)] if metric_rows else []
    return {
        "id": section_id,
        "title": title,
        "subtitle": subtitle,
        "status": status_class(status),
        "status_text": status_text(status),
        "summary": key_metric_summary(title, result_metrics, status, hint),
        "metrics": metrics,
        "details": details,
        "tables": tables,
        "hint": "" if status == "success" else text(result.get("error_message"), ""),
    }


def key_metric_summary(title: str, metrics: dict[str, Any], status: str, hint: str) -> str:
    if status != "success":
        return hint
    if "CPU" in title:
        return f"单核 {fmt_number(metrics.get('single_core_score'))}，多核 {fmt_number(metrics.get('multi_core_score'))}，总分 {fmt_number(metrics.get('total_score'))}。"
    if "内存" in title:
        return f"读取 {fmt_number(metrics.get('read_speed_mbps'))} MB/s，写入 {fmt_number(metrics.get('write_speed_mbps'))} MB/s。"
    if "磁盘" in title:
        return f"顺序读取 {fmt_number(metrics.get('sequential_read_mbps') or metrics.get('read_speed_mbps'))} MB/s，顺序写入 {fmt_number(metrics.get('sequential_write_mbps') or metrics.get('write_speed_mbps'))} MB/s，随机 {fmt_int(metrics.get('random_iops'))} IOPS。"
    if "网络" in title:
        return f"延迟 {fmt_number(metrics.get('latency_ms'))} ms，下载 {fmt_number(metrics.get('download_speed_mbps'))} Mbps，上传 {fmt_number(metrics.get('upload_speed_mbps'))} Mbps。"
    return "本模块已执行。"


def simple_value_rows(value: Any) -> list[list[str]]:
    if isinstance(value, dict):
        rows = []
        for key, item in value.items():
            if isinstance(item, (dict, list)):
                rows.append([key, json.dumps(item, ensure_ascii=False)])
            else:
                rows.append([key, text(item)])
        return rows
    if isinstance(value, list):
        rows = []
        for index, item in enumerate(value, 1):
            rows.append([str(index), json.dumps(item, ensure_ascii=False) if isinstance(item, (dict, list)) else text(item)])
        return rows
    return [["结果", text(value)]]


def optional_section(section_id: str, title: str, subtitle: str, value: Any, hint: str) -> dict[str, Any]:
    if not value:
        return {
            "id": section_id,
            "title": title,
            "subtitle": subtitle,
            "status": "skipped",
            "status_text": "未执行",
            "summary": hint,
            "metrics": [],
            "details": [],
            "tables": [],
            "hint": hint,
        }
    row_count = len(value) if isinstance(value, (dict, list)) else 1
    return {
        "id": section_id,
        "title": title,
        "subtitle": subtitle,
        "status": "success",
        "status_text": "已执行",
        "summary": f"已生成 {row_count} 项检测数据。",
        "metrics": [metric("数据项", row_count, "", "primary")],
        "details": [],
        "tables": [table("检测结果", ["项目", "值"], simple_value_rows(value))],
        "hint": "",
    }


def ip_quality_section(summary: dict[str, Any]) -> dict[str, Any]:
    report = summary.get("ip_quality_report")
    hint = "本次未启用 IP 质量检测。使用 --ip-quality 或 --full 启用。"
    if not isinstance(report, dict):
        return optional_section("ip-quality", "IP 节点分析报告", "公网 IP、归属地、机房类型、欺诈风险、DNSBL 和邮件连通性。", None, hint)

    blacklist = report.get("blacklist_summary") if isinstance(report.get("blacklist_summary"), dict) else {}
    risk_factors = report.get("risk_factors") if isinstance(report.get("risk_factors"), list) else []
    blacklists = report.get("blacklist_checks") if isinstance(report.get("blacklist_checks"), list) else []
    mail_checks = report.get("mail_checks") if isinstance(report.get("mail_checks"), list) else []
    level = text(report.get("risk_level"), "unknown")
    status = "failed" if level == "high" else "warning" if level == "medium" else "success"
    tables = []
    if risk_factors:
        tables.append(table("风险因子", ["名称", "命中", "置信度", "来源", "说明"], [
            [item.get("name"), "是" if item.get("detected") else "否", item.get("confidence"), item.get("source"), item.get("detail")]
            for item in risk_factors if isinstance(item, dict)
        ]))
    if blacklists:
        tables.append(table("DNSBL 黑名单", ["区域", "状态", "命中", "说明"], [
            [item.get("zone"), item.get("status"), "是" if item.get("listed") else "否", item.get("detail")]
            for item in blacklists if isinstance(item, dict)
        ]))
    if mail_checks:
        tables.append(table("邮件端口连通性", ["目标", "端口", "状态", "可达", "说明"], [
            [item.get("target"), item.get("port"), item.get("status"), "是" if item.get("reachable") else "否", item.get("detail")]
            for item in mail_checks if isinstance(item, dict)
        ]))
    return {
        "id": "ip-quality",
        "title": "IP 节点分析报告",
        "subtitle": "公网 IP、归属地、机房类型、欺诈风险、DNSBL 和邮件连通性。",
        "status": status,
        "status_text": {"low": "低风险", "medium": "中风险", "high": "高风险"}.get(level, "未知"),
        "summary": f"{ip_node_type_summary(report)}，欺诈风险分 {text(report.get('risk_score'))}/100，综合评级 {ip_quality_grade(report)}。",
        "metrics": [
            metric("欺诈风险分", report.get("risk_score"), "/ 100", "red" if status == "failed" else "amber" if status == "warning" else "green"),
            metric("综合评级", ip_quality_grade(report), {"low": "低风险", "medium": "中风险", "high": "高风险"}.get(level, "未知"), "red" if status == "failed" else "amber" if status == "warning" else "green"),
            metric("DNSBL 命中", blacklist.get("listed", 0), f"/ {blacklist.get('total', len(blacklists))}", "amber"),
            metric("邮件可连", count_reachable_mail(mail_checks), f"/ {len(mail_checks)}", "cyan"),
        ],
        "details": [
            detail("IP 地址", report.get("public_ip")),
            detail("国家/地区", ", ".join(part for part in [text(report.get("country"), ""), text(report.get("city"), "")] if part)),
            detail("运营商/ASN", ip_asn_label(report)),
            detail("IP 类型", ip_node_type_summary(report)),
            detail("代理/VPN 标记", "是" if risk_factor_detected(report, "proxy") or risk_factor_detected(report, "vpn") else "否"),
            detail("机房/托管标记", "是" if risk_factor_detected(report, "datacenter") else "否"),
            detail("评级依据", ip_quality_basis(report, blacklist, mail_checks)),
        ],
        "tables": tables,
        "hint": " ".join(text(note, "") for note in report.get("notes", []) if note),
    }


def risk_factor_detected(report: dict[str, Any], name: str) -> bool:
    factors = report.get("risk_factors") if isinstance(report.get("risk_factors"), list) else []
    return any(isinstance(item, dict) and item.get("name") == name and item.get("detected") for item in factors)


def count_reachable_mail(checks: list[Any]) -> int:
    return sum(1 for item in checks if isinstance(item, dict) and item.get("reachable"))


def ip_asn_label(report: dict[str, Any]) -> str:
    parts = []
    if report.get("isp"):
        parts.append(text(report.get("isp")))
    if report.get("organization"):
        parts.append(text(report.get("organization")))
    if report.get("asn"):
        parts.append("AS" + text(report.get("asn")))
    return " / ".join(parts) if parts else "-"


def ip_node_type_summary(report: dict[str, Any]) -> str:
    value = report.get("ip_type")
    if value == "datacenter_likely":
        return "机房 IP (Datacenter/IDC)"
    if value == "residential_or_isp_likely":
        return "住宅或运营商 IP"
    return "IP 类型未知"


def ip_quality_grade(report: dict[str, Any]) -> str:
    level = text(report.get("risk_level"), "unknown")
    blacklist = report.get("blacklist_summary") if isinstance(report.get("blacklist_summary"), dict) else {}
    listed = blacklist.get("listed", 0)
    if level == "low" and listed == 0:
        return "A"
    if level == "low":
        return "B"
    if level == "medium":
        return "C"
    return "D"


def ip_quality_basis(report: dict[str, Any], blacklist: dict[str, Any], mail_checks: list[Any]) -> str:
    parts = [ip_node_type_summary(report), f"欺诈风险分 {text(report.get('risk_score'))}/100"]
    if blacklist:
        parts.append(f"DNSBL 命中 {text(blacklist.get('listed'), '0')}/{text(blacklist.get('total'), '0')}")
    if mail_checks:
        parts.append(f"邮件端口可连 {count_reachable_mail(mail_checks)}/{len(mail_checks)}")
    return " + ".join(parts)


def build_report_sections(report: dict[str, Any]) -> list[dict[str, Any]]:
    summary = report.get("summary") if isinstance(report.get("summary"), dict) else {}
    test_results = report.get("test_results") if isinstance(report.get("test_results"), dict) else {}
    system = report.get("system_info") if isinstance(report.get("system_info"), dict) else {}
    overall = summary.get("overall_score") if isinstance(summary.get("overall_score"), dict) else {}

    sections: list[dict[str, Any]] = [
        {
            "id": "overview",
            "title": "总览",
            "subtitle": "本次服务器测评的综合摘要。",
            "status": "success",
            "status_text": "已生成",
            "summary": f"总分 {fmt_number(summary.get('total_score') or overall.get('total_score'))} / 100，等级 {text(summary.get('grade') or overall.get('grade'))}，置信度 {text(data_get(summary, 'confidence_level', 'level'))}。",
            "metrics": [
                metric("总分", fmt_number(summary.get("total_score") or overall.get("total_score")), "/ 100", "primary"),
                metric("CPU", fmt_number(summary.get("cpu_score") or overall.get("cpu_score")), "/ 100", "primary"),
                metric("内存", fmt_number(summary.get("memory_score") or overall.get("memory_score")), "/ 100", "green"),
                metric("磁盘", fmt_number(summary.get("disk_score") or overall.get("disk_score")), "/ 100", "amber"),
                metric("网络", fmt_number(summary.get("network_score") or overall.get("network_score")), "/ 100", "cyan"),
            ],
            "details": [
                detail("评分基准", summary.get("score_profile")),
                detail("评测档位", data_get(summary, "benchmark_profile", "name")),
                detail("校准版本", data_get(summary, "score_calibration", "version")),
                detail("成功/失败/跳过", f"{text(summary.get('tests_success'), '0')} / {text(summary.get('tests_failed'), '0')} / {text(summary.get('tests_skipped'), '0')}"),
            ],
            "tables": [],
            "hint": text(summary.get("performance_note"), ""),
        }
    ]

    cpu = system.get("cpu") if isinstance(system.get("cpu"), dict) else {}
    memory_info = system.get("memory") if isinstance(system.get("memory"), dict) else {}
    disk_info = system.get("disk") if isinstance(system.get("disk"), dict) else {}
    os_info = system.get("os") if isinstance(system.get("os"), dict) else {}
    virt = system.get("virtualization") if isinstance(system.get("virtualization"), dict) else {}
    ip_info = system.get("ip_info") if isinstance(system.get("ip_info"), dict) else {}
    geo = ip_info.get("geo_location") if isinstance(ip_info.get("geo_location"), dict) else {}
    sections.append({
        "id": "system",
        "title": "系统信息",
        "subtitle": "硬件、系统、虚拟化和公网 IP 信息。",
        "status": "success" if system else "skipped",
        "status_text": "已采集" if system else "无数据",
        "summary": f"{text(cpu.get('model'))}，{text(cpu.get('cores'))}C/{text(cpu.get('threads'))}T，{fmt_number(memory_info.get('total_mb'), 0)} MB 内存，{fmt_number(disk_info.get('total_gb'))} GB 磁盘。",
        "metrics": [
            metric("CPU 核心", cpu.get("cores"), "C", "primary"),
            metric("线程", cpu.get("threads"), "T", "primary"),
            metric("内存", fmt_number(memory_info.get("total_mb"), 0), "MB", "green"),
            metric("磁盘", fmt_number(disk_info.get("total_gb")), "GB", "amber"),
        ],
        "details": [
            detail("CPU 型号", cpu.get("model")),
            detail("CPU 频率", f"{fmt_number(cpu.get('frequency_mhz'))} MHz"),
            detail("内存类型", memory_info.get("memory_type")),
            detail("可用内存", f"{fmt_number(memory_info.get('available_mb'), 0)} MB"),
            detail("磁盘类型", disk_info.get("disk_type")),
            detail("可用磁盘", f"{fmt_number(disk_info.get('available_gb'))} GB"),
            detail("系统", f"{text(os_info.get('name'))} {text(os_info.get('version'))}"),
            detail("架构", os_info.get("architecture")),
            detail("虚拟化", f"{text(virt.get('type'))} / {text(virt.get('vendor'))}"),
            detail("公网 IP", ip_info.get("public_ip")),
            detail("ISP", ip_info.get("isp")),
            detail("位置", ", ".join(part for part in [text(geo.get("country"), ""), text(geo.get("city"), "")] if part)),
        ],
        "tables": [],
        "hint": "",
    })

    cpu_result = test_results.get("cpu_result") if isinstance(test_results.get("cpu_result"), dict) else None
    cpu_metrics = cpu_result.get("metrics", {}) if cpu_result else {}
    sections.append(result_section("cpu", "CPU 测试", "单核、多核性能和采样稳定性。", cpu_result, [
        metric("CPU 评分", fmt_number(summary.get("cpu_score")), "/ 100", "primary"),
        metric("单核", fmt_number(cpu_metrics.get("single_core_score")), "", "primary"),
        metric("多核", fmt_number(cpu_metrics.get("multi_core_score")), "", "primary"),
        metric("后端", cpu_metrics.get("backend"), "", "primary"),
    ], [], "本次未执行 CPU 测试。"))

    memory_result = test_results.get("memory_result") if isinstance(test_results.get("memory_result"), dict) else None
    memory_metrics = memory_result.get("metrics", {}) if memory_result else {}
    sections.append(result_section("memory", "内存测试", "内存读写吞吐和采样稳定性。", memory_result, [
        metric("内存评分", fmt_number(summary.get("memory_score")), "/ 100", "green"),
        metric("读取速度", fmt_number(memory_metrics.get("read_speed_mbps")), "MB/s", "green"),
        metric("写入速度", fmt_number(memory_metrics.get("write_speed_mbps")), "MB/s", "green"),
        metric("后端", memory_metrics.get("backend"), "", "green"),
    ], [], "本次未执行内存测试。"))

    disk_result = test_results.get("disk_result") if isinstance(test_results.get("disk_result"), dict) else None
    disk_metrics = disk_result.get("metrics", {}) if disk_result else {}
    sections.append(result_section("disk", "磁盘测试", "顺序读写、随机 IOPS 和磁盘评分。", disk_result, [
        metric("磁盘评分", fmt_number(summary.get("disk_score")), "/ 100", "amber"),
        metric("顺序读取", fmt_number(disk_metrics.get("sequential_read_mbps") or disk_metrics.get("read_speed_mbps")), "MB/s", "amber"),
        metric("顺序写入", fmt_number(disk_metrics.get("sequential_write_mbps") or disk_metrics.get("write_speed_mbps")), "MB/s", "amber"),
        metric("随机 IOPS", fmt_int(disk_metrics.get("random_iops")), "IOPS", "amber"),
    ], [], "本次未执行磁盘测试。"))

    network_result = test_results.get("network_result") if isinstance(test_results.get("network_result"), dict) else None
    network_metrics = network_result.get("metrics", {}) if network_result else {}
    sections.append(result_section("network", "网络测试", "延迟、下载、上传、IPv4/IPv6 和多节点质量。", network_result, [
        metric("网络评分", fmt_number(summary.get("network_score")), "/ 100", "cyan"),
        metric("平均延迟", fmt_number(network_metrics.get("latency_ms")), "ms", "cyan"),
        metric("下载速度", fmt_number(network_metrics.get("download_speed_mbps")), "Mbps", "cyan"),
        metric("上传速度", fmt_number(network_metrics.get("upload_speed_mbps")), "Mbps", "cyan"),
    ], [], "本次未执行网络测试。"))

    sections.extend([
        optional_section("route", "路由追踪", "到主要地区和节点的网络路径质量。", summary.get("route_trace_results"), "本次未启用路由追踪。"),
        optional_section("streaming", "流媒体解锁", "Netflix、Disney+、YouTube 等平台的区域访问能力。", summary.get("streaming_results"), "本次未启用流媒体检测。"),
        optional_section("ai", "AI 服务检测", "OpenAI、Gemini 等 AI 服务的可访问性。", summary.get("ai_results"), "本次未启用 AI 服务检测。"),
        ip_quality_section(summary),
        optional_section("stress", "压力测试", "长时间 CPU、内存、磁盘压力下的稳定性。", summary.get("stress_report"), "本次未启用压力测试。"),
        optional_section("security", "安全体检", "端口、SSH 配置和基础安全风险检查。", summary.get("security_report"), "本次未启用安全体检。"),
    ])
    return sections


def render_report(report: dict[str, Any]) -> str:
    summary = report.get("summary") if isinstance(report.get("summary"), dict) else {}
    overall = summary.get("overall_score") if isinstance(summary.get("overall_score"), dict) else {}
    session_id = text(report.get("session_id"))
    timestamp = text(report.get("timestamp"))
    total_score = fmt_number(summary.get("total_score") or overall.get("total_score"), 0)
    grade = text(summary.get("grade") or overall.get("grade"))
    sections = build_report_sections(report)
    nav = "".join(
        f'<a class="nav-link" href="#{esc(section["id"])}"><span>{esc(section["title"])}</span>'
        f'<span class="status-badge status-{esc(section["status"])}">{esc(section["status_text"])}</span></a>'
        for section in sections
    )
    cards = []
    for section in sections:
        hint = f'<div class="hint-box">{esc(section.get("hint"))}</div>' if section.get("hint") else ""
        cards.append(
            f'<section id="{esc(section["id"])}" class="module-card">'
            '<div class="module-head"><div>'
            f'<h2 class="module-title">{esc(section["title"])}</h2>'
            f'<p class="module-subtitle">{esc(section["subtitle"])}</p>'
            f'</div><span class="status-badge status-{esc(section["status"])}">{esc(section["status_text"])}</span></div>'
            f'<p class="module-summary">{esc(section["summary"])}</p>'
            f'{render_metrics(section.get("metrics", []))}'
            f'{render_details(section.get("details", []))}'
            f'{render_tables(section.get("tables", []))}'
            f'{hint}</section>'
        )

    quality_notes = summary.get("quality_notes") if isinstance(summary.get("quality_notes"), list) else []
    if quality_notes:
        items = "".join(f"<li>{esc(note)}</li>" for note in quality_notes)
        nav += '<a class="nav-link" href="#quality"><span>质量提示</span><span class="status-badge status-skipped">提示</span></a>'
        cards.append(
            '<section id="quality" class="module-card quality-notes"><div class="module-head"><div>'
            '<h2 class="module-title">质量提示</h2><p class="module-subtitle">影响报告置信度和可比性的说明。</p>'
            '</div><span class="status-badge status-skipped">提示</span></div>'
            f"<ul>{items}</ul></section>"
        )

    share = data_get(summary, "share_templates", "plain_text", default="")
    if share:
        nav += '<a class="nav-link" href="#share"><span>分享模板</span><span class="status-badge status-success">可复制</span></a>'
        cards.append(
            '<section id="share" class="module-card"><div class="module-head"><div>'
            '<h2 class="module-title">分享模板</h2><p class="module-subtitle">可直接复制到论坛、工单或聊天窗口。</p>'
            '</div><span class="status-badge status-success">可复制</span></div>'
            f'<div class="share-box">{esc(share)}</div></section>'
        )

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Perfassess 完整报告 - {esc(session_id)}</title>
  <style>
    :root {{
      --bg: #f7faff;
      --surface: #ffffff;
      --surface-2: #f1f4f9;
      --surface-3: #e9eef6;
      --text: #1f1f1f;
      --muted: #5f6368;
      --primary: #0b57d0;
      --primary-container: #d3e3fd;
      --green: #146c2e;
      --amber: #b06000;
      --cyan: #00639b;
      --red: #b3261e;
      --outline: #d9dee7;
    }}
    * {{ box-sizing: border-box; }}
    html {{ scroll-behavior: smooth; }}
    body {{
      margin: 0;
      min-height: 100vh;
      color: var(--text);
      background: linear-gradient(180deg, #f7faff 0%, #fffdf7 54%, #f4f7fb 100%);
      font-family: Inter, Roboto, "Noto Sans SC", Arial, sans-serif;
    }}
    .app-shell {{ width: min(1440px, calc(100% - 32px)); margin: 0 auto; padding: 24px 0 42px; }}
    .top-app-bar {{ display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; }}
    .brand {{ display: flex; align-items: center; gap: 12px; min-width: 0; }}
    .brand-mark {{ width: 44px; height: 44px; border-radius: 8px; display: grid; place-items: center; color: #fff; background: var(--primary); font-weight: 800; }}
    .brand-title {{ font-size: 18px; font-weight: 760; }}
    .brand-subtitle {{ color: var(--muted); font-size: 13px; margin-top: 2px; }}
    .actions {{ display: flex; flex-wrap: wrap; gap: 8px; justify-content: flex-end; }}
    .chip, .action-link, .status-badge {{ display: inline-flex; align-items: center; min-height: 30px; padding: 5px 11px; border-radius: 999px; font-size: 12px; font-weight: 720; }}
    .chip, .action-link {{ color: #041e49; background: var(--primary-container); text-decoration: none; }}
    .hero {{ overflow: hidden; border-radius: 8px; padding: 32px; color: #fff; background: linear-gradient(135deg, rgba(11, 87, 208, 0.96), rgba(0, 99, 155, 0.92)); box-shadow: 0 2px 6px rgba(60, 64, 67, 0.16), 0 8px 24px rgba(60, 64, 67, 0.10); }}
    .hero-content {{ display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 28px; align-items: end; }}
    .eyebrow {{ display: inline-flex; border-radius: 999px; padding: 7px 12px; background: rgba(255, 255, 255, 0.16); font-size: 13px; font-weight: 700; }}
    .hero h1 {{ margin: 18px 0 10px; font-size: clamp(34px, 6vw, 64px); line-height: 0.98; letter-spacing: 0; }}
    .hero p {{ margin: 0; max-width: 760px; color: rgba(255, 255, 255, 0.86); font-size: 15px; line-height: 1.8; }}
    .chip-row {{ display: flex; flex-wrap: wrap; gap: 10px; margin-top: 20px; }}
    .hero-score {{ width: 190px; min-height: 190px; border-radius: 8px; padding: 22px; display: grid; align-content: center; text-align: center; background: rgba(255, 255, 255, 0.17); border: 1px solid rgba(255, 255, 255, 0.26); }}
    .score-number {{ font-size: 64px; line-height: 1; font-weight: 820; }}
    .score-label {{ margin-top: 8px; color: rgba(255, 255, 255, 0.82); font-size: 13px; }}
    .report-layout {{ display: grid; grid-template-columns: 260px minmax(0, 1fr); gap: 22px; align-items: start; margin-top: 22px; }}
    .sidebar {{ position: sticky; top: 18px; max-height: calc(100vh - 36px); overflow: auto; border-radius: 8px; padding: 14px; background: rgba(255, 255, 255, 0.86); border: 1px solid var(--outline); box-shadow: 0 1px 2px rgba(60, 64, 67, 0.10); }}
    .sidebar-title {{ padding: 8px 10px 12px; color: var(--muted); font-size: 12px; font-weight: 800; }}
    .nav-link {{ display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 38px; padding: 8px 10px; border-radius: 8px; color: var(--text); text-decoration: none; font-size: 14px; font-weight: 680; }}
    .nav-link:hover {{ background: var(--primary-container); color: #041e49; }}
    .content-stack {{ display: grid; gap: 18px; }}
    .module-card {{ scroll-margin-top: 20px; border-radius: 8px; padding: 22px; background: rgba(255, 255, 255, 0.86); border: 1px solid var(--outline); box-shadow: 0 1px 2px rgba(60, 64, 67, 0.10); }}
    .module-head {{ display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; margin-bottom: 18px; }}
    .module-title {{ margin: 0; font-size: 24px; line-height: 1.2; }}
    .module-subtitle {{ margin: 7px 0 0; color: var(--muted); font-size: 14px; line-height: 1.6; }}
    .module-summary {{ margin: 0 0 16px; color: var(--text); font-size: 14px; line-height: 1.7; }}
    .metric-grid {{ display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 14px; }}
    .metric-card {{ min-height: 104px; border-radius: 8px; padding: 16px; background: var(--surface-2); border: 1px solid var(--outline); }}
    .metric-card h3 {{ margin: 0 0 8px; color: var(--muted); font-size: 12px; font-weight: 760; }}
    .metric-value {{ color: var(--primary); font-size: 34px; line-height: 1.05; font-weight: 840; overflow-wrap: anywhere; }}
    .metric-unit {{ margin-top: 5px; color: var(--muted); font-size: 12px; font-weight: 650; }}
    .tone-green .metric-value {{ color: var(--green); }}
    .tone-amber .metric-value {{ color: var(--amber); }}
    .tone-cyan .metric-value {{ color: var(--cyan); }}
    .tone-red .metric-value {{ color: var(--red); }}
    .detail-grid {{ display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; margin-top: 4px; }}
    .detail-row {{ display: flex; justify-content: space-between; gap: 14px; padding: 11px 0; border-bottom: 1px solid var(--outline); }}
    .detail-label {{ color: var(--muted); font-size: 13px; white-space: nowrap; }}
    .detail-value {{ min-width: 0; color: var(--text); font-size: 13px; font-weight: 650; text-align: right; overflow-wrap: anywhere; }}
    .status-success {{ color: #0d3b1e; background: #c4e7c6; }}
    .status-degraded, .status-warning {{ color: #003355; background: #c2e7ff; }}
    .status-skipped {{ color: #4a3000; background: #fdd663; }}
    .status-failed {{ color: #601410; background: #f9dedc; }}
    .status-running {{ color: #003355; background: #c2e7ff; }}
    .status-unknown {{ color: var(--muted); background: var(--surface-3); }}
    .hint-box {{ margin-top: 14px; border-radius: 8px; padding: 14px 16px; color: #4a3000; background: #fff7df; border: 1px solid #fdd663; font-size: 14px; line-height: 1.65; }}
    .data-table-wrap {{ margin-top: 18px; overflow-x: auto; }}
    .table-title {{ margin: 0 0 10px; color: var(--muted); font-size: 13px; font-weight: 780; }}
    .data-table {{ width: 100%; border-collapse: collapse; overflow: hidden; border-radius: 8px; }}
    .data-table th, .data-table td {{ padding: 13px 12px; text-align: left; border-bottom: 1px solid var(--outline); vertical-align: top; font-size: 13px; line-height: 1.5; }}
    .data-table th {{ color: var(--muted); background: var(--surface-2); font-size: 12px; font-weight: 820; }}
    .quality-notes {{ border-color: #fdd663; background: #fffaf0; }}
    .quality-notes ul {{ margin: 10px 0 0; padding-left: 20px; color: #4a3000; line-height: 1.7; }}
    .share-box {{ white-space: pre-wrap; overflow-x: auto; border-radius: 8px; padding: 16px; color: var(--text); background: var(--surface-2); border: 1px solid var(--outline); font: 13px/1.65 "Roboto Mono", Consolas, monospace; }}
    .footer {{ margin-top: 24px; padding: 22px; text-align: center; color: var(--muted); font-size: 13px; }}
    @media (max-width: 1100px) {{
      .report-layout {{ grid-template-columns: 1fr; }}
      .sidebar {{ position: static; max-height: none; display: flex; gap: 8px; overflow-x: auto; }}
      .sidebar-title {{ display: none; }}
      .nav-link {{ flex: 0 0 auto; }}
    }}
    @media (max-width: 760px) {{
      .app-shell {{ width: min(100% - 20px, 1440px); padding-top: 12px; }}
      .hero, .module-card {{ padding: 18px; }}
      .hero-content, .detail-grid {{ grid-template-columns: 1fr; }}
      .hero-score {{ width: 100%; min-height: 130px; }}
      .metric-grid {{ grid-template-columns: repeat(2, minmax(0, 1fr)); }}
      .module-head {{ flex-direction: column; }}
    }}
    @media (max-width: 480px) {{ .metric-grid {{ grid-template-columns: 1fr; }} }}
  </style>
</head>
<body>
  <main class="app-shell">
    <div class="top-app-bar">
      <div class="brand">
        <div class="brand-mark">P</div>
        <div><div class="brand-title">Perfassess</div><div class="brand-subtitle">Material Design 3 Web Report</div></div>
      </div>
      <div class="actions">
        <a class="action-link" href="/progress">实时进度</a>
        <a class="action-link" href="/artifacts/default.json" target="_blank" rel="noreferrer">JSON</a>
        <a class="action-link" href="/artifacts/default.txt" target="_blank" rel="noreferrer">文本报告</a>
        <a class="action-link" href="/artifacts/acceptance/summary.md" target="_blank" rel="noreferrer">验收摘要</a>
        <span class="chip">Session {esc(session_id)}</span>
      </div>
    </div>
    <section class="hero">
      <div class="hero-content">
        <div>
          <span class="eyebrow">VPS Benchmark Report</span>
          <h1>性能评估报告</h1>
          <p>生成时间 {esc(timestamp)}。报告按模块组织系统信息、CPU、内存、磁盘、网络和扩展检测，左侧目录可快速跳转。</p>
          <div class="chip-row">
            <span class="chip">评分基准 {esc(summary.get("score_profile"))}</span>
            <span class="chip">评测档位 {esc(data_get(summary, "benchmark_profile", "name"))}</span>
            <span class="chip">置信度 {esc(data_get(summary, "confidence_level", "level"))}</span>
            <span class="chip">校准 {esc(data_get(summary, "score_calibration", "version"))}</span>
          </div>
        </div>
        <div class="hero-score">
          <div class="score-number">{esc(total_score)}</div>
          <div class="score-label">总体评分 / 100</div>
          <div style="margin-top: 14px;"><span class="status-badge status-success">{esc(grade)}</span></div>
        </div>
      </div>
    </section>
    <section class="report-layout">
      <nav class="sidebar" aria-label="报告目录">
        <div class="sidebar-title">报告目录</div>
        {nav}
      </nav>
      <div class="content-stack">{''.join(cards)}</div>
    </section>
    <footer class="footer">Perfassess Web 报告</footer>
  </main>
</body>
</html>"""


def load_json_file(path: Path) -> Optional[dict[str, Any]]:
    if not path.exists():
        return None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
    return data if isinstance(data, dict) else None


def report_ready(output_dir: Path, progress_file: Path) -> bool:
    if not (output_dir / "default.json").is_file():
        return False
    progress = load_json_file(progress_file)
    return bool(progress and progress.get("status") == "success")


def safe_artifact_path(output_dir: Path, raw_path: str) -> Optional[Path]:
    relative = Path(unquote(raw_path)).as_posix().lstrip("/")
    candidate = (output_dir / relative).resolve()
    try:
        candidate.relative_to(output_dir.resolve())
    except ValueError:
        return None
    return candidate


class ProgressHandler(BaseHTTPRequestHandler):
    output_dir: Path
    progress_file: Path

    def log_message(self, fmt, *args):
        return

    def send_bytes(self, body: bytes, content_type: str, status=HTTPStatus.OK):
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path == "/":
            if report_ready(self.output_dir, self.progress_file):
                report = load_json_file(self.output_dir / "default.json")
                if report is not None:
                    self.send_bytes(render_report(report).encode("utf-8"), "text/html; charset=utf-8")
                    return
            self.send_bytes(PROGRESS_HTML.encode("utf-8"), "text/html; charset=utf-8")
            return
        if parsed.path == "/progress":
            self.send_bytes(PROGRESS_HTML.encode("utf-8"), "text/html; charset=utf-8")
            return
        if parsed.path == "/report":
            report = load_json_file(self.output_dir / "default.json")
            if report is None:
                self.send_bytes(PROGRESS_HTML.encode("utf-8"), "text/html; charset=utf-8")
                return
            self.send_bytes(render_report(report).encode("utf-8"), "text/html; charset=utf-8")
            return
        if parsed.path == "/api/progress":
            if self.progress_file.exists():
                body = self.progress_file.read_bytes()
            else:
                body = json.dumps({
                    "title": "Perfassess 实时测评",
                    "status": "pending",
                    "message": "等待自动测评开始。",
                    "output_dir": str(self.output_dir),
                    "steps": [],
                    "artifacts": [],
                }, ensure_ascii=False).encode("utf-8")
            self.send_bytes(body, "application/json; charset=utf-8")
            return
        if parsed.path.startswith("/artifacts/"):
            artifact = safe_artifact_path(self.output_dir, parsed.path.removeprefix("/artifacts/"))
            if artifact is None or not artifact.is_file():
                self.send_error(HTTPStatus.NOT_FOUND)
                return
            content_type = mimetypes.guess_type(artifact.name)[0] or "text/plain"
            self.send_bytes(artifact.read_bytes(), content_type)
            return
        self.send_error(HTTPStatus.NOT_FOUND)


def main():
    parser = argparse.ArgumentParser(description="Serve Perfassess auto progress.")
    parser.add_argument("--dir", default="/tmp/perfassess-auto", help="Auto output directory.")
    parser.add_argument("--progress-file", default="", help="Progress JSON path.")
    parser.add_argument("--host", default="0.0.0.0", help="Listen host.")
    parser.add_argument("--port", type=int, default=8080, help="Listen port.")
    args = parser.parse_args()

    output_dir = Path(args.dir).resolve()
    progress_file = Path(args.progress_file).resolve() if args.progress_file else output_dir / "progress.json"
    output_dir.mkdir(parents=True, exist_ok=True)

    ProgressHandler.output_dir = output_dir
    ProgressHandler.progress_file = progress_file
    server = ThreadingHTTPServer((args.host, args.port), ProgressHandler)
    print(f"Perfassess progress server: http://{args.host}:{args.port}", flush=True)
    print(f"Output directory: {output_dir}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
