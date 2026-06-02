package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"performance-assessment-system/internal/models"
)

const (
	geekbenchBinary              = "geekbench6"
	geekbenchSingleCoreBaseScore = 2500.0
	geekbenchMultiCoreBasePerCPU = 2500.0
)

type GeekbenchCPUBackend struct {
	timeout time.Duration
	runner  commandRunner
	result  *geekbenchCPUScore
}

type GeekbenchCPUConfig struct {
	Timeout time.Duration
	Runner  commandRunner
}

type geekbenchCPUScore struct {
	SingleCoreScore float64
	MultiCoreScore  float64
}

func NewGeekbenchCPUBackend(cfg GeekbenchCPUConfig) *GeekbenchCPUBackend {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 15 * time.Minute
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}
	return &GeekbenchCPUBackend{
		timeout: timeout,
		runner:  runner,
	}
}

func (b *GeekbenchCPUBackend) Name() string {
	return models.CPUBackendGeekbench
}

func (b *GeekbenchCPUBackend) SingleCoreSource() string {
	return models.CPUSourceGeekbench
}

func (b *GeekbenchCPUBackend) MultiCoreSource() string {
	return models.CPUSourceGeekbench
}

func (b *GeekbenchCPUBackend) MeasureSingleCore() (CPUBackendResult, error) {
	score, err := b.loadScore()
	if err != nil {
		return CPUBackendResult{}, err
	}
	return CPUBackendResult{
		Score:    scoreGeekbenchSingleCore(score.SingleCoreScore),
		RawScore: score.SingleCoreScore,
	}, nil
}

func (b *GeekbenchCPUBackend) MeasureMultiCore() (CPUBackendResult, error) {
	score, err := b.loadScore()
	if err != nil {
		return CPUBackendResult{}, err
	}
	return CPUBackendResult{
		Score:    scoreGeekbenchMultiCore(score.MultiCoreScore, runtime.NumCPU()),
		RawScore: score.MultiCoreScore,
	}, nil
}

func (b *GeekbenchCPUBackend) loadScore() (geekbenchCPUScore, error) {
	if b.result != nil {
		return *b.result, nil
	}
	score, err := b.runGeekbench()
	if err != nil {
		return geekbenchCPUScore{}, err
	}
	b.result = &score
	return score, nil
}

func (b *GeekbenchCPUBackend) runGeekbench() (geekbenchCPUScore, error) {
	if _, err := b.runner.LookPath(geekbenchBinary); err != nil {
		return geekbenchCPUScore{}, fmt.Errorf("geekbench6 is not installed; install it manually before using --cpu-backend geekbench. Download Geekbench 6 from https://www.geekbench.com/download/ and confirm geekbench6 is available in PATH")
	}

	outputFile, err := os.CreateTemp("", "perfassess-geekbench-*.json")
	if err != nil {
		return geekbenchCPUScore{}, fmt.Errorf("create geekbench export file: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		return geekbenchCPUScore{}, fmt.Errorf("close geekbench export file: %w", err)
	}
	defer os.Remove(outputPath)

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	if _, err := b.runner.Run(ctx, geekbenchBinary, "--no-upload", "--export-json", outputPath); err != nil {
		return geekbenchCPUScore{}, fmt.Errorf("geekbench6 cpu benchmark failed: %w", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		return geekbenchCPUScore{}, fmt.Errorf("read geekbench json export: %w", err)
	}

	score, err := parseGeekbenchCPUScores(content)
	if err != nil {
		return geekbenchCPUScore{}, fmt.Errorf("parse geekbench cpu result: %w", err)
	}
	return score, nil
}

func parseGeekbenchCPUScores(content []byte) (geekbenchCPUScore, error) {
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return geekbenchCPUScore{}, err
	}

	single := findGeekbenchScore(data, func(key string) bool {
		return strings.Contains(key, "single")
	})
	multi := findGeekbenchScore(data, func(key string) bool {
		return strings.Contains(key, "multi")
	})
	if single <= 0 || multi <= 0 {
		return geekbenchCPUScore{}, fmt.Errorf("missing single or multi core score")
	}
	return geekbenchCPUScore{
		SingleCoreScore: single,
		MultiCoreScore:  multi,
	}, nil
}

func findGeekbenchScore(value interface{}, keyMatches func(string) bool) float64 {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			if keyMatches(normalizeGeekbenchKey(key)) {
				if score := numericValue(child); score > 0 {
					return score
				}
				if score := findGenericScore(child); score > 0 {
					return score
				}
			}
		}
		for _, child := range typed {
			if score := findGeekbenchScore(child, keyMatches); score > 0 {
				return score
			}
		}
	case []interface{}:
		for _, child := range typed {
			if score := findGeekbenchScore(child, keyMatches); score > 0 {
				return score
			}
		}
	}
	return 0
}

func findGenericScore(value interface{}) float64 {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			normalized := normalizeGeekbenchKey(key)
			if normalized == "score" || strings.HasSuffix(normalized, "score") {
				if score := numericValue(child); score > 0 {
					return score
				}
			}
		}
		for _, child := range typed {
			if score := findGenericScore(child); score > 0 {
				return score
			}
		}
	case []interface{}:
		for _, child := range typed {
			if score := findGenericScore(child); score > 0 {
				return score
			}
		}
	}
	return 0
}

func normalizeGeekbenchKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")
	return key
}

func numericValue(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case json.Number:
		score, _ := typed.Float64()
		return score
	default:
		return 0
	}
}

func scoreGeekbenchSingleCore(rawScore float64) float64 {
	return capCPUScore(rawScore / geekbenchSingleCoreBaseScore * 100)
}

func scoreGeekbenchMultiCore(rawScore float64, cpuCount int) float64 {
	if cpuCount < 1 {
		cpuCount = 1
	}
	return capCPUScore(rawScore / (geekbenchMultiCoreBasePerCPU * float64(cpuCount)) * 100)
}

func capCPUScore(score float64) float64 {
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}
