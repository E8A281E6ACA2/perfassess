// Package tests 提供网络性能测试功能
package tests

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
	"performance-assessment-system/pkg/utils"
)

// NetworkBenchmarkBackend 定义网络评测后端接口
// 当前默认使用内置实现，后续可扩展到 iperf3 等主流工具。
type NetworkBenchmarkBackend interface {
	Name() string
	Server() string
	DownloadSource() string
	UploadSource(estimated bool) string
	MeasureLatency(hosts []string) (float64, error)
	MeasureDownload() (float64, error)
	MeasureUpload(downloadSpeed float64) (float64, bool, error)
}

// BuiltinNetworkBackend 使用当前项目内置的网络测试逻辑
type BuiltinNetworkBackend struct {
	test *NetworkTest
}

func (b *BuiltinNetworkBackend) Name() string {
	return models.NetworkBackendBuiltin
}

func (b *BuiltinNetworkBackend) Server() string {
	return ""
}

func (b *BuiltinNetworkBackend) DownloadSource() string {
	return models.NetworkDownloadSourceHTTP
}

func (b *BuiltinNetworkBackend) UploadSource(estimated bool) string {
	if estimated {
		return models.NetworkUploadSourceEstimated
	}
	return models.NetworkUploadSourceHTTP
}

func (b *BuiltinNetworkBackend) MeasureLatency(hosts []string) (float64, error) {
	return b.test.TestLatency(hosts)
}

func (b *BuiltinNetworkBackend) MeasureDownload() (float64, error) {
	return b.test.TestDownloadSpeed()
}

func (b *BuiltinNetworkBackend) MeasureUpload(downloadSpeed float64) (float64, bool, error) {
	return b.test.TestUploadSpeed(downloadSpeed)
}

// NetworkTest 网络性能测试
// 测试网络延迟、下载和上传速度
type NetworkTest struct {
	*BaseTest
	httpClient *http.Client
	testHosts  []string // 测试主机列表
	backend    NetworkBenchmarkBackend
	latencyFn  func([]string) (float64, error)
	downloadFn func() (float64, error)
	uploadFn   func(float64) (float64, bool, error)
}

// NewNetworkTest 创建网络性能测试
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *NetworkTest: 网络测试实例
func NewNetworkTest(logger *logger.Logger) *NetworkTest {
	return NewNetworkTestWithBackend(logger, nil)
}

func NewNetworkTestWithBackend(logger *logger.Logger, backend NetworkBenchmarkBackend) *NetworkTest {
	test := &NetworkTest{
		BaseTest: NewBaseTest("网络性能测试", 60*time.Second, logger),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		testHosts: []string{
			"8.8.8.8:53",         // Google DNS
			"1.1.1.1:53",         // Cloudflare DNS
			"114.114.114.114:53", // 114 DNS
		},
	}
	if backend != nil {
		test.backend = backend
	} else {
		test.backend = &BuiltinNetworkBackend{test: test}
	}
	return test
}

func (nt *NetworkTest) resolveLatencyFunc() func([]string) (float64, error) {
	if nt.latencyFn != nil {
		return nt.latencyFn
	}
	if nt.backend != nil {
		return nt.backend.MeasureLatency
	}
	return nt.TestLatency
}

func (nt *NetworkTest) resolveDownloadFunc() func() (float64, error) {
	if nt.downloadFn != nil {
		return nt.downloadFn
	}
	if nt.backend != nil {
		return nt.backend.MeasureDownload
	}
	return nt.TestDownloadSpeed
}

func (nt *NetworkTest) resolveUploadFunc() func(float64) (float64, bool, error) {
	if nt.uploadFn != nil {
		return nt.uploadFn
	}
	if nt.backend != nil {
		return nt.backend.MeasureUpload
	}
	return nt.TestUploadSpeed
}

// Setup 测试前的准备工作
// 检查网络连接是否可用
// 返回:
//   - error: 准备错误
func (nt *NetworkTest) Setup() error {
	// 检查网络连接
	if !nt.CheckConnectivity() {
		return utils.WrapError(
			utils.ErrNetworkUnavailable,
			"网络连接不可用",
		)
	}

	nt.GetLogger().Info("网络连接正常")
	return nil
}

// Execute 执行网络性能测试
// 返回:
//   - *models.TestResult: 测试结果
//   - error: 测试错误
func (nt *NetworkTest) Execute() (*models.TestResult, error) {
	nt.MarkStart()
	defer func() {
		nt.MarkEnd("success")
	}()

	metrics := &models.NetworkMetrics{
		Backend:           models.NetworkBackendBuiltin,
		LatencyMs:         -1.0,
		AverageLatencyMs:  -1.0,
		LatencySource:     models.NetworkLatencySourceTCPConnect,
		DownloadSpeedMbps: -1.0,
		DownloadSource:    models.NetworkDownloadSourceHTTPFailed,
		UploadSpeedMbps:   -1.0,
		UploadSource:      models.NetworkUploadSourceUnavailable,
		UploadEstimated:   false,
		Score:             0.0,
	}
	if nt.backend != nil {
		metrics.Backend = nt.backend.Name()
		metrics.BackendServer = nt.backend.Server()
	}
	avgLatency := -1.0
	downloadSpeed := -1.0
	uploadSpeed := -1.0
	uploadEstimated := false

	// 测试网络延迟
	nt.GetLogger().Info("开始网络延迟测试...")
	var err error
	avgLatency, err = nt.resolveLatencyFunc()(nt.testHosts)
	if err != nil {
		nt.GetLogger().Warn(fmt.Sprintf("延迟测试失败: %v", err))
		metrics.AppendError("延迟测试失败: " + err.Error())
	} else {
		metrics.LatencyMs = avgLatency
		metrics.AverageLatencyMs = avgLatency
		nt.GetLogger().Info(fmt.Sprintf("平均延迟: %.2f ms", avgLatency))
	}

	// 测试下载速度
	nt.GetLogger().Info("开始下载速度测试...")
	downloadSpeed, err = nt.resolveDownloadFunc()()
	if err != nil {
		nt.GetLogger().Warn(fmt.Sprintf("下载速度测试失败: %v", err))
		metrics.AppendError("下载速度测试失败: " + err.Error())
	} else {
		metrics.DownloadSpeedMbps = downloadSpeed
		metrics.DownloadSource = nt.resolveDownloadSource()
		nt.GetLogger().Info(fmt.Sprintf("下载速度: %.2f Mbps", downloadSpeed))
	}

	// 测试上传速度（当前为降级估算模式）
	nt.GetLogger().Info("开始上传速度测试...")
	uploadSpeed, uploadEstimated, err = nt.resolveUploadFunc()(downloadSpeed)
	if err != nil {
		nt.GetLogger().Warn(fmt.Sprintf("上传速度测试失败: %v", err))
		metrics.AppendError("上传速度测试失败: " + err.Error())
	} else {
		metrics.UploadSpeedMbps = uploadSpeed
		if uploadEstimated {
			metrics.UploadSource = nt.resolveUploadSource(uploadEstimated)
			metrics.UploadEstimated = true
			nt.GetLogger().Warn(fmt.Sprintf("上传速度为估算值: %.2f Mbps", uploadSpeed))
		} else {
			metrics.UploadSource = nt.resolveUploadSource(uploadEstimated)
			metrics.UploadEstimated = false
			nt.GetLogger().Info(fmt.Sprintf("上传速度: %.2f Mbps", uploadSpeed))
		}
	}

	// 计算总体评分
	score := nt.calculateScore(avgLatency, downloadSpeed, uploadSpeed, uploadEstimated)
	metrics.Score = score

	nt.GetLogger().Info(fmt.Sprintf("网络测试完成，评分: %.2f", score))

	return nt.CreateResult("success", metrics.ToMetricsMap(), ""), nil
}

func (nt *NetworkTest) resolveDownloadSource() string {
	if nt.backend != nil {
		return nt.backend.DownloadSource()
	}
	return models.NetworkDownloadSourceHTTP
}

func (nt *NetworkTest) resolveUploadSource(estimated bool) string {
	if nt.backend != nil {
		return nt.backend.UploadSource(estimated)
	}
	if estimated {
		return models.NetworkUploadSourceEstimated
	}
	return models.NetworkUploadSourceHTTP
}

// CheckConnectivity 检查网络连接
// 返回:
//   - bool: 是否连接
func (nt *NetworkTest) CheckConnectivity() bool {
	// 尝试连接到Google DNS
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		// 尝试连接到Cloudflare DNS
		conn, err = net.DialTimeout("tcp", "1.1.1.1:53", 3*time.Second)
		if err != nil {
			return false
		}
	}
	conn.Close()
	return true
}

// TestLatency 测试网络延迟
// 参数:
//   - hosts: 测试主机列表
//
// 返回:
//   - float64: 平均延迟（毫秒）
//   - error: 测试错误
func (nt *NetworkTest) TestLatency(hosts []string) (float64, error) {
	var totalLatency time.Duration
	successCount := 0

	for _, host := range hosts {
		latency, err := nt.pingHost(host)
		if err != nil {
			nt.GetLogger().Debug(fmt.Sprintf("Ping %s 失败: %v", host, err))
			continue
		}

		totalLatency += latency
		successCount++
		nt.GetLogger().Debug(fmt.Sprintf("Ping %s: %.2f ms", host, float64(latency.Microseconds())/1000.0))
	}

	if successCount == 0 {
		return 0, fmt.Errorf("所有主机ping失败")
	}

	// 计算平均延迟（毫秒）
	avgLatency := float64(totalLatency.Microseconds()) / float64(successCount) / 1000.0

	return avgLatency, nil
}

// pingHost 测试单个主机的延迟
// 参数:
//   - host: 主机地址
//
// 返回:
//   - time.Duration: 延迟时间
//   - error: 测试错误
func (nt *NetworkTest) pingHost(host string) (time.Duration, error) {
	startTime := time.Now()

	conn, err := net.DialTimeout("tcp", host, 3*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	latency := time.Since(startTime)
	return latency, nil
}

// TestDownloadSpeed 测试下载速度
// 使用公共测试文件
// 返回:
//   - float64: 下载速度（Mbps）
//   - error: 测试错误
func (nt *NetworkTest) TestDownloadSpeed() (float64, error) {
	testURL := "http://speedtest.tele2.net/1GB.zip"

	startTime := time.Now()

	resp, err := nt.httpClient.Get(testURL)
	if err != nil {
		return 0, fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 读取所有数据
	bytesRead, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取数据失败: %w", err)
	}

	duration := time.Since(startTime)

	// 计算速度（Mbps）
	// 字节 -> 比特 -> Mbps
	speed := float64(bytesRead*8) / duration.Seconds() / 1000000

	return speed, nil
}

// TestUploadSpeed 测试上传速度
// 当前实现会在缺乏可靠公共上传端点时回退到估算模式
// 返回:
//   - float64: 上传速度（Mbps）
//   - bool: 是否为估算值
//   - error: 测试错误
func (nt *NetworkTest) TestUploadSpeed(downloadSpeed float64) (float64, bool, error) {
	if downloadSpeed <= 0 {
		return 0, false, fmt.Errorf("缺少可用的下载测速结果，无法估算上传速度")
	}

	nt.GetLogger().Debug("上传速度测试当前使用估算值")

	estimatedUpload := downloadSpeed * 0.7
	if estimatedUpload <= 0 {
		return 0, false, fmt.Errorf("上传速度估算失败")
	}

	return estimatedUpload, true, nil
}

// calculateScore 计算网络性能评分
// 参数:
//   - latency: 延迟（毫秒）
//   - downloadSpeed: 下载速度（Mbps）
//   - uploadSpeed: 上传速度（Mbps）
//
// 返回:
//   - float64: 评分（0-100）
func (nt *NetworkTest) calculateScore(latency, downloadSpeed, uploadSpeed float64, uploadEstimated bool) float64 {
	// 基准值
	const baseLatency = 50.0        // ms（越低越好）
	const baseDownloadSpeed = 100.0 // Mbps
	const baseUploadSpeed = 50.0    // Mbps

	// 计算延迟评分（延迟越低分数越高）
	latencyScore := 0.0
	if latency > 0 {
		latencyScore = (baseLatency / latency) * 100
		if latencyScore > 100 {
			latencyScore = 100
		}
	}

	// 计算下载速度评分
	downloadScore := 0.0
	if downloadSpeed > 0 {
		downloadScore = (downloadSpeed / baseDownloadSpeed) * 100
		if downloadScore > 100 {
			downloadScore = 100
		}
	}

	// 计算上传速度评分
	uploadScore := 0.0
	if uploadSpeed > 0 {
		uploadScore = (uploadSpeed / baseUploadSpeed) * 100
		if uploadScore > 100 {
			uploadScore = 100
		}
	}

	latencyWeight := 0.4
	downloadWeight := 0.4
	uploadWeight := 0.2
	if uploadEstimated {
		uploadWeight = 0.0
	}

	totalWeight := latencyWeight + downloadWeight + uploadWeight
	if totalWeight == 0 {
		return 0
	}

	totalScore := latencyScore*latencyWeight + downloadScore*downloadWeight + uploadScore*uploadWeight

	return totalScore / totalWeight

}
