package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
)

// AIService 表示单个 AI 服务的检测配置
type AIService struct {
	Name      string
	TestURL   string
	CheckFunc func(resp *http.Response, body string) (bool, string)
}

// AIServiceDetector 负责检测 AI 服务可用性
type AIServiceDetector struct {
	logger     *logger.Logger
	httpClient *http.Client
	services   map[string]AIService
}

// NewAIServiceDetector 创建 AI 服务检测器
func NewAIServiceDetector(logger *logger.Logger) *AIServiceDetector {
	detector := &AIServiceDetector{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("重定向次数过多")
				}
				return nil
			},
		},
		services: make(map[string]AIService),
	}

	detector.initDefaultServices()
	return detector
}

// initDefaultServices 初始化默认 AI 服务列表
func (ad *AIServiceDetector) initDefaultServices() {
	ad.services["ChatGPT"] = AIService{
		Name:    "ChatGPT",
		TestURL: "https://chat.openai.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string) {
			return interpretGenericAIResponse(resp, body, "需要登录")
		},
	}

	ad.services["Claude"] = AIService{
		Name:    "Claude",
		TestURL: "https://claude.ai/",
		CheckFunc: func(resp *http.Response, body string) (bool, string) {
			return interpretGenericAIResponse(resp, body, "需要登录")
		},
	}

	ad.services["Gemini"] = AIService{
		Name:    "Gemini",
		TestURL: "https://gemini.google.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string) {
			if strings.Contains(body, "region is not yet supported") {
				return false, "地区暂不支持"
			}
			return interpretGenericAIResponse(resp, body, "需要登录 Google 账号")
		},
	}

	ad.services["Mistral"] = AIService{
		Name:    "Mistral",
		TestURL: "https://chat.mistral.ai/",
		CheckFunc: func(resp *http.Response, body string) (bool, string) {
			return interpretGenericAIResponse(resp, body, "需要登录")
		},
	}

	ad.services["Copilot"] = AIService{
		Name:    "Copilot",
		TestURL: "https://copilot.microsoft.com/",
		CheckFunc: func(resp *http.Response, body string) (bool, string) {
			if strings.Contains(body, "Copilot is not available in your country") {
				return false, "地区限制"
			}
			return interpretGenericAIResponse(resp, body, "需要登录 Microsoft 账号")
		},
	}
}

// interpretGenericAIResponse 根据状态码输出通用信息
func interpretGenericAIResponse(resp *http.Response, body string, loginHint string) (bool, string) {
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 400:
		return true, "可访问"
	case resp.StatusCode == 401:
		return true, loginHint
	case resp.StatusCode == 403:
		if strings.Contains(body, "not available") || strings.Contains(body, "Not available") {
			return false, "地区限制"
		}
		return false, "访问被拒绝"
	case resp.StatusCode == 429:
		return true, "请求过于频繁或地区限制"
	default:
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}

// CheckService 检测单个 AI 服务
func (ad *AIServiceDetector) CheckService(name string) (*models.AIServiceResult, error) {
	service, ok := ad.services[name]
	if !ok {
		return nil, fmt.Errorf("未知的 AI 服务: %s", name)
	}

	ad.logger.Info(fmt.Sprintf("检测 AI 服务: %s", name))

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", service.TestURL, nil)
	if err != nil {
		return &models.AIServiceResult{
			Service:   name,
			Available: false,
			Message:   fmt.Sprintf("创建请求失败: %v", err),
		}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")

	resp, err := ad.httpClient.Do(req)
	if err != nil {
		ad.logger.Warn(fmt.Sprintf("检测 %s 失败: %v", name, err))
		return &models.AIServiceResult{
			Service:   name,
			Available: false,
			Message:   fmt.Sprintf("请求失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	limitedBody := io.LimitReader(resp.Body, 512*1024)
	bodyBytes, err := io.ReadAll(limitedBody)
	if err != nil {
		return &models.AIServiceResult{
			Service:   name,
			Available: false,
			Message:   fmt.Sprintf("读取响应失败: %v", err),
		}, nil
	}

	available, message := service.CheckFunc(resp, string(bodyBytes))

	result := &models.AIServiceResult{
		Service:   name,
		Available: available,
		Message:   message,
	}

	ad.logger.Info(fmt.Sprintf("检测 %s 完成: %s", name, message))
	return result, nil
}

// CheckAll 检测所有 AI 服务
func (ad *AIServiceDetector) CheckAll() (map[string]*models.AIServiceResult, error) {
	results := make(map[string]*models.AIServiceResult)
	for name := range ad.services {
		result, err := ad.CheckService(name)
		if err != nil {
			ad.logger.Warn(fmt.Sprintf("检测 %s 出错: %v", name, err))
			continue
		}
		results[name] = result
		time.Sleep(400 * time.Millisecond)
	}

	return results, nil
}

// CheckConnectivity 检查基础网络连通性
func (ad *AIServiceDetector) CheckConnectivity() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com", nil)
	if err != nil {
		return false
	}

	resp, err := ad.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
