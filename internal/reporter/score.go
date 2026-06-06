// Package reporter 提供评分计算功能
package reporter

import (
	"fmt"

	"github.com/E8A281E6ACA2/perfassess/internal/config"
	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

const scoreCalibrationVersion = "2026-06-v1"

// ScoreCalculator 评分计算器
// 负责根据测试结果计算各项性能评分和综合评分
type ScoreCalculator struct {
	// weights 各项测试的权重配置
	// 默认: CPU 30%, Memory 20%, Disk 25%, Network 25%
	weights map[string]float64

	profile ScoreProfile
}

type ScoreProfile struct {
	Name                    string
	MemoryReadBaseMBps      float64
	MemoryWriteBaseMBps     float64
	DiskSequentialReadMBps  float64
	DiskSequentialWriteMBps float64
	DiskRandomIOPS          float64
	NetworkLatencyBaseMs    float64
	NetworkDownloadMbps     float64
	NetworkUploadMbps       float64
}

type gradeThreshold struct {
	Name    string
	Minimum float64
}

// NewScoreCalculator 创建新的评分计算器
// 使用默认权重配置
func NewScoreCalculator() *ScoreCalculator {
	return NewScoreCalculatorWithWeightsAndProfile(config.DefaultScoreWeights(), config.DefaultScoreProfile)
}

// NewScoreCalculatorWithWeights 创建带自定义权重的评分计算器
func NewScoreCalculatorWithWeights(weights map[string]float64) *ScoreCalculator {
	return NewScoreCalculatorWithWeightsAndProfile(weights, config.DefaultScoreProfile)
}

func NewScoreCalculatorWithWeightsAndProfile(weights map[string]float64, profileName string) *ScoreCalculator {
	if err := config.ValidateScoreWeights(weights); err != nil {
		weights = config.DefaultScoreWeights()
	}
	profile := scoreProfileByName(profileName)
	return &ScoreCalculator{
		weights: cloneWeights(weights),
		profile: profile,
	}
}

func scoreProfileByName(name string) ScoreProfile {
	switch name {
	case "vps":
		return ScoreProfile{
			Name:                    "vps",
			MemoryReadBaseMBps:      3000,
			MemoryWriteBaseMBps:     2000,
			DiskSequentialReadMBps:  300,
			DiskSequentialWriteMBps: 200,
			DiskRandomIOPS:          3000,
			NetworkLatencyBaseMs:    80,
			NetworkDownloadMbps:     50,
			NetworkUploadMbps:       25,
		}
	case "workstation":
		return ScoreProfile{
			Name:                    "workstation",
			MemoryReadBaseMBps:      8000,
			MemoryWriteBaseMBps:     6000,
			DiskSequentialReadMBps:  1500,
			DiskSequentialWriteMBps: 1000,
			DiskRandomIOPS:          20000,
			NetworkLatencyBaseMs:    30,
			NetworkDownloadMbps:     300,
			NetworkUploadMbps:       100,
		}
	default:
		return ScoreProfile{
			Name:                    "server",
			MemoryReadBaseMBps:      5000,
			MemoryWriteBaseMBps:     3000,
			DiskSequentialReadMBps:  500,
			DiskSequentialWriteMBps: 300,
			DiskRandomIOPS:          5000,
			NetworkLatencyBaseMs:    50,
			NetworkDownloadMbps:     100,
			NetworkUploadMbps:       50,
		}
	}
}

func gradeThresholds() []gradeThreshold {
	return []gradeThreshold{
		{Name: "优秀", Minimum: 90},
		{Name: "良好", Minimum: 75},
		{Name: "一般", Minimum: 60},
		{Name: "较差", Minimum: 0},
	}
}

func scoreProfilesByName() map[string]ScoreProfile {
	profiles := map[string]ScoreProfile{}
	for _, name := range []string{"vps", "server", "workstation"} {
		profiles[name] = scoreProfileByName(name)
	}
	return profiles
}

func (sc *ScoreCalculator) BuildScoreCalibration() map[string]interface{} {
	thresholds := make([]map[string]interface{}, 0, len(gradeThresholds()))
	for _, threshold := range gradeThresholds() {
		thresholds = append(thresholds, map[string]interface{}{
			"grade":     threshold.Name,
			"min_score": threshold.Minimum,
		})
	}

	return map[string]interface{}{
		"version":          scoreCalibrationVersion,
		"active_profile":   sc.profile.Name,
		"active_baselines": scoreProfileToMap(sc.profile),
		"grade_thresholds": thresholds,
		"profiles":         scoreProfilesToMap(scoreProfilesByName()),
		"notes": []string{
			"CPU 分数由测试后端归一化输出，score_profile 当前主要校准内存、磁盘和网络基准线。",
			"未执行、失败或降级测试不参与总分权重归一化，但会降低报告置信度并可能使等级标记为未完成。",
			"当前阈值是工程默认基线，后续应基于真实 VPS 样本回测继续校准。",
		},
	}
}

func scoreProfilesToMap(profiles map[string]ScoreProfile) map[string]interface{} {
	result := make(map[string]interface{}, len(profiles))
	for name, profile := range profiles {
		result[name] = scoreProfileToMap(profile)
	}
	return result
}

func scoreProfileToMap(profile ScoreProfile) map[string]interface{} {
	return map[string]interface{}{
		"memory_read_base_mbps":      profile.MemoryReadBaseMBps,
		"memory_write_base_mbps":     profile.MemoryWriteBaseMBps,
		"disk_sequential_read_mbps":  profile.DiskSequentialReadMBps,
		"disk_sequential_write_mbps": profile.DiskSequentialWriteMBps,
		"disk_random_iops":           profile.DiskRandomIOPS,
		"network_latency_base_ms":    profile.NetworkLatencyBaseMs,
		"network_download_base_mbps": profile.NetworkDownloadMbps,
		"network_upload_base_mbps":   profile.NetworkUploadMbps,
	}
}

// CalculateCPUScore 计算CPU性能评分
// 基于单核和多核测试结果计算评分（0-100）
func (sc *ScoreCalculator) CalculateCPUScore(result *models.TestResult) float64 {
	if result == nil || result.Status != models.TestStatusSuccess {
		return 0.0
	}

	// 优先使用测试已经计算好的总评分
	if totalScore, ok := getCPUScore(result); ok {
		return totalScore
	}

	// 备用方案：从单核和多核评分计算
	singleCoreScore, ok1 := result.Metrics["single_core_score"].(float64)
	multiCoreScore, ok2 := result.Metrics["multi_core_score"].(float64)

	if ok1 && ok2 {
		// 单核占30%，多核占70%
		score := singleCoreScore*0.3 + multiCoreScore*0.7

		// 限制在0-100范围内
		if score > 100.0 {
			score = 100.0
		}
		if score < 0.0 {
			score = 0.0
		}

		return score
	}

	return 0.0
}

// CalculateMemoryScore 计算内存性能评分
// 基于读写速度计算评分（0-100）
func (sc *ScoreCalculator) CalculateMemoryScore(result *models.TestResult) float64 {
	if result == nil || result.Status != models.TestStatusSuccess {
		return 0.0
	}

	// 从Metrics中提取内存测试结果（MB/s）
	readSpeed, ok1 := getMemoryReadSpeed(result)
	writeSpeed, ok2 := getMemoryWriteSpeed(result)

	if !ok1 || !ok2 {
		return 0.0
	}

	readScore := (readSpeed / sc.profile.MemoryReadBaseMBps) * 50.0
	writeScore := (writeSpeed / sc.profile.MemoryWriteBaseMBps) * 50.0

	score := readScore + writeScore

	// 限制在0-100范围内
	if score > 100.0 {
		score = 100.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// CalculateDiskScore 计算磁盘性能评分
// 基于顺序读写和随机IOPS计算评分（0-100）
func (sc *ScoreCalculator) CalculateDiskScore(result *models.TestResult) float64 {
	if result == nil || result.Status != models.TestStatusSuccess {
		return 0.0
	}

	// 从Metrics中提取磁盘测试结果
	seqRead, ok1 := getDiskReadSpeed(result)
	seqWrite, ok2 := getDiskWriteSpeed(result)
	randomIOPS, ok3 := getDiskRandomIOPS(result)

	if !ok1 || !ok2 || !ok3 {
		return 0.0
	}

	seqReadScore := (seqRead / sc.profile.DiskSequentialReadMBps) * 35.0
	seqWriteScore := (seqWrite / sc.profile.DiskSequentialWriteMBps) * 35.0
	iopsScore := (float64(randomIOPS) / sc.profile.DiskRandomIOPS) * 30.0

	score := seqReadScore + seqWriteScore + iopsScore

	// 限制在0-100范围内
	if score > 100.0 {
		score = 100.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// CalculateNetworkScore 计算网络性能评分
// 基于延迟、下载和上传速度计算评分（0-100）
func (sc *ScoreCalculator) CalculateNetworkScore(result *models.TestResult) float64 {
	if result == nil || result.Status != models.TestStatusSuccess {
		return 0.0
	}

	// 从Metrics中提取网络测试结果
	avgLatency, ok1 := getNetworkLatency(result)
	downloadSpeed, ok2 := getNetworkDownloadSpeed(result)
	uploadSpeed, ok3 := getNetworkUploadSpeed(result)
	uploadEstimated := isNetworkUploadEstimated(result)

	if !ok1 || !ok2 {
		return 0.0
	}
	if avgLatency <= 0 || downloadSpeed < 0 {
		return 0.0
	}

	latencyScore := (sc.profile.NetworkLatencyBaseMs / avgLatency) * 30.0
	if latencyScore > 30.0 {
		latencyScore = 30.0
	}

	downloadScore := (downloadSpeed / sc.profile.NetworkDownloadMbps) * 40.0

	score := latencyScore + downloadScore
	maxScore := 70.0
	if ok3 && uploadSpeed >= 0 && !uploadEstimated {
		uploadScore := (uploadSpeed / sc.profile.NetworkUploadMbps) * 30.0
		score += uploadScore
		maxScore = 100.0
	}

	if maxScore > 0 {
		score = score / maxScore * 100.0
	}

	// 限制在0-100范围内
	if score > 100.0 {
		score = 100.0
	}
	if score < 0.0 {
		score = 0.0
	}
	if uploadEstimated && score > 85.0 {
		score = 85.0
	}

	return score
}

// CalculateOverallScore 计算综合性能评分
// 使用加权平均算法计算总分，并确定性能等级
func (sc *ScoreCalculator) CalculateOverallScore(results *models.TestResults) *models.OverallScore {
	if results == nil {
		return &models.OverallScore{
			Grade: "未知",
		}
	}

	// 计算各项评分
	cpuScore := sc.CalculateCPUScore(results.CPUResult)
	memoryScore := sc.CalculateMemoryScore(results.MemoryResult)
	diskScore := sc.CalculateDiskScore(results.DiskResult)
	networkScore := sc.CalculateNetworkScore(results.NetworkResult)

	// 计算加权总分
	totalScore := cpuScore*sc.weights["cpu"] +
		memoryScore*sc.weights["memory"] +
		diskScore*sc.weights["disk"] +
		networkScore*sc.weights["network"]

	// 如果某些测试未执行，需要调整权重
	totalWeight := 0.0
	if results.CPUResult != nil && results.CPUResult.Status == models.TestStatusSuccess {
		totalWeight += sc.weights["cpu"]
	}
	if results.MemoryResult != nil && results.MemoryResult.Status == models.TestStatusSuccess {
		totalWeight += sc.weights["memory"]
	}
	if results.DiskResult != nil && results.DiskResult.Status == models.TestStatusSuccess {
		totalWeight += sc.weights["disk"]
	}
	if results.NetworkResult != nil && results.NetworkResult.Status == models.TestStatusSuccess {
		totalWeight += sc.weights["network"]
	}

	// 归一化总分
	if totalWeight > 0 {
		totalScore = totalScore / totalWeight
	} else {
		totalScore = 0.0
	}

	// 确定性能等级
	grade := sc.GetPerformanceGrade(totalScore)

	return &models.OverallScore{
		CPUScore:     cpuScore,
		MemoryScore:  memoryScore,
		DiskScore:    diskScore,
		NetworkScore: networkScore,
		TotalScore:   totalScore,
		Grade:        grade,
		Weights:      sc.weights,
	}
}

func (sc *ScoreCalculator) BuildScoreBreakdown(results *models.TestResults, overall *models.OverallScore) map[string]interface{} {
	if results == nil {
		return map[string]interface{}{
			"normalized_total": map[string]interface{}{
				"score":         0.0,
				"active_weight": 0.0,
				"formula":       "未获取测试结果，无法计算总分。",
			},
		}
	}

	breakdown := map[string]interface{}{
		"score_profile":       sc.profile.Name,
		"calibration_version": scoreCalibrationVersion,
		"cpu":                 sc.buildCPUScoreBreakdown(results.CPUResult),
		"memory":              sc.buildMemoryScoreBreakdown(results.MemoryResult),
		"disk":                sc.buildDiskScoreBreakdown(results.DiskResult),
		"network":             sc.buildNetworkScoreBreakdown(results.NetworkResult),
	}

	activeWeight := 0.0
	weightedSum := 0.0
	for _, key := range []string{"cpu", "memory", "disk", "network"} {
		item := breakdown[key].(map[string]interface{})
		active, _ := item["active"].(bool)
		score, _ := item["score"].(float64)
		weight := sc.weights[key]
		item["weight"] = weight
		if active {
			activeWeight += weight
			weightedSum += score * weight
		}
	}

	totalScore := 0.0
	if activeWeight > 0 {
		totalScore = weightedSum / activeWeight
	}
	if overall != nil {
		totalScore = overall.TotalScore
	}

	breakdown["normalized_total"] = map[string]interface{}{
		"score":               totalScore,
		"active_weight":       activeWeight,
		"weighted_sum":        weightedSum,
		"score_profile":       sc.profile.Name,
		"calibration_version": scoreCalibrationVersion,
		"formula":             "总分 = 已成功测试分项加权和 / 已成功测试权重和。",
	}
	return breakdown
}

func (sc *ScoreCalculator) buildCPUScoreBreakdown(result *models.TestResult) map[string]interface{} {
	score := sc.CalculateCPUScore(result)
	item := baseBreakdown(result, score, "CPU 分数优先使用 total_score；缺失时按单核 30% + 多核 70% 计算。")
	if result != nil && result.Metrics != nil {
		if single, ok := metricFloat64(result.Metrics, "single_core_score"); ok {
			item["single_core_score"] = single
		}
		if multi, ok := metricFloat64(result.Metrics, "multi_core_score"); ok {
			item["multi_core_score"] = multi
		}
		if events, ok := getCPUMultiCoreEvents(result); ok {
			item["multi_core_events_per_sec"] = events
		}
	}
	return item
}

func (sc *ScoreCalculator) buildMemoryScoreBreakdown(result *models.TestResult) map[string]interface{} {
	score := sc.CalculateMemoryScore(result)
	item := baseBreakdown(result, score, fmt.Sprintf("内存分数 = 读取相对 %.0f MB/s 最高 50 分 + 写入相对 %.0f MB/s 最高 50 分。", sc.profile.MemoryReadBaseMBps, sc.profile.MemoryWriteBaseMBps))
	item["read_base_mbps"] = sc.profile.MemoryReadBaseMBps
	item["write_base_mbps"] = sc.profile.MemoryWriteBaseMBps
	if result != nil && result.Metrics != nil {
		if read, ok := getMemoryReadSpeed(result); ok {
			item["read_speed_mbps"] = read
			item["read_score"] = clampMax(read/sc.profile.MemoryReadBaseMBps*50.0, 50.0)
		}
		if write, ok := getMemoryWriteSpeed(result); ok {
			item["write_speed_mbps"] = write
			item["write_score"] = clampMax(write/sc.profile.MemoryWriteBaseMBps*50.0, 50.0)
		}
	}
	return item
}

func (sc *ScoreCalculator) buildDiskScoreBreakdown(result *models.TestResult) map[string]interface{} {
	score := sc.CalculateDiskScore(result)
	item := baseBreakdown(result, score, fmt.Sprintf("磁盘分数 = 顺序读相对 %.0f MB/s 最高 35 分 + 顺序写相对 %.0f MB/s 最高 35 分 + 随机 IOPS 相对 %.0f 最高 30 分。", sc.profile.DiskSequentialReadMBps, sc.profile.DiskSequentialWriteMBps, sc.profile.DiskRandomIOPS))
	item["sequential_read_base_mbps"] = sc.profile.DiskSequentialReadMBps
	item["sequential_write_base_mbps"] = sc.profile.DiskSequentialWriteMBps
	item["random_iops_base"] = sc.profile.DiskRandomIOPS
	if result != nil && result.Metrics != nil {
		if read, ok := getDiskReadSpeed(result); ok {
			item["sequential_read_mbps"] = read
			item["sequential_read_score"] = clampMax(read/sc.profile.DiskSequentialReadMBps*35.0, 35.0)
		}
		if write, ok := getDiskWriteSpeed(result); ok {
			item["sequential_write_mbps"] = write
			item["sequential_write_score"] = clampMax(write/sc.profile.DiskSequentialWriteMBps*35.0, 35.0)
		}
		if iops, ok := getDiskRandomIOPS(result); ok {
			item["random_iops"] = iops
			item["random_iops_score"] = clampMax(float64(iops)/sc.profile.DiskRandomIOPS*30.0, 30.0)
		}
	}
	return item
}

func (sc *ScoreCalculator) buildNetworkScoreBreakdown(result *models.TestResult) map[string]interface{} {
	score := sc.CalculateNetworkScore(result)
	item := baseBreakdown(result, score, "网络分数 = 延迟、下载、真实上传归一化计算；估算上传不参与真实上传评分且网络评分上限为 85。")
	item["latency_base_ms"] = sc.profile.NetworkLatencyBaseMs
	item["download_base_mbps"] = sc.profile.NetworkDownloadMbps
	item["upload_base_mbps"] = sc.profile.NetworkUploadMbps
	if result != nil && result.Metrics != nil {
		uploadEstimated := isNetworkUploadEstimated(result)
		if latency, ok := getNetworkLatency(result); ok {
			item["average_latency_ms"] = latency
			if latency > 0 {
				item["latency_score"] = minFloat(sc.profile.NetworkLatencyBaseMs/latency*30.0, 30.0)
			}
		}
		if download, ok := getNetworkDownloadSpeed(result); ok {
			item["download_speed_mbps"] = download
			item["download_score"] = clampMax(download/sc.profile.NetworkDownloadMbps*40.0, 40.0)
		}
		if upload, ok := getNetworkUploadSpeed(result); ok {
			item["upload_speed_mbps"] = upload
			if uploadEstimated {
				item["upload_score"] = 0.0
			} else {
				item["upload_score"] = clampMax(upload/sc.profile.NetworkUploadMbps*30.0, 30.0)
			}
		}
		item["upload_estimated"] = uploadEstimated
	}
	return item
}

func baseBreakdown(result *models.TestResult, score float64, formula string) map[string]interface{} {
	active := result != nil && result.Status == models.TestStatusSuccess
	status := "not_run"
	if result != nil {
		status = result.Status
	}
	return map[string]interface{}{
		"active":  active,
		"status":  status,
		"score":   score,
		"formula": formula,
	}
}

func clampMax(score float64, max float64) float64 {
	if score > max {
		return max
	}
	if score < 0 {
		return 0
	}
	return score
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func cloneWeights(weights map[string]float64) map[string]float64 {
	cloned := make(map[string]float64, len(weights))
	for key, value := range weights {
		cloned[key] = value
	}
	return cloned
}

// GetPerformanceGrade 根据评分获取性能等级
// 评分范围：优秀(90+), 良好(75-89), 一般(60-74), 较差(<60)
func (sc *ScoreCalculator) GetPerformanceGrade(score float64) string {
	if score >= 90.0 {
		return "优秀"
	} else if score >= 75.0 {
		return "良好"
	} else if score >= 60.0 {
		return "一般"
	} else {
		return "较差"
	}
}
