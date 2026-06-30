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
    .activity {
      margin-top: 8px;
      color: var(--text);
      font-weight: 700;
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
        <p class="message activity" id="activity">当前活动：等待测评日志。</p>
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
      const activeStep = steps.find(s => s.id === active) || {};
      const activity = data.activity || activeStep.activity || "";
      document.getElementById("activity").textContent = activity ? `当前活动：${activity}` : "当前活动：等待测评日志。";
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


def module_assessment_tables(assessment: Any) -> list[dict[str, Any]]:
    if not isinstance(assessment, dict):
        return []

    tables: list[dict[str, Any]] = [
        table("模块结论", ["项目", "内容"], [
            ["状态", module_status_label(assessment.get("status"))],
            ["置信度", assessment.get("confidence")],
            ["结论", assessment.get("summary")],
        ])
    ]

    evidence = assessment.get("evidence") if isinstance(assessment.get("evidence"), list) else []
    evidence_rows = []
    for item in evidence:
        if not isinstance(item, dict):
            continue
        evidence_rows.append([
            item.get("label"),
            item.get("value"),
            evidence_label(text(item.get("status"))),
            item.get("detail"),
        ])
    if evidence_rows:
        tables.append(table("证据审计", ["证据", "结果", "状态", "说明"], evidence_rows))

    limitations = assessment.get("limitations") if isinstance(assessment.get("limitations"), list) else []
    recommendations = assessment.get("recommendations") if isinstance(assessment.get("recommendations"), list) else []
    action_rows = []
    for item in limitations:
        action_rows.append(["限制", item])
    for item in recommendations:
        action_rows.append(["建议", item])
    if action_rows:
        tables.append(table("限制与建议", ["类型", "内容"], action_rows))

    return tables


def render_artifact_links(artifacts: list[dict[str, str]]) -> str:
    if not artifacts:
        return ""
    links = []
    for item in artifacts:
        available = item.get("available") == "true"
        href = f'/artifacts/{item.get("path", "")}'
        attrs = f'href="{esc(href)}" target="_blank" rel="noreferrer"' if available else 'aria-disabled="true"'
        status = "可下载" if available else "未生成"
        links.append(
            f'<a class="artifact-link {"artifact-ready" if available else "artifact-missing"}" {attrs}>'
            f'<span><strong>{esc(item.get("label"))}</strong><small>{esc(item.get("path"))}</small></span>'
            f'<span>{esc(status)}</span></a>'
        )
    return f'<div class="artifact-grid">{"".join(links)}</div>'


def first_text(items: Any, fallback: str = "-") -> str:
    if isinstance(items, list):
        for item in items:
            value = text(item, "")
            if value:
                return value
    return fallback


def top_evidence_items(conclusion: dict[str, Any], limit: int = 4) -> list[dict[str, str]]:
    evidence = conclusion.get("evidence") if isinstance(conclusion.get("evidence"), list) else []
    rows = []
    for item in evidence[:limit]:
        if not isinstance(item, dict):
            continue
        rows.append({
            "label": text(item.get("label")),
            "value": text(item.get("value")),
            "status": evidence_label(text(item.get("status"))),
            "status_class": status_class(text(item.get("status"))),
        })
    return rows


def module_health_items(modules: dict[str, Any]) -> list[dict[str, str]]:
    rows = []
    for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
        item = modules.get(key)
        if not isinstance(item, dict):
            continue
        rows.append({
            "title": text(item.get("title")),
            "status": module_status_label(item.get("status")),
            "status_class": status_class(text(item.get("status"))),
            "confidence": text(item.get("confidence")),
        })
    return rows


def evidence_map_groups(summary: dict[str, Any]) -> list[dict[str, Any]]:
    conclusion = summary.get("assessment_conclusion") if isinstance(summary.get("assessment_conclusion"), dict) else {}
    modules = summary.get("module_assessments") if isinstance(summary.get("module_assessments"), dict) else {}
    quality_notes = summary.get("quality_notes") if isinstance(summary.get("quality_notes"), list) else []

    groups = [
        {
            "title": "核心性能证据",
            "subtitle": "CPU、内存、磁盘的后端、关键指标和可信度。",
            "keys": ["cpu", "memory", "disk"],
            "rows": [],
        },
        {
            "title": "网络与连通证据",
            "subtitle": "网络吞吐、IP 质量、流媒体和 AI 服务可达性。",
            "keys": ["network", "route", "ip_quality", "streaming", "ai_services"],
            "rows": [],
        },
    ]
    for group in groups:
        for key in group["keys"]:
            item = modules.get(key)
            if not isinstance(item, dict) or item.get("status") == "skipped":
                continue
            evidence = item.get("evidence") if isinstance(item.get("evidence"), list) else []
            if evidence:
                for evidence_item in evidence[:3]:
                    if not isinstance(evidence_item, dict):
                        continue
                    group["rows"].append({
                        "module": text(item.get("title")),
                        "evidence": text(evidence_item.get("label")),
                        "value": text(evidence_item.get("value")),
                        "status": evidence_label(text(evidence_item.get("status"))),
                        "status_class": status_class(text(evidence_item.get("status"))),
                    })
            else:
                group["rows"].append({
                    "module": text(item.get("title")),
                    "evidence": "模块结论",
                    "value": text(item.get("summary")),
                    "status": module_status_label(item.get("status")),
                    "status_class": status_class(text(item.get("status"))),
                })

    conclusion_rows = []
    evidence = conclusion.get("evidence") if isinstance(conclusion.get("evidence"), list) else []
    for item in evidence[:6]:
        if isinstance(item, dict):
            conclusion_rows.append({
                "module": "总评",
                "evidence": text(item.get("label")),
                "value": text(item.get("value")),
                "status": evidence_label(text(item.get("status"))),
                "status_class": status_class(text(item.get("status"))),
            })
    if conclusion_rows:
        groups.insert(0, {
            "title": "总评证据",
            "subtitle": "支撑最终等级、适用场景和主要短板的证据。",
            "keys": [],
            "rows": conclusion_rows,
        })

    risk_rows = []
    limitations = conclusion.get("limitations") if isinstance(conclusion.get("limitations"), list) else []
    recommendations = conclusion.get("recommendations") if isinstance(conclusion.get("recommendations"), list) else []
    for item in limitations[:3]:
        risk_rows.append({
            "module": "限制",
            "evidence": "主要限制",
            "value": text(item),
            "status": "注意",
            "status_class": "warning",
        })
    for item in recommendations[:3]:
        risk_rows.append({
            "module": "建议",
            "evidence": "优先建议",
            "value": text(item),
            "status": "建议",
            "status_class": "success",
        })
    for item in quality_notes[:3]:
        risk_rows.append({
            "module": "质量提示",
            "evidence": "可比性",
            "value": text(item),
            "status": "提示",
            "status_class": "skipped",
        })
    if risk_rows:
        groups.append({
            "title": "限制与建议",
            "subtitle": "影响置信度、可比性和下一步复测策略的提示。",
            "keys": [],
            "rows": risk_rows,
        })

    return [group for group in groups if group["rows"]]


def render_evidence_map(summary: dict[str, Any]) -> str:
    groups = evidence_map_groups(summary)
    if not groups:
        return ""
    rendered_groups = []
    for group in groups:
        rows = []
        for row in group["rows"]:
            rows.append(
                "<tr>"
                f"<td>{esc(row['module'])}</td>"
                f"<td>{esc(row['evidence'])}</td>"
                f"<td>{esc(row['value'])}</td>"
                f'<td><span class="status-badge status-{esc(row["status_class"])}">{esc(row["status"])}</span></td>'
                "</tr>"
            )
        rendered_groups.append(
            '<div class="evidence-map-group">'
            f'<div class="evidence-map-head"><h3>{esc(group["title"])}</h3><p>{esc(group["subtitle"])}</p></div>'
            '<div class="evidence-table-wrap"><table class="evidence-table">'
            '<thead><tr><th>模块</th><th>证据</th><th>结果</th><th>状态</th></tr></thead>'
            f'<tbody>{"".join(rows)}</tbody></table></div>'
            '</div>'
        )
    return (
        '<section class="evidence-map" aria-label="证据地图">'
        '<div class="evidence-map-title"><span>Evidence Map</span><h2>证据地图</h2>'
        '<p>把总评、核心性能、网络与连通、限制建议放在同一张阅读路径里，方便判断报告结论是否有足够支撑。</p></div>'
        f'{"".join(rendered_groups)}</section>'
    )


def budget_summary(summary: dict[str, Any]) -> str:
    value = summary.get("budget_summary")
    return text(value, "") if value else ""


def collect_error_diagnostics(report: dict[str, Any]) -> list[dict[str, str]]:
    test_results = report.get("test_results") if isinstance(report.get("test_results"), dict) else {}
    rows = []
    for key, title in [
        ("cpu_result", "CPU"),
        ("memory_result", "内存"),
        ("disk_result", "磁盘"),
        ("network_result", "网络"),
    ]:
        result = test_results.get(key)
        if not isinstance(result, dict):
            continue
        metrics = result.get("metrics") if isinstance(result.get("metrics"), dict) else {}
        category = text(metrics.get("error_category"), "")
        stage = text(metrics.get("error_stage"), "")
        hint = text(metrics.get("error_hint"), "")
        message = text(result.get("error_message"), "") or text(metrics.get("network_error"), "")
        if category or stage or hint or message:
            rows.append({
                "module": title,
                "status": test_status_label(result.get("status")),
                "category": error_category_label(category),
                "stage": stage or "-",
                "message": message or "-",
                "hint": hint or "-",
            })
    return rows


def error_category_label(value: str) -> str:
    return {
        "missing_dependency": "依赖缺失",
        "invalid_config": "配置错误",
        "command_failed": "命令执行失败",
        "permission_denied": "权限不足",
        "resource_limited": "资源不足",
        "network_unavailable": "网络不可达",
        "parse_failed": "结果解析失败",
        "timeout": "执行超时",
        "runtime_error": "运行异常",
    }.get(text(value, ""), text(value, "-"))


def render_decision_panel(summary: dict[str, Any]) -> str:
    conclusion = summary.get("assessment_conclusion") if isinstance(summary.get("assessment_conclusion"), dict) else {}
    modules = summary.get("module_assessments") if isinstance(summary.get("module_assessments"), dict) else {}
    if not conclusion:
        return ""

    suitability = conclusion.get("suitability") if isinstance(conclusion.get("suitability"), list) else []
    recommendations = conclusion.get("recommendations") if isinstance(conclusion.get("recommendations"), list) else []
    limitations = conclusion.get("limitations") if isinstance(conclusion.get("limitations"), list) else []
    budget = budget_summary(summary)
    evidence_cards = []
    for item in top_evidence_items(conclusion):
        evidence_cards.append(
            f'<div class="evidence-card"><div class="evidence-head"><span>{esc(item["label"])}</span>'
            f'<span class="status-badge status-{esc(item["status_class"])}">{esc(item["status"])}</span></div>'
            f'<div class="evidence-value">{esc(item["value"])}</div></div>'
        )

    module_cards = []
    for item in module_health_items(modules):
        module_cards.append(
            f'<div class="module-health-card"><div class="module-health-title">{esc(item["title"])}</div>'
            f'<span class="status-badge status-{esc(item["status_class"])}">{esc(item["status"])}</span>'
            f'<div class="module-health-confidence">置信度 {esc(item["confidence"])}</div></div>'
        )

    return (
        '<section class="decision-panel" aria-label="结论优先摘要">'
        '<div class="decision-main">'
        '<div class="decision-block decision-primary">'
        '<div class="decision-label">适用判断</div>'
        f'<h2>{esc(conclusion.get("scenario"))}</h2>'
        f'<p>{esc(first_text(suitability, "已完成分项可作为局部参考。"))}</p>'
        '</div>'
        '<div class="decision-block">'
        '<div class="decision-label">优先建议</div>'
        f'<p>{esc(first_text(recommendations, "保留本次 JSON 报告，用同一档位横向比较。"))}</p>'
        '<div class="decision-label decision-gap">主要限制</div>'
        f'<p>{esc(first_text(limitations, "未发现明显限制。"))}</p>'
        '</div>'
        '</div>'
        f'<div class="budget-strip"><span>测评预算</span><strong>{esc(budget or "未提供预算说明")}</strong></div>'
        f'<div class="decision-evidence">{"".join(evidence_cards)}</div>'
        f'<div class="decision-modules">{"".join(module_cards)}</div>'
        '</section>'
    )


def result_section(
    section_id: str,
    title: str,
    subtitle: str,
    result: Optional[dict[str, Any]],
    metrics: list[dict[str, str]],
    extra_details: list[dict[str, str]],
    hint: str,
    assessment: Any = None,
) -> dict[str, Any]:
    assessment_tables = module_assessment_tables(assessment)
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
            "tables": assessment_tables,
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
    tables = assessment_tables
    if metric_rows:
        tables.append(table("原始指标", ["指标", "值"], metric_rows))
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


def safe_result_metrics(result: Optional[dict[str, Any]]) -> dict[str, Any]:
    if not isinstance(result, dict):
        return {}
    metrics = result.get("metrics")
    return metrics if isinstance(metrics, dict) else {}


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


def optional_section(section_id: str, title: str, subtitle: str, value: Any, hint: str, assessment: Any = None) -> dict[str, Any]:
    assessment_tables = module_assessment_tables(assessment)
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
            "tables": assessment_tables,
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
        "tables": assessment_tables + [table("检测结果", ["项目", "值"], simple_value_rows(value))],
        "hint": "",
    }


def availability_section(section_id: str, title: str, subtitle: str, value: Any, hint: str, name_key: str, region_label: str, assessment: Any = None) -> dict[str, Any]:
    if not isinstance(value, dict) or not value:
        return optional_section(section_id, title, subtitle, None, hint, assessment)

    items = [item for item in value.values() if isinstance(item, dict)]
    available_count = sum(1 for item in items if item.get("available"))
    status = "success" if available_count == len(items) else "warning" if available_count else "failed"
    rows = [
        [
            item.get(name_key) or item.get("platform") or item.get("service"),
            "可用" if item.get("available") else "不可用",
            item.get("region") or "-",
            item.get("message") or "-",
        ]
        for item in sorted(items, key=lambda row: text(row.get(name_key) or row.get("platform") or row.get("service")))
    ]
    return {
        "id": section_id,
        "title": title,
        "subtitle": subtitle,
        "status": status,
        "status_text": f"{available_count}/{len(items)} 可用",
        "summary": f"检测 {len(items)} 项，可用 {available_count} 项。",
        "metrics": [
            metric("检测项", len(items), "", "primary"),
            metric("可用", available_count, f"/ {len(items)}", "green" if available_count else "red"),
            metric("不可用", len(items) - available_count, "", "amber" if available_count else "red"),
        ],
        "details": [],
        "tables": module_assessment_tables(assessment) + [table("检测结果", ["项目", "状态", region_label, "说明"], rows)],
        "hint": "",
    }


def streaming_category(value: Any) -> str:
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


def streaming_unlock_type(value: Any) -> str:
    return {
        "full": "完整解锁",
        "partial": "部分解锁",
        "limited": "受限",
        "blocked": "不可用",
        "login_required": "需要登录",
        "available": "可访问",
    }.get(text(value, ""), text(value))


def streaming_section(value: Any, assessment: Any = None) -> dict[str, Any]:
    hint = "本次未启用流媒体检测。使用 --streaming、--full 或 bootstrap 的 standard/full 档位启用。"
    if not isinstance(value, dict) or not value:
        return optional_section("streaming", "流媒体解锁", "Netflix、Disney+、YouTube 等平台的区域访问能力。", None, hint, assessment)

    items = [item for item in value.values() if isinstance(item, dict)]
    available_count = sum(1 for item in items if item.get("available"))
    status = "success" if available_count == len(items) else "warning" if available_count else "failed"
    rows = [
        [
            streaming_category(item.get("category")),
            item.get("platform") or "-",
            "可用" if item.get("available") else "不可用",
            item.get("region") or "-",
            streaming_unlock_type(item.get("unlock_type")),
            item.get("protocol") or "-",
            item.get("message") or "-",
        ]
        for item in sorted(items, key=lambda row: (streaming_category(row.get("category")), text(row.get("platform"))))
    ]
    return {
        "id": "streaming",
        "title": "流媒体解锁",
        "subtitle": "Netflix、Disney+、YouTube 等平台的区域访问能力。",
        "status": status,
        "status_text": f"{available_count}/{len(items)} 可用",
        "summary": f"检测 {len(items)} 个平台，可用 {available_count} 个。",
        "metrics": [
            metric("平台数", len(items), "", "primary"),
            metric("可用", available_count, f"/ {len(items)}", "green" if available_count else "red"),
            metric("不可用", len(items) - available_count, "", "amber" if available_count else "red"),
        ],
        "details": [],
        "tables": module_assessment_tables(assessment) + [table("平台结果", ["分组", "平台", "状态", "区域", "解锁类型", "协议", "说明"], rows)],
        "hint": "",
    }


def ai_category(value: Any) -> str:
    return {
        "chatbot": "对话",
        "assistant": "助手",
        "search": "搜索",
        "coding": "编程",
    }.get(text(value, ""), text(value))


def ai_access_type(value: Any) -> str:
    return {
        "full": "可访问",
        "login_required": "需要登录",
        "verification_required": "需验证",
        "rate_limited": "限流",
        "restricted": "受限",
        "available": "可用",
    }.get(text(value, ""), text(value))


def ai_section(value: Any, assessment: Any = None) -> dict[str, Any]:
    hint = "本次未启用 AI 服务检测。使用 --ai-services、--full 或 bootstrap 的 standard/full 档位启用。"
    if not isinstance(value, dict) or not value:
        return optional_section("ai", "AI 服务检测", "OpenAI、Gemini 等 AI 服务的可访问性。", None, hint, assessment)

    items = [item for item in value.values() if isinstance(item, dict)]
    available_count = sum(1 for item in items if item.get("available"))
    status = "success" if available_count == len(items) else "warning" if available_count else "failed"
    rows = [
        [
            ai_category(item.get("category")),
            item.get("service") or "-",
            "可用" if item.get("available") else "不可用",
            ai_access_type(item.get("access_type")),
            item.get("region_hint") or "-",
            item.get("message") or "-",
        ]
        for item in sorted(items, key=lambda row: (ai_category(row.get("category")), text(row.get("service"))))
    ]
    return {
        "id": "ai",
        "title": "AI 服务检测",
        "subtitle": "OpenAI、Gemini 等 AI 服务的可访问性。",
        "status": status,
        "status_text": f"{available_count}/{len(items)} 可用",
        "summary": f"检测 {len(items)} 个 AI 服务，可用 {available_count} 个。",
        "metrics": [
            metric("服务数", len(items), "", "primary"),
            metric("可用", available_count, f"/ {len(items)}", "green" if available_count else "red"),
            metric("不可用", len(items) - available_count, "", "amber" if available_count else "red"),
        ],
        "details": [],
        "tables": module_assessment_tables(assessment) + [table("服务结果", ["分组", "服务", "状态", "访问类型", "区域提示", "说明"], rows)],
        "hint": "",
    }


def stress_section(value: Any) -> dict[str, Any]:
    hint = "本次未启用压力测试。使用 --stress、--full 或 bootstrap 的 full 档位启用。"
    if not isinstance(value, dict):
        return optional_section("stress", "压力测试", "长时间 CPU、内存、磁盘压力下的稳定性。", None, hint)

    components = value.get("components") if isinstance(value.get("components"), list) else []
    failed_count = sum(1 for item in components if isinstance(item, dict) and item.get("status") == "failed")
    degraded_count = sum(1 for item in components if isinstance(item, dict) and item.get("status") == "degraded")
    status = "failed" if failed_count else "warning" if degraded_count else "success"
    rows = [
        [
            item.get("name"),
            status_text(text(item.get("status"))),
            f"{fmt_number(item.get('duration_seconds'), 0)} 秒",
            item.get("notes") or "-",
        ]
        for item in components if isinstance(item, dict)
    ]
    return {
        "id": "stress",
        "title": "压力测试",
        "subtitle": "长时间 CPU、内存、磁盘压力下的稳定性。",
        "status": status,
        "status_text": "稳定" if status == "success" else "注意",
        "summary": f"压力测试持续 {fmt_number(value.get('total_duration_seconds'), 0)} 秒，组件 {len(components)} 个。",
        "metrics": [
            metric("总耗时", fmt_number(value.get("total_duration_seconds"), 0), "秒", "primary"),
            metric("组件数", len(components), "", "primary"),
            metric("失败", failed_count, "", "red" if failed_count else "green"),
            metric("温度数据", "有" if value.get("temperature_available") else "无", "", "cyan"),
        ],
        "details": [],
        "tables": [table("压力组件", ["组件", "状态", "耗时", "备注"], rows)] if rows else [],
        "hint": "",
    }


def security_section(value: Any) -> dict[str, Any]:
    hint = "本次未启用安全体检。使用 --security、--full 或 bootstrap 的 standard/full 档位启用。"
    if not isinstance(value, dict):
        return optional_section("security", "安全体检", "端口、SSH 配置和基础安全风险检查。", None, hint)

    findings = value.get("findings") if isinstance(value.get("findings"), list) else []
    severity_counts = {"high": 0, "medium": 0, "low": 0, "info": 0}
    for item in findings:
        if not isinstance(item, dict):
            continue
        severity = text(item.get("severity"), "info").lower()
        if severity not in severity_counts:
            severity = "info"
        severity_counts[severity] += 1
    status = "failed" if severity_counts["high"] else "warning" if severity_counts["medium"] else "success"
    rows = [
        [
            item.get("category"),
            severity_label(item.get("severity")),
            item.get("title"),
            item.get("detail") or "-",
            item.get("advice") or "-",
        ]
        for item in findings if isinstance(item, dict)
    ]
    return {
        "id": "security",
        "title": "安全体检",
        "subtitle": "端口、SSH 配置和基础安全风险检查。",
        "status": status,
        "status_text": "无明显风险" if status == "success" else "注意",
        "summary": f"发现 {len(findings)} 条安全提示：高危 {severity_counts['high']}，中危 {severity_counts['medium']}，低危 {severity_counts['low']}，信息 {severity_counts['info']}。",
        "metrics": [
            metric("提示数", len(findings), "", "primary"),
            metric("高危", severity_counts["high"], "", "red" if severity_counts["high"] else "green"),
            metric("中危", severity_counts["medium"], "", "amber" if severity_counts["medium"] else "green"),
            metric("信息", severity_counts["info"], "", "cyan"),
        ],
        "details": [],
        "tables": [table("安全发现", ["类别", "级别", "标题", "详情", "建议"], rows)] if rows else [],
        "hint": "",
    }


def severity_label(value: Any) -> str:
    return {
        "high": "高危",
        "medium": "中危",
        "low": "低危",
        "info": "信息",
    }.get(text(value, "info").lower(), text(value, "信息"))


def route_section(value: Any, assessment: Any = None) -> dict[str, Any]:
    hint = "本次未启用路由追踪。使用 --route-trace、--full 或 bootstrap 的 standard/full 档位启用。"
    if not isinstance(value, list) or not value:
        return optional_section("route", "路由追踪", "到主要地区和节点的网络路径质量。", None, hint, assessment)

    success_count = sum(1 for item in value if isinstance(item, dict) and item.get("success"))
    total_hops = sum(int(item.get("total_hops") or len(item.get("hops") or [])) for item in value if isinstance(item, dict) and item.get("success"))
    avg_hops = total_hops / success_count if success_count else 0
    status = "success" if success_count == len(value) else "warning" if success_count else "failed"
    timeout_hops = sum(int(item.get("timeout_hops") or 0) for item in value if isinstance(item, dict))
    avg_latency_values = [
        float(item.get("average_latency_ms"))
        for item in value
        if isinstance(item, dict) and isinstance(item.get("average_latency_ms"), (int, float)) and item.get("average_latency_ms") > 0
    ]
    avg_latency = sum(avg_latency_values) / len(avg_latency_values) if avg_latency_values else 0
    tables = module_assessment_tables(assessment) + [
        table("追踪目标", ["方向", "目标", "评级", "状态", "跳数", "超时", "平均延迟", "最后一跳/错误"], [
            [
                route_direction_label(item),
                item.get("target"),
                route_quality_label(item),
                "成功" if item.get("success") else "失败",
                item.get("total_hops") or len(item.get("hops") or []),
                item.get("timeout_hops", 0),
                format_route_average_latency(item),
                last_route_hop(item) if item.get("success") else item.get("error_message") or "-",
            ]
            for item in value if isinstance(item, dict)
        ])
    ]
    for item in value:
        if not isinstance(item, dict):
            continue
        hops = item.get("hops") if isinstance(item.get("hops"), list) else []
        if not hops:
            continue
        rows = [
            [
                hop.get("number"),
                hop.get("ip"),
                hop.get("hostname"),
                format_route_latency(hop.get("latency")),
            ]
            for hop in hops if isinstance(hop, dict)
        ]
        if rows:
            tables.append(table(f"{text(item.get('target'))} 路由跳点", ["跳数", "IP", "主机名", "延迟"], rows))
    return {
        "id": "route",
        "title": "路由追踪",
        "subtitle": "到主要地区和节点的网络路径质量。",
        "status": status,
        "status_text": f"{success_count}/{len(value)} 成功",
        "summary": route_summary(value, success_count, avg_hops, avg_latency, timeout_hops),
        "metrics": [
            metric("目标数", len(value), "", "primary"),
            metric("成功", success_count, f"/ {len(value)}", "green" if success_count == len(value) else "amber"),
            metric("平均跳数", f"{avg_hops:.1f}", "hops", "cyan"),
            metric("超时跳", timeout_hops, "hops", "amber" if timeout_hops else "green"),
            metric("平均延迟", f"{avg_latency:.2f}" if avg_latency else "-", "ms", "cyan"),
        ],
        "details": [],
        "tables": tables,
        "hint": "内置 traceroute 展示本机出站路径；国内方向参考不等同于真实回程。",
    }


def route_summary(items: list[Any], success_count: int, avg_hops: float, avg_latency: float, timeout_hops: int) -> str:
    summaries = [
        item.get("quality", {}).get("summary")
        for item in items
        if isinstance(item, dict) and isinstance(item.get("quality"), dict) and item.get("quality", {}).get("summary")
    ]
    if summaries:
        return f"完成 {len(items)} 个目标追踪，成功 {success_count} 个；代表结论：{text(summaries[0])}"
    latency_part = f"，平均延迟 {avg_latency:.2f} ms" if avg_latency else ""
    timeout_part = f"，超时跳 {timeout_hops}" if timeout_hops else ""
    return f"完成 {len(items)} 个目标追踪，成功 {success_count} 个，平均 {avg_hops:.1f} 跳{latency_part}{timeout_part}。"


def route_direction_label(item: dict[str, Any]) -> str:
    group = text(item.get("direction_group"), "")
    if group == "china_reference":
        return "国内方向参考"
    target = text(item.get("target"), "").lower()
    if any(marker in target for marker in ("189.cn", "10086.cn", "chinaunicom", "ctyun", "qq.com")):
        return "国内方向参考"
    return "公共方向"


def route_quality_label(item: dict[str, Any]) -> str:
    quality = item.get("quality") if isinstance(item.get("quality"), dict) else {}
    grade = text(quality.get("grade"), "")
    status = route_status_label(quality.get("status"))
    if grade and status:
        return f"{grade}/{status}"
    return grade or ("完成" if item.get("success") else "失败")


def route_status_label(value: Any) -> str:
    lowered = text(value, "").lower()
    if lowered == "success":
        return "良好"
    if lowered == "warning":
        return "需关注"
    if lowered == "failed":
        return "异常"
    return text(value, "")


def format_route_average_latency(item: dict[str, Any]) -> str:
    value = item.get("average_latency_ms")
    if isinstance(value, bool) or value is None:
        return "-"
    try:
        numeric = float(value)
    except (TypeError, ValueError):
        return "-"
    return f"{numeric:.2f} ms" if numeric > 0 else "-"


def last_route_hop(item: dict[str, Any]) -> str:
    if text(item.get("last_visible_hop"), ""):
        return text(item.get("last_visible_hop"))
    hops = item.get("hops") if isinstance(item.get("hops"), list) else []
    for hop in reversed(hops):
        if not isinstance(hop, dict):
            continue
        ip = text(hop.get("ip"), "")
        if not ip or ip == "*":
            continue
        hostname = text(hop.get("hostname"), "")
        return f"{ip} ({hostname})" if hostname else ip
    return "-"


def format_route_latency(value: Any) -> str:
    if isinstance(value, bool) or value is None:
        return "-"
    if isinstance(value, (int, float)):
        if value <= 0:
            return "-"
        return f"{float(value) / 1_000_000:.2f} ms"
    return text(value)


def ip_quality_section(summary: dict[str, Any], assessment: Any = None) -> dict[str, Any]:
    report = summary.get("ip_quality_report")
    hint = "本次未启用 IP 质量检测。使用 --ip-quality 或 --full 启用。"
    if not isinstance(report, dict):
        return optional_section("ip-quality", "IP 节点分析报告", "公网 IP、归属地、机房类型、欺诈风险、DNSBL 和邮件连通性。", None, hint, assessment)

    tables = module_assessment_tables(assessment)
    blacklist = report.get("blacklist_summary") if isinstance(report.get("blacklist_summary"), dict) else {}
    verdict = report.get("verdict") if isinstance(report.get("verdict"), dict) else {}
    evidence = report.get("evidence") if isinstance(report.get("evidence"), list) else []
    recommendations = report.get("recommendations") if isinstance(report.get("recommendations"), list) else []
    risk_factors = report.get("risk_factors") if isinstance(report.get("risk_factors"), list) else []
    risk_sources = report.get("risk_sources") if isinstance(report.get("risk_sources"), list) else []
    blacklists = report.get("blacklist_checks") if isinstance(report.get("blacklist_checks"), list) else []
    mail_checks = report.get("mail_checks") if isinstance(report.get("mail_checks"), list) else []
    mail_summary = report.get("mail_summary") if isinstance(report.get("mail_summary"), dict) else {}
    network_stack = report.get("network_stack") if isinstance(report.get("network_stack"), dict) else {}
    level = text(report.get("risk_level"), "unknown")
    status = "failed" if level == "high" else "warning" if level == "medium" else "success"
    if risk_sources:
        tables.append(table("风险来源", ["来源", "类型", "状态", "信号", "说明"], [
            [item.get("name"), item.get("type"), risk_source_status_label(item.get("status")), item.get("signal"), item.get("detail")]
            for item in risk_sources if isinstance(item, dict)
        ]))
    if evidence:
        tables.append(table("结论证据", ["证据", "结果", "状态", "说明"], [
            [item.get("name"), item.get("value"), item.get("status"), item.get("detail")]
            for item in evidence if isinstance(item, dict)
        ]))
    if recommendations:
        tables.append(table("建议", ["序号", "建议"], [
            [index + 1, item]
            for index, item in enumerate(recommendations)
        ]))
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
        tables.append(table("邮件端口连通性", ["服务商", "目标", "端口", "状态", "可达", "说明"], [
            [item.get("provider"), item.get("target"), item.get("port"), item.get("status"), "是" if item.get("reachable") else "否", item.get("detail")]
            for item in mail_checks if isinstance(item, dict)
        ]))
    return {
        "id": "ip-quality",
        "title": "IP 节点分析报告",
        "subtitle": "公网 IP、归属地、机房类型、欺诈风险、DNSBL 和邮件连通性。",
        "status": status,
        "status_text": {"low": "低风险", "medium": "中风险", "high": "高风险"}.get(level, "未知"),
        "summary": text(verdict.get("summary"), f"{ip_node_type_summary(report)}，欺诈风险分 {text(report.get('risk_score'))}/100，综合评级 {ip_quality_grade(report)}。"),
        "metrics": [
            metric("欺诈风险分", report.get("risk_score"), "/ 100", "red" if status == "failed" else "amber" if status == "warning" else "green"),
            metric("综合评级", verdict.get("grade") or ip_quality_grade(report), verdict.get("risk_label") or {"low": "低风险", "medium": "中风险", "high": "高风险"}.get(level, "未知"), "red" if status == "failed" else "amber" if status == "warning" else "green"),
            metric("DNSBL 命中", blacklist.get("listed", 0), f"/ {blacklist.get('total', len(blacklists))}", "amber"),
            metric("邮件可连", count_reachable_mail(mail_checks), f"/ {len(mail_checks)}", "cyan"),
            metric("可连服务商", mail_summary.get("provider_open", 0), f"/ {mail_summary.get('providers', 0)}", "cyan"),
        ],
        "details": [
            detail("IP 地址", report.get("public_ip")),
            detail("国家/地区", ", ".join(part for part in [text(report.get("country"), ""), text(report.get("city"), "")] if part)),
            detail("运营商/ASN", ip_asn_label(report)),
            detail("IP 类型", verdict.get("ip_type_label") or ip_node_type_summary(report)),
            detail("代理/VPN 标记", "是" if verdict.get("proxy_hint") or risk_factor_detected(report, "proxy") or risk_factor_detected(report, "vpn") else "否"),
            detail("机房/托管标记", "是" if verdict.get("hosting_hint") or risk_factor_detected(report, "datacenter") else "否"),
            detail("网络栈", network_stack_label(network_stack)),
            detail("邮件可用", "是" if verdict.get("mail_usable") else "否"),
            detail("邮件汇总", f"服务商 {text(mail_summary.get('providers'), '0')} 个，可连服务商 {text(mail_summary.get('provider_open'), '0')} 个；可连端口 {text(mail_summary.get('reachable'), '0')}/{text(mail_summary.get('total'), '0')}"),
            detail("评级依据", ip_quality_basis(report, blacklist, mail_checks)),
        ],
        "tables": tables,
        "hint": " ".join(text(note, "") for note in report.get("notes", []) if note),
    }


def risk_factor_detected(report: dict[str, Any], name: str) -> bool:
    factors = report.get("risk_factors") if isinstance(report.get("risk_factors"), list) else []
    return any(isinstance(item, dict) and item.get("name") == name and item.get("detected") for item in factors)


def risk_source_status_label(value: Any) -> str:
    return {
        "available": "可用",
        "missing": "缺失",
        "clean": "正常",
        "listed": "命中",
        "partial": "部分",
        "disabled": "未启用",
    }.get(text(value, ""), text(value))


def network_stack_label(value: dict[str, Any]) -> str:
    if not isinstance(value, dict) or not value:
        return "-"
    if value.get("dual_stack"):
        return "IPv4/IPv6 双栈"
    return text(value.get("detected_version"))


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


def bottleneck_label(value: str) -> str:
    return {
        "good": "表现正常",
        "watch": "需要关注",
        "weak": "明显短板",
    }.get(value, text(value))


def evidence_label(value: str) -> str:
    return {
        "success": "通过",
        "partial": "部分",
        "warning": "注意",
        "failed": "风险",
    }.get(value, text(value))


def module_status_label(value: Any) -> str:
    return {
        "success": "通过",
        "warning": "注意",
        "failed": "风险",
        "skipped": "未执行",
    }.get(text(value, ""), text(value))


def test_status_label(value: Any) -> str:
    return {
        "success": "成功",
        "warning": "注意",
        "failed": "失败",
        "skipped": "跳过",
        "partial": "部分完成",
    }.get(text(value, ""), text(value, "-"))


REPORT_GROUPS = [
    ("overview", "概览", "评分、系统和报告结论。", {"overview", "conclusion", "system"}),
    ("core", "核心性能", "CPU、内存、磁盘和网络基础测评。", {"cpu", "memory", "disk", "network"}),
    ("network", "网络扩展", "路由、流媒体、AI 服务、IP 质量和安全体检。", {"route", "streaming", "ai", "ip-quality", "security"}),
    ("stability", "稳定性", "压力测试和长时间负载表现。", {"stress"}),
    ("delivery", "交付", "报告产物、质量提示和分享模板。", {"artifacts", "diagnostics", "quality", "share"}),
]


def group_sections(sections: list[dict[str, Any]], extra_ids: Optional[set[str]] = None) -> list[dict[str, Any]]:
    remaining = list(sections)
    groups = []
    extra_ids = extra_ids or set()
    for group_id, title, subtitle, ids in REPORT_GROUPS:
        selected = [section for section in remaining if section.get("id") in ids]
        if group_id == "delivery" and extra_ids:
            selected.extend({"id": item_id, "title": "质量提示" if item_id == "quality" else "分享模板", "status": "success", "status_text": "可用"} for item_id in sorted(extra_ids))
        if selected:
            groups.append({"id": group_id, "title": title, "subtitle": subtitle, "sections": selected})
            remaining = [section for section in remaining if section.get("id") not in ids]
    if remaining:
        groups.append({"id": "other", "title": "其他", "subtitle": "其他报告模块。", "sections": remaining})
    return groups


def report_status_counts(sections: list[dict[str, Any]]) -> dict[str, int]:
    counts = {"success": 0, "warning": 0, "failed": 0, "skipped": 0, "total": len(sections)}
    for section in sections:
        status = status_class(text(section.get("status"), "unknown"))
        if status in {"success", "running"}:
            counts["success"] += 1
        elif status in {"warning", "degraded"}:
            counts["warning"] += 1
        elif status == "failed":
            counts["failed"] += 1
        else:
            counts["skipped"] += 1
    return counts


def artifact_section(output_dir: Optional[Path]) -> dict[str, Any]:
    artifact_defs = [
        ("终端彩色报告", "console.ansi"),
        ("终端纯文本报告", "console.txt"),
        ("Markdown 摘要", "summary.md"),
        ("报告压缩包", "perfassess-report.zip"),
        ("报告产物清单", "artifact_manifest.json"),
        ("完整 JSON", "default.json"),
        ("完整文本报告", "default.txt"),
        ("硬件质量模块", "hardware_quality.json"),
        ("网络质量模块", "net_quality.json"),
        ("脱敏校准样本", "calibration_sample.json"),
        ("路由追踪模块", "route_trace.json"),
        ("国内方向参考", "backroute_trace.json"),
        ("IP 质量模块", "ip_quality.json"),
        ("流媒体模块", "streaming_unlock.json"),
        ("AI 服务模块", "ai_services.json"),
        ("安全体检模块", "security_scan.json"),
        ("压力测试模块", "stress_test.json"),
        ("快速测评 JSON", "quick.json"),
        ("验收摘要", "acceptance/summary.md"),
    ]
    artifacts = []
    for label, path in artifact_defs:
        available = bool(output_dir and (output_dir / path).is_file())
        artifacts.append({"label": label, "path": path, "available": "true" if available else "false"})
    available_count = sum(1 for item in artifacts if item["available"] == "true")
    return {
        "id": "artifacts",
        "title": "报告产物",
        "subtitle": "控制台、Markdown、JSON、模块化报告和压缩包。",
        "status": "success" if available_count else "skipped",
        "status_text": f"{available_count}/{len(artifacts)} 可下载" if available_count else "未生成",
        "summary": "测评完成后可直接下载结构化报告、终端报告和模块化产物。",
        "metrics": [
            metric("可下载", available_count, "files", "primary"),
            metric("总产物", len(artifacts), "files", "primary"),
            metric("模块 JSON", sum(1 for item in artifacts if item["path"].endswith(".json") and item["available"] == "true"), "files", "green"),
            metric("压缩包", "有" if any(item["path"] == "perfassess-report.zip" and item["available"] == "true" for item in artifacts) else "无", "", "amber"),
        ],
        "details": [],
        "tables": [],
        "artifacts": artifacts,
        "hint": "",
    }


def build_report_sections(report: dict[str, Any], output_dir: Optional[Path] = None) -> list[dict[str, Any]]:
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
                detail("测评预算", budget_summary(summary) or "-"),
                detail("成功/失败/跳过", f"{text(summary.get('tests_success'), '0')} / {text(summary.get('tests_failed'), '0')} / {text(summary.get('tests_skipped'), '0')}"),
            ],
            "tables": [],
            "hint": text(summary.get("performance_note"), ""),
        }
    ]

    conclusion = summary.get("assessment_conclusion") if isinstance(summary.get("assessment_conclusion"), dict) else {}
    if conclusion:
        bottlenecks = conclusion.get("bottlenecks") if isinstance(conclusion.get("bottlenecks"), list) else []
        evidence = conclusion.get("evidence") if isinstance(conclusion.get("evidence"), list) else []
        limitations = conclusion.get("limitations") if isinstance(conclusion.get("limitations"), list) else []
        recommendations = conclusion.get("recommendations") if isinstance(conclusion.get("recommendations"), list) else []
        suitability = conclusion.get("suitability") if isinstance(conclusion.get("suitability"), list) else []
        sections.append({
            "id": "conclusion",
            "title": "测评结论",
            "subtitle": "适用场景、主要短板、置信度限制和下一步建议。",
            "status": "success" if text(conclusion.get("grade")) != "未完成" else "warning",
            "status_text": text(conclusion.get("grade"), "已生成"),
            "summary": text(conclusion.get("headline")),
            "metrics": [
                metric("总分", fmt_number(conclusion.get("total_score")), "/ 100", "primary"),
                metric("等级", conclusion.get("grade"), "", "primary"),
                metric("置信度", conclusion.get("confidence"), "", "cyan"),
                metric("短板", text(data_get(bottlenecks[0], "label") if bottlenecks else "-"), "", "amber"),
            ],
            "details": [
                detail("适用判断", conclusion.get("scenario")),
                detail("适合场景", "、".join(text(item) for item in suitability) if suitability else "-"),
                detail("主要限制", "；".join(text(item) for item in limitations[:3]) if limitations else "-"),
                detail("优先建议", text(recommendations[0]) if recommendations else "-"),
            ],
            "tables": [
                {
                    "title": "短板排序",
                    "headers": ["模块", "评分", "状态"],
                    "rows": [
                        [text(item.get("label")), fmt_number(item.get("score")), bottleneck_label(text(item.get("severity")))]
                        for item in bottlenecks
                        if isinstance(item, dict)
                    ],
                },
                {
                    "title": "关键证据",
                    "headers": ["证据", "结果", "状态", "说明"],
                    "rows": [
                        [text(item.get("label")), text(item.get("value")), evidence_label(text(item.get("status"))), text(item.get("detail"))]
                        for item in evidence[:6]
                        if isinstance(item, dict)
                    ],
                },
                {
                    "title": "建议",
                    "headers": ["序号", "内容"],
                    "rows": [[str(idx + 1), text(item)] for idx, item in enumerate(recommendations[:5])],
                },
            ],
            "hint": "",
        })

    module_assessments = summary.get("module_assessments") if isinstance(summary.get("module_assessments"), dict) else {}
    module_rows = []
    module_evidence_rows = []
    for key in ["cpu", "memory", "disk", "network", "route", "ip_quality", "streaming", "ai_services"]:
        item = module_assessments.get(key)
        if not isinstance(item, dict) or item.get("status") == "skipped":
            continue
        module_rows.append([
            text(item.get("title")),
            module_status_label(item.get("status")),
            text(item.get("confidence")),
            text(item.get("summary")),
        ])
        evidence = item.get("evidence") if isinstance(item.get("evidence"), list) else []
        for evidence_item in evidence[:3]:
            if isinstance(evidence_item, dict):
                module_evidence_rows.append([
                    text(item.get("title")),
                    text(evidence_item.get("label")),
                    text(evidence_item.get("value")),
                    evidence_label(text(evidence_item.get("status"))),
                    text(evidence_item.get("detail")),
                ])
    if module_rows:
        sections.append({
            "id": "module-assessments",
            "title": "模块可信度",
            "subtitle": "网络、IP、流媒体和 AI 服务的模块级结论、证据和限制。",
            "status": "success" if all(row[1] == "通过" for row in module_rows) else "warning",
            "status_text": f"{len(module_rows)} 个模块",
            "summary": "模块级评估用于解释扩展检测结果是否可直接作为结论。",
            "metrics": [
                metric("模块数", len(module_rows), "", "primary"),
                metric("高置信", sum(1 for row in module_rows if row[2] == "high"), "", "green"),
                metric("注意", sum(1 for row in module_rows if row[1] != "通过"), "", "amber"),
            ],
            "details": [],
            "tables": [
                table("模块判断", ["模块", "状态", "置信度", "结论"], module_rows),
                table("模块证据", ["模块", "证据", "结果", "状态", "说明"], module_evidence_rows),
            ],
            "hint": "",
        })

    diagnostics = collect_error_diagnostics(report)
    if diagnostics:
        sections.append({
            "id": "diagnostics",
            "title": "诊断提示",
            "subtitle": "外部后端、配置、命令执行和解析失败的结构化原因。",
            "status": "warning",
            "status_text": f"{len(diagnostics)} 条提示",
            "summary": "这些提示用于判断本次报告的失败或降级原因，以及下一步如何处理。",
            "metrics": [
                metric("提示数", len(diagnostics), "", "amber"),
                metric("依赖缺失", sum(1 for row in diagnostics if row["category"] == "依赖缺失"), "", "red"),
                metric("配置错误", sum(1 for row in diagnostics if row["category"] == "配置错误"), "", "amber"),
                metric("解析失败", sum(1 for row in diagnostics if row["category"] == "结果解析失败"), "", "amber"),
            ],
            "details": [],
            "tables": [
                table("错误诊断", ["模块", "状态", "分类", "阶段", "错误", "建议"], [
                    [row["module"], row["status"], row["category"], row["stage"], row["message"], row["hint"]]
                    for row in diagnostics
                ]),
            ],
            "hint": "缺失可选依赖不会阻止 builtin 路径生成报告，但会降低主流后端可比性。",
        })

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
    cpu_metrics = safe_result_metrics(cpu_result)
    sections.append(result_section("cpu", "CPU 测试", "单核、多核性能和采样稳定性。", cpu_result, [
        metric("CPU 评分", fmt_number(summary.get("cpu_score")), "/ 100", "primary"),
        metric("单核", fmt_number(cpu_metrics.get("single_core_score")), "", "primary"),
        metric("多核", fmt_number(cpu_metrics.get("multi_core_score")), "", "primary"),
        metric("后端", cpu_metrics.get("backend"), "", "primary"),
    ], [], "本次未执行 CPU 测试。", module_assessments.get("cpu")))

    memory_result = test_results.get("memory_result") if isinstance(test_results.get("memory_result"), dict) else None
    memory_metrics = safe_result_metrics(memory_result)
    sections.append(result_section("memory", "内存测试", "内存读写吞吐和采样稳定性。", memory_result, [
        metric("内存评分", fmt_number(summary.get("memory_score")), "/ 100", "green"),
        metric("读取速度", fmt_number(memory_metrics.get("read_speed_mbps")), "MB/s", "green"),
        metric("写入速度", fmt_number(memory_metrics.get("write_speed_mbps")), "MB/s", "green"),
        metric("后端", memory_metrics.get("backend"), "", "green"),
    ], [], "本次未执行内存测试。", module_assessments.get("memory")))

    disk_result = test_results.get("disk_result") if isinstance(test_results.get("disk_result"), dict) else None
    disk_metrics = safe_result_metrics(disk_result)
    sections.append(result_section("disk", "磁盘测试", "顺序读写、随机 IOPS 和磁盘评分。", disk_result, [
        metric("磁盘评分", fmt_number(summary.get("disk_score")), "/ 100", "amber"),
        metric("顺序读取", fmt_number(disk_metrics.get("sequential_read_mbps") or disk_metrics.get("read_speed_mbps")), "MB/s", "amber"),
        metric("顺序写入", fmt_number(disk_metrics.get("sequential_write_mbps") or disk_metrics.get("write_speed_mbps")), "MB/s", "amber"),
        metric("随机 IOPS", fmt_int(disk_metrics.get("random_iops")), "IOPS", "amber"),
    ], [], "本次未执行磁盘测试。", module_assessments.get("disk")))

    network_result = test_results.get("network_result") if isinstance(test_results.get("network_result"), dict) else None
    network_metrics = safe_result_metrics(network_result)
    sections.append(result_section("network", "网络测试", "延迟、下载、上传、IPv4/IPv6 和多节点质量。", network_result, [
        metric("网络评分", fmt_number(summary.get("network_score")), "/ 100", "cyan"),
        metric("平均延迟", fmt_number(network_metrics.get("latency_ms")), "ms", "cyan"),
        metric("下载速度", fmt_number(network_metrics.get("download_speed_mbps")), "Mbps", "cyan"),
        metric("上传速度", fmt_number(network_metrics.get("upload_speed_mbps")), "Mbps", "cyan"),
    ], [], "本次未执行网络测试。", module_assessments.get("network")))

    sections.extend([
        route_section(summary.get("route_trace_results"), module_assessments.get("route")),
        streaming_section(summary.get("streaming_results"), module_assessments.get("streaming")),
        ai_section(summary.get("ai_results"), module_assessments.get("ai_services")),
        ip_quality_section(summary, module_assessments.get("ip_quality")),
        stress_section(summary.get("stress_report")),
        security_section(summary.get("security_report")),
        artifact_section(output_dir),
    ])
    return sections


def render_report(report: dict[str, Any], output_dir: Optional[Path] = None) -> str:
    summary = report.get("summary") if isinstance(report.get("summary"), dict) else {}
    overall = summary.get("overall_score") if isinstance(summary.get("overall_score"), dict) else {}
    session_id = text(report.get("session_id"))
    timestamp = text(report.get("timestamp"))
    total_score = fmt_number(summary.get("total_score") or overall.get("total_score"), 0)
    grade = text(summary.get("grade") or overall.get("grade"))
    sections = build_report_sections(report, output_dir)
    quality_notes = summary.get("quality_notes") if isinstance(summary.get("quality_notes"), list) else []
    share = data_get(summary, "share_templates", "plain_text", default="")
    decision_panel = render_decision_panel(summary)
    evidence_map = render_evidence_map(summary)
    extra_ids = set()
    if quality_notes:
        extra_ids.add("quality")
    if share:
        extra_ids.add("share")
    groups = group_sections(sections, extra_ids)
    status_counts = report_status_counts(sections)
    nav_groups = []
    for group in groups:
        links = []
        for section in group["sections"]:
            links.append(
                f'<a class="nav-link" href="#{esc(section["id"])}" data-section="{esc(section["id"])}"><span>{esc(section["title"])}</span>'
                f'<span class="status-badge status-{esc(status_class(text(section.get("status"))))}">{esc(section.get("status_text"))}</span></a>'
            )
        nav_groups.append(
            f'<div class="nav-group"><div class="nav-group-title">{esc(group["title"])}</div>{"".join(links)}</div>'
        )
    nav = "".join(nav_groups)
    cards = []
    for section in sections:
        hint = f'<div class="hint-box">{esc(section.get("hint"))}</div>' if section.get("hint") else ""
        cards.append(
            f'<section id="{esc(section["id"])}" class="module-card module-panel module-{esc(status_class(text(section.get("status"))))}" data-panel="{esc(section["id"])}">'
            '<div class="module-head"><div>'
            f'<h2 class="module-title">{esc(section["title"])}</h2>'
            f'<p class="module-subtitle">{esc(section["subtitle"])}</p>'
            f'</div><span class="status-badge status-{esc(status_class(text(section.get("status"))))}">{esc(section["status_text"])}</span></div>'
            f'<p class="module-summary">{esc(section["summary"])}</p>'
            f'{render_metrics(section.get("metrics", []))}'
            f'{render_details(section.get("details", []))}'
            f'{render_tables(section.get("tables", []))}'
            f'{render_artifact_links(section.get("artifacts", []))}'
            f'{hint}</section>'
        )

    if quality_notes:
        items = "".join(f"<li>{esc(note)}</li>" for note in quality_notes)
        cards.append(
            '<section id="quality" class="module-card module-panel quality-notes" data-panel="quality"><div class="module-head"><div>'
            '<h2 class="module-title">质量提示</h2><p class="module-subtitle">影响报告置信度和可比性的说明。</p>'
            '</div><span class="status-badge status-skipped">提示</span></div>'
            f"<ul>{items}</ul></section>"
        )

    if share:
        cards.append(
            '<section id="share" class="module-card module-panel" data-panel="share"><div class="module-head"><div>'
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
    .hero {{ overflow: visible; border-radius: 8px; padding: 28px 32px; color: #fff; background: linear-gradient(135deg, rgba(11, 87, 208, 0.96), rgba(0, 99, 155, 0.92)); box-shadow: 0 2px 6px rgba(60, 64, 67, 0.16), 0 8px 24px rgba(60, 64, 67, 0.10); }}
    .hero-content {{ display: grid; grid-template-columns: minmax(0, 1fr) 170px; gap: 24px; align-items: end; }}
    .eyebrow {{ display: inline-flex; border-radius: 999px; padding: 7px 12px; background: rgba(255, 255, 255, 0.16); font-size: 13px; font-weight: 700; }}
    .hero h1 {{ margin: 14px 0 10px; padding-left: 2px; font-size: clamp(34px, 4.8vw, 56px); line-height: 1.08; letter-spacing: 0; overflow-wrap: anywhere; }}
    .hero p {{ margin: 0; max-width: 760px; color: rgba(255, 255, 255, 0.86); font-size: 15px; line-height: 1.8; }}
    .chip-row {{ display: flex; flex-wrap: wrap; gap: 10px; margin-top: 20px; }}
    .hero-score {{ width: 170px; min-height: 150px; border-radius: 8px; padding: 18px; display: grid; align-content: center; text-align: center; background: rgba(255, 255, 255, 0.17); border: 1px solid rgba(255, 255, 255, 0.26); }}
    .score-number {{ font-size: 58px; line-height: 1; font-weight: 820; }}
    .score-label {{ margin-top: 8px; color: rgba(255, 255, 255, 0.82); font-size: 13px; }}
    .report-layout {{ display: grid; grid-template-columns: 248px minmax(0, 1fr); gap: 28px; align-items: start; margin-top: 24px; }}
    .decision-panel {{ margin-top: 18px; display: grid; gap: 14px; border-radius: 8px; padding: 18px; background: rgba(255, 255, 255, 0.92); border: 1px solid var(--outline); box-shadow: 0 1px 2px rgba(60, 64, 67, 0.10); }}
    .decision-main {{ display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(280px, 0.65fr); gap: 14px; }}
    .decision-block {{ min-width: 0; border-radius: 8px; padding: 16px; background: var(--surface-2); border: 1px solid var(--outline); }}
    .decision-primary {{ background: var(--primary-container); border-color: #a8c7fa; }}
    .decision-label {{ color: var(--muted); font-size: 12px; font-weight: 820; }}
    .decision-gap {{ margin-top: 14px; }}
    .decision-block h2 {{ margin: 8px 0 8px; font-size: 22px; line-height: 1.35; letter-spacing: 0; }}
    .decision-block p {{ margin: 8px 0 0; color: var(--text); font-size: 14px; line-height: 1.7; }}
    .budget-strip {{ display: grid; grid-template-columns: auto minmax(0, 1fr); gap: 12px; align-items: start; border-radius: 8px; padding: 12px 14px; background: #fff8e1; border: 1px solid #fdd663; color: var(--text); }}
    .budget-strip span {{ color: var(--amber); font-size: 12px; font-weight: 840; white-space: nowrap; }}
    .budget-strip strong {{ font-size: 13px; line-height: 1.6; font-weight: 680; overflow-wrap: anywhere; }}
    .decision-evidence, .decision-modules {{ display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }}
    .evidence-card, .module-health-card {{ min-width: 0; min-height: 92px; border-radius: 8px; padding: 13px; background: #fff; border: 1px solid var(--outline); }}
    .evidence-head {{ display: flex; align-items: center; justify-content: space-between; gap: 8px; color: var(--muted); font-size: 12px; font-weight: 800; }}
    .evidence-value {{ margin-top: 12px; color: var(--text); font-size: 17px; line-height: 1.3; font-weight: 820; overflow-wrap: anywhere; }}
    .module-health-title {{ color: var(--muted); font-size: 12px; font-weight: 820; margin-bottom: 10px; }}
    .module-health-confidence {{ margin-top: 12px; color: var(--text); font-size: 13px; font-weight: 720; }}
    .evidence-map {{ margin-top: 18px; display: grid; gap: 14px; border-radius: 8px; padding: 18px; background: rgba(255, 255, 255, 0.92); border: 1px solid var(--outline); box-shadow: 0 1px 2px rgba(60, 64, 67, 0.10); }}
    .evidence-map-title span {{ color: var(--primary); font-size: 12px; font-weight: 840; }}
    .evidence-map-title h2 {{ margin: 5px 0 6px; font-size: 22px; letter-spacing: 0; }}
    .evidence-map-title p {{ margin: 0; color: var(--muted); font-size: 13px; line-height: 1.7; }}
    .evidence-map-group {{ min-width: 0; border-radius: 8px; background: #fff; border: 1px solid var(--outline); overflow: hidden; }}
    .evidence-map-head {{ padding: 14px 16px; background: var(--surface-2); border-bottom: 1px solid var(--outline); }}
    .evidence-map-head h3 {{ margin: 0; font-size: 15px; }}
    .evidence-map-head p {{ margin: 5px 0 0; color: var(--muted); font-size: 12px; line-height: 1.55; }}
    .evidence-table-wrap {{ overflow-x: auto; }}
    .evidence-table {{ width: 100%; border-collapse: collapse; }}
    .evidence-table th, .evidence-table td {{ padding: 11px 12px; text-align: left; border-bottom: 1px solid var(--outline); vertical-align: top; font-size: 12px; line-height: 1.5; }}
    .evidence-table th {{ color: var(--muted); font-weight: 820; white-space: nowrap; }}
    .evidence-table td:nth-child(1), .evidence-table td:nth-child(2) {{ white-space: nowrap; font-weight: 700; }}
    .evidence-table td:nth-child(3) {{ min-width: 240px; overflow-wrap: anywhere; }}
    .sidebar {{ position: sticky; top: 16px; padding: 2px 12px 12px 0; background: transparent; border-right: 1px solid var(--outline); box-shadow: none; overflow: visible; }}
    .sidebar-title {{ padding: 4px 8px 10px; color: var(--muted); font-size: 12px; font-weight: 800; }}
    .nav-group + .nav-group {{ margin-top: 12px; padding-top: 10px; border-top: 1px solid var(--outline); }}
    .nav-group-title {{ padding: 2px 8px 5px; color: var(--muted); font-size: 11px; font-weight: 820; text-transform: uppercase; }}
    .nav-link {{ display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 34px; padding: 6px 8px 6px 12px; border-radius: 999px; color: var(--text); text-decoration: none; font-size: 13px; font-weight: 680; }}
    .nav-link:hover, .nav-link.active {{ background: var(--primary-container); color: #041e49; }}
    .nav-link.active {{ box-shadow: inset 3px 0 0 var(--primary); }}
    .content-stack {{ display: grid; gap: 18px; }}
    .report-health {{ display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-top: 16px; }}
    .health-item {{ min-height: 66px; border-radius: 8px; padding: 10px 12px; color: var(--text); background: rgba(255, 255, 255, 0.16); border: 1px solid rgba(255, 255, 255, 0.24); }}
    .health-label {{ font-size: 12px; color: rgba(255, 255, 255, 0.78); font-weight: 720; }}
    .health-value {{ margin-top: 4px; font-size: 24px; font-weight: 820; }}
    .section-group {{ display: grid; gap: 14px; }}
    .section-group-head {{ padding: 2px 2px 0; }}
    .section-group-head h2 {{ margin: 0; font-size: 18px; }}
    .section-group-head p {{ margin: 5px 0 0; color: var(--muted); font-size: 13px; line-height: 1.6; }}
    .module-card {{ scroll-margin-top: 20px; border-radius: 8px; padding: 22px; background: rgba(255, 255, 255, 0.86); border: 1px solid var(--outline); box-shadow: 0 1px 2px rgba(60, 64, 67, 0.10); }}
    .module-panel[hidden] {{ display: none; }}
    .module-skipped {{ background: rgba(255, 250, 240, 0.88); border-color: #fdd663; }}
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
    .artifact-grid {{ display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 16px; }}
    .artifact-link {{ display: flex; align-items: center; justify-content: space-between; gap: 14px; min-height: 64px; padding: 12px 14px; border-radius: 8px; color: var(--text); text-decoration: none; border: 1px solid var(--outline); background: var(--surface-2); }}
    .artifact-link strong {{ display: block; font-size: 13px; }}
    .artifact-link small {{ display: block; margin-top: 4px; color: var(--muted); font-size: 11px; overflow-wrap: anywhere; }}
    .artifact-ready:hover {{ border-color: var(--primary); background: var(--primary-container); }}
    .artifact-missing {{ color: var(--muted); pointer-events: none; opacity: 0.72; }}
    .footer {{ margin-top: 24px; padding: 22px; text-align: center; color: var(--muted); font-size: 13px; }}
    @media (max-width: 1100px) {{
      .report-layout {{ grid-template-columns: 1fr; }}
      .decision-main {{ grid-template-columns: 1fr; }}
      .decision-evidence, .decision-modules {{ grid-template-columns: repeat(2, minmax(0, 1fr)); }}
      .sidebar {{ position: static; display: flex; gap: 8px; padding: 0 0 10px; border-right: 0; border-bottom: 1px solid var(--outline); overflow-x: auto; }}
      .sidebar-title {{ display: none; }}
      .nav-link {{ flex: 0 0 auto; }}
    }}
    @media (max-width: 760px) {{
      .app-shell {{ width: min(100% - 20px, 1440px); padding-top: 12px; }}
      .top-app-bar {{ align-items: flex-start; flex-direction: column; gap: 12px; }}
      .actions {{ justify-content: flex-start; }}
      .hero {{ padding: 18px; }}
      .module-card {{ padding: 18px; }}
      .hero-content, .detail-grid {{ grid-template-columns: 1fr; }}
      .hero h1 {{ font-size: 36px; margin-top: 12px; }}
      .hero p {{ font-size: 14px; line-height: 1.65; }}
      .chip-row {{ gap: 8px; margin-top: 14px; }}
      .report-health {{ display: none; }}
      .hero-score {{ width: 100%; min-height: 82px; padding: 12px; grid-template-columns: auto 1fr auto; align-items: center; align-content: center; gap: 12px; text-align: left; }}
      .score-number {{ font-size: 42px; }}
      .score-label {{ margin-top: 0; }}
      .health-item {{ min-height: 58px; }}
      .metric-grid, .artifact-grid {{ grid-template-columns: repeat(2, minmax(0, 1fr)); }}
      .decision-evidence, .decision-modules {{ grid-template-columns: 1fr; }}
      .module-head {{ flex-direction: column; }}
    }}
    @media (max-width: 480px) {{ .metric-grid, .report-health, .artifact-grid {{ grid-template-columns: 1fr; }} }}
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
          <p>生成时间 {esc(timestamp)}。报告按模块组织系统信息、CPU、内存、磁盘、网络和扩展检测，左侧目录点击后右侧只显示当前模块。</p>
          <div class="chip-row">
            <span class="chip">评分基准 {esc(summary.get("score_profile"))}</span>
            <span class="chip">评测档位 {esc(data_get(summary, "benchmark_profile", "name"))}</span>
            <span class="chip">置信度 {esc(data_get(summary, "confidence_level", "level"))}</span>
            <span class="chip">校准 {esc(data_get(summary, "score_calibration", "version"))}</span>
          </div>
          <div class="report-health">
            <div class="health-item"><div class="health-label">成功模块</div><div class="health-value">{status_counts["success"]}</div></div>
            <div class="health-item"><div class="health-label">注意模块</div><div class="health-value">{status_counts["warning"]}</div></div>
            <div class="health-item"><div class="health-label">失败模块</div><div class="health-value">{status_counts["failed"]}</div></div>
            <div class="health-item"><div class="health-label">未执行模块</div><div class="health-value">{status_counts["skipped"]}</div></div>
          </div>
        </div>
        <div class="hero-score">
          <div class="score-number">{esc(total_score)}</div>
          <div class="score-label">总体评分 / 100</div>
          <div style="margin-top: 14px;"><span class="status-badge status-success">{esc(grade)}</span></div>
        </div>
      </div>
    </section>
    {decision_panel}
    {evidence_map}
    <section class="report-layout">
      <nav class="sidebar" aria-label="报告目录">
        <div class="sidebar-title">报告目录</div>
        {nav}
      </nav>
      <div class="content-stack">{''.join(cards)}</div>
    </section>
    <footer class="footer">Perfassess Web 报告</footer>
  </main>
  <script>
    const panels = Array.from(document.querySelectorAll(".module-panel"));
    const links = Array.from(document.querySelectorAll(".nav-link[data-section]"));
    function showPanel(id) {{
      const target = panels.some((panel) => panel.dataset.panel === id) ? id : (panels[0]?.dataset.panel || "");
      panels.forEach((panel) => {{
        panel.hidden = panel.dataset.panel !== target;
      }});
      links.forEach((link) => {{
        link.classList.toggle("active", link.dataset.section === target);
      }});
      if (target && location.hash !== "#" + target) {{
        history.replaceState(null, "", "#" + target);
      }}
    }}
    links.forEach((link) => {{
      link.addEventListener("click", (event) => {{
        event.preventDefault();
        showPanel(link.dataset.section);
      }});
    }});
    showPanel((location.hash || "").replace("#", ""));
  </script>
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
                    self.send_bytes(render_report(report, self.output_dir).encode("utf-8"), "text/html; charset=utf-8")
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
            self.send_bytes(render_report(report, self.output_dir).encode("utf-8"), "text/html; charset=utf-8")
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
