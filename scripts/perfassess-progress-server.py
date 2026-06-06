#!/usr/bin/env python3
import argparse
import json
import mimetypes
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Optional
from urllib.parse import unquote, urlparse


HTML = """<!doctype html>
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
    <span class="badge pending" id="overall">等待中</span>
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
            self.send_bytes(HTML.encode("utf-8"), "text/html; charset=utf-8")
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
