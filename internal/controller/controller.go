// Package controller 提供核心控制器功能
// 负责协调整个评估流程，管理各模块生命周期
package controller

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"

	"performance-assessment-system/internal/collector"
	"performance-assessment-system/internal/config"
	"performance-assessment-system/internal/models"
	"performance-assessment-system/internal/platform"
	"performance-assessment-system/internal/reporter"
	"performance-assessment-system/internal/tests"
	"performance-assessment-system/pkg/logger"
	"performance-assessment-system/pkg/utils"
)

// AssessmentController 核心控制器
// 协调整个评估流程，管理各模块生命周期
type AssessmentController struct {
	config          *config.Config
	logger          *logger.Logger
	session         *models.AssessmentSession
	platformAdapter platform.PlatformAdapter
	errorHandler    *utils.ErrorHandler
}

// NewAssessmentController 创建新的评估控制器
func NewAssessmentController(cfg *config.Config) (*AssessmentController, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建日志管理器
	log, err := logger.NewLogger("logs", cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("创建日志管理器失败: %w", err)
	}

	// 检测平台
	detector := platform.NewPlatformDetector()
	if !detector.IsSupported() {
		return nil, fmt.Errorf("不支持的操作系统平台: %s", detector.DetectPlatform())
	}

	platformAdapter, err := detector.GetPlatformAdapter()
	if err != nil {
		return nil, fmt.Errorf("获取平台适配器失败: %w", err)
	}

	// 创建错误处理器
	errorHandler := utils.NewErrorHandler(log)

	return &AssessmentController{
		config:          cfg,
		logger:          log,
		platformAdapter: platformAdapter,
		errorHandler:    errorHandler,
	}, nil
}

// StartSession 启动评估会话
func (ac *AssessmentController) StartSession() (*models.AssessmentSession, error) {
	// 生成会话ID（使用时间戳）
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	// 创建会话日志文件
	logFile, err := ac.logger.CreateSessionLog()
	if err != nil {
		return nil, fmt.Errorf("创建会话日志失败: %w", err)
	}

	// 创建会话对象
	session := &models.AssessmentSession{
		SessionID: sessionID,
		StartTime: time.Now(),
		Config:    ac.config,
		LogFile:   logFile,
	}

	ac.session = session

	ac.logger.Info("评估会话已启动",
		zap.String("session_id", sessionID),
		zap.String("log_file", logFile),
	)

	return session, nil
}

// RunAssessment 运行完整的评估流程
func (ac *AssessmentController) RunAssessment(session *models.AssessmentSession) (*models.Report, error) {
	if session == nil {
		return nil, fmt.Errorf("会话不能为空")
	}

	ac.logger.Info("开始性能评估")

	// 步骤1: 收集系统信息
	ac.logger.Info("步骤 1/3: 收集系统信息")
	systemInfo, err := ac.collectSystemInfo()
	if err != nil {
		ac.logger.Error("收集系统信息失败", err)
		return nil, fmt.Errorf("收集系统信息失败: %w", err)
	}
	ac.logger.Info("系统信息收集完成")

	// 步骤2: 执行性能测试
	ac.logger.Info("步骤 2/3: 执行性能测试")
	testResults, err := ac.runPerformanceTests()
	if err != nil {
		ac.logger.Error("性能测试失败", err)
		// 非致命错误，继续生成报告
		ac.logger.Warn("部分测试失败，继续生成报告")
	}
	ac.logger.Info("性能测试完成")

	// 可选步骤: 路由追踪
	var routeTraceResults []*models.TraceResult
	if ac.config.EnableRouteTrace {
		ac.logger.Info("可选步骤: 执行路由追踪")
		routeTraceResults, err = ac.runRouteTrace()
		if err != nil {
			ac.logger.Warn("路由追踪失败", zap.Error(err))
		} else {
			ac.logger.Info("路由追踪完成")
		}
	}

	// 可选步骤: 流媒体检测
	var streamingResults map[string]*models.StreamingResult
	var aiResults map[string]*models.AIServiceResult
	var stressReport *models.StressTestReport
	var securityReport *models.SecurityReport
	if ac.config.EnableStreaming {
		ac.logger.Info("可选步骤: 执行流媒体检测")
		streamingResults, err = ac.runStreamingDetection()
		if err != nil {
			ac.logger.Warn("流媒体检测失败", zap.Error(err))
		} else {
			ac.logger.Info("流媒体检测完成")
		}
	}

	if ac.config.EnableAIServices {
		ac.logger.Info("可选步骤: 执行AI服务检测")
		aiResults, err = ac.runAIServiceDetection()
		if err != nil {
			ac.logger.Warn("AI服务检测失败", zap.Error(err))
		} else {
			ac.logger.Info("AI服务检测完成")
		}
	}

	if ac.config.EnableStressTest {
		ac.logger.Info("可选步骤: 执行长时间压力测试")
		stressReport, err = ac.runStressTest()
		if err != nil {
			ac.logger.Warn("压力测试失败", zap.Error(err))
		} else {
			ac.logger.Info("压力测试完成")
		}
	}

	if ac.config.EnableSecurityScan {
		ac.logger.Info("可选步骤: 执行安全体检")
		securityReport = ac.runSecurityScan(systemInfo)
		ac.logger.Info("安全体检完成")
	}

	// 步骤3: 生成报告
	ac.logger.Info("步骤 3/3: 生成评估报告")
	report, err := ac.generateReport(session.SessionID, systemInfo, testResults)
	if err != nil {
		ac.logger.Error("生成报告失败", err)
		return nil, fmt.Errorf("生成报告失败: %w", err)
	}

	// 添加可选功能结果到报告
	if routeTraceResults != nil {
		report.Summary["route_trace_results"] = routeTraceResults
	}
	if streamingResults != nil {
		report.Summary["streaming_results"] = streamingResults
	}
	if aiResults != nil {
		report.Summary["ai_results"] = aiResults
	}
	if stressReport != nil {
		report.Summary["stress_report"] = stressReport
	}
	if securityReport != nil {
		report.Summary["security_report"] = securityReport
	}

	// 更新格式化报告以包含新增的摘要内容
	report.FormattedContent = reporter.NewReportGenerator().FormatReport(report)

	ac.logger.Info("评估报告生成完成")

	// 输出报告
	if err := ac.outputReport(report); err != nil {
		ac.logger.Error("输出报告失败", err)
		return nil, fmt.Errorf("输出报告失败: %w", err)
	}

	// 如果启用了 Web 服务器，启动它
	if ac.config.EnableWeb {
		webServer := reporter.NewWebServer(ac.config.WebPort, ac.logger)
		if err := webServer.Start(report); err != nil {
			ac.logger.Error("Web 服务器启动失败", err)
		}
	}

	// 记录会话摘要
	ac.logger.LogSessionSummary(report.Summary)

	ac.logger.Info("性能评估完成",
		zap.String("session_id", session.SessionID),
		zap.Duration("total_duration", time.Since(session.StartTime)),
	)

	return report, nil
}

// collectSystemInfo 收集系统信息
func (ac *AssessmentController) collectSystemInfo() (*models.SystemInfo, error) {
	// 创建系统信息收集器
	sysInfoCollector := collector.NewSystemInfoCollector(ac.platformAdapter)

	// 使用超时控制
	timeout := 10 * time.Second
	var systemInfo *models.SystemInfo
	var err error

	ctx := context.Background()
	err = utils.RunWithTimeout(ctx, timeout, func() error {
		systemInfo, err = sysInfoCollector.CollectAll()
		return err
	})

	if err != nil {
		return nil, err
	}

	// 收集虚拟化信息
	virtDetector := collector.NewVirtualizationDetector(ac.platformAdapter)
	virtInfo, err := virtDetector.Detect()
	if err != nil {
		ac.logger.Warn("虚拟化检测失败", zap.Error(err))
	} else {
		systemInfo.Virtualization = virtInfo
	}

	// 收集IP信息
	ipCollector, err := collector.NewIPInfoCollector(ac.config.GeoIPDBPath)
	if err != nil {
		ac.logger.Warn("创建IP信息收集器失败", zap.Error(err))
	} else {
		ipInfo, err := ipCollector.CollectAll()
		if err != nil {
			ac.logger.Warn("IP信息收集失败", zap.Error(err))
		} else {
			systemInfo.IPInfo = ipInfo
		}
	}

	return systemInfo, nil
}

// runPerformanceTests 执行性能测试
func (ac *AssessmentController) runPerformanceTests() (*models.TestResults, error) {
	// 创建测试运行器
	testRunner := tests.NewTestRunner(ac.logger)

	// 确定要运行的测试类型
	testTypes := ac.config.Tests
	if len(testTypes) == 1 && testTypes[0] == "all" {
		testTypes = []string{"cpu", "memory", "disk", "network"}
	}

	// 创建测试实例列表
	var testList []tests.PerformanceTest
	for _, testType := range testTypes {
		switch testType {
		case "cpu":
			testList = append(testList, tests.NewCPUTest(ac.logger))
		case "memory":
			testList = append(testList, tests.NewMemoryTest(ac.logger))
		case "disk":
			testList = append(testList, ac.newDiskTest())
		case "network":
			testList = append(testList, ac.newNetworkTest())
		}
	}

	// 运行测试
	results, err := testRunner.RunTests(testList)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (ac *AssessmentController) newDiskTest() tests.PerformanceTest {
	if ac.config.DiskBackend == models.DiskBackendFio {
		return tests.NewDiskTestWithBackend(ac.logger, tests.NewFioDiskBackend(tests.FioConfig{}))
	}

	return tests.NewDiskTest(ac.logger)
}

func (ac *AssessmentController) newNetworkTest() tests.PerformanceTest {
	if ac.config.NetworkBackend == models.NetworkBackendIperf3 {
		latencyFallback := tests.NewNetworkTest(ac.logger)
		backend := tests.NewIperf3NetworkBackend(tests.Iperf3Config{
			Server:    ac.config.Iperf3Server,
			LatencyFn: latencyFallback.TestLatency,
		})
		return tests.NewNetworkTestWithBackend(ac.logger, backend)
	}

	return tests.NewNetworkTest(ac.logger)
}

// runRouteTrace 执行路由追踪
func (ac *AssessmentController) runRouteTrace() ([]*models.TraceResult, error) {
	// 创建路由追踪器
	tracer := tests.NewRouteTracer(ac.logger)

	// 获取目标列表
	targets := ac.config.RouteTraceTargets
	if len(targets) == 0 {
		// 使用默认目标
		targets = []string{"8.8.8.8", "1.1.1.1", "cloudflare.com"}
	}

	// 执行追踪
	results, err := tracer.TraceMultiple(targets)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// runStreamingDetection 执行流媒体检测
func (ac *AssessmentController) runStreamingDetection() (map[string]*models.StreamingResult, error) {
	// 创建流媒体检测器
	detector := tests.NewStreamingDetector(ac.logger)

	// 检查网络连接
	if !detector.CheckConnectivity() {
		return nil, fmt.Errorf("网络连接不可用")
	}

	// 执行检测
	results, err := detector.CheckAll()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// runAIServiceDetection 执行 AI 服务检测
func (ac *AssessmentController) runAIServiceDetection() (map[string]*models.AIServiceResult, error) {
	detector := tests.NewAIServiceDetector(ac.logger)

	if !detector.CheckConnectivity() {
		return nil, fmt.Errorf("网络连接不可用")
	}

	results, err := detector.CheckAll()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// runStressTest 执行长时间压力测试
func (ac *AssessmentController) runStressTest() (*models.StressTestReport, error) {
	stressTester := tests.NewStressTest(ac.logger)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	return stressTester.Run(ctx)
}

func (ac *AssessmentController) runSecurityScan(systemInfo *models.SystemInfo) *models.SecurityReport {
	scanner := tests.NewSecurityScanner(ac.logger, systemInfo)
	return scanner.Run()
}

// generateReport 生成评估报告
func (ac *AssessmentController) generateReport(sessionID string, systemInfo *models.SystemInfo, testResults *models.TestResults) (*models.Report, error) {
	// 创建报告生成器
	reportGenerator := reporter.NewReportGenerator()

	// 生成报告
	report, err := reportGenerator.GenerateReport(sessionID, systemInfo, testResults)
	if err != nil {
		return nil, err
	}

	return report, nil
}

// outputReport 输出报告
func (ac *AssessmentController) outputReport(report *models.Report) error {
	// 创建输出格式化器
	formatter := reporter.NewOutputFormatter()

	// 输出到控制台
	if err := formatter.OutputToConsoleWithFormat(report, ac.config.OutputFormat); err != nil {
		return fmt.Errorf("输出到控制台失败: %w", err)
	}

	// 如果指定了输出文件，保存到文件
	if ac.config.Output != "" {
		if err := formatter.OutputToFileWithFormat(report, ac.config.Output, ac.config.OutputFormat); err != nil {
			return fmt.Errorf("输出到文件失败: %w", err)
		}
		ac.logger.Info("报告已保存到文件", zap.String("file", ac.config.Output))
	}

	return nil
}

// HandleError 处理错误
func (ac *AssessmentController) HandleError(err error) utils.ErrorAction {
	return ac.errorHandler.HandleError(err, "评估控制器")
}

// Cleanup 清理资源
func (ac *AssessmentController) Cleanup() error {
	ac.logger.Info("清理资源")

	// 同步日志
	if err := ac.logger.Sync(); err != nil {
		return fmt.Errorf("同步日志失败: %w", err)
	}

	// 强制垃圾回收
	runtime.GC()

	ac.logger.Info("资源清理完成")
	return nil
}

// GetSession 获取当前会话
func (ac *AssessmentController) GetSession() *models.AssessmentSession {
	return ac.session
}

// GetLogger 获取日志管理器
func (ac *AssessmentController) GetLogger() *logger.Logger {
	return ac.logger
}
