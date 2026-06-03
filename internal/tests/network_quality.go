package tests

import (
	"fmt"
	"math"
	"net"
	"time"
)

const networkQualityProfile = "tcp_connect_matrix"

type networkQualityTarget struct {
	Name     string
	Address  string
	Protocol string
}

type networkQualityResult struct {
	Target       string
	Address      string
	Protocol     string
	Available    bool
	SuccessCount int
	FailureCount int
	FailureRate  float64
	AvgLatencyMs float64
	MinLatencyMs float64
	MaxLatencyMs float64
	JitterMs     float64
}

func defaultNetworkQualityTargets() []networkQualityTarget {
	return []networkQualityTarget{
		{Name: "cloudflare_ipv4_https", Address: "1.1.1.1:443", Protocol: "ipv4"},
		{Name: "google_ipv4_dns", Address: "8.8.8.8:53", Protocol: "ipv4"},
		{Name: "cloudflare_ipv6_https", Address: "[2606:4700:4700::1111]:443", Protocol: "ipv6"},
		{Name: "google_ipv6_dns", Address: "[2001:4860:4860::8888]:53", Protocol: "ipv6"},
	}
}

func (nt *NetworkTest) MeasureNetworkQuality() []networkQualityResult {
	if len(nt.qualityTargets) == 0 {
		return nil
	}

	samples := nt.qualitySamples
	if samples <= 0 {
		samples = 1
	}

	timeout := nt.qualityDialTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	results := make([]networkQualityResult, 0, len(nt.qualityTargets))
	for _, target := range nt.qualityTargets {
		results = append(results, nt.measureNetworkQualityTarget(target, samples, timeout))
	}
	return results
}

func (nt *NetworkTest) measureNetworkQualityTarget(target networkQualityTarget, samples int, timeout time.Duration) networkQualityResult {
	latencies := make([]float64, 0, samples)
	failures := 0

	for i := 0; i < samples; i++ {
		latency, err := nt.dialQualityTarget(target.Address, timeout)
		if err != nil {
			failures++
			nt.GetLogger().Debug(fmt.Sprintf("网络质量采样失败: %s %s: %v", target.Name, target.Address, err))
			continue
		}
		latencies = append(latencies, float64(latency.Microseconds())/1000.0)
	}

	result := networkQualityResult{
		Target:       target.Name,
		Address:      target.Address,
		Protocol:     target.Protocol,
		Available:    len(latencies) > 0,
		SuccessCount: len(latencies),
		FailureCount: failures,
	}
	totalAttempts := len(latencies) + failures
	if totalAttempts > 0 {
		result.FailureRate = float64(failures) / float64(totalAttempts)
	}
	if len(latencies) == 0 {
		return result
	}

	result.AvgLatencyMs = averageFloat64(latencies)
	result.MinLatencyMs = minFloat64(latencies)
	result.MaxLatencyMs = maxFloat64(latencies)
	result.JitterMs = stddevFloat64(latencies, result.AvgLatencyMs)
	return result
}

func (nt *NetworkTest) dialQualityTarget(address string, timeout time.Duration) (time.Duration, error) {
	if nt.qualityDialFn != nil {
		return nt.qualityDialFn(address, timeout)
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}

func (nt *NetworkTest) appendNetworkQualityMetrics(metrics map[string]interface{}) {
	results := nt.MeasureNetworkQuality()
	if len(results) == 0 {
		return
	}

	metrics["network_quality_profile"] = networkQualityProfile
	metrics["network_quality_target_count"] = len(results)

	totalSuccess := 0
	totalFailure := 0
	weightedLatency := 0.0
	weightedJitter := 0.0
	protocolStats := map[string]struct {
		success int
		failure int
	}{}

	for index, result := range results {
		prefix := fmt.Sprintf("network_quality_%d", index+1)
		metrics[prefix+"_target"] = result.Target
		metrics[prefix+"_address"] = result.Address
		metrics[prefix+"_protocol"] = result.Protocol
		metrics[prefix+"_available"] = result.Available
		metrics[prefix+"_success_count"] = result.SuccessCount
		metrics[prefix+"_failure_count"] = result.FailureCount
		metrics[prefix+"_failure_rate"] = result.FailureRate
		metrics[prefix+"_avg_latency_ms"] = result.AvgLatencyMs
		metrics[prefix+"_min_latency_ms"] = result.MinLatencyMs
		metrics[prefix+"_max_latency_ms"] = result.MaxLatencyMs
		metrics[prefix+"_jitter_ms"] = result.JitterMs

		totalSuccess += result.SuccessCount
		totalFailure += result.FailureCount
		if result.SuccessCount > 0 {
			weightedLatency += result.AvgLatencyMs * float64(result.SuccessCount)
			weightedJitter += result.JitterMs * float64(result.SuccessCount)
		}
		stats := protocolStats[result.Protocol]
		stats.success += result.SuccessCount
		stats.failure += result.FailureCount
		protocolStats[result.Protocol] = stats
	}

	metrics["network_quality_ipv4_available"] = protocolStats["ipv4"].success > 0
	metrics["network_quality_ipv6_available"] = protocolStats["ipv6"].success > 0
	metrics["network_quality_ipv4_failure_rate"] = protocolFailureRate(protocolStats["ipv4"].success, protocolStats["ipv4"].failure)
	metrics["network_quality_ipv6_failure_rate"] = protocolFailureRate(protocolStats["ipv6"].success, protocolStats["ipv6"].failure)
	metrics["network_quality_failure_rate"] = protocolFailureRate(totalSuccess, totalFailure)

	if totalSuccess > 0 {
		metrics["network_quality_avg_latency_ms"] = weightedLatency / float64(totalSuccess)
		metrics["network_quality_jitter_ms"] = weightedJitter / float64(totalSuccess)
	} else {
		metrics["network_quality_avg_latency_ms"] = 0.0
		metrics["network_quality_jitter_ms"] = 0.0
	}
}

func protocolFailureRate(successCount, failureCount int) float64 {
	total := successCount + failureCount
	if total == 0 {
		return 0
	}
	return float64(failureCount) / float64(total)
}

func averageFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func minFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func maxFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func stddevFloat64(values []float64, average float64) float64 {
	if len(values) == 0 {
		return 0
	}
	totalVariance := 0.0
	for _, value := range values {
		delta := value - average
		totalVariance += delta * delta
	}
	return math.Sqrt(totalVariance / float64(len(values)))
}
