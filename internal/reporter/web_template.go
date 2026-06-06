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
            --md-sys-color-primary: #0b57d0;
            --md-sys-color-on-primary: #ffffff;
            --md-sys-color-primary-container: #d3e3fd;
            --md-sys-color-on-primary-container: #041e49;
            --md-sys-color-secondary: #00639b;
            --md-sys-color-tertiary: #146c2e;
            --md-sys-color-error: #b3261e;
            --md-sys-color-surface: #fffdf7;
            --md-sys-color-surface-dim: #ddd9d0;
            --md-sys-color-surface-container-low: #f8fafd;
            --md-sys-color-surface-container: #f1f4f9;
            --md-sys-color-surface-container-high: #e9eef6;
            --md-sys-color-on-surface: #1f1f1f;
            --md-sys-color-on-surface-variant: #5f6368;
            --md-sys-color-outline: #c4c7c5;
            --md-sys-color-outline-variant: #dfe3eb;
            --md-sys-elevation-1: 0 1px 2px rgba(60, 64, 67, 0.18), 0 1px 3px rgba(60, 64, 67, 0.12);
            --md-sys-elevation-2: 0 2px 6px rgba(60, 64, 67, 0.16), 0 8px 24px rgba(60, 64, 67, 0.10);
            --md-sys-shape-xl: 28px;
            --md-sys-shape-lg: 22px;
            --md-sys-shape-md: 16px;
        }

        * {
            box-sizing: border-box;
        }

        body {
            margin: 0;
            min-height: 100vh;
            color: var(--md-sys-color-on-surface);
            font-family: "Google Sans", "Noto Sans SC", "Product Sans", "Helvetica Neue", sans-serif;
            background:
                radial-gradient(circle at top left, rgba(211, 227, 253, 0.95), transparent 32rem),
                radial-gradient(circle at 85% 10%, rgba(196, 231, 198, 0.8), transparent 28rem),
                linear-gradient(180deg, #f7faff 0%, #fffdf7 52%, #f4f7fb 100%);
        }

        .app-shell {
            width: min(1240px, calc(100% - 32px));
            margin: 0 auto;
            padding: 28px 0 40px;
        }

        .top-app-bar {
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 16px;
            margin-bottom: 22px;
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
            color: var(--md-sys-color-on-primary);
            background: var(--md-sys-color-primary);
            font-weight: 800;
            box-shadow: var(--md-sys-elevation-1);
        }

        .brand-title {
            font-size: 18px;
            font-weight: 700;
            letter-spacing: -0.01em;
        }

        .brand-subtitle {
            color: var(--md-sys-color-on-surface-variant);
            font-size: 13px;
            margin-top: 2px;
        }

        .hero {
            position: relative;
            overflow: hidden;
            border-radius: var(--md-sys-shape-xl);
            padding: 34px;
            background:
                linear-gradient(135deg, rgba(11, 87, 208, 0.94), rgba(0, 99, 155, 0.90)),
                radial-gradient(circle at 80% 20%, rgba(255, 255, 255, 0.35), transparent 18rem);
            color: var(--md-sys-color-on-primary);
            box-shadow: var(--md-sys-elevation-2);
        }

        .hero::after {
            content: "";
            position: absolute;
            right: -80px;
            bottom: -140px;
            width: 320px;
            height: 320px;
            border-radius: 50%;
            background: rgba(255, 255, 255, 0.14);
        }

        .hero-content {
            position: relative;
            z-index: 1;
            display: grid;
            grid-template-columns: minmax(0, 1fr) auto;
            gap: 28px;
            align-items: end;
        }

        .eyebrow {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            border-radius: 999px;
            padding: 7px 12px;
            background: rgba(255, 255, 255, 0.16);
            font-size: 13px;
            font-weight: 600;
        }

        .hero h1 {
            margin: 18px 0 10px;
            font-size: clamp(34px, 7vw, 68px);
            line-height: 0.95;
            letter-spacing: -0.055em;
        }

        .hero p {
            margin: 0;
            max-width: 720px;
            font-size: 15px;
            line-height: 1.8;
            color: rgba(255, 255, 255, 0.86);
        }

        .hero-score {
            width: 190px;
            min-height: 190px;
            border-radius: 34px;
            padding: 22px;
            display: grid;
            align-content: center;
            text-align: center;
            background: rgba(255, 255, 255, 0.17);
            border: 1px solid rgba(255, 255, 255, 0.26);
            backdrop-filter: blur(14px);
        }

        .hero-score .score-number {
            font-size: 64px;
            line-height: 1;
            font-weight: 800;
            letter-spacing: -0.06em;
        }

        .hero-score .score-label {
            margin-top: 8px;
            font-size: 13px;
            color: rgba(255, 255, 255, 0.82);
        }

        .chip-row {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
            margin-top: 20px;
        }

        .chip {
            display: inline-flex;
            align-items: center;
            min-height: 32px;
            padding: 6px 12px;
            border-radius: 999px;
            color: var(--md-sys-color-on-primary-container);
            background: rgba(211, 227, 253, 0.92);
            font-size: 13px;
            font-weight: 650;
        }

        .page-grid {
            display: grid;
            grid-template-columns: 1.15fr 0.85fr;
            gap: 22px;
            margin-top: 22px;
        }

        .section {
            margin-top: 22px;
        }

        .section-title {
            margin: 0 0 14px;
            font-size: 22px;
            line-height: 1.2;
            letter-spacing: -0.02em;
        }

        .card {
            border-radius: var(--md-sys-shape-lg);
            padding: 22px;
            background: rgba(255, 255, 255, 0.76);
            border: 1px solid rgba(196, 199, 197, 0.70);
            box-shadow: var(--md-sys-elevation-1);
        }

        .tonal-card {
            background: var(--md-sys-color-primary-container);
            color: var(--md-sys-color-on-primary-container);
            border: 0;
        }

        .score-grid {
            display: grid;
            grid-template-columns: repeat(4, minmax(0, 1fr));
            gap: 14px;
        }

        .score-card {
            border-radius: var(--md-sys-shape-md);
            padding: 18px;
            background: var(--md-sys-color-surface-container-low);
            border: 1px solid var(--md-sys-color-outline-variant);
        }

        .score-card h3,
        .info-card h4 {
            margin: 0 0 10px;
            color: var(--md-sys-color-on-surface-variant);
            font-size: 13px;
            font-weight: 700;
            letter-spacing: 0.02em;
        }

        .score-value {
            font-size: 38px;
            font-weight: 800;
            letter-spacing: -0.05em;
        }

        .score-grade,
        .status-badge {
            display: inline-flex;
            align-items: center;
            min-height: 28px;
            padding: 5px 12px;
            border-radius: 999px;
            font-size: 12px;
            font-weight: 700;
        }

        .grade-excellent,
        .status-success {
            color: #0d3b1e;
            background: #c4e7c6;
        }

        .grade-good,
        .status-degraded {
            color: #003355;
            background: #c2e7ff;
        }

        .grade-fair,
        .status-skipped {
            color: #4a3000;
            background: #fdd663;
        }

        .grade-poor,
        .status-failed {
            color: #601410;
            background: #f9dedc;
        }

        .meta-grid,
        .info-grid {
            display: grid;
            grid-template-columns: repeat(2, minmax(0, 1fr));
            gap: 14px;
        }

        .meta-item,
        .info-item {
            display: flex;
            justify-content: space-between;
            gap: 12px;
            padding: 11px 0;
            border-bottom: 1px solid var(--md-sys-color-outline-variant);
        }

        .meta-item:last-child,
        .info-item:last-child {
            border-bottom: 0;
        }

        .info-label {
            color: var(--md-sys-color-on-surface-variant);
            font-size: 13px;
        }

        .info-value {
            min-width: 0;
            color: var(--md-sys-color-on-surface);
            font-size: 13px;
            font-weight: 650;
            text-align: right;
            overflow-wrap: anywhere;
        }

        .score-row {
            margin-bottom: 18px;
        }

        .score-row-head {
            display: flex;
            justify-content: space-between;
            gap: 12px;
            margin-bottom: 8px;
            font-size: 14px;
            font-weight: 650;
        }

        .progress-bar {
            height: 12px;
            overflow: hidden;
            border-radius: 999px;
            background: var(--md-sys-color-surface-container-high);
        }

        .progress-fill {
            height: 100%;
            min-width: 4px;
            border-radius: inherit;
            background: linear-gradient(90deg, var(--md-sys-color-primary), #34a853);
        }

        .quality-notes {
            background: #fff7df;
            border-color: #fdd663;
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
            border-radius: var(--md-sys-shape-md);
            padding: 16px;
            color: #1f1f1f;
            background: var(--md-sys-color-surface-container);
            border: 1px solid var(--md-sys-color-outline-variant);
            font: 13px/1.65 "Roboto Mono", "SFMono-Regular", Consolas, monospace;
        }

        .test-table {
            width: 100%;
            border-collapse: collapse;
            overflow: hidden;
            border-radius: var(--md-sys-shape-md);
        }

        .test-table th,
        .test-table td {
            padding: 15px 14px;
            text-align: left;
            border-bottom: 1px solid var(--md-sys-color-outline-variant);
            vertical-align: top;
        }

        .test-table th {
            color: var(--md-sys-color-on-surface-variant);
            background: var(--md-sys-color-surface-container);
            font-size: 12px;
            font-weight: 800;
            letter-spacing: 0.04em;
        }

        .test-table td {
            font-size: 14px;
            line-height: 1.55;
        }

        .footer {
            margin-top: 24px;
            padding: 22px;
            text-align: center;
            color: var(--md-sys-color-on-surface-variant);
            font-size: 13px;
        }

        @media (max-width: 920px) {
            .hero-content,
            .page-grid {
                grid-template-columns: 1fr;
            }

            .hero-score {
                width: 100%;
                min-height: 132px;
            }

            .score-grid {
                grid-template-columns: repeat(2, minmax(0, 1fr));
            }
        }

        @media (max-width: 640px) {
            .app-shell {
                width: min(100% - 20px, 1240px);
                padding-top: 12px;
            }

            .hero,
            .card {
                padding: 18px;
                border-radius: 22px;
            }

            .score-grid,
            .meta-grid,
            .info-grid {
                grid-template-columns: 1fr;
            }

            .test-table {
                display: block;
                overflow-x: auto;
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
                    <p>生成时间 {{.Timestamp}}。报告聚合系统信息、性能评分、测试置信度、评分校准版本和可复制分享模板。</p>
                    <div class="chip-row">
                        <span class="chip">评分基准 {{.ScoreProfile}}</span>
                        <span class="chip">评测档位 {{.BenchmarkProfileName}}</span>
                        <span class="chip">置信度 {{.ConfidenceLevel}}</span>
                        <span class="chip">校准 {{.CalibrationVersion}}</span>
                    </div>
                </div>
                <div class="hero-score">
                    <div class="score-number">{{printf "%.0f" .OverallScore.TotalScore}}</div>
                    <div class="score-label">总体评分 / 100</div>
                    <div style="margin-top: 14px;"><span class="score-grade {{.OverallGradeClass}}">{{.OverallScore.Grade}}</span></div>
                </div>
            </div>
        </section>

        <section class="section">
            <div class="score-grid">
                <div class="score-card">
                    <h3>CPU</h3>
                    <div class="score-value" style="color: #0b57d0;">{{printf "%.0f" .OverallScore.CPUScore}}</div>
                </div>
                <div class="score-card">
                    <h3>内存</h3>
                    <div class="score-value" style="color: #146c2e;">{{printf "%.0f" .OverallScore.MemoryScore}}</div>
                </div>
                <div class="score-card">
                    <h3>磁盘</h3>
                    <div class="score-value" style="color: #b06000;">{{printf "%.0f" .OverallScore.DiskScore}}</div>
                </div>
                <div class="score-card">
                    <h3>网络</h3>
                    <div class="score-value" style="color: #00639b;">{{printf "%.0f" .OverallScore.NetworkScore}}</div>
                </div>
            </div>
        </section>

        <section class="page-grid">
            <div class="card">
                <h2 class="section-title">性能评分详情</h2>
                <div class="score-row">
                    <div class="score-row-head"><span>CPU 性能</span><span>{{printf "%.1f" .OverallScore.CPUScore}} / 100</span></div>
                    <div class="progress-bar"><div class="progress-fill" style="width: {{.OverallScore.CPUScore}}%;"></div></div>
                </div>
                <div class="score-row">
                    <div class="score-row-head"><span>内存性能</span><span>{{printf "%.1f" .OverallScore.MemoryScore}} / 100</span></div>
                    <div class="progress-bar"><div class="progress-fill" style="width: {{.OverallScore.MemoryScore}}%;"></div></div>
                </div>
                <div class="score-row">
                    <div class="score-row-head"><span>磁盘性能</span><span>{{printf "%.1f" .OverallScore.DiskScore}} / 100</span></div>
                    <div class="progress-bar"><div class="progress-fill" style="width: {{.OverallScore.DiskScore}}%;"></div></div>
                </div>
                <div class="score-row">
                    <div class="score-row-head"><span>网络性能</span><span>{{printf "%.1f" .OverallScore.NetworkScore}} / 100</span></div>
                    <div class="progress-bar"><div class="progress-fill" style="width: {{.OverallScore.NetworkScore}}%;"></div></div>
                </div>
            </div>

            <div class="card tonal-card">
                <h2 class="section-title">报告摘要</h2>
                <div class="meta-grid">
                    <div class="meta-item"><span class="info-label">评分基准</span><span class="info-value">{{.ScoreProfile}}</span></div>
                    <div class="meta-item"><span class="info-label">评测档位</span><span class="info-value">{{.BenchmarkProfileName}}</span></div>
                    <div class="meta-item"><span class="info-label">置信等级</span><span class="info-value">{{.ConfidenceLevel}}</span></div>
                    <div class="meta-item"><span class="info-label">校准版本</span><span class="info-value">{{.CalibrationVersion}}</span></div>
                </div>
            </div>
        </section>

        {{if .QualityNotes}}
        <section class="section card quality-notes">
            <h2 class="section-title">质量提示</h2>
            <ul>
                {{range .QualityNotes}}
                <li>{{.}}</li>
                {{end}}
            </ul>
        </section>
        {{end}}

        <section class="section card">
            <h2 class="section-title">系统信息</h2>
            <div class="info-grid">
                {{if .SystemInfo.CPU}}
                <div class="score-card info-card">
                    <h4>处理器</h4>
                    <div class="info-item"><span class="info-label">型号</span><span class="info-value">{{.SystemInfo.CPU.Model}}</span></div>
                    <div class="info-item"><span class="info-label">核心数</span><span class="info-value">{{.SystemInfo.CPU.Cores}} 核心</span></div>
                    <div class="info-item"><span class="info-label">线程数</span><span class="info-value">{{.SystemInfo.CPU.Threads}} 线程</span></div>
                    <div class="info-item"><span class="info-label">频率</span><span class="info-value">{{printf "%.0f" .SystemInfo.CPU.FrequencyMHz}} MHz</span></div>
                </div>
                {{end}}
                {{if .SystemInfo.Memory}}
                <div class="score-card info-card">
                    <h4>内存</h4>
                    <div class="info-item"><span class="info-label">总容量</span><span class="info-value">{{.SystemInfo.Memory.TotalMB}} MB</span></div>
                    <div class="info-item"><span class="info-label">可用容量</span><span class="info-value">{{.SystemInfo.Memory.AvailableMB}} MB</span></div>
                    {{if .SystemInfo.Memory.MemoryType}}<div class="info-item"><span class="info-label">类型</span><span class="info-value">{{.SystemInfo.Memory.MemoryType}}</span></div>{{end}}
                </div>
                {{end}}
                {{if .SystemInfo.Disk}}
                <div class="score-card info-card">
                    <h4>磁盘</h4>
                    <div class="info-item"><span class="info-label">总容量</span><span class="info-value">{{printf "%.2f" .SystemInfo.Disk.TotalGB}} GB</span></div>
                    <div class="info-item"><span class="info-label">可用空间</span><span class="info-value">{{printf "%.2f" .SystemInfo.Disk.AvailableGB}} GB</span></div>
                    {{if .SystemInfo.Disk.DiskType}}<div class="info-item"><span class="info-label">类型</span><span class="info-value">{{.SystemInfo.Disk.DiskType}}</span></div>{{end}}
                </div>
                {{end}}
                {{if .SystemInfo.OS}}
                <div class="score-card info-card">
                    <h4>操作系统</h4>
                    <div class="info-item"><span class="info-label">系统</span><span class="info-value">{{.SystemInfo.OS.Name}}</span></div>
                    <div class="info-item"><span class="info-label">版本</span><span class="info-value">{{.SystemInfo.OS.Version}}</span></div>
                    <div class="info-item"><span class="info-label">架构</span><span class="info-value">{{.SystemInfo.OS.Architecture}}</span></div>
                </div>
                {{end}}
            </div>
        </section>

        {{if .SharePlainText}}
        <section class="section card">
            <h2 class="section-title">分享模板</h2>
            <div class="share-box">{{.SharePlainText}}</div>
        </section>
        {{end}}

        <section class="section card">
            <h2 class="section-title">测试结果详情</h2>
            <table class="test-table">
                <thead>
                    <tr>
                        <th>测试项目</th>
                        <th>状态</th>
                        <th>耗时</th>
                        <th>关键指标</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .TestResultsList}}
                    <tr>
                        <td>{{.TestName}}</td>
                        <td><span class="status-badge status-{{.Status}}">{{.StatusText}}</span></td>
                        <td>{{printf "%.2f" .DurationSeconds}} 秒</td>
                        <td>{{.KeyMetrics}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </section>

        <footer class="footer">
            <p>Perfassess Web 报告</p>
        </footer>
    </main>
</body>
</html>`
