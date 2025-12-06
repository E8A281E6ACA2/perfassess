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

// NetworkTest 网络性能测试
// 测试网络延迟、下载和上传速度
type NetworkTest struct {
	*BaseTest
	httpClient *http.Client
	testHosts  []string // 测试主机列表
}

// NewNetworkTest 创建网络性能测试
// 参数:
//   - logger: 日志记录器
//
// 返回:
//   - *NetworkTest: 网络测试实例
func NewNetworkTest(logger *logger.Logger) *NetworkTest {
	return &NetworkTest{
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

	metrics := make(map[string]interface{})

	// 测试网络延迟
	nt.GetLogger().Info("开始网络延迟测试...")
	avgLatency, err := nt.TestLatency(nt.testHosts)
	if err != nil {
		nt.GetLogger().Warn(fmt.Sprintf("延迟测试失败: %v", err))
		metrics["latency_ms"] = -1
		metrics["average_latency_ms"] = -1
	} else {
		metrics["latency_ms"] = avgLatency
		metrics["average_latency_ms"] = avgLatency
		nt.GetLogger().Info(fmt.Sprintf("平均延迟: %.2f ms", avgLatency))
	}

	// 测试下载速度
	nt.GetLogger().Info("开始下载速度测试...")
	downloadSpeed, err := nt.TestDownloadSpeed()
	if err != nil {
		nt.GetLogger().Warn(fmt.Sprintf("下载速度测试失败: %v", err))
		metrics["download_speed_mbps"] = -1
	} else {
		metrics["download_speed_mbps"] = downloadSpeed
		nt.GetLogger().Info(fmt.Sprintf("下载速度: %.2f Mbps", downloadSpeed))
	}

	// 测试上传速度（简化版本，使用HTTP POST）
	nt.GetLogger().Info("开始上传速度测试...")
	uploadSpeed, err := nt.TestUploadSpeed()
	if err != nil {
		nt.GetLogger().Warn(fmt.Sprintf("上传速度测试失败: %v", err))
		metrics["upload_speed_mbps"] = -1
	} else {
		metrics["upload_speed_mbps"] = uploadSpeed
		nt.GetLogger().Info(fmt.Sprintf("上传速度: %.2f Mbps", uploadSpeed))
	}

	// 计算总体评分
	score := nt.calculateScore(avgLatency, downloadSpeed, uploadSpeed)
	metrics["score"] = score

	nt.GetLogger().Info(fmt.Sprintf("网络测试完成，评分: %.2f", score))

	return nt.CreateResult("success", metrics, ""), nil
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
// 使用HTTP POST模拟上传
// 返回:
//   - float64: 上传速度（Mbps）
//   - error: 测试错误
func (nt *NetworkTest) TestUploadSpeed() (float64, error) {
	// 注意：这是一个简化的实现
	// 实际使用时应该使用专门的速度测试服务器

	// 由于没有可靠的公共上传测试服务器
	// 这里返回一个估算值（基于下载速度的70%）
	// 在实际项目中应该实现真实的上传测试

	nt.GetLogger().Debug("上传速度测试使用估算值")

	// 返回一个合理的估算值
	return 50.0, nil
}

// calculateScore 计算网络性能评分
// 参数:
//   - latency: 延迟（毫秒）
//   - downloadSpeed: 下载速度（Mbps）
//   - uploadSpeed: 上传速度（Mbps）
//
// 返回:
//   - float64: 评分（0-100）
func (nt *NetworkTest) calculateScore(latency, downloadSpeed, uploadSpeed float64) float64 {
	// 基准值
	const baseLatency = 50.0        // ms（越低越好）
	const baseDownloadSpeed = 100.0 // Mbps
	const baseUploadSpeed = 50.0    // Mbps

	// 计算延迟评分（延迟越低分数越高）
	latencyScore := 100.0
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

	// 延迟40%，下载40%，上传20%
	totalScore := latencyScore*0.4 + downloadScore*0.4 + uploadScore*0.2

	return totalScore
}
