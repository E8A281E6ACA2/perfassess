// Package models 定义压力测试相关的数据结构
package models

// StressComponentResult 表示单个压力测试组件的结果
type StressComponentResult struct {
	Name               string                 `json:"name"`
	DurationSeconds    float64                `json:"duration_seconds"`
	Status             string                 `json:"status"`
	AverageTemperature float64                `json:"average_temperature_c,omitempty"`
	PeakTemperature    float64                `json:"peak_temperature_c,omitempty"`
	Notes              string                 `json:"notes,omitempty"`
	Metrics            map[string]interface{} `json:"metrics,omitempty"`
}

// StressTestReport 表示整体压力测试的结果
type StressTestReport struct {
	TotalDurationSeconds float64                   `json:"total_duration_seconds"`
	Components           []*StressComponentResult  `json:"components"`
	TemperatureAvailable bool                      `json:"temperature_available"`
}
