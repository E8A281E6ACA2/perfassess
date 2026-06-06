// Package collector 提供IP信息收集功能
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/oschwald/geoip2-golang"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
	"github.com/E8A281E6ACA2/perfassess/pkg/utils"
)

// IPInfoCollector IP信息收集器
// 负责获取公网IP地址和地理位置信息
type IPInfoCollector struct {
	// httpClient HTTP客户端
	httpClient *http.Client

	// geoipDB GeoIP2数据库读取器（可选）
	geoipDB *geoip2.Reader

	// timeout 查询超时时间
	timeout time.Duration
}

// NewIPInfoCollector 创建IP信息收集器
// 参数:
//   - geoipDBPath: GeoIP2数据库文件路径（可选，为空则使用在线API）
//
// 返回:
//   - *IPInfoCollector: IP信息收集器实例
//   - error: 初始化错误
func NewIPInfoCollector(geoipDBPath string) (*IPInfoCollector, error) {
	collector := &IPInfoCollector{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		timeout: 5 * time.Second,
	}

	// 如果提供了GeoIP数据库路径，尝试加载
	if geoipDBPath != "" {
		db, err := geoip2.Open(geoipDBPath)
		if err != nil {
			// 数据库加载失败不是致命错误，降级使用在线API
			return collector, nil
		}
		collector.geoipDB = db
	}

	return collector, nil
}

// CollectAll 收集所有IP信息
// 包括公网IP、地理位置和ISP信息
// 返回:
//   - *models.IPInfo: IP信息
//   - error: 收集错误
func (iic *IPInfoCollector) CollectAll() (*models.IPInfo, error) {
	ctx := context.Background()

	result, err := utils.RunWithTimeoutAndResult(ctx, iic.timeout, func() (interface{}, error) {
		// 获取公网IP
		publicIP, err := iic.GetPublicIP()
		if err != nil {
			return nil, utils.WrapError(err, "获取公网IP失败")
		}

		ipInfo := &models.IPInfo{
			PublicIP:  publicIP,
			QueryTime: time.Now(),
		}

		// 获取地理位置信息
		geoLocation, err := iic.GetGeoLocation(publicIP)
		if err == nil {
			ipInfo.GeoLocation = geoLocation
		}

		// 获取ISP信息
		isp, err := iic.GetISPInfo(publicIP)
		if err == nil {
			ipInfo.ISP = isp
		}

		return ipInfo, nil
	})

	if err != nil {
		if utils.IsTimeout(err) {
			return nil, utils.WrapError(err, "IP信息查询超时")
		}
		return nil, err
	}

	return result.(*models.IPInfo), nil
}

// GetPublicIP 获取公网IP地址
// 使用 ipify.org API 获取
// 返回:
//   - string: 公网IP地址
//   - error: 获取错误
func (iic *IPInfoCollector) GetPublicIP() (string, error) {
	// 尝试多个API服务，提高成功率
	apis := []string{
		"https://api.ipify.org?format=text",
		"https://api64.ipify.org?format=text",
		"https://ifconfig.me/ip",
	}

	var lastErr error
	for _, apiURL := range apis {
		resp, err := iic.httpClient.Get(apiURL)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API返回状态码: %d", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		ip := string(body)
		if ip != "" {
			return ip, nil
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("获取公网IP失败: %w", lastErr)
	}
	return "", fmt.Errorf("获取公网IP失败: 所有API都不可用")
}

// GetGeoLocation 获取IP的地理位置信息
// 优先使用本地GeoIP数据库，否则使用在线API
// 参数:
//   - ip: IP地址
//
// 返回:
//   - *models.GeoLocation: 地理位置信息
//   - error: 查询错误
func (iic *IPInfoCollector) GetGeoLocation(ip string) (*models.GeoLocation, error) {
	// 如果有本地GeoIP数据库，优先使用
	if iic.geoipDB != nil {
		return iic.getGeoLocationFromDB(ip)
	}

	// 否则使用在线API
	return iic.getGeoLocationFromAPI(ip)
}

// getGeoLocationFromDB 从本地GeoIP数据库获取地理位置
// 参数:
//   - ip: IP地址
//
// 返回:
//   - *models.GeoLocation: 地理位置信息
//   - error: 查询错误
func (iic *IPInfoCollector) getGeoLocationFromDB(ip string) (*models.GeoLocation, error) {
	// 解析IP地址
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return nil, fmt.Errorf("无效的IP地址: %s", ip)
	}

	// 查询城市信息
	record, err := iic.geoipDB.City(netIP)
	if err != nil {
		return nil, fmt.Errorf("GeoIP查询失败: %w", err)
	}

	// 提取国家和城市名称（优先使用中文）
	country := record.Country.Names["zh-CN"]
	if country == "" {
		country = record.Country.Names["en"]
	}

	city := record.City.Names["zh-CN"]
	if city == "" {
		city = record.City.Names["en"]
	}

	return &models.GeoLocation{
		Country:     country,
		CountryCode: record.Country.IsoCode,
		City:        city,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
	}, nil
}

// getGeoLocationFromAPI 从在线API获取地理位置
// 使用 ip-api.com 免费API
// 参数:
//   - ip: IP地址
//
// 返回:
//   - *models.GeoLocation: 地理位置信息
//   - error: 查询错误
func (iic *IPInfoCollector) getGeoLocationFromAPI(ip string) (*models.GeoLocation, error) {
	// 使用 ip-api.com API（免费，无需密钥）
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)

	resp, err := iic.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("地理位置API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("地理位置API返回状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		ISP         string  `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析地理位置响应失败: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("地理位置查询失败")
	}

	return &models.GeoLocation{
		Country:     result.Country,
		CountryCode: result.CountryCode,
		City:        result.City,
		Latitude:    result.Lat,
		Longitude:   result.Lon,
	}, nil
}

// GetISPInfo 获取ISP信息
// 参数:
//   - ip: IP地址
//
// 返回:
//   - string: ISP名称
//   - error: 查询错误
func (iic *IPInfoCollector) GetISPInfo(ip string) (string, error) {
	// 使用 ip-api.com API 获取ISP信息
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?fields=isp", ip)

	resp, err := iic.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("ISP信息API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ISP信息API返回状态码: %d", resp.StatusCode)
	}

	var result struct {
		ISP string `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析ISP信息响应失败: %w", err)
	}

	return result.ISP, nil
}

// Close 关闭IP信息收集器
// 释放GeoIP数据库资源
func (iic *IPInfoCollector) Close() error {
	if iic.geoipDB != nil {
		return iic.geoipDB.Close()
	}
	return nil
}
