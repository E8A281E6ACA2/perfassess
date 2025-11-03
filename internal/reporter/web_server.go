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
	case "cpu":
		if ops, ok := result.Metrics["single_core_operations"].(int64); ok {
			return fmt.Sprintf("单核: %d 次操作", ops)
		}
	case "memory":
		if speed, ok := result.Metrics["read_speed_mbps"].(float64); ok {
			return fmt.Sprintf("读取: %.2f MB/s", speed)
		}
	case "disk":
		if speed, ok := result.Metrics["sequential_read_mbps"].(float64); ok {
			return fmt.Sprintf("顺序读: %.2f MB/s", speed)
		}
	case "network":
		if latency, ok := result.Metrics["average_latency_ms"].(float64); ok {
			return fmt.Sprintf("延迟: %.2f ms", latency)
		}
	}
	
	return "-"
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
