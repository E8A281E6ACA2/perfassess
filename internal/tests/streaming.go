// Package tests 提供流媒体解锁检测功能
package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/logger"
)

// StreamingPlatform 表示一个流媒体平台的检测配置
type StreamingPlatform struct {
	// Name 平台名称
	Name string

	// TestURL 测试URL
	TestURL string

	// CheckFunc 检测函数，根据HTTP响应判断是否可访问
	CheckFunc func(response *http.Response, body string) (bool, string, string)
}

// StreamingDetector 流媒体检测器
// 负责检测各流媒体平台的可访问性
type StreamingDetector struct {
	// logger 日志记录器
	logger *logger.Logger

	// httpClient HTTP客户端
	httpClient *http.Client

	// platforms 支持的平台配置
	platforms map[string]StreamingPlatform

	// platformOrder 固定检测顺序，避免 map 遍历顺序导致报告抖动
	platformOrder []string

	// profile 当前检测档位
	profile string
}

// NewStreamingDetector 创建新的流媒体检测器
func NewStreamingDetector(logger *logger.Logger) *StreamingDetector {
	detector := &StreamingDetector{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// 允许重定向，但记录重定向链
				if len(via) >= 10 {
					return fmt.Errorf("重定向次数过多")
				}
				return nil
			},
		},
		platforms:     make(map[string]StreamingPlatform),
		platformOrder: []string{},
		profile:       "quick",
	}

	// 初始化默认平台
	detector.initDefaultPlatforms()

	return detector
}

// NewStreamingDetectorWithProfile 创建指定档位的流媒体检测器
func NewStreamingDetectorWithProfile(logger *logger.Logger, profile string) *StreamingDetector {
	detector := NewStreamingDetector(logger)
	detector.SetProfile(profile)
	return detector
}

// SetProfile 设置检测档位。quick 保持低耗时，standard 覆盖主流平台，full 增加区域型平台。
func (sd *StreamingDetector) SetProfile(profile string) {
	switch profile {
	case "quick", "standard", "full":
		sd.profile = profile
	default:
		sd.profile = "quick"
	}
}

// initDefaultPlatforms 初始化默认支持的流媒体平台
func (sd *StreamingDetector) initDefaultPlatforms() {
	// Netflix
	sd.platforms["Netflix"] = StreamingPlatform{
		Name:    "Netflix",
		TestURL: "https://www.netflix.com/title/70143836",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				if strings.Contains(body, "Not Available") || strings.Contains(body, "not available") {
					return false, "", "不可用"
				}
				// 尝试检测区域
				region := "Unknown"
				if strings.Contains(body, "\"country\":\"US\"") {
					region = "US"
				} else if strings.Contains(body, "\"country\":\"JP\"") {
					region = "JP"
				}
				return true, region, "完全解锁"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// YouTube Premium
	sd.platforms["YouTube"] = StreamingPlatform{
		Name:    "YouTube",
		TestURL: "https://www.youtube.com/premium",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				return true, "Global", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// Disney+
	sd.platforms["Disney+"] = StreamingPlatform{
		Name:    "Disney+",
		TestURL: "https://www.disneyplus.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				if strings.Contains(body, "not available") || strings.Contains(body, "unavailable") {
					return false, "", "不可用"
				}
				return true, "Unknown", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// HBO Max
	sd.platforms["HBO Max"] = StreamingPlatform{
		Name:    "HBO Max",
		TestURL: "https://www.hbomax.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				if strings.Contains(body, "not available") {
					return false, "", "不可用"
				}
				return true, "Unknown", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// Amazon Prime Video
	sd.platforms["Prime Video"] = StreamingPlatform{
		Name:    "Prime Video",
		TestURL: "https://www.primevideo.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				return true, "Unknown", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// Hulu
	sd.platforms["Hulu"] = StreamingPlatform{
		Name:    "Hulu",
		TestURL: "https://www.hulu.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				if strings.Contains(body, "not available in your region") {
					return false, "", "地区限制"
				}
				return true, "US", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// Paramount+
	sd.platforms["Paramount+"] = StreamingPlatform{
		Name:    "Paramount+",
		TestURL: "https://www.paramountplus.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				if strings.Contains(body, "not yet available") {
					return false, "", "地区限制"
				}
				return true, "Unknown", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	// BBC iPlayer
	sd.platforms["BBC iPlayer"] = StreamingPlatform{
		Name:    "BBC iPlayer",
		TestURL: "https://www.bbc.co.uk/iplayer",
		CheckFunc: func(resp *http.Response, body string) (bool, string, string) {
			if resp.StatusCode == 200 {
				if strings.Contains(body, "BBC iPlayer only works in the UK") {
					return false, "", "地区限制"
				}
				return true, "UK", "可访问"
			}
			return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
		},
	}

	sd.platforms["Apple TV+"] = StreamingPlatform{
		Name:      "Apple TV+",
		TestURL:   "https://tv.apple.com/",
		CheckFunc: basicStatusCheck("Unknown"),
	}

	sd.platforms["Spotify"] = StreamingPlatform{
		Name:      "Spotify",
		TestURL:   "https://www.spotify.com/",
		CheckFunc: basicStatusCheck("Unknown"),
	}

	sd.platforms["TikTok"] = StreamingPlatform{
		Name:      "TikTok",
		TestURL:   "https://www.tiktok.com/",
		CheckFunc: basicStatusCheck("Unknown"),
	}

	sd.platforms["DAZN"] = StreamingPlatform{
		Name:      "DAZN",
		TestURL:   "https://www.dazn.com/",
		CheckFunc: basicStatusCheck("Unknown"),
	}

	sd.platforms["Abema"] = StreamingPlatform{
		Name:      "Abema",
		TestURL:   "https://abema.tv/",
		CheckFunc: basicStatusCheck("JP"),
	}

	sd.platforms["Niconico"] = StreamingPlatform{
		Name:      "Niconico",
		TestURL:   "https://www.nicovideo.jp/",
		CheckFunc: basicStatusCheck("JP"),
	}

	sd.platforms["Bilibili"] = StreamingPlatform{
		Name:      "Bilibili",
		TestURL:   "https://www.bilibili.com/",
		CheckFunc: basicStatusCheck("CN"),
	}

	sd.platforms["TVB Anywhere"] = StreamingPlatform{
		Name:      "TVB Anywhere",
		TestURL:   "https://www.tvbanywhere.com/",
		CheckFunc: basicStatusCheck("HK"),
	}

	sd.platformOrder = []string{
		"Netflix",
		"YouTube",
		"Disney+",
		"Prime Video",
		"HBO Max",
		"Hulu",
		"Paramount+",
		"BBC iPlayer",
		"Apple TV+",
		"Spotify",
		"TikTok",
		"DAZN",
		"Abema",
		"Niconico",
		"Bilibili",
		"TVB Anywhere",
	}
}

func basicStatusCheck(region string) func(*http.Response, string) (bool, string, string) {
	return func(resp *http.Response, body string) (bool, string, string) {
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			lowered := strings.ToLower(body)
			if strings.Contains(lowered, "not available") || strings.Contains(lowered, "unavailable in your") || strings.Contains(lowered, "not yet available") {
				return false, "", "地区限制"
			}
			return true, region, "可访问"
		}
		return false, "", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}

func (sd *StreamingDetector) platformsForProfile() []string {
	switch sd.profile {
	case "full":
		return sd.orderedPlatforms()
	case "standard":
		return []string{"Netflix", "YouTube", "Disney+", "Prime Video", "HBO Max", "Hulu", "Paramount+", "BBC iPlayer", "Apple TV+", "Spotify", "TikTok", "DAZN"}
	default:
		return []string{"Netflix", "YouTube", "Disney+", "Prime Video", "HBO Max", "Hulu", "Paramount+", "BBC iPlayer"}
	}
}

func (sd *StreamingDetector) orderedPlatforms() []string {
	seen := make(map[string]bool, len(sd.platformOrder))
	platforms := make([]string, 0, len(sd.platforms))
	for _, name := range sd.platformOrder {
		if _, exists := sd.platforms[name]; exists {
			platforms = append(platforms, name)
			seen[name] = true
		}
	}
	custom := make([]string, 0)
	for name := range sd.platforms {
		if !seen[name] {
			custom = append(custom, name)
		}
	}
	sort.Strings(custom)
	return append(platforms, custom...)
}

// CheckPlatform 检测单个平台
// 参数:
//   - platformName: 平台名称
//
// 返回:
//   - *models.StreamingResult: 检测结果
//   - error: 检测错误
func (sd *StreamingDetector) CheckPlatform(platformName string) (*models.StreamingResult, error) {
	platform, exists := sd.platforms[platformName]
	if !exists {
		return nil, fmt.Errorf("不支持的平台: %s", platformName)
	}

	sd.logger.Info(fmt.Sprintf("检测流媒体平台: %s", platformName))

	// 创建请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", platform.TestURL, nil)
	if err != nil {
		return &models.StreamingResult{
			Platform:  platformName,
			Available: false,
			Message:   fmt.Sprintf("创建请求失败: %v", err),
		}, err
	}

	// 设置User-Agent模拟浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// 发送请求
	resp, err := sd.httpClient.Do(req)
	if err != nil {
		sd.logger.Warn(fmt.Sprintf("检测 %s 失败: %v", platformName, err))
		return &models.StreamingResult{
			Platform:  platformName,
			Available: false,
			Message:   fmt.Sprintf("请求失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &models.StreamingResult{
			Platform:  platformName,
			Available: false,
			Message:   fmt.Sprintf("读取响应失败: %v", err),
		}, nil
	}
	body := string(bodyBytes)

	// 使用平台特定的检测函数
	available, region, message := platform.CheckFunc(resp, body)

	result := &models.StreamingResult{
		Platform:  platformName,
		Available: available,
		Region:    region,
		Message:   message,
	}

	sd.logger.Info(fmt.Sprintf("检测 %s 完成: %s", platformName, message))

	return result, nil
}

// CheckAll 检测所有平台
// 返回:
//   - map[string]*models.StreamingResult: 所有平台的检测结果
//   - error: 检测错误
func (sd *StreamingDetector) CheckAll() (map[string]*models.StreamingResult, error) {
	platforms := sd.platformsForProfile()
	sd.logger.Info(fmt.Sprintf("开始检测流媒体平台，档位: %s，数量: %d", sd.profile, len(platforms)))

	results := make(map[string]*models.StreamingResult)

	// 串行检测（避免并发请求被识别为攻击）
	for _, platformName := range platforms {
		result, err := sd.CheckPlatform(platformName)
		if err != nil {
			sd.logger.Error(fmt.Sprintf("检测 %s 出错", platformName), err)
			// 继续检测其他平台
			continue
		}
		results[platformName] = result

		// 短暂延迟，避免请求过快
		time.Sleep(500 * time.Millisecond)
	}

	sd.logger.Info(fmt.Sprintf("流媒体检测完成，成功: %d/%d", len(results), len(platforms)))

	return results, nil
}

// AddCustomPlatform 添加自定义平台
// 参数:
//   - platform: 平台配置
func (sd *StreamingDetector) AddCustomPlatform(platform StreamingPlatform) {
	sd.platforms[platform.Name] = platform
	sd.platformOrder = append(sd.platformOrder, platform.Name)
	sd.logger.Info(fmt.Sprintf("添加自定义平台: %s", platform.Name))
}

// GetSupportedPlatforms 获取支持的平台列表
// 返回:
//   - []string: 平台名称列表
func (sd *StreamingDetector) GetSupportedPlatforms() []string {
	return sd.orderedPlatforms()
}

// CheckConnectivity 检查网络连接性
// 返回:
//   - bool: 是否有网络连接
func (sd *StreamingDetector) CheckConnectivity() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com", nil)
	if err != nil {
		return false
	}

	resp, err := sd.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
