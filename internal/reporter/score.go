// Package reporter 提供评分计算功能
package reporter

import (
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
		weights: map[string]float64{
			"cpu":     0.30,
			"memory":  0.20,
			"disk":    0.25,
			"network": 0.25,
		},
	}
}

// NewScoreCalculatorWithWeights 创建带自定义权重的评分计算器
func NewScoreCalculatorWithWeights(weights map[string]float64) *ScoreCalculator {
	return &ScoreCalculator{
		weights: weights,
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
