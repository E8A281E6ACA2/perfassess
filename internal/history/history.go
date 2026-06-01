package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"performance-assessment-system/internal/models"
)

const DefaultStoreFile = "history.jsonl"

type Entry struct {
	ReportPath       string                 `json:"report_path"`
	SessionID        string                 `json:"session_id"`
	Timestamp        time.Time              `json:"timestamp"`
	HostID           string                 `json:"host_id"`
	CPUModel         string                 `json:"cpu_model,omitempty"`
	OS               string                 `json:"os,omitempty"`
	Architecture     string                 `json:"architecture,omitempty"`
	ScoreProfile     string                 `json:"score_profile"`
	BenchmarkProfile map[string]interface{} `json:"benchmark_profile,omitempty"`
	Grade            string                 `json:"grade"`
	TotalScore       float64                `json:"total_score"`
	CPUScore         float64                `json:"cpu_score"`
	MemoryScore      float64                `json:"memory_score"`
	DiskScore        float64                `json:"disk_score"`
	NetworkScore     float64                `json:"network_score"`
}

type Trend struct {
	HostID       string      `json:"host_id"`
	Count        int         `json:"count"`
	First        *Entry      `json:"first,omitempty"`
	Last         *Entry      `json:"last,omitempty"`
	TotalDelta   MetricDelta `json:"total_delta"`
	CPUDelta     MetricDelta `json:"cpu_delta"`
	MemoryDelta  MetricDelta `json:"memory_delta"`
	DiskDelta    MetricDelta `json:"disk_delta"`
	NetworkDelta MetricDelta `json:"network_delta"`
	Entries      []Entry     `json:"entries"`
}

type MetricDelta struct {
	First        float64 `json:"first"`
	Last         float64 `json:"last"`
	Delta        float64 `json:"delta"`
	DeltaPercent float64 `json:"delta_percent"`
}

func DefaultStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return DefaultStoreFile
	}
	return filepath.Join(home, ".perfassess", DefaultStoreFile)
}

func AddReport(storePath string, reportPath string) (*Entry, error) {
	entry, err := EntryFromFile(reportPath)
	if err != nil {
		return nil, err
	}
	if err := AppendEntry(storePath, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func EntryFromFile(reportPath string) (*Entry, error) {
	report, err := ReadReport(reportPath)
	if err != nil {
		return nil, err
	}
	entry := EntryFromReport(report, reportPath)
	return &entry, nil
}

func ReadReport(reportPath string) (*models.Report, error) {
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	var report models.Report
	if err := json.Unmarshal(content, &report); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	if report.Summary == nil {
		return nil, errors.New("报告缺少 summary")
	}
	return &report, nil
}

func EntryFromReport(report *models.Report, reportPath string) Entry {
	entry := Entry{
		ReportPath:       reportPath,
		SessionID:        report.SessionID,
		Timestamp:        report.Timestamp,
		HostID:           hostID(report),
		ScoreProfile:     scoreProfile(report),
		BenchmarkProfile: summaryObject(report, "benchmark_profile"),
		Grade:            summaryString(report, "grade"),
		TotalScore:       score(report, "total_score"),
		CPUScore:         score(report, "cpu_score"),
		MemoryScore:      score(report, "memory_score"),
		DiskScore:        score(report, "disk_score"),
		NetworkScore:     score(report, "network_score"),
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if report.SystemInfo != nil {
		if report.SystemInfo.CPU != nil {
			entry.CPUModel = report.SystemInfo.CPU.Model
		}
		if report.SystemInfo.OS != nil {
			entry.OS = strings.TrimSpace(strings.Join([]string{report.SystemInfo.OS.Name, report.SystemInfo.OS.Version}, " "))
			entry.Architecture = report.SystemInfo.OS.Architecture
		}
	}
	return entry
}

func AppendEntry(storePath string, entry *Entry) error {
	if entry == nil {
		return errors.New("历史条目为空")
	}
	if storePath == "" {
		storePath = DefaultStorePath()
	}
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(storePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(content, '\n')); err != nil {
		return err
	}
	return nil
}

func LoadEntries(storePath string) ([]Entry, error) {
	if storePath == "" {
		storePath = DefaultStorePath()
	}
	file, err := os.Open(storePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("解析历史库第 %d 行失败: %w", lineNo, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sortEntries(entries)
	return entries, nil
}

func BuildTrend(entries []Entry, hostID string) (*Trend, error) {
	if hostID == "" {
		hosts := map[string]bool{}
		for _, entry := range entries {
			hosts[entry.HostID] = true
		}
		if len(hosts) > 1 {
			return nil, errors.New("历史库包含多个 host_id，请使用 --host 指定主机")
		}
	}

	var filtered []Entry
	for _, entry := range entries {
		if hostID == "" || entry.HostID == hostID {
			filtered = append(filtered, entry)
		}
	}
	sortEntries(filtered)
	if len(filtered) == 0 {
		if hostID == "" {
			return nil, errors.New("历史库为空")
		}
		return nil, fmt.Errorf("未找到主机历史: %s", hostID)
	}

	first := filtered[0]
	last := filtered[len(filtered)-1]
	return &Trend{
		HostID:       first.HostID,
		Count:        len(filtered),
		First:        &first,
		Last:         &last,
		TotalDelta:   delta(first.TotalScore, last.TotalScore),
		CPUDelta:     delta(first.CPUScore, last.CPUScore),
		MemoryDelta:  delta(first.MemoryScore, last.MemoryScore),
		DiskDelta:    delta(first.DiskScore, last.DiskScore),
		NetworkDelta: delta(first.NetworkScore, last.NetworkScore),
		Entries:      filtered,
	}, nil
}

func EntriesFromDir(dir string) ([]Entry, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var entries []Entry
	var parseErrors []string
	for _, path := range matches {
		entry, err := EntryFromFile(path)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			continue
		}
		entries = append(entries, *entry)
	}
	if len(entries) == 0 && len(parseErrors) > 0 {
		return nil, fmt.Errorf("目录中没有可用 JSON 报告: %s", strings.Join(parseErrors, "; "))
	}
	return entries, nil
}

func SortEntries(entries []Entry, by string, desc bool) error {
	value := func(entry Entry) float64 {
		switch by {
		case "total":
			return entry.TotalScore
		case "cpu":
			return entry.CPUScore
		case "memory":
			return entry.MemoryScore
		case "disk":
			return entry.DiskScore
		case "network":
			return entry.NetworkScore
		default:
			return 0
		}
	}
	if !validSortKey(by) {
		return fmt.Errorf("无效排序字段: %s", by)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if desc {
			return value(entries[i]) > value(entries[j])
		}
		return value(entries[i]) < value(entries[j])
	})
	return nil
}

func validSortKey(key string) bool {
	switch key {
	case "total", "cpu", "memory", "disk", "network":
		return true
	default:
		return false
	}
}

func FormatEntriesText(entries []Entry) string {
	if len(entries) == 0 {
		return "暂无历史记录\n"
	}
	var sb strings.Builder
	sb.WriteString("=== 历史报告列表 ===\n\n")
	for _, entry := range entries {
		sb.WriteString(formatEntryLine(entry))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func FormatTrendText(trend *Trend) string {
	if trend == nil {
		return "趋势结果不可用\n"
	}
	var sb strings.Builder
	sb.WriteString("=== 历史趋势 ===\n\n")
	sb.WriteString(fmt.Sprintf("主机: %s\n", trend.HostID))
	sb.WriteString(fmt.Sprintf("样本数: %d\n", trend.Count))
	if trend.First != nil && trend.Last != nil {
		sb.WriteString(fmt.Sprintf("时间范围: %s -> %s\n\n", formatTime(trend.First.Timestamp), formatTime(trend.Last.Timestamp)))
	}
	sb.WriteString(fmt.Sprintf("总体评分: %.2f -> %.2f, 差值=%+.2f (%+.2f%%)\n", trend.TotalDelta.First, trend.TotalDelta.Last, trend.TotalDelta.Delta, trend.TotalDelta.DeltaPercent))
	sb.WriteString(fmt.Sprintf("CPU评分: %.2f -> %.2f, 差值=%+.2f (%+.2f%%)\n", trend.CPUDelta.First, trend.CPUDelta.Last, trend.CPUDelta.Delta, trend.CPUDelta.DeltaPercent))
	sb.WriteString(fmt.Sprintf("内存评分: %.2f -> %.2f, 差值=%+.2f (%+.2f%%)\n", trend.MemoryDelta.First, trend.MemoryDelta.Last, trend.MemoryDelta.Delta, trend.MemoryDelta.DeltaPercent))
	sb.WriteString(fmt.Sprintf("磁盘评分: %.2f -> %.2f, 差值=%+.2f (%+.2f%%)\n", trend.DiskDelta.First, trend.DiskDelta.Last, trend.DiskDelta.Delta, trend.DiskDelta.DeltaPercent))
	sb.WriteString(fmt.Sprintf("网络评分: %.2f -> %.2f, 差值=%+.2f (%+.2f%%)\n", trend.NetworkDelta.First, trend.NetworkDelta.Last, trend.NetworkDelta.Delta, trend.NetworkDelta.DeltaPercent))
	return sb.String()
}

func FormatRankText(entries []Entry, sortBy string) string {
	if len(entries) == 0 {
		return "目录中没有可用 JSON 报告\n"
	}
	var sb strings.Builder
	sb.WriteString("=== 报告批量排序 ===\n\n")
	sb.WriteString(fmt.Sprintf("排序字段: %s\n\n", sortBy))
	for i, entry := range entries {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, formatEntryLine(entry)))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func FormatJSON(value interface{}) (string, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func formatEntryLine(entry Entry) string {
	return fmt.Sprintf("%s | host=%s | profile=%s | benchmark=%s | total=%.2f | cpu=%.2f | mem=%.2f | disk=%.2f | net=%.2f | %s",
		formatTime(entry.Timestamp),
		emptyAsUnknown(entry.HostID),
		emptyAsUnknown(entry.ScoreProfile),
		benchmarkName(entry.BenchmarkProfile),
		entry.TotalScore,
		entry.CPUScore,
		entry.MemoryScore,
		entry.DiskScore,
		entry.NetworkScore,
		entry.ReportPath,
	)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "unknown-time"
	}
	return value.UTC().Format(time.RFC3339)
}

func benchmarkName(profile map[string]interface{}) string {
	if value, ok := profile["name"].(string); ok && value != "" {
		return value
	}
	return "unknown"
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
}

func delta(first float64, last float64) MetricDelta {
	d := last - first
	percent := 0.0
	if first != 0 {
		percent = d / first * 100
	}
	return MetricDelta{
		First:        first,
		Last:         last,
		Delta:        d,
		DeltaPercent: percent,
	}
}

func hostID(report *models.Report) string {
	if report == nil || report.SystemInfo == nil {
		return "unknown"
	}
	parts := []string{}
	if report.SystemInfo.OS != nil {
		parts = append(parts, report.SystemInfo.OS.Name, report.SystemInfo.OS.Architecture)
	}
	if report.SystemInfo.CPU != nil {
		parts = append(parts, report.SystemInfo.CPU.Model, strconv.Itoa(report.SystemInfo.CPU.Threads))
	}
	if report.SystemInfo.Memory != nil {
		parts = append(parts, strconv.FormatInt(report.SystemInfo.Memory.TotalMB, 10))
	}
	joined := strings.TrimSpace(strings.Join(parts, "|"))
	if joined == "" {
		return "unknown"
	}
	return joined
}

func score(report *models.Report, key string) float64 {
	if report == nil || report.Summary == nil {
		return 0
	}
	if value, ok := number(report.Summary[key]); ok {
		return value
	}
	if overall, ok := report.Summary["overall_score"].(map[string]interface{}); ok {
		if value, ok := number(overall[key]); ok {
			return value
		}
	}
	if breakdown, ok := report.Summary["score_breakdown"].(map[string]interface{}); ok {
		if key == "total_score" {
			if normalized, ok := breakdown["normalized_total"].(map[string]interface{}); ok {
				if value, ok := number(normalized["score"]); ok {
					return value
				}
			}
		}
		if component := scoreComponent(key); component != "" {
			if item, ok := breakdown[component].(map[string]interface{}); ok {
				if value, ok := number(item["score"]); ok {
					return value
				}
			}
		}
	}
	return 0
}

func scoreComponent(key string) string {
	switch key {
	case "cpu_score":
		return "cpu"
	case "memory_score":
		return "memory"
	case "disk_score":
		return "disk"
	case "network_score":
		return "network"
	default:
		return ""
	}
}

func scoreProfile(report *models.Report) string {
	if value := summaryString(report, "score_profile"); value != "" {
		return value
	}
	if report == nil || report.Summary == nil {
		return ""
	}
	if breakdown, ok := report.Summary["score_breakdown"].(map[string]interface{}); ok {
		if value, ok := breakdown["score_profile"].(string); ok {
			return value
		}
	}
	return ""
}

func summaryString(report *models.Report, key string) string {
	if report == nil || report.Summary == nil {
		return ""
	}
	if value, ok := report.Summary[key].(string); ok {
		return value
	}
	return ""
}

func summaryObject(report *models.Report, key string) map[string]interface{} {
	if report == nil || report.Summary == nil {
		return nil
	}
	if value, ok := report.Summary[key].(map[string]interface{}); ok {
		return value
	}
	return nil
}

func number(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
