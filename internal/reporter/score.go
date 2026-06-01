// Package reporter 提供评分计算功能
package reporter

import (
	"performance-assessment-system/internal/config"
	"performance-assessment-system/internal/models"
)

// ScoreCalculator 评分计算器
// 负责根据测试结果计算各项性能评分和综合评分
type ScoreCalculator struct {
	// weights 各项测试的权重配置
	// 默认: CPU 30%, Memory 20%, Disk 25%, Network 25%
	weights map[string]float64
}

// NewScoreCalculator 创建新的评分计算器
// 使用默认权重配置
func NewScoreCalculator() *ScoreCalculator {
	return &ScoreCalculator{
		weights: config.DefaultScoreWeights(),
	}
}

// NewScoreCalculatorWithWeights 创建带自定义权重的评分计算器
func NewScoreCalculatorWithWeights(weights map[string]float64) *ScoreCalculator {
	if err := config.ValidateScoreWeights(weights); err != nil {
		weights = config.DefaultScoreWeights()
	}
	return &ScoreCalculator{
		weights: cloneWeights(weights),
	}
}

// CalculateCPUScore 计算CPU性能评分
// 基于单核和多核测试结果计算评分（0-100）
func (sc *ScoreCalculator) CalculateCPUScore(result *models.TestResult) float64 {
	if result == nil || result.Status != "success" {
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
	if result == nil || result.Status != "success" {
		return 0.0
	}

	// 从Metrics中提取内存测试结果（MB/s）
	readSpeed, ok1 := getMemoryReadSpeed(result)
	writeSpeed, ok2 := getMemoryWriteSpeed(result)

	if !ok1 || !ok2 {
		return 0.0
	}

	// 基准值：读取5000 MB/s，写入3000 MB/s为60分
	readBase := 5000.0
	writeBase := 3000.0

	readScore := (readSpeed / readBase) * 50.0
	writeScore := (writeSpeed / writeBase) * 50.0

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
	if result == nil || result.Status != "success" {
		return 0.0
	}

	// 从Metrics中提取磁盘测试结果
	seqRead, ok1 := getDiskReadSpeed(result)
	seqWrite, ok2 := getDiskWriteSpeed(result)
	randomIOPS, ok3 := getDiskRandomIOPS(result)

	if !ok1 || !ok2 || !ok3 {
		return 0.0
	}

	// 基准值：顺序读500 MB/s，顺序写300 MB/s，随机IOPS 5000为60分
	seqReadBase := 500.0
	seqWriteBase := 300.0
	iopsBase := 5000.0

	seqReadScore := (seqRead / seqReadBase) * 35.0
	seqWriteScore := (seqWrite / seqWriteBase) * 35.0
	iopsScore := (float64(randomIOPS) / iopsBase) * 30.0

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
	if result == nil || result.Status != "success" {
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

	// 延迟评分：延迟越低分数越高（50ms为60分，反比例）
	latencyBase := 50.0
	latencyScore := (latencyBase / avgLatency) * 30.0
	if latencyScore > 30.0 {
		latencyScore = 30.0
	}

	// 下载速度评分：100 Mbps为60分
	downloadBase := 100.0
	downloadScore := (downloadSpeed / downloadBase) * 40.0

	score := latencyScore + downloadScore
	maxScore := 70.0
	if ok3 && uploadSpeed >= 0 && !uploadEstimated {
		uploadBase := 50.0
		uploadScore := (uploadSpeed / uploadBase) * 30.0
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
	if results.CPUResult != nil && results.CPUResult.Status == "success" {
		totalWeight += sc.weights["cpu"]
	}
	if results.MemoryResult != nil && results.MemoryResult.Status == "success" {
		totalWeight += sc.weights["memory"]
	}
	if results.DiskResult != nil && results.DiskResult.Status == "success" {
		totalWeight += sc.weights["disk"]
	}
	if results.NetworkResult != nil && results.NetworkResult.Status == "success" {
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
		"cpu":     sc.buildCPUScoreBreakdown(results.CPUResult),
		"memory":  sc.buildMemoryScoreBreakdown(results.MemoryResult),
		"disk":    sc.buildDiskScoreBreakdown(results.DiskResult),
		"network": sc.buildNetworkScoreBreakdown(results.NetworkResult),
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
		"score":         totalScore,
		"active_weight": activeWeight,
		"weighted_sum":  weightedSum,
		"formula":       "总分 = 已成功测试分项加权和 / 已成功测试权重和。",
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
	item := baseBreakdown(result, score, "内存分数 = 读取相对 5000 MB/s 最高 50 分 + 写入相对 3000 MB/s 最高 50 分。")
	item["read_base_mbps"] = 5000.0
	item["write_base_mbps"] = 3000.0
	if result != nil && result.Metrics != nil {
		if read, ok := getMemoryReadSpeed(result); ok {
			item["read_speed_mbps"] = read
			item["read_score"] = clampMax(read/5000.0*50.0, 50.0)
		}
		if write, ok := getMemoryWriteSpeed(result); ok {
			item["write_speed_mbps"] = write
			item["write_score"] = clampMax(write/3000.0*50.0, 50.0)
		}
	}
	return item
}

func (sc *ScoreCalculator) buildDiskScoreBreakdown(result *models.TestResult) map[string]interface{} {
	score := sc.CalculateDiskScore(result)
	item := baseBreakdown(result, score, "磁盘分数 = 顺序读相对 500 MB/s 最高 35 分 + 顺序写相对 300 MB/s 最高 35 分 + 随机 IOPS 相对 5000 最高 30 分。")
	item["sequential_read_base_mbps"] = 500.0
	item["sequential_write_base_mbps"] = 300.0
	item["random_iops_base"] = 5000.0
	if result != nil && result.Metrics != nil {
		if read, ok := getDiskReadSpeed(result); ok {
			item["sequential_read_mbps"] = read
			item["sequential_read_score"] = clampMax(read/500.0*35.0, 35.0)
		}
		if write, ok := getDiskWriteSpeed(result); ok {
			item["sequential_write_mbps"] = write
			item["sequential_write_score"] = clampMax(write/300.0*35.0, 35.0)
		}
		if iops, ok := getDiskRandomIOPS(result); ok {
			item["random_iops"] = iops
			item["random_iops_score"] = clampMax(float64(iops)/5000.0*30.0, 30.0)
		}
	}
	return item
}

func (sc *ScoreCalculator) buildNetworkScoreBreakdown(result *models.TestResult) map[string]interface{} {
	score := sc.CalculateNetworkScore(result)
	item := baseBreakdown(result, score, "网络分数 = 延迟、下载、真实上传归一化计算；估算上传不参与真实上传评分且网络评分上限为 85。")
	item["latency_base_ms"] = 50.0
	item["download_base_mbps"] = 100.0
	item["upload_base_mbps"] = 50.0
	if result != nil && result.Metrics != nil {
		if latency, ok := getNetworkLatency(result); ok {
			item["average_latency_ms"] = latency
			if latency > 0 {
				item["latency_score"] = minFloat(50.0/latency*30.0, 30.0)
			}
		}
		if download, ok := getNetworkDownloadSpeed(result); ok {
			item["download_speed_mbps"] = download
			item["download_score"] = download / 100.0 * 40.0
		}
		if upload, ok := getNetworkUploadSpeed(result); ok {
			item["upload_speed_mbps"] = upload
			item["upload_score"] = upload / 50.0 * 30.0
		}
		item["upload_estimated"] = isNetworkUploadEstimated(result)
	}
	return item
}

func baseBreakdown(result *models.TestResult, score float64, formula string) map[string]interface{} {
	active := result != nil && result.Status == "success"
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
