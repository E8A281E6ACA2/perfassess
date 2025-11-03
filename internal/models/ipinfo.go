// Package models 定义IP和地理位置相关的数据结构
package models

import "time"

// GeoLocation 表示地理位置信息
// 包含国家、城市和经纬度坐标
type GeoLocation struct {
	// Country 国家名称，如 "中国", "United States"
	Country string `json:"country"`
	
	// CountryCode 国家代码，如 "CN", "US"
	CountryCode string `json:"country_code"`
	
	// City 城市名称，如 "北京", "New York"
	City string `json:"city"`
	
	// Latitude 纬度坐标
	Latitude float64 `json:"latitude"`
	
	// Longitude 经度坐标
	Longitude float64 `json:"longitude"`
}

// IPInfo 表示IP地址和相关信息
// 包含公网IP、地理位置和ISP信息
type IPInfo struct {
	// PublicIP 公网IP地址
	PublicIP string `json:"public_ip"`
	
	// GeoLocation 地理位置信息
	GeoLocation *GeoLocation `json:"geo_location,omitempty"`
	
	// ISP 互联网服务提供商名称
	// 如 "中国电信", "Comcast"
	ISP string `json:"isp,omitempty"`
	
	// QueryTime 查询时间戳
	QueryTime time.Time `json:"query_time"`
}
