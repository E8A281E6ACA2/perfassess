// Package reporter 提供 Web 服务器功能
package reporter

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

// WebServer Web 服务器
type WebServer struct {
	port   int
	logger *logger.Logger
	server *http.Server
	report *models.Report
}

type webMetricCard struct {
	Label string
	Value string
	Unit  string
	Tone  string
}

type webDetailRow struct {
	Label string
	Value string
}

type webTable struct {
	Title   string
	Headers []string
	Rows    [][]string
}

type webReportSection struct {
	ID         string
	Title      string
	Subtitle   string
	Status     string
	StatusText string
	Summary    string
	Hint       string
	Metrics    []webMetricCard
	Details    []webDetailRow
	Tables     []webTable
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
	data["ReportSections"] = ws.buildReportSections(overallScore)

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

func (ws *WebServer) buildReportSections(overallScore *models.OverallScore) []webReportSection {
	sections := []webReportSection{
		ws.overviewSection(overallScore),
		ws.systemSection(),
	}

	var cpuResult, memoryResult, diskResult, networkResult *models.TestResult
	if ws.report != nil && ws.report.TestResults != nil {
		cpuResult = ws.report.TestResults.CPUResult
		memoryResult = ws.report.TestResults.MemoryResult
		diskResult = ws.report.TestResults.DiskResult
		networkResult = ws.report.TestResults.NetworkResult
	}

	sections = append(sections,
		ws.cpuSection(cpuResult, overallScore),
		ws.memorySection(memoryResult, overallScore),
		ws.diskSection(diskResult, overallScore),
		ws.networkSection(networkResult, overallScore),
		ws.routeSection(),
		ws.streamingSection(),
		ws.aiSection(),
		ws.ipQualitySection(),
		ws.stressSection(),
		ws.securitySection(),
	)

	return sections
}

func (ws *WebServer) overviewSection(overallScore *models.OverallScore) webReportSection {
	if overallScore == nil {
		overallScore = &models.OverallScore{}
	}
	return webReportSection{
		ID:         "overview",
		Title:      "总览",
		Subtitle:   "本次服务器测评的综合摘要。",
		Status:     "success",
		StatusText: "已生成",
		Summary:    fmt.Sprintf("总分 %.2f / 100，等级 %s，置信度 %s。", overallScore.TotalScore, overallScore.Grade, ws.confidenceLevel()),
		Metrics: []webMetricCard{
			{Label: "总体评分", Value: fmt.Sprintf("%.0f", overallScore.TotalScore), Unit: "/ 100", Tone: "primary"},
			{Label: "CPU", Value: fmt.Sprintf("%.0f", overallScore.CPUScore), Unit: "/ 100", Tone: "blue"},
			{Label: "内存", Value: fmt.Sprintf("%.0f", overallScore.MemoryScore), Unit: "/ 100", Tone: "green"},
			{Label: "磁盘", Value: fmt.Sprintf("%.0f", overallScore.DiskScore), Unit: "/ 100", Tone: "amber"},
			{Label: "网络", Value: fmt.Sprintf("%.0f", overallScore.NetworkScore), Unit: "/ 100", Tone: "cyan"},
		},
		Details: []webDetailRow{
			{Label: "评分基准", Value: ws.summaryString("score_profile")},
			{Label: "评测档位", Value: ws.benchmarkProfileName()},
			{Label: "校准版本", Value: ws.calibrationVersion()},
			{Label: "会话 ID", Value: ws.report.SessionID},
		},
	}
}

func (ws *WebServer) systemSection() webReportSection {
	section := webReportSection{
		ID:         "system",
		Title:      "系统信息",
		Subtitle:   "硬件、系统、虚拟化和公网 IP 信息。",
		Status:     "success",
		StatusText: "已采集",
		Summary:    "系统基础信息已采集完成。",
	}
	if ws.report == nil || ws.report.SystemInfo == nil {
		section.Status = "skipped"
		section.StatusText = "无数据"
		section.Hint = "本次报告没有系统信息。"
		return section
	}
	sys := ws.report.SystemInfo
	if sys.CPU != nil {
		section.Details = append(section.Details,
			webDetailRow{Label: "CPU 型号", Value: sys.CPU.Model},
			webDetailRow{Label: "CPU 核心/线程", Value: fmt.Sprintf("%d 核 / %d 线程", sys.CPU.Cores, sys.CPU.Threads)},
			webDetailRow{Label: "CPU 频率", Value: fmt.Sprintf("%.0f MHz", sys.CPU.FrequencyMHz)},
		)
	}
	if sys.Memory != nil {
		section.Details = append(section.Details,
			webDetailRow{Label: "内存总量", Value: fmt.Sprintf("%d MB", sys.Memory.TotalMB)},
			webDetailRow{Label: "内存可用", Value: fmt.Sprintf("%d MB", sys.Memory.AvailableMB)},
		)
		if sys.Memory.MemoryType != "" {
			section.Details = append(section.Details, webDetailRow{Label: "内存类型", Value: sys.Memory.MemoryType})
		}
	}
	if sys.Disk != nil {
		section.Details = append(section.Details,
			webDetailRow{Label: "磁盘总量", Value: fmt.Sprintf("%.2f GB", sys.Disk.TotalGB)},
			webDetailRow{Label: "磁盘可用", Value: fmt.Sprintf("%.2f GB", sys.Disk.AvailableGB)},
		)
		if sys.Disk.DiskType != "" {
			section.Details = append(section.Details, webDetailRow{Label: "磁盘类型", Value: sys.Disk.DiskType})
		}
	}
	if sys.OS != nil {
		section.Details = append(section.Details,
			webDetailRow{Label: "操作系统", Value: fmt.Sprintf("%s %s", sys.OS.Name, sys.OS.Version)},
			webDetailRow{Label: "系统架构", Value: sys.OS.Architecture},
		)
	}
	if sys.Virtualization != nil {
		section.Details = append(section.Details,
			webDetailRow{Label: "虚拟化", Value: sys.Virtualization.Type},
			webDetailRow{Label: "虚拟化厂商", Value: fallback(sys.Virtualization.Vendor, "-")},
		)
	}
	if sys.IPInfo != nil {
		section.Details = append(section.Details,
			webDetailRow{Label: "公网 IP", Value: sys.IPInfo.PublicIP},
			webDetailRow{Label: "ISP", Value: fallback(sys.IPInfo.ISP, "-")},
		)
		if sys.IPInfo.GeoLocation != nil {
			geo := sys.IPInfo.GeoLocation
			section.Details = append(section.Details, webDetailRow{Label: "位置", Value: fmt.Sprintf("%s, %s", fallback(geo.Country, "-"), fallback(geo.City, "-"))})
		}
	}
	return section
}

func (ws *WebServer) cpuSection(result *models.TestResult, overallScore *models.OverallScore) webReportSection {
	section := ws.testSectionBase("cpu", "CPU 测试", "单核、多核性能和采样稳定性。", result, "本次未执行 CPU 测试。使用 -b cpu 或 -b all 启用。")
	if result == nil || result.Status != models.TestStatusSuccess {
		return section
	}
	section.Metrics = append(section.Metrics, webMetricCard{Label: "CPU 评分", Value: fmt.Sprintf("%.0f", overallScore.CPUScore), Unit: "/ 100", Tone: "blue"})
	if score, ok := getCPUScore(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "测试总分", Value: fmt.Sprintf("%.2f", score)})
	}
	if backend := getCPUBackend(result); backend != "" {
		section.Details = append(section.Details, webDetailRow{Label: "测试后端", Value: backend})
	}
	if events, ok := getCPUSingleCoreEvents(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "单核事件", Value: fmt.Sprintf("%.2f", events), Unit: "events/s", Tone: "primary"})
	}
	if events, ok := getCPUMultiCoreEvents(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "多核事件", Value: fmt.Sprintf("%.2f", events), Unit: "events/s", Tone: "primary"})
	}
	if single, ok := getCPUSingleCoreRawScore(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "单核原始分", Value: fmt.Sprintf("%.0f", single)})
	}
	if multi, ok := getCPUMultiCoreRawScore(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "多核原始分", Value: fmt.Sprintf("%.0f", multi)})
	}
	if stddev, ok := getCPUScoreStdDev(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "采样波动", Value: fmt.Sprintf("%.2f", stddev)})
	}
	return section
}

func (ws *WebServer) memorySection(result *models.TestResult, overallScore *models.OverallScore) webReportSection {
	section := ws.testSectionBase("memory", "内存测试", "内存读写吞吐和采样稳定性。", result, "本次未执行内存测试。使用 -b memory 或 -b all 启用。")
	if result == nil || result.Status != models.TestStatusSuccess {
		return section
	}
	section.Metrics = append(section.Metrics, webMetricCard{Label: "内存评分", Value: fmt.Sprintf("%.0f", overallScore.MemoryScore), Unit: "/ 100", Tone: "green"})
	if read, ok := getMemoryReadSpeed(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "读取速度", Value: fmt.Sprintf("%.2f", read), Unit: "MB/s", Tone: "green"})
	}
	if write, ok := getMemoryWriteSpeed(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "写入速度", Value: fmt.Sprintf("%.2f", write), Unit: "MB/s", Tone: "green"})
	}
	if backend := getMemoryBackend(result); backend != "" {
		section.Details = append(section.Details, webDetailRow{Label: "测试后端", Value: backend})
	}
	if source := getMemoryReadSource(result); source != "" {
		section.Details = append(section.Details, webDetailRow{Label: "读取来源", Value: source})
	}
	if source := getMemoryWriteSource(result); source != "" {
		section.Details = append(section.Details, webDetailRow{Label: "写入来源", Value: source})
	}
	if stddev, ok := getMemoryReadStdDev(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "读取波动", Value: fmt.Sprintf("%.2f MB/s", stddev)})
	}
	if stddev, ok := getMemoryWriteStdDev(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "写入波动", Value: fmt.Sprintf("%.2f MB/s", stddev)})
	}
	return section
}

func (ws *WebServer) diskSection(result *models.TestResult, overallScore *models.OverallScore) webReportSection {
	section := ws.testSectionBase("disk", "磁盘测试", "顺序读写、随机 IOPS 和 fio mixed 矩阵。", result, "本次未执行磁盘测试。使用 -b disk 或 -b all 启用。")
	if result == nil || result.Status != models.TestStatusSuccess {
		return section
	}
	section.Metrics = append(section.Metrics, webMetricCard{Label: "磁盘评分", Value: fmt.Sprintf("%.0f", overallScore.DiskScore), Unit: "/ 100", Tone: "amber"})
	if read, ok := getDiskReadSpeed(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "顺序读取", Value: fmt.Sprintf("%.2f", read), Unit: "MB/s", Tone: "amber"})
	}
	if write, ok := getDiskWriteSpeed(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "顺序写入", Value: fmt.Sprintf("%.2f", write), Unit: "MB/s", Tone: "amber"})
	}
	if iops, ok := getDiskRandomIOPS(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "随机 IOPS", Value: fmt.Sprintf("%d", iops), Unit: "IOPS", Tone: "amber"})
	}
	if backend := getDiskBackend(result); backend != "" {
		section.Details = append(section.Details, webDetailRow{Label: "测试后端", Value: backend})
	}
	if readIOPS, ok := getDiskRandomReadIOPS(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "随机读 IOPS", Value: fmt.Sprintf("%.0f", readIOPS)})
	}
	if writeIOPS, ok := getDiskRandomWriteIOPS(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "随机写 IOPS", Value: fmt.Sprintf("%.0f", writeIOPS)})
	}
	if latency, ok := getDiskRandomReadP95Latency(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "随机读 P95 延迟", Value: fmt.Sprintf("%.2f ms", latency)})
	}
	if latency, ok := getDiskRandomWriteP95Latency(result); ok {
		section.Details = append(section.Details, webDetailRow{Label: "随机写 P95 延迟", Value: fmt.Sprintf("%.2f ms", latency)})
	}
	if rows := getDiskFioMixedRows(result); len(rows) > 0 {
		table := webTable{Title: "fio mixed randrw 50/50 矩阵", Headers: []string{"块大小", "读 MB/s", "写 MB/s", "总 MB/s", "读 IOPS", "写 IOPS", "总 IOPS"}}
		for _, row := range rows {
			table.Rows = append(table.Rows, []string{
				row.BlockSize,
				fmt.Sprintf("%.2f", row.ReadMBps),
				fmt.Sprintf("%.2f", row.WriteMBps),
				fmt.Sprintf("%.2f", row.TotalMBps),
				fmt.Sprintf("%.0f", row.ReadIOPS),
				fmt.Sprintf("%.0f", row.WriteIOPS),
				fmt.Sprintf("%.0f", row.TotalIOPS),
			})
		}
		section.Tables = append(section.Tables, table)
	}
	return section
}

func (ws *WebServer) networkSection(result *models.TestResult, overallScore *models.OverallScore) webReportSection {
	section := ws.testSectionBase("network", "网络测试", "延迟、下载、上传、IPv4/IPv6 和多节点质量。", result, "本次未执行网络测试。使用 -b network 或 -b all 启用。")
	if result == nil || result.Status != models.TestStatusSuccess {
		return section
	}
	section.Metrics = append(section.Metrics, webMetricCard{Label: "网络评分", Value: fmt.Sprintf("%.0f", overallScore.NetworkScore), Unit: "/ 100", Tone: "cyan"})
	if latency, ok := getNetworkLatency(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "平均延迟", Value: fmt.Sprintf("%.2f", latency), Unit: "ms", Tone: "cyan"})
	}
	if download, ok := getNetworkDownloadSpeed(result); ok {
		section.Metrics = append(section.Metrics, webMetricCard{Label: "下载速度", Value: fmt.Sprintf("%.2f", download), Unit: "Mbps", Tone: "cyan"})
	}
	if upload, ok := getNetworkUploadSpeed(result); ok {
		unit := "Mbps"
		if isNetworkUploadEstimated(result) {
			unit = "Mbps 估算"
		}
		section.Metrics = append(section.Metrics, webMetricCard{Label: "上传速度", Value: fmt.Sprintf("%.2f", upload), Unit: unit, Tone: "cyan"})
	}
	if backend := getNetworkBackend(result); backend != "" {
		section.Details = append(section.Details, webDetailRow{Label: "测试后端", Value: backend})
	}
	if server := getNetworkBackendServer(result); server != "" {
		section.Details = append(section.Details, webDetailRow{Label: "测试服务端", Value: server})
	}
	if source := getNetworkLatencySource(result); source != "" {
		section.Details = append(section.Details, webDetailRow{Label: "延迟来源", Value: source})
	}
	if source := getNetworkDownloadSource(result); source != "" {
		section.Details = append(section.Details, webDetailRow{Label: "下载来源", Value: source})
	}
	if source := getNetworkUploadSource(result); source != "" {
		section.Details = append(section.Details, webDetailRow{Label: "上传来源", Value: source})
	}
	if node, ok := getNetworkSpeedtestNode(result); ok {
		section.Details = append(section.Details,
			webDetailRow{Label: "Speedtest 节点", Value: networkSpeedtestServerLabel(node)},
			webDetailRow{Label: "节点位置", Value: fmt.Sprintf("%s %s", node.Location, node.Country)},
			webDetailRow{Label: "ISP", Value: fallback(node.ISP, "-")},
			webDetailRow{Label: "结果 URL", Value: fallback(node.ResultURL, "-")},
			webDetailRow{Label: "出口 IP", Value: fallback(node.ExternalIP, "-")},
			webDetailRow{Label: "Ping Jitter", Value: fmt.Sprintf("%.2f ms", node.PingJitter)},
		)
	}
	if rows := getNetworkQualityRows(result); len(rows) > 0 {
		table := webTable{Title: "TCP connect 网络质量矩阵", Headers: []string{"目标", "协议", "可用", "成功/失败", "平均延迟", "抖动", "失败率"}}
		for _, row := range rows {
			table.Rows = append(table.Rows, []string{
				fmt.Sprintf("%s %s", row.Target, row.Address),
				row.Protocol,
				availabilityLabel(row.Available),
				fmt.Sprintf("%d/%d", row.SuccessCount, row.FailureCount),
				fmt.Sprintf("%.2f ms", row.AvgLatencyMs),
				fmt.Sprintf("%.2f ms", row.JitterMs),
				fmt.Sprintf("%.2f%%", row.FailureRate*100),
			})
		}
		section.Tables = append(section.Tables, table)
	}
	if rows := getNetworkIperf3MatrixRows(result); len(rows) > 0 {
		table := webTable{Title: "iperf3 多节点矩阵", Headers: []string{"节点", "协议", "下载", "上传", "延迟", "错误"}}
		for _, row := range rows {
			table.Rows = append(table.Rows, []string{
				row.Server,
				row.Protocol,
				fmt.Sprintf("%.2f Mbps", row.DownloadMbps),
				fmt.Sprintf("%.2f Mbps", row.UploadMbps),
				fmt.Sprintf("%.2f ms", row.LatencyMs),
				fallback(row.Error, "-"),
			})
		}
		section.Tables = append(section.Tables, table)
	}
	if err := getNetworkError(result); err != "" {
		section.Details = append(section.Details, webDetailRow{Label: "网络错误", Value: err})
	}
	return section
}

func (ws *WebServer) testSectionBase(id string, title string, subtitle string, result *models.TestResult, hint string) webReportSection {
	section := webReportSection{
		ID:       id,
		Title:    title,
		Subtitle: subtitle,
		Hint:     hint,
	}
	if result == nil {
		section.Status = "skipped"
		section.StatusText = "未执行"
		section.Summary = hint
		return section
	}
	section.Status = result.Status
	section.StatusText = ws.getStatusText(result.Status)
	section.Summary = ws.getKeyMetrics(result)
	if result.Status == models.TestStatusSuccess {
		section.Hint = ""
	}
	section.Details = append(section.Details,
		webDetailRow{Label: "测试状态", Value: section.StatusText},
		webDetailRow{Label: "测试耗时", Value: fmt.Sprintf("%.2f 秒", result.DurationSeconds)},
	)
	if result.ErrorMessage != "" {
		section.Details = append(section.Details, webDetailRow{Label: "错误信息", Value: result.ErrorMessage})
	}
	if section.Summary == "-" {
		section.Summary = "本模块已执行，暂无可展示的关键指标。"
	}
	return section
}

func (ws *WebServer) optionalSection(id string, title string, subtitle string, summaryKey string, hint string) webReportSection {
	section := webReportSection{
		ID:       id,
		Title:    title,
		Subtitle: subtitle,
		Hint:     hint,
	}
	if ws.report == nil || ws.report.Summary == nil {
		section.Status = "skipped"
		section.StatusText = "未执行"
		section.Summary = hint
		return section
	}
	value, ok := ws.report.Summary[summaryKey]
	if !ok || value == nil {
		section.Status = "skipped"
		section.StatusText = "未执行"
		section.Summary = hint
		return section
	}
	section.Status = "success"
	section.StatusText = "已执行"
	section.Summary = "本模块已执行，详细结构会在后续版本中继续展开。"
	section.Details = append(section.Details, webDetailRow{Label: "报告字段", Value: summaryKey})
	section.Details = append(section.Details, webDetailRow{Label: "结果摘要", Value: fmt.Sprintf("%v", value)})
	return section
}

func optionalModuleBase(id string, title string, subtitle string, hint string) webReportSection {
	return webReportSection{
		ID:       id,
		Title:    title,
		Subtitle: subtitle,
		Hint:     hint,
	}
}

func skippedModule(section webReportSection) webReportSection {
	section.Status = "skipped"
	section.StatusText = "未执行"
	section.Summary = section.Hint
	return section
}

func (ws *WebServer) summaryValue(key string) interface{} {
	if ws.report == nil || ws.report.Summary == nil {
		return nil
	}
	return ws.report.Summary[key]
}

func (ws *WebServer) routeSection() webReportSection {
	hint := "本次未启用路由追踪。使用 --route-trace 启用。"
	section := optionalModuleBase("route", "路由追踪", "到主要地区和节点的网络路径质量。", hint)
	results, ok := ws.summaryValue("route_trace_results").([]*models.TraceResult)
	if !ok || len(results) == 0 {
		return skippedModule(section)
	}

	successCount := 0
	totalHops := 0
	for _, result := range results {
		if result != nil && result.Success {
			successCount++
			totalHops += result.TotalHops
		}
	}
	avgHops := 0.0
	if successCount > 0 {
		avgHops = float64(totalHops) / float64(successCount)
	}

	section.Status = webAvailabilityStatus(successCount, len(results))
	section.StatusText = fmt.Sprintf("%d/%d 成功", successCount, len(results))
	section.Summary = fmt.Sprintf("完成 %d 个目标追踪，成功 %d 个，平均 %.1f 跳。", len(results), successCount, avgHops)
	section.Hint = ""
	section.Metrics = append(section.Metrics,
		webMetricCard{Label: "目标数", Value: fmt.Sprintf("%d", len(results)), Tone: "primary"},
		webMetricCard{Label: "成功", Value: fmt.Sprintf("%d", successCount), Unit: fmt.Sprintf("/ %d", len(results)), Tone: "green"},
		webMetricCard{Label: "平均跳数", Value: fmt.Sprintf("%.1f", avgHops), Unit: "hops", Tone: "cyan"},
	)

	table := webTable{Title: "追踪目标", Headers: []string{"目标", "状态", "跳数", "最后一跳", "错误"}}
	for _, result := range results {
		if result == nil {
			continue
		}
		table.Rows = append(table.Rows, []string{
			result.Target,
			successLabel(result.Success),
			fmt.Sprintf("%d", result.TotalHops),
			lastTraceHop(result),
			fallback(result.ErrorMessage, "-"),
		})
	}
	section.Tables = append(section.Tables, table)
	return section
}

func webAvailabilityStatus(available int, total int) string {
	switch {
	case total == 0:
		return "skipped"
	case available == total:
		return "success"
	case available > 0:
		return "warning"
	default:
		return "failed"
	}
}

func availabilityTone(available int, total int) string {
	switch webAvailabilityStatus(available, total) {
	case "success":
		return "green"
	case "warning":
		return "amber"
	case "failed":
		return "red"
	default:
		return "primary"
	}
}

func successLabel(success bool) string {
	if success {
		return "成功"
	}
	return "失败"
}

func lastTraceHop(result *models.TraceResult) string {
	if result == nil || len(result.Hops) == 0 {
		return "-"
	}
	for i := len(result.Hops) - 1; i >= 0; i-- {
		hop := result.Hops[i]
		if hop == nil || hop.IP == "" || hop.IP == "*" {
			continue
		}
		if hop.Hostname != "" {
			return fmt.Sprintf("%s (%s)", hop.IP, hop.Hostname)
		}
		return hop.IP
	}
	return "-"
}

func (ws *WebServer) streamingSection() webReportSection {
	hint := "本次未启用流媒体检测。使用 --streaming 启用。"
	section := optionalModuleBase("streaming", "流媒体解锁", "Netflix、Disney+、YouTube 等平台的区域访问能力。", hint)
	results, ok := ws.summaryValue("streaming_results").(map[string]*models.StreamingResult)
	if !ok || len(results) == 0 {
		return skippedModule(section)
	}

	availableCount := 0
	for _, result := range results {
		if result != nil && result.Available {
			availableCount++
		}
	}

	section.Status = webAvailabilityStatus(availableCount, len(results))
	section.StatusText = fmt.Sprintf("%d/%d 可用", availableCount, len(results))
	section.Summary = fmt.Sprintf("检测 %d 个流媒体平台，可用 %d 个。", len(results), availableCount)
	section.Hint = ""
	section.Metrics = append(section.Metrics,
		webMetricCard{Label: "平台数", Value: fmt.Sprintf("%d", len(results)), Tone: "primary"},
		webMetricCard{Label: "可用", Value: fmt.Sprintf("%d", availableCount), Unit: fmt.Sprintf("/ %d", len(results)), Tone: availabilityTone(availableCount, len(results))},
	)

	table := webTable{Title: "平台结果", Headers: []string{"平台", "状态", "区域", "说明"}}
	for _, pair := range sortedStreamingResults(results) {
		table.Rows = append(table.Rows, []string{
			pair.key,
			availabilityLabel(pair.result != nil && pair.result.Available),
			streamingRegion(pair.result),
			streamingMessage(pair.result),
		})
	}
	section.Tables = append(section.Tables, table)
	return section
}

type streamingResultPair struct {
	key    string
	result *models.StreamingResult
}

func sortedStreamingResults(results map[string]*models.StreamingResult) []streamingResultPair {
	pairs := make([]streamingResultPair, 0, len(results))
	for key, result := range results {
		pairs = append(pairs, streamingResultPair{key: key, result: result})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return strings.ToLower(pairs[i].key) < strings.ToLower(pairs[j].key)
	})
	return pairs
}

func streamingRegion(result *models.StreamingResult) string {
	if result == nil {
		return "-"
	}
	return fallback(result.Region, "-")
}

func streamingMessage(result *models.StreamingResult) string {
	if result == nil {
		return "-"
	}
	return fallback(result.Message, "-")
}

func (ws *WebServer) aiSection() webReportSection {
	hint := "本次未启用 AI 服务检测。使用 --ai-services 启用。"
	section := optionalModuleBase("ai", "AI 服务检测", "OpenAI、Gemini 等 AI 服务的可访问性。", hint)
	results, ok := ws.summaryValue("ai_results").(map[string]*models.AIServiceResult)
	if !ok || len(results) == 0 {
		return skippedModule(section)
	}

	availableCount := 0
	for _, result := range results {
		if result != nil && result.Available {
			availableCount++
		}
	}

	section.Status = webAvailabilityStatus(availableCount, len(results))
	section.StatusText = fmt.Sprintf("%d/%d 可用", availableCount, len(results))
	section.Summary = fmt.Sprintf("检测 %d 个 AI 服务，可访问 %d 个。", len(results), availableCount)
	section.Hint = ""
	section.Metrics = append(section.Metrics,
		webMetricCard{Label: "服务数", Value: fmt.Sprintf("%d", len(results)), Tone: "primary"},
		webMetricCard{Label: "可访问", Value: fmt.Sprintf("%d", availableCount), Unit: fmt.Sprintf("/ %d", len(results)), Tone: availabilityTone(availableCount, len(results))},
	)

	table := webTable{Title: "服务结果", Headers: []string{"服务", "状态", "说明"}}
	for _, pair := range sortedAIResults(results) {
		table.Rows = append(table.Rows, []string{
			pair.key,
			availabilityLabel(pair.result != nil && pair.result.Available),
			aiMessage(pair.result),
		})
	}
	section.Tables = append(section.Tables, table)
	return section
}

type aiResultPair struct {
	key    string
	result *models.AIServiceResult
}

func sortedAIResults(results map[string]*models.AIServiceResult) []aiResultPair {
	pairs := make([]aiResultPair, 0, len(results))
	for key, result := range results {
		pairs = append(pairs, aiResultPair{key: key, result: result})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return strings.ToLower(pairs[i].key) < strings.ToLower(pairs[j].key)
	})
	return pairs
}

func aiMessage(result *models.AIServiceResult) string {
	if result == nil {
		return "-"
	}
	return fallback(result.Message, "-")
}

func (ws *WebServer) ipQualitySection() webReportSection {
	hint := "本次未启用 IP 质量检测。使用 --ip-quality 启用，或使用 --full / --vps-profile 预设。"
	section := webReportSection{
		ID:       "ip-quality",
		Title:    "IP 质量",
		Subtitle: "DNSBL 黑名单、邮件端口连通性、IP 类型和风险评分。",
		Hint:     hint,
	}
	if ws.report == nil || ws.report.Summary == nil {
		section.Status = "skipped"
		section.StatusText = "未执行"
		section.Summary = hint
		return section
	}
	report, ok := ws.report.Summary["ip_quality_report"].(*models.IPQualityReport)
	if !ok || report == nil {
		section.Status = "skipped"
		section.StatusText = "未执行"
		section.Summary = hint
		return section
	}

	section.Status = ipQualityWebStatus(report.RiskLevel)
	section.StatusText = ipRiskLabel(report.RiskLevel)
	section.Summary = fmt.Sprintf("%s，风险分 %d/100，IP 类型：%s。", ipRiskLabel(report.RiskLevel), report.RiskScore, ipQualityLabel(report.IPType))
	section.Hint = ""
	section.Metrics = append(section.Metrics,
		webMetricCard{Label: "风险分", Value: fmt.Sprintf("%d", report.RiskScore), Unit: "/ 100", Tone: ipQualityMetricTone(report.RiskLevel)},
		webMetricCard{Label: "黑名单命中", Value: fmt.Sprintf("%d", countListedBlacklists(report.BlacklistChecks)), Unit: fmt.Sprintf("/ %d", len(report.BlacklistChecks)), Tone: "amber"},
		webMetricCard{Label: "邮件可连", Value: fmt.Sprintf("%d", countReachableMailChecks(report.MailChecks)), Unit: fmt.Sprintf("/ %d", len(report.MailChecks)), Tone: "cyan"},
	)
	section.Details = append(section.Details,
		webDetailRow{Label: "公网 IP", Value: fallback(report.PublicIP, "-")},
		webDetailRow{Label: "IP 版本", Value: fallback(report.IPVersion, "-")},
		webDetailRow{Label: "ISP", Value: fallback(report.ISP, "-")},
		webDetailRow{Label: "位置", Value: fallback(strings.Trim(strings.Join([]string{report.Country, report.City}, " "), " "), "-")},
		webDetailRow{Label: "IP 类型", Value: ipQualityLabel(report.IPType)},
	)
	if len(report.BlacklistChecks) > 0 {
		table := webTable{Title: "DNSBL 黑名单", Headers: []string{"名单", "状态", "详情"}}
		for _, check := range report.BlacklistChecks {
			table.Rows = append(table.Rows, []string{
				check.Zone,
				ipBlacklistStatusLabel(check),
				fallback(check.Detail, "-"),
			})
		}
		section.Tables = append(section.Tables, table)
	}
	if len(report.MailChecks) > 0 {
		table := webTable{Title: "邮件端口连通性", Headers: []string{"目标", "端口", "状态", "详情"}}
		for _, check := range report.MailChecks {
			table.Rows = append(table.Rows, []string{
				check.Target,
				fmt.Sprintf("%d", check.Port),
				mailCheckStatusLabel(check),
				fallback(check.Detail, "-"),
			})
		}
		section.Tables = append(section.Tables, table)
	}
	for _, note := range report.Notes {
		section.Details = append(section.Details, webDetailRow{Label: "说明", Value: note})
	}
	return section
}

func (ws *WebServer) stressSection() webReportSection {
	hint := "本次未启用压力测试。使用 --stress 启用。"
	section := optionalModuleBase("stress", "压力测试", "长时间 CPU、内存、磁盘压力下的稳定性。", hint)
	report, ok := ws.summaryValue("stress_report").(*models.StressTestReport)
	if !ok || report == nil {
		return skippedModule(section)
	}

	failedCount := 0
	degradedCount := 0
	for _, component := range report.Components {
		if component == nil {
			continue
		}
		switch component.Status {
		case models.TestStatusFailed:
			failedCount++
		case models.TestStatusDegraded:
			degradedCount++
		}
	}
	section.Status = stressWebStatus(failedCount, degradedCount)
	section.StatusText = stressWebStatusText(failedCount, degradedCount)
	section.Summary = fmt.Sprintf("压力测试持续 %.0f 秒，组件 %d 个，温度数据%s。", report.TotalDurationSeconds, len(report.Components), temperatureStatus(report.TemperatureAvailable))
	section.Hint = ""
	section.Metrics = append(section.Metrics,
		webMetricCard{Label: "总耗时", Value: fmt.Sprintf("%.0f", report.TotalDurationSeconds), Unit: "秒", Tone: "primary"},
		webMetricCard{Label: "组件数", Value: fmt.Sprintf("%d", len(report.Components)), Tone: "cyan"},
		webMetricCard{Label: "异常组件", Value: fmt.Sprintf("%d", failedCount+degradedCount), Unit: fmt.Sprintf("/ %d", len(report.Components)), Tone: stressMetricTone(failedCount, degradedCount)},
	)

	table := webTable{Title: "压力组件", Headers: []string{"组件", "状态", "耗时", "平均温度", "峰值温度", "备注"}}
	for _, component := range report.Components {
		if component == nil {
			continue
		}
		table.Rows = append(table.Rows, []string{
			component.Name,
			ws.getStatusText(component.Status),
			fmt.Sprintf("%.0f 秒", component.DurationSeconds),
			temperatureValue(component.AverageTemperature),
			temperatureValue(component.PeakTemperature),
			fallback(component.Notes, "-"),
		})
	}
	section.Tables = append(section.Tables, table)
	return section
}

func (ws *WebServer) securitySection() webReportSection {
	hint := "本次未启用安全体检。使用 --security 启用。"
	section := optionalModuleBase("security", "安全体检", "端口、SSH 配置和基础安全风险检查。", hint)
	report, ok := ws.summaryValue("security_report").(*models.SecurityReport)
	if !ok || report == nil {
		return skippedModule(section)
	}

	counts := countSecuritySeverities(report.Findings)
	section.Status = securityWebStatus(counts)
	section.StatusText = securityWebStatusText(counts)
	section.Summary = fmt.Sprintf("发现 %d 条安全提示：高危 %d，中危 %d，低危 %d，信息 %d。", len(report.Findings), counts["high"], counts["medium"], counts["low"], counts["info"])
	section.Hint = ""
	section.Metrics = append(section.Metrics,
		webMetricCard{Label: "提示数", Value: fmt.Sprintf("%d", len(report.Findings)), Tone: "primary"},
		webMetricCard{Label: "高危", Value: fmt.Sprintf("%d", counts["high"]), Tone: severityTone("high")},
		webMetricCard{Label: "中危", Value: fmt.Sprintf("%d", counts["medium"]), Tone: severityTone("medium")},
	)

	table := webTable{Title: "安全发现", Headers: []string{"类别", "级别", "标题", "详情", "建议"}}
	for _, finding := range report.Findings {
		if finding == nil {
			continue
		}
		table.Rows = append(table.Rows, []string{
			finding.Category,
			securitySeverityLabel(finding.Severity),
			finding.Title,
			fallback(finding.Detail, "-"),
			fallback(finding.Advice, "-"),
		})
	}
	section.Tables = append(section.Tables, table)
	return section
}

func (ws *WebServer) placeholderSection(id string, title string, subtitle string, hint string) webReportSection {
	return webReportSection{
		ID:         id,
		Title:      title,
		Subtitle:   subtitle,
		Status:     "skipped",
		StatusText: "规划中",
		Summary:    hint,
		Hint:       hint,
	}
}

func ipQualityWebStatus(level string) string {
	if level == "high" {
		return "failed"
	}
	if level == "medium" {
		return "warning"
	}
	return "success"
}

func ipQualityMetricTone(level string) string {
	if level == "high" {
		return "red"
	}
	if level == "medium" {
		return "amber"
	}
	return "green"
}

func countListedBlacklists(checks []*models.IPBlacklistCheck) int {
	count := 0
	for _, check := range checks {
		if check != nil && check.Listed {
			count++
		}
	}
	return count
}

func countReachableMailChecks(checks []*models.MailPortCheck) int {
	count := 0
	for _, check := range checks {
		if check != nil && check.Reachable {
			count++
		}
	}
	return count
}

func stressWebStatus(failedCount int, degradedCount int) string {
	if failedCount > 0 {
		return "failed"
	}
	if degradedCount > 0 {
		return "warning"
	}
	return "success"
}

func stressWebStatusText(failedCount int, degradedCount int) string {
	if failedCount > 0 {
		return fmt.Sprintf("%d 个失败", failedCount)
	}
	if degradedCount > 0 {
		return fmt.Sprintf("%d 个降级", degradedCount)
	}
	return "稳定"
}

func stressMetricTone(failedCount int, degradedCount int) string {
	if failedCount > 0 {
		return "red"
	}
	if degradedCount > 0 {
		return "amber"
	}
	return "green"
}

func temperatureStatus(available bool) string {
	if available {
		return "可获取"
	}
	return "未提供"
}

func temperatureValue(value float64) string {
	if value <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f°C", value)
}

func countSecuritySeverities(findings []*models.SecurityFinding) map[string]int {
	counts := map[string]int{
		"high":   0,
		"medium": 0,
		"low":    0,
		"info":   0,
	}
	for _, finding := range findings {
		if finding == nil {
			continue
		}
		severity := strings.ToLower(strings.TrimSpace(finding.Severity))
		if severity == "" {
			severity = "info"
		}
		if _, ok := counts[severity]; !ok {
			severity = "info"
		}
		counts[severity]++
	}
	return counts
}

func securityWebStatus(counts map[string]int) string {
	if counts["high"] > 0 {
		return "failed"
	}
	if counts["medium"] > 0 {
		return "warning"
	}
	return "success"
}

func securityWebStatusText(counts map[string]int) string {
	if counts["high"] > 0 {
		return fmt.Sprintf("%d 高危", counts["high"])
	}
	if counts["medium"] > 0 {
		return fmt.Sprintf("%d 中危", counts["medium"])
	}
	return "无明显风险"
}

func severityTone(severity string) string {
	switch severity {
	case "high":
		return "red"
	case "medium":
		return "amber"
	case "low":
		return "cyan"
	default:
		return "green"
	}
}

func securitySeverityLabel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		return "高危"
	case "medium":
		return "中危"
	case "low":
		return "低危"
	case "info":
		return "信息"
	default:
		return fallback(severity, "信息")
	}
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
	case "warning":
		return "注意"
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

func fallback(value string, fallbackValue string) string {
	if value != "" {
		return value
	}
	return fallbackValue
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
