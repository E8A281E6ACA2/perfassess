// Package reporter 提供 Web 报告模板
package reporter

// HTMLTemplate Web 报告的 HTML 模板
const HTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>性能评估报告 - {{.SessionID}}</title>
    <style>
        :root {
            --md-primary: #0b57d0;
            --md-on-primary: #ffffff;
            --md-primary-container: #d3e3fd;
            --md-on-primary-container: #041e49;
            --md-secondary: #00639b;
            --md-green: #146c2e;
            --md-amber: #b06000;
            --md-error: #b3261e;
            --md-surface: #fffdf7;
            --md-surface-low: #f8fafd;
            --md-surface-container: #f1f4f9;
            --md-surface-high: #e9eef6;
            --md-text: #1f1f1f;
            --md-muted: #5f6368;
            --md-outline: #c4c7c5;
            --md-outline-variant: #dfe3eb;
            --md-shadow-1: 0 1px 2px rgba(60, 64, 67, 0.18), 0 1px 3px rgba(60, 64, 67, 0.12);
            --md-shadow-2: 0 2px 6px rgba(60, 64, 67, 0.16), 0 8px 24px rgba(60, 64, 67, 0.10);
        }

        * { box-sizing: border-box; }

        html { scroll-behavior: smooth; }

        body {
            margin: 0;
            min-height: 100vh;
            color: var(--md-text);
            font-family: "Google Sans", "Noto Sans SC", "Product Sans", "Helvetica Neue", sans-serif;
            background:
                radial-gradient(circle at top left, rgba(211, 227, 253, 0.95), transparent 32rem),
                radial-gradient(circle at 85% 10%, rgba(196, 231, 198, 0.78), transparent 28rem),
                linear-gradient(180deg, #f7faff 0%, #fffdf7 52%, #f4f7fb 100%);
        }

        .app-shell {
            width: min(1440px, calc(100% - 32px));
            margin: 0 auto;
            padding: 24px 0 42px;
        }

        .top-app-bar {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 16px;
            margin-bottom: 20px;
        }

        .brand {
            display: flex;
            align-items: center;
            gap: 12px;
            min-width: 0;
        }

        .brand-mark {
            width: 44px;
            height: 44px;
            border-radius: 14px;
            display: grid;
            place-items: center;
            color: var(--md-on-primary);
            background: var(--md-primary);
            font-weight: 800;
            box-shadow: var(--md-shadow-1);
        }

        .brand-title {
            font-size: 18px;
            font-weight: 760;
        }

        .brand-subtitle {
            color: var(--md-muted);
            font-size: 13px;
            margin-top: 2px;
        }

        .chip,
        .status-badge {
            display: inline-flex;
            align-items: center;
            min-height: 30px;
            padding: 5px 11px;
            border-radius: 999px;
            font-size: 12px;
            font-weight: 720;
        }

        .chip {
            color: var(--md-on-primary-container);
            background: rgba(211, 227, 253, 0.92);
        }

        .hero {
            overflow: hidden;
            border-radius: 28px;
            padding: 32px;
            color: var(--md-on-primary);
            background:
                linear-gradient(135deg, rgba(11, 87, 208, 0.95), rgba(0, 99, 155, 0.90)),
                radial-gradient(circle at 86% 18%, rgba(255, 255, 255, 0.32), transparent 18rem);
            box-shadow: var(--md-shadow-2);
        }

        .hero-content {
            display: grid;
            grid-template-columns: minmax(0, 1fr) auto;
            gap: 28px;
            align-items: end;
        }

        .eyebrow {
            display: inline-flex;
            align-items: center;
            border-radius: 999px;
            padding: 7px 12px;
            background: rgba(255, 255, 255, 0.16);
            font-size: 13px;
            font-weight: 700;
        }

        .hero h1 {
            margin: 18px 0 10px;
            font-size: clamp(34px, 6vw, 64px);
            line-height: 0.98;
        }

        .hero p {
            margin: 0;
            max-width: 760px;
            color: rgba(255, 255, 255, 0.86);
            font-size: 15px;
            line-height: 1.8;
        }

        .chip-row {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
            margin-top: 20px;
        }

        .hero-score {
            width: 190px;
            min-height: 190px;
            border-radius: 28px;
            padding: 22px;
            display: grid;
            align-content: center;
            text-align: center;
            background: rgba(255, 255, 255, 0.17);
            border: 1px solid rgba(255, 255, 255, 0.26);
            backdrop-filter: blur(14px);
        }

        .score-number {
            font-size: 64px;
            line-height: 1;
            font-weight: 820;
        }

        .score-label {
            margin-top: 8px;
            color: rgba(255, 255, 255, 0.82);
            font-size: 13px;
        }

        .report-layout {
            display: grid;
            grid-template-columns: 248px minmax(0, 1fr);
            gap: 28px;
            align-items: start;
            margin-top: 24px;
        }

        .decision-panel {
            display: grid;
            grid-template-columns: minmax(0, 1.15fr) minmax(280px, 0.85fr);
            gap: 16px;
            margin-top: 18px;
        }

        .decision-main,
        .decision-side {
            border-radius: 24px;
            padding: 20px;
            background: rgba(255, 255, 255, 0.86);
            border: 1px solid rgba(196, 199, 197, 0.72);
            box-shadow: var(--md-shadow-1);
        }

        .decision-kicker {
            color: var(--md-primary);
            font-size: 12px;
            font-weight: 820;
        }

        .decision-title {
            margin: 8px 0 8px;
            font-size: 24px;
            line-height: 1.25;
        }

        .decision-scenario {
            margin: 0;
            color: var(--md-muted);
            font-size: 14px;
            line-height: 1.7;
        }

        .pill-list,
        .decision-list {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-top: 14px;
        }

        .decision-pill {
            border-radius: 999px;
            padding: 7px 11px;
            color: var(--md-on-primary-container);
            background: var(--md-primary-container);
            font-size: 12px;
            font-weight: 720;
        }

        .evidence-grid {
            display: grid;
            grid-template-columns: repeat(2, minmax(0, 1fr));
            gap: 10px;
            margin-top: 16px;
        }

        .evidence-item {
            min-height: 82px;
            border-radius: 16px;
            padding: 12px;
            background: var(--md-surface-low);
            border: 1px solid var(--md-outline-variant);
        }

        .evidence-label {
            display: flex;
            justify-content: space-between;
            gap: 8px;
            color: var(--md-muted);
            font-size: 12px;
            font-weight: 760;
        }

        .evidence-value {
            margin-top: 7px;
            color: var(--md-text);
            font-size: 13px;
            font-weight: 720;
            overflow-wrap: anywhere;
        }

        .decision-side h2 {
            margin: 0 0 10px;
            font-size: 16px;
        }

        .decision-side ul {
            margin: 8px 0 0;
            padding-left: 19px;
            color: var(--md-text);
            font-size: 13px;
            line-height: 1.7;
        }

        .bottleneck-row {
            display: grid;
            grid-template-columns: 64px 1fr auto;
            gap: 10px;
            align-items: center;
            padding: 9px 0;
            border-bottom: 1px solid var(--md-outline-variant);
            font-size: 13px;
        }

        .bar-track {
            height: 8px;
            border-radius: 999px;
            background: var(--md-surface-container);
            overflow: hidden;
        }

        .bar-fill {
            height: 100%;
            border-radius: inherit;
            background: var(--md-primary);
        }

        .sidebar {
            position: sticky;
            top: 16px;
            overflow: visible;
            padding: 2px 12px 12px 0;
            background: transparent;
            border-right: 1px solid var(--md-outline-variant);
            box-shadow: none;
        }

        .sidebar-title {
            padding: 4px 8px 10px;
            color: var(--md-muted);
            font-size: 12px;
            font-weight: 800;
        }

        .nav-group + .nav-group {
            margin-top: 12px;
            padding-top: 10px;
            border-top: 1px solid var(--md-outline-variant);
        }

        .nav-group-title {
            padding: 2px 8px 5px;
            color: var(--md-muted);
            font-size: 11px;
            font-weight: 820;
        }

        .nav-link {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 8px;
            min-height: 34px;
            padding: 6px 8px 6px 12px;
            border-radius: 999px;
            color: var(--md-text);
            text-decoration: none;
            font-size: 13px;
            font-weight: 680;
        }

        .nav-link:hover {
            background: var(--md-primary-container);
            color: var(--md-on-primary-container);
        }

        .content-stack {
            display: grid;
            gap: 18px;
        }

        .report-health {
            display: grid;
            grid-template-columns: repeat(4, minmax(0, 1fr));
            gap: 10px;
            margin-top: 18px;
        }

        .health-item {
            min-height: 74px;
            border-radius: 18px;
            padding: 12px;
            background: rgba(255, 255, 255, 0.16);
            border: 1px solid rgba(255, 255, 255, 0.24);
        }

        .health-label {
            color: rgba(255, 255, 255, 0.78);
            font-size: 12px;
            font-weight: 720;
        }

        .health-value {
            margin-top: 6px;
            font-size: 26px;
            font-weight: 820;
        }

        .section-group {
            display: grid;
            gap: 14px;
        }

        .section-group-head {
            padding: 2px 2px 0;
        }

        .section-group-head h2 {
            margin: 0;
            font-size: 18px;
        }

        .section-group-head p {
            margin: 5px 0 0;
            color: var(--md-muted);
            font-size: 13px;
            line-height: 1.6;
        }

        .module-card {
            scroll-margin-top: 20px;
            border-radius: 24px;
            padding: 22px;
            background: rgba(255, 255, 255, 0.80);
            border: 1px solid rgba(196, 199, 197, 0.72);
            box-shadow: var(--md-shadow-1);
        }

        .module-skipped {
            background: rgba(255, 250, 240, 0.88);
            border-color: #fdd663;
        }

        .module-head {
            display: flex;
            justify-content: space-between;
            gap: 16px;
            align-items: flex-start;
            margin-bottom: 18px;
        }

        .module-title {
            margin: 0;
            font-size: 24px;
            line-height: 1.2;
        }

        .module-subtitle {
            margin: 7px 0 0;
            color: var(--md-muted);
            font-size: 14px;
            line-height: 1.6;
        }

        .module-summary {
            margin: 0 0 16px;
            color: var(--md-text);
            font-size: 14px;
            line-height: 1.7;
        }

        .metric-grid {
            display: grid;
            grid-template-columns: repeat(4, minmax(0, 1fr));
            gap: 12px;
            margin-bottom: 14px;
        }

        .metric-card {
            min-height: 104px;
            border-radius: 18px;
            padding: 16px;
            background: var(--md-surface-low);
            border: 1px solid var(--md-outline-variant);
        }

        .metric-card h3 {
            margin: 0 0 8px;
            color: var(--md-muted);
            font-size: 12px;
            font-weight: 760;
        }

        .metric-value {
            color: var(--md-primary);
            font-size: 34px;
            line-height: 1.05;
            font-weight: 840;
            overflow-wrap: anywhere;
        }

        .metric-unit {
            margin-top: 5px;
            color: var(--md-muted);
            font-size: 12px;
            font-weight: 650;
        }

        .tone-green .metric-value { color: var(--md-green); }
        .tone-amber .metric-value { color: var(--md-amber); }
        .tone-cyan .metric-value { color: var(--md-secondary); }
        .tone-red .metric-value { color: var(--md-error); }

        .detail-grid {
            display: grid;
            grid-template-columns: repeat(2, minmax(0, 1fr));
            gap: 0 18px;
            margin-top: 4px;
        }

        .detail-row {
            display: flex;
            justify-content: space-between;
            gap: 14px;
            padding: 11px 0;
            border-bottom: 1px solid var(--md-outline-variant);
        }

        .detail-label {
            color: var(--md-muted);
            font-size: 13px;
            white-space: nowrap;
        }

        .detail-value {
            min-width: 0;
            color: var(--md-text);
            font-size: 13px;
            font-weight: 650;
            text-align: right;
            overflow-wrap: anywhere;
        }

        .assessment-grid {
            display: grid;
            grid-template-columns: minmax(0, 1.2fr) minmax(220px, 0.8fr) minmax(220px, 0.8fr);
            gap: 14px;
            margin-top: 16px;
            padding-top: 16px;
            border-top: 1px solid var(--md-outline-variant);
        }

        .assessment-block h3 {
            margin: 0 0 10px;
            color: var(--md-muted);
            font-size: 13px;
            font-weight: 820;
        }

        .assessment-row {
            display: grid;
            grid-template-columns: auto minmax(0, 1fr);
            gap: 10px;
            align-items: start;
            padding: 9px 0;
            border-bottom: 1px solid var(--md-outline-variant);
        }

        .assessment-row strong {
            display: block;
            color: var(--md-text);
            font-size: 13px;
        }

        .assessment-row p {
            margin: 3px 0 0;
            color: var(--md-muted);
            font-size: 13px;
            line-height: 1.5;
            overflow-wrap: anywhere;
        }

        .assessment-block ul {
            margin: 0;
            padding-left: 18px;
            color: var(--md-text);
            font-size: 13px;
            line-height: 1.7;
        }

        .status-success {
            color: #0d3b1e;
            background: #c4e7c6;
        }

        .grade-excellent,
        .grade-good,
        .grade-fair,
        .grade-poor {
            color: #041e49;
            background: rgba(211, 227, 253, 0.92);
        }

        .grade-excellent {
            color: #0d3b1e;
            background: #c4e7c6;
        }

        .grade-good {
            color: #003355;
            background: #c2e7ff;
        }

        .grade-poor {
            color: #601410;
            background: #f9dedc;
        }

        .status-degraded,
        .status-warning,
        .status-fair {
            color: #003355;
            background: #c2e7ff;
        }

        .status-skipped {
            color: #4a3000;
            background: #fdd663;
        }

        .status-failed {
            color: #601410;
            background: #f9dedc;
        }

        .status-unknown {
            color: var(--md-muted);
            background: var(--md-surface-container);
        }

        .hint-box {
            margin-top: 14px;
            border-radius: 16px;
            padding: 14px 16px;
            color: #4a3000;
            background: #fff7df;
            border: 1px solid #fdd663;
            font-size: 14px;
            line-height: 1.65;
        }

        .data-table-wrap {
            margin-top: 18px;
            overflow-x: auto;
        }

        .table-title {
            margin: 0 0 10px;
            color: var(--md-muted);
            font-size: 13px;
            font-weight: 780;
        }

        .data-table {
            width: 100%;
            border-collapse: collapse;
            overflow: hidden;
            border-radius: 16px;
        }

        .data-table th,
        .data-table td {
            padding: 13px 12px;
            text-align: left;
            border-bottom: 1px solid var(--md-outline-variant);
            vertical-align: top;
            font-size: 13px;
            line-height: 1.5;
        }

        .data-table th {
            color: var(--md-muted);
            background: var(--md-surface-container);
            font-size: 12px;
            font-weight: 820;
        }

        .quality-notes {
            border-color: #fdd663;
            background: #fffaf0;
        }

        .quality-notes ul {
            margin: 10px 0 0;
            padding-left: 20px;
            color: #4a3000;
            line-height: 1.7;
        }

        .share-box {
            white-space: pre-wrap;
            overflow-x: auto;
            border-radius: 18px;
            padding: 16px;
            color: var(--md-text);
            background: var(--md-surface-container);
            border: 1px solid var(--md-outline-variant);
            font: 13px/1.65 "Roboto Mono", "SFMono-Regular", Consolas, monospace;
        }

        .footer {
            margin-top: 24px;
            padding: 22px;
            text-align: center;
            color: var(--md-muted);
            font-size: 13px;
        }

        @media (max-width: 1100px) {
            .report-layout {
                grid-template-columns: 1fr;
            }

            .decision-panel {
                grid-template-columns: 1fr;
            }

            .evidence-grid {
                grid-template-columns: 1fr;
            }

            .sidebar {
                position: static;
                display: flex;
                gap: 8px;
                padding: 0 0 10px;
                border-right: 0;
                border-bottom: 1px solid var(--md-outline-variant);
                overflow-x: auto;
            }

            .sidebar-title {
                display: none;
            }

            .nav-link {
                flex: 0 0 auto;
            }
        }

        @media (max-width: 760px) {
            .app-shell {
                width: min(100% - 20px, 1440px);
                padding-top: 12px;
            }

            .hero,
            .module-card {
                padding: 18px;
                border-radius: 22px;
            }

            .hero-content,
            .detail-grid,
            .assessment-grid {
                grid-template-columns: 1fr;
            }

            .report-health {
                grid-template-columns: repeat(2, minmax(0, 1fr));
            }

            .hero-score {
                width: 100%;
                min-height: 130px;
            }

            .metric-grid {
                grid-template-columns: repeat(2, minmax(0, 1fr));
            }

            .module-head {
                flex-direction: column;
            }
        }

        @media (max-width: 480px) {
            .metric-grid,
            .report-health {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <main class="app-shell">
        <div class="top-app-bar">
            <div class="brand">
                <div class="brand-mark">P</div>
                <div>
                    <div class="brand-title">Perfassess</div>
                    <div class="brand-subtitle">Material Design 3 Web Report</div>
                </div>
            </div>
            <span class="chip">Session {{.SessionID}}</span>
        </div>

        <section class="hero">
            <div class="hero-content">
                <div>
                    <span class="eyebrow">VPS Benchmark Report</span>
                    <h1>性能评估报告</h1>
                    <p>生成时间 {{.Timestamp}}。报告按模块组织系统信息、CPU、内存、磁盘、网络和扩展检测，左侧目录可快速跳转。</p>
                    <div class="chip-row">
                        <span class="chip">评分基准 {{.ScoreProfile}}</span>
                        <span class="chip">评测档位 {{.BenchmarkProfileName}}</span>
                        <span class="chip">置信度 {{.ConfidenceLevel}}</span>
                        <span class="chip">校准 {{.CalibrationVersion}}</span>
                    </div>
                    <div class="report-health">
                        <div class="health-item">
                            <div class="health-label">成功模块</div>
                            <div class="health-value">{{index .ReportStatusCounts "success"}}</div>
                        </div>
                        <div class="health-item">
                            <div class="health-label">注意模块</div>
                            <div class="health-value">{{index .ReportStatusCounts "warning"}}</div>
                        </div>
                        <div class="health-item">
                            <div class="health-label">失败模块</div>
                            <div class="health-value">{{index .ReportStatusCounts "failed"}}</div>
                        </div>
                        <div class="health-item">
                            <div class="health-label">未执行模块</div>
                            <div class="health-value">{{index .ReportStatusCounts "skipped"}}</div>
                        </div>
                    </div>
                </div>
                <div class="hero-score">
                    <div class="score-number">{{printf "%.0f" .OverallScore.TotalScore}}</div>
                    <div class="score-label">总体评分 / 100</div>
                    <div style="margin-top: 14px;"><span class="status-badge {{.OverallGradeClass}}">{{.OverallScore.Grade}}</span></div>
                </div>
            </div>
        </section>

        <section class="decision-panel" aria-label="测评决策面板">
            <div class="decision-main">
                <div class="decision-kicker">决策摘要</div>
                <h2 class="decision-title">{{.DecisionPanel.Headline}}</h2>
                <p class="decision-scenario">{{.DecisionPanel.Scenario}}</p>
                {{if .DecisionPanel.Suitability}}
                <div class="pill-list">
                    {{range .DecisionPanel.Suitability}}
                    <span class="decision-pill">{{.}}</span>
                    {{end}}
                </div>
                {{end}}
                {{if .DecisionPanel.Evidence}}
                <div class="decision-kicker" style="margin-top: 16px;">关键证据</div>
                <div class="evidence-grid">
                    {{range .DecisionPanel.Evidence}}
                    <div class="evidence-item">
                        <div class="evidence-label">
                            <span>{{.Label}}</span>
                            <span class="status-badge status-{{.Status}}">{{.StatusText}}</span>
                        </div>
                        <div class="evidence-value">{{.Value}}</div>
                    </div>
                    {{end}}
                </div>
                {{end}}
            </div>
            <aside class="decision-side">
                <h2>短板排序</h2>
                {{range .DecisionPanel.Bottlenecks}}
                <div class="bottleneck-row">
                    <span>{{.Label}}</span>
                    <span class="bar-track"><span class="bar-fill" style="width: {{.Percent}};"></span></span>
                    <strong>{{printf "%.0f" .Score}}</strong>
                </div>
                {{end}}
                {{if .DecisionPanel.Recommendations}}
                <h2 style="margin-top: 18px;">优先建议</h2>
                <ul>
                    {{range .DecisionPanel.Recommendations}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
                {{end}}
                {{if .DecisionPanel.Limitations}}
                <h2 style="margin-top: 18px;">限制说明</h2>
                <ul>
                    {{range .DecisionPanel.Limitations}}
                    <li>{{.}}</li>
                    {{end}}
                </ul>
                {{end}}
            </aside>
        </section>

        <section class="report-layout">
            <nav class="sidebar" aria-label="报告目录">
                <div class="sidebar-title">报告目录</div>
                {{range .ReportGroups}}
                <div class="nav-group">
                    <div class="nav-group-title">{{.Title}}</div>
                    {{range .Sections}}
                    <a class="nav-link" href="#{{.ID}}">
                        <span>{{.Title}}</span>
                        <span class="status-badge status-{{.Status}}">{{.StatusText}}</span>
                    </a>
                    {{end}}
                </div>
                {{end}}
            </nav>

            <div class="content-stack">
                {{range .ReportGroups}}
                {{if ne .ID "delivery"}}
                <div class="section-group" id="group-{{.ID}}">
                    <div class="section-group-head">
                        <h2>{{.Title}}</h2>
                        <p>{{.Subtitle}}</p>
                    </div>
                    {{range .Sections}}
                    <section id="{{.ID}}" class="module-card module-{{.Status}}">
                    <div class="module-head">
                        <div>
                            <h2 class="module-title">{{.Title}}</h2>
                            <p class="module-subtitle">{{.Subtitle}}</p>
                        </div>
                        <span class="status-badge status-{{.Status}}">{{.StatusText}}</span>
                    </div>

                    <p class="module-summary">{{.Summary}}</p>

                    {{if .Metrics}}
                    <div class="metric-grid">
                        {{range .Metrics}}
                        <div class="metric-card tone-{{.Tone}}">
                            <h3>{{.Label}}</h3>
                            <div class="metric-value">{{.Value}}</div>
                            {{if .Unit}}<div class="metric-unit">{{.Unit}}</div>{{end}}
                        </div>
                        {{end}}
                    </div>
                    {{end}}

                    {{if .Details}}
                    <div class="detail-grid">
                        {{range .Details}}
                        <div class="detail-row">
                            <span class="detail-label">{{.Label}}</span>
                            <span class="detail-value">{{.Value}}</span>
                        </div>
                        {{end}}
                    </div>
                    {{end}}

                    {{if .Evidence}}
                    <div class="assessment-grid">
                        <div class="assessment-block">
                            <h3>证据</h3>
                            {{range .Evidence}}
                            <div class="assessment-row">
                                <span class="status-badge status-{{.Status}}">{{.StatusText}}</span>
                                <div>
                                    <strong>{{.Label}}</strong>
                                    <p>{{.Value}}</p>
                                </div>
                            </div>
                            {{end}}
                        </div>
                        {{if .Recommendations}}
                        <div class="assessment-block">
                            <h3>建议</h3>
                            <ul>
                                {{range .Recommendations}}<li>{{.}}</li>{{end}}
                            </ul>
                        </div>
                        {{end}}
                        {{if .Limitations}}
                        <div class="assessment-block">
                            <h3>限制</h3>
                            <ul>
                                {{range .Limitations}}<li>{{.}}</li>{{end}}
                            </ul>
                        </div>
                        {{end}}
                    </div>
                    {{end}}

                    {{range .Tables}}
                    <div class="data-table-wrap">
                        <h3 class="table-title">{{.Title}}</h3>
                        <table class="data-table">
                            <thead>
                                <tr>
                                    {{range .Headers}}<th>{{.}}</th>{{end}}
                                </tr>
                            </thead>
                            <tbody>
                                {{range .Rows}}
                                <tr>
                                    {{range .}}<td>{{.}}</td>{{end}}
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                    {{end}}

                    {{if .Hint}}
                    <div class="hint-box">{{.Hint}}</div>
                    {{end}}
                    </section>
                    {{end}}
                </div>
                {{end}}
                {{end}}

                {{if .QualityNotes}}
                <section id="quality" class="module-card quality-notes">
                    <div class="module-head">
                        <div>
                            <h2 class="module-title">质量提示</h2>
                            <p class="module-subtitle">影响报告置信度和可比性的说明。</p>
                        </div>
                        <span class="status-badge status-skipped">提示</span>
                    </div>
                    <ul>
                        {{range .QualityNotes}}
                        <li>{{.}}</li>
                        {{end}}
                    </ul>
                </section>
                {{end}}

                {{if .SharePlainText}}
                <section id="share" class="module-card">
                    <div class="module-head">
                        <div>
                            <h2 class="module-title">分享模板</h2>
                            <p class="module-subtitle">可直接复制到论坛、工单或聊天窗口。</p>
                        </div>
                        <span class="status-badge status-success">可复制</span>
                    </div>
                    <div class="share-box">{{.SharePlainText}}</div>
                </section>
                {{end}}
            </div>
        </section>

        <footer class="footer">
            <p>Perfassess Web 报告</p>
        </footer>
    </main>
</body>
</html>`
