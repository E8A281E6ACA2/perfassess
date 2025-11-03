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
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
            min-height: 100vh;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 32px;
            margin-bottom: 10px;
        }
        
        .header p {
            opacity: 0.9;
            font-size: 14px;
        }
        
        .content {
            padding: 40px;
        }
        
        .score-dashboard {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        
        .score-card {
            background: #f8f9fa;
            border-radius: 12px;
            padding: 24px;
            text-align: center;
            transition: transform 0.2s;
        }
        
        .score-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 16px rgba(0,0,0,0.1);
        }
        
        .score-card h3 {
            font-size: 14px;
            color: #6c757d;
            margin-bottom: 12px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        
        .score-value {
            font-size: 48px;
            font-weight: bold;
            margin-bottom: 8px;
        }
        
        .score-grade {
            display: inline-block;
            padding: 6px 16px;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 600;
        }
        
        .grade-excellent { background: #d4edda; color: #155724; }
        .grade-good { background: #d1ecf1; color: #0c5460; }
        .grade-fair { background: #fff3cd; color: #856404; }
        .grade-poor { background: #f8d7da; color: #721c24; }
        
        .section {
            margin-bottom: 40px;
        }
        
        .section-title {
            font-size: 24px;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid #667eea;
            color: #333;
        }
        
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .info-card {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
        }
        
        .info-card h4 {
            color: #667eea;
            margin-bottom: 12px;
            font-size: 16px;
        }
        
        .info-item {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #e9ecef;
        }
        
        .info-item:last-child {
            border-bottom: none;
        }
        
        .info-label {
            color: #6c757d;
            font-size: 14px;
        }
        
        .info-value {
            color: #333;
            font-weight: 500;
            font-size: 14px;
        }
        
        .progress-bar {
            background: #e9ecef;
            border-radius: 10px;
            height: 20px;
            overflow: hidden;
            margin: 10px 0;
        }
        
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            transition: width 1s ease;
            display: flex;
            align-items: center;
            justify-content: flex-end;
            padding-right: 10px;
            color: white;
            font-size: 12px;
            font-weight: bold;
        }
        
        .test-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        
        .test-table th,
        .test-table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #e9ecef;
        }
        
        .test-table th {
            background: #f8f9fa;
            color: #495057;
            font-weight: 600;
            font-size: 14px;
        }
        
        .test-table td {
            font-size: 14px;
        }
        
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
        }
        
        .status-success { background: #d4edda; color: #155724; }
        .status-failed { background: #f8d7da; color: #721c24; }
        .status-skipped { background: #fff3cd; color: #856404; }
        
        .footer {
            text-align: center;
            padding: 20px;
            color: #6c757d;
            font-size: 14px;
            border-top: 1px solid #e9ecef;
        }
        
        @media (max-width: 768px) {
            .score-dashboard {
                grid-template-columns: 1fr;
            }
            
            .info-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 性能评估报告</h1>
            <p>会话ID: {{.SessionID}} | 生成时间: {{.Timestamp}}</p>
        </div>
        
        <div class="content">
            <!-- 评分仪表盘 -->
            <div class="score-dashboard">
                <div class="score-card">
                    <h3>总体评分</h3>
                    <div class="score-value" style="color: {{.OverallScoreColor}};">{{printf "%.0f" .OverallScore.TotalScore}}</div>
                    <span class="score-grade {{.OverallGradeClass}}">{{.OverallScore.Grade}}</span>
                </div>
                
                <div class="score-card">
                    <h3>CPU 评分</h3>
                    <div class="score-value" style="color: #667eea;">{{printf "%.0f" .OverallScore.CPUScore}}</div>
                </div>
                
                <div class="score-card">
                    <h3>内存评分</h3>
                    <div class="score-value" style="color: #764ba2;">{{printf "%.0f" .OverallScore.MemoryScore}}</div>
                </div>
                
                <div class="score-card">
                    <h3>磁盘评分</h3>
                    <div class="score-value" style="color: #f093fb;">{{printf "%.0f" .OverallScore.DiskScore}}</div>
                </div>
                
                <div class="score-card">
                    <h3>网络评分</h3>
                    <div class="score-value" style="color: #4facfe;">{{printf "%.0f" .OverallScore.NetworkScore}}</div>
                </div>
            </div>
            
            <!-- 系统信息 -->
            <div class="section">
                <h2 class="section-title">💻 系统信息</h2>
                <div class="info-grid">
                    {{if .SystemInfo.CPU}}
                    <div class="info-card">
                        <h4>处理器</h4>
                        <div class="info-item">
                            <span class="info-label">型号</span>
                            <span class="info-value">{{.SystemInfo.CPU.Model}}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">核心数</span>
                            <span class="info-value">{{.SystemInfo.CPU.Cores}} 核心</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">线程数</span>
                            <span class="info-value">{{.SystemInfo.CPU.Threads}} 线程</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">频率</span>
                            <span class="info-value">{{printf "%.0f" .SystemInfo.CPU.FrequencyMHz}} MHz</span>
                        </div>
                    </div>
                    {{end}}
                    
                    {{if .SystemInfo.Memory}}
                    <div class="info-card">
                        <h4>内存</h4>
                        <div class="info-item">
                            <span class="info-label">总容量</span>
                            <span class="info-value">{{.SystemInfo.Memory.TotalMB}} MB</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">可用容量</span>
                            <span class="info-value">{{.SystemInfo.Memory.AvailableMB}} MB</span>
                        </div>
                        {{if .SystemInfo.Memory.MemoryType}}
                        <div class="info-item">
                            <span class="info-label">类型</span>
                            <span class="info-value">{{.SystemInfo.Memory.MemoryType}}</span>
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                    
                    {{if .SystemInfo.Disk}}
                    <div class="info-card">
                        <h4>磁盘</h4>
                        <div class="info-item">
                            <span class="info-label">总容量</span>
                            <span class="info-value">{{printf "%.2f" .SystemInfo.Disk.TotalGB}} GB</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">可用空间</span>
                            <span class="info-value">{{printf "%.2f" .SystemInfo.Disk.AvailableGB}} GB</span>
                        </div>
                        {{if .SystemInfo.Disk.DiskType}}
                        <div class="info-item">
                            <span class="info-label">类型</span>
                            <span class="info-value">{{.SystemInfo.Disk.DiskType}}</span>
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                    
                    {{if .SystemInfo.OS}}
                    <div class="info-card">
                        <h4>操作系统</h4>
                        <div class="info-item">
                            <span class="info-label">系统</span>
                            <span class="info-value">{{.SystemInfo.OS.Name}}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">版本</span>
                            <span class="info-value">{{.SystemInfo.OS.Version}}</span>
                        </div>
                        <div class="info-item">
                            <span class="info-label">架构</span>
                            <span class="info-value">{{.SystemInfo.OS.Architecture}}</span>
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            
            <!-- 性能评分详情 -->
            <div class="section">
                <h2 class="section-title">📊 性能评分详情</h2>
                
                <div style="margin-bottom: 20px;">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 5px;">
                        <span>CPU 性能</span>
                        <span>{{printf "%.1f" .OverallScore.CPUScore}} / 100</span>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: {{.OverallScore.CPUScore}}%;"></div>
                    </div>
                </div>
                
                <div style="margin-bottom: 20px;">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 5px;">
                        <span>内存性能</span>
                        <span>{{printf "%.1f" .OverallScore.MemoryScore}} / 100</span>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: {{.OverallScore.MemoryScore}}%;"></div>
                    </div>
                </div>
                
                <div style="margin-bottom: 20px;">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 5px;">
                        <span>磁盘性能</span>
                        <span>{{printf "%.1f" .OverallScore.DiskScore}} / 100</span>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: {{.OverallScore.DiskScore}}%;"></div>
                    </div>
                </div>
                
                <div style="margin-bottom: 20px;">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 5px;">
                        <span>网络性能</span>
                        <span>{{printf "%.1f" .OverallScore.NetworkScore}} / 100</span>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: {{.OverallScore.NetworkScore}}%;"></div>
                    </div>
                </div>
            </div>
            
            <!-- 测试结果 -->
            <div class="section">
                <h2 class="section-title">📝 测试结果详情</h2>
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
            </div>
        </div>
        
        <div class="footer">
            <p>高性能多终端自动化性能评估系统 v1.0</p>
            <p>© 2025 Performance Assessment System</p>
        </div>
    </div>
</body>
</html>`
