// Package reporter 提供 Web 服务器功能
package reporter

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
)

// WebServer Web 服务器
type WebServer struct {
	port   int
	logger *logger.Logger
	server *http.Server
	report *models.Report
}

// NewWebServer 创建新的 Web 服务器
func NewWebServer(port int, logger *logger.Logger) *WebServer {
	return &WebServer{
		port:   port,
		logger: logger,
	}
}

// Start 启动 Web 服务器
func (ws *WebServer) Start(report *models.Report) error {
	ws.report = report

	// 创建 HTTP 处理器
	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.handleReport)

	// 创建服务器
	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ws.port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		ws.logger.Info(fmt.Sprintf("Web 服务器启动在端口 %d", ws.port))
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ws.logger.Error("Web 服务器错误", err)
		}
	}()

	// 打印访问地址
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 报告已生成")
	fmt.Printf("🌐 Web 服务器已启动: http://localhost:%d\n", ws.port)
	fmt.Println("💡 在浏览器中打开上述地址查看报告")
	fmt.Println("⏹  按 Ctrl+C 停止服务器")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 等待中断信号
	ws.waitForShutdown()

	return nil
}

// handleReport 处理报告请求
func (ws *WebServer) handleReport(w http.ResponseWriter, r *http.Request) {
	// 解析模板
	tmpl, err := template.New("report").Parse(HTMLTemplate)
	if err != nil {
		http.Error(w, "模板解析错误", http.StatusInternalServerError)
		ws.logger.Error("模板解析错误", err)
		return
	}

	// 准备模板数据
	data := ws.prepareTemplateData()

	// 渲染模板
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "模板渲染错误", http.StatusInternalServerError)
		ws.logger.Error("模板渲染错误", err)
		return
	}
}

// prepareTemplateData 准备模板数据
func (ws *WebServer) prepareTemplateData() map[string]interface{} {
	data := make(map[string]interface{})

	// 基本信息
	data["SessionID"] = ws.report.SessionID
	data["Timestamp"] = ws.report.Timestamp.Format("2006-01-02 15:04:05")
	data["SystemInfo"] = ws.report.SystemInfo

	// 获取综合评分
	overallScore := &models.OverallScore{}
	if score, ok := ws.report.Summary["overall_score"].(*models.OverallScore); ok {
		overallScore = score
	}
	data["OverallScore"] = overallScore

	// 评分颜色
	data["OverallScoreColor"] = ws.getScoreColor(overallScore.TotalScore)
	data["OverallGradeClass"] = ws.getGradeClass(overallScore.Grade)
	data["QualityNotes"] = ws.getQualityNotes()
	data["ScoreProfile"] = ws.summaryString("score_profile")
	data["BenchmarkProfileName"] = ws.benchmarkProfileName()
	data["ConfidenceLevel"] = ws.confidenceLevel()
	data["CalibrationVersion"] = ws.calibrationVersion()
	data["SharePlainText"] = ws.sharePlainText()

	// 测试结果列表
	testResultsList := []map[string]interface{}{}

	if ws.report.TestResults != nil {
		if ws.report.TestResults.CPUResult != nil {
			testResultsList = append(testResultsList, ws.formatTestResult("CPU性能测试", ws.report.TestResults.CPUResult))
		}
		if ws.report.TestResults.MemoryResult != nil {
			testResultsList = append(testResultsList, ws.formatTestResult("内存性能测试", ws.report.TestResults.MemoryResult))
		}
		if ws.report.TestResults.DiskResult != nil {
			testResultsList = append(testResultsList, ws.formatTestResult("磁盘性能测试", ws.report.TestResults.DiskResult))
		}
		if ws.report.TestResults.NetworkResult != nil {
			testResultsList = append(testResultsList, ws.formatTestResult("网络性能测试", ws.report.TestResults.NetworkResult))
		}
	}

	data["TestResultsList"] = testResultsList

	return data
}

func (ws *WebServer) summaryString(key string) string {
	if ws.report == nil || ws.report.Summary == nil {
		return "-"
	}
	if value, ok := ws.report.Summary[key].(string); ok && value != "" {
		return value
	}
	return "-"
}

func (ws *WebServer) benchmarkProfileName() string {
	if ws.report == nil || ws.report.Summary == nil {
		return "-"
	}
	if profile, ok := ws.report.Summary["benchmark_profile"].(map[string]interface{}); ok {
		if name, ok := profile["name"].(string); ok && name != "" {
			return name
		}
	}
	return "-"
}

func (ws *WebServer) confidenceLevel() string {
	if ws.report == nil || ws.report.Summary == nil {
		return "-"
	}
	if confidence, ok := ws.report.Summary["confidence_level"].(map[string]interface{}); ok {
		if level, ok := confidence["level"].(string); ok && level != "" {
			return level
		}
	}
	return "-"
}

func (ws *WebServer) calibrationVersion() string {
	if ws.report == nil || ws.report.Summary == nil {
		return "-"
	}
	if calibration, ok := ws.report.Summary["score_calibration"].(map[string]interface{}); ok {
		if version, ok := calibration["version"].(string); ok && version != "" {
			return version
		}
	}
	return "-"
}

func (ws *WebServer) sharePlainText() string {
	if ws.report == nil || ws.report.Summary == nil {
		return ""
	}
	if templates, ok := ws.report.Summary["share_templates"].(map[string]interface{}); ok {
		if plain, ok := templates["plain_text"].(string); ok {
			return plain
		}
	}
	return ""
}

func (ws *WebServer) getQualityNotes() []string {
	if ws.report == nil || ws.report.Summary == nil {
		return nil
	}
	switch notes := ws.report.Summary["quality_notes"].(type) {
	case []string:
		return notes
	case []interface{}:
		result := make([]string, 0, len(notes))
		for _, note := range notes {
			if text, ok := note.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

// formatTestResult 格式化测试结果
func (ws *WebServer) formatTestResult(name string, result *models.TestResult) map[string]interface{} {
	formatted := make(map[string]interface{})

	formatted["TestName"] = name
	formatted["Status"] = result.Status
	formatted["StatusText"] = ws.getStatusText(result.Status)
	formatted["DurationSeconds"] = result.DurationSeconds
	formatted["KeyMetrics"] = ws.getKeyMetrics(result)

	return formatted
}

// getStatusText 获取状态文本
func (ws *WebServer) getStatusText(status string) string {
	switch status {
	case "success":
		return "成功"
	case "failed":
		return "失败"
	case "skipped":
		return "跳过"
	case "degraded":
		return "降级"
	default:
		return "未知"
	}
}

// getKeyMetrics 获取关键指标
func (ws *WebServer) getKeyMetrics(result *models.TestResult) string {
	if result.Status != "success" || result.Metrics == nil {
		return "-"
	}

	switch result.TestName {
	case "cpu", "CPU性能测试":
		if score, ok := getCPUScore(result); ok {
			prefix := ""
			if backend := getCPUBackend(result); backend != "" {
				prefix = backend + " | "
			}
			if events, ok := getCPUMultiCoreEvents(result); ok {
				return fmt.Sprintf("%s总分: %.2f | 多核 %.2f events/s", prefix, score, events)
			}
			if singleRaw, singleOK := getCPUSingleCoreRawScore(result); singleOK {
				if multiRaw, multiOK := getCPUMultiCoreRawScore(result); multiOK {
					return fmt.Sprintf("%s总分: %.2f | Geekbench %.0f / %.0f", prefix, score, singleRaw, multiRaw)
				}
			}
			if stddev, ok := getCPUScoreStdDev(result); ok {
				return fmt.Sprintf("%s总分: %.2f (stddev %.2f)", prefix, score, stddev)
			}
			return fmt.Sprintf("%s总分: %.2f", prefix, score)
		}
	case "memory", "内存性能测试":
		readSpeed, readOK := getMemoryReadSpeed(result)
		writeSpeed, writeOK := getMemoryWriteSpeed(result)
		prefix := ""
		if backend := getMemoryBackend(result); backend != "" {
			prefix = backend + " | "
		}
		if readOK && writeOK {
			readStdDev, readStdOK := getMemoryReadStdDev(result)
			writeStdDev, writeStdOK := getMemoryWriteStdDev(result)
			if readStdOK && writeStdOK {
				return fmt.Sprintf("%s读/写: %.2f / %.2f MB/s (stddev %.2f / %.2f)", prefix, readSpeed, writeSpeed, readStdDev, writeStdDev)
			}
			return fmt.Sprintf("%s读/写: %.2f / %.2f MB/s", prefix, readSpeed, writeSpeed)
		}
		if readOK {
			return fmt.Sprintf("%s读取: %.2f MB/s", prefix, readSpeed)
		}
	case "disk", "磁盘性能测试":
		if rows := getDiskFioMixedRows(result); len(rows) > 0 {
			prefix := ""
			if backend := getDiskBackend(result); backend != "" {
				prefix = backend + " | "
			}
			first := rows[0]
			last := rows[len(rows)-1]
			return fmt.Sprintf("%sfio mixed: %s %.0f IOPS | %s %.2f MB/s", prefix, first.BlockSize, first.TotalIOPS, last.BlockSize, last.TotalMBps)
		}
		if speed, ok := getDiskReadSpeed(result); ok {
			if backend := getDiskBackend(result); backend != "" {
				if readIOPS, ok := getDiskRandomReadIOPS(result); ok {
					return fmt.Sprintf("%s | 顺序读: %.2f MB/s | 随机读 %.0f IOPS", backend, speed, readIOPS)
				}
				return fmt.Sprintf("%s | 顺序读: %.2f MB/s", backend, speed)
			}
			return fmt.Sprintf("顺序读: %.2f MB/s", speed)
		}
	case "network", "网络性能测试":
		if message := getNetworkError(result); message != "" {
			return message
		}
		backend := getNetworkBackend(result)
		server := getNetworkBackendServer(result)
		if rows := getNetworkIperf3MatrixRows(result); len(rows) > 0 {
			download, _ := metricFloat64(result.Metrics, "iperf3_matrix_avg_download_mbps")
			upload, _ := metricFloat64(result.Metrics, "iperf3_matrix_avg_upload_mbps")
			prefix := ""
			if backend != "" {
				prefix = backend + " | "
			}
			return fmt.Sprintf("%siperf3 matrix: %d nodes | 下载 %.2f Mbps | 上传 %.2f Mbps", prefix, len(rows), download, upload)
		}
		if node, ok := getNetworkSpeedtestNode(result); ok {
			download, _ := getNetworkDownloadSpeed(result)
			upload, _ := getNetworkUploadSpeed(result)
			prefix := ""
			if backend != "" {
				prefix = backend + " | "
			}
			location := node.Location
			if node.Country != "" {
				if location != "" {
					location += ", "
				}
				location += node.Country
			}
			if location == "" {
				location = "-"
			}
			return fmt.Sprintf("%sspeedtest: %s | %s | 下载 %.2f Mbps | 上传 %.2f Mbps", prefix, networkSpeedtestServerLabel(node), location, download, upload)
		}
		if rows := getNetworkQualityRows(result); len(rows) > 0 {
			avgLatency, _ := metricFloat64(result.Metrics, "network_quality_avg_latency_ms")
			jitter, _ := metricFloat64(result.Metrics, "network_quality_jitter_ms")
			ipv4Available, _ := metricBool(result.Metrics, "network_quality_ipv4_available")
			ipv6Available, _ := metricBool(result.Metrics, "network_quality_ipv6_available")
			prefix := ""
			if backend != "" {
				prefix = backend + " | "
			}
			return fmt.Sprintf("%s网络质量: IPv4 %s | IPv6 %s | 延迟 %.2f ms | 抖动 %.2f ms",
				prefix, availabilityLabel(ipv4Available), availabilityLabel(ipv6Available), avgLatency, jitter)
		}
		if latency, ok := getNetworkLatency(result); ok {
			prefix := ""
			if backend != "" {
				prefix = backend + " | "
			}
			if server != "" {
				prefix = prefix + server + " | "
			}
			if isNetworkUploadEstimated(result) {
				return fmt.Sprintf("%s延迟: %.2f ms | 上传: 估算值", prefix, latency)
			}
			return fmt.Sprintf("%s延迟: %.2f ms", prefix, latency)
		}
	}

	return "-"
}

func availabilityLabel(available bool) string {
	if available {
		return "可用"
	}
	return "不可用"
}

// getScoreColor 获取评分颜色
func (ws *WebServer) getScoreColor(score float64) string {
	if score >= 90 {
		return "#28a745"
	} else if score >= 75 {
		return "#17a2b8"
	} else if score >= 60 {
		return "#ffc107"
	}
	return "#dc3545"
}

// getGradeClass 获取等级样式类
func (ws *WebServer) getGradeClass(grade string) string {
	switch grade {
	case "优秀":
		return "grade-excellent"
	case "良好":
		return "grade-good"
	case "一般":
		return "grade-fair"
	case "较差":
		return "grade-poor"
	default:
		return "grade-fair"
	}
}

// waitForShutdown 等待关闭信号
func (ws *WebServer) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n正在关闭 Web 服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ws.server.Shutdown(ctx); err != nil {
		ws.logger.Error("Web 服务器关闭错误", err)
	}

	fmt.Println("Web 服务器已关闭")
}
