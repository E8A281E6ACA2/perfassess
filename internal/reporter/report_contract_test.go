package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

func TestReportJSONSampleContract(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "report-json-sample.json"))
	if err != nil {
		t.Fatalf("failed to read sample report: %v", err)
	}
	schemaContent, err := os.ReadFile(filepath.Join("..", "..", "docs", "report.schema.json"))
	if err != nil {
		t.Fatalf("failed to read report schema: %v", err)
	}

	var sample map[string]interface{}
	if err := json.Unmarshal(content, &sample); err != nil {
		t.Fatalf("sample report is not valid JSON: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaContent, &schema); err != nil {
		t.Fatalf("report schema is not valid JSON: %v", err)
	}
	if err := validateJSONSchemaSubset(sample, schema, schema, "$"); err != nil {
		t.Fatalf("sample report does not match schema: %v", err)
	}

	requiredTopLevel := []string{"session_id", "timestamp", "system_info", "test_results", "summary"}
	for _, key := range requiredTopLevel {
		if _, ok := sample[key]; !ok {
			t.Fatalf("sample report missing top-level key %q", key)
		}
	}

	summary := objectAt(t, sample, "summary")
	for _, key := range []string{"benchmark_profile", "confidence_level", "score_calibration", "score_breakdown", "vps_benchmark_summary", "assessment_conclusion", "module_assessments", "share_templates"} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("sample summary missing key %q", key)
		}
	}
	assertExternalEvidenceSchema(t, schema)
	if summary["score_profile"] != "server" {
		t.Fatalf("expected sample score profile server, got %#v", summary["score_profile"])
	}

	profile := objectAt(t, summary, "benchmark_profile")
	if profile["name"] != "full_iperf3" {
		t.Fatalf("expected sample profile full_iperf3, got %#v", profile["name"])
	}

	confidence := objectAt(t, summary, "confidence_level")
	if confidence["level"] != "high" {
		t.Fatalf("expected sample confidence high, got %#v", confidence["level"])
	}

	calibration := objectAt(t, summary, "score_calibration")
	if calibration["version"] != scoreCalibrationVersion {
		t.Fatalf("expected sample calibration version %q, got %#v", scoreCalibrationVersion, calibration["version"])
	}
	if calibration["active_profile"] != "server" {
		t.Fatalf("expected sample calibration active profile server, got %#v", calibration["active_profile"])
	}
	profiles := objectAt(t, calibration, "profiles")
	if _, ok := profiles["server"]; !ok {
		t.Fatalf("expected sample calibration to include server profile, got %#v", profiles)
	}

	breakdown := objectAt(t, summary, "score_breakdown")
	if breakdown["calibration_version"] != scoreCalibrationVersion {
		t.Fatalf("expected score breakdown calibration version %q, got %#v", scoreCalibrationVersion, breakdown["calibration_version"])
	}
	normalized := objectAt(t, breakdown, "normalized_total")
	if _, ok := normalized["active_weight"].(float64); !ok {
		t.Fatalf("expected normalized active_weight number, got %#v", normalized["active_weight"])
	}

	vpsSummary := objectAt(t, summary, "vps_benchmark_summary")
	cpu := objectAt(t, vpsSummary, "cpu")
	if cpu["backend"] != "sysbench" {
		t.Fatalf("expected VPS summary CPU backend sysbench, got %#v", cpu["backend"])
	}
	network := objectAt(t, vpsSummary, "network")
	if network["backend"] != "iperf3" {
		t.Fatalf("expected VPS summary network backend iperf3, got %#v", network["backend"])
	}
	conclusion := objectAt(t, summary, "assessment_conclusion")
	if _, ok := conclusion["headline"].(string); !ok {
		t.Fatalf("expected assessment conclusion headline, got %#v", conclusion["headline"])
	}
	if evidence, ok := conclusion["evidence"].([]interface{}); !ok || len(evidence) == 0 {
		t.Fatalf("expected assessment conclusion evidence, got %#v", conclusion["evidence"])
	}
	if bottlenecks, ok := conclusion["bottlenecks"].([]interface{}); !ok || len(bottlenecks) != 4 {
		t.Fatalf("expected four bottlenecks, got %#v", conclusion["bottlenecks"])
	}
	modules := objectAt(t, summary, "module_assessments")
	networkModule := objectAt(t, modules, "network")
	if networkModule["confidence"] != "high" {
		t.Fatalf("expected network module confidence high, got %#v", networkModule["confidence"])
	}
	assertExternalEvidenceSample(t, summary)

	share := objectAt(t, summary, "share_templates")
	if _, ok := share["plain_text"].(string); !ok {
		t.Fatalf("expected plain text share template, got %#v", share["plain_text"])
	}
	if markdown, ok := share["markdown"].(string); !ok || !strings.Contains(markdown, "| 项目 | 结果 |") {
		t.Fatalf("expected markdown share template table, got %#v", share["markdown"])
	}
}

func TestGeneratedReportJSONMatchesSchema(t *testing.T) {
	generator := NewReportGeneratorWithWeights(map[string]float64{
		"cpu":     0.30,
		"memory":  0.20,
		"disk":    0.25,
		"network": 0.25,
	})
	report, err := generator.GenerateReport("schema_session", snapshotSystemInfo(), snapshotTestResults())
	if err != nil {
		t.Fatalf("expected report generation to succeed, got %v", err)
	}
	report.Timestamp = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	content, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}
	var actual map[string]interface{}
	if err := json.Unmarshal(content, &actual); err != nil {
		t.Fatalf("generated report is not valid JSON: %v", err)
	}

	schemaContent, err := os.ReadFile(filepath.Join("..", "..", "docs", "report.schema.json"))
	if err != nil {
		t.Fatalf("failed to read report schema: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaContent, &schema); err != nil {
		t.Fatalf("report schema is not valid JSON: %v", err)
	}
	if err := validateJSONSchemaSubset(actual, schema, schema, "$"); err != nil {
		t.Fatalf("generated report does not match schema: %v", err)
	}
}

func TestFormatReportSnapshot(t *testing.T) {
	generator := NewReportGeneratorWithWeightsAndProfile(map[string]float64{
		"cpu":     0.30,
		"memory":  0.20,
		"disk":    0.25,
		"network": 0.25,
	}, "server")
	report, err := generator.GenerateReport("snapshot_session", snapshotSystemInfo(), snapshotTestResults())
	if err != nil {
		t.Fatalf("expected report generation to succeed, got %v", err)
	}
	report.Timestamp = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	report.FormattedContent = generator.FormatReport(report)

	expected, err := os.ReadFile(filepath.Join("testdata", "text_report_snapshot.txt"))
	if err != nil {
		t.Fatalf("failed to read text report snapshot: %v", err)
	}

	actual := strings.TrimSpace(report.FormattedContent)
	want := strings.TrimSpace(string(expected))
	if actual != want {
		t.Fatalf("text report snapshot mismatch\n--- got ---\n%s\n--- want ---\n%s", actual, want)
	}
}

func objectAt(t *testing.T, source map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := source[key]
	if !ok {
		t.Fatalf("missing object key %q", key)
	}
	object, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected %q to be object, got %T", key, value)
	}
	return object
}

func assertExternalEvidenceSchema(t *testing.T, schema map[string]interface{}) {
	t.Helper()
	properties := objectAt(t, objectAt(t, schema, "properties"), "summary")
	summaryProperties := objectAt(t, properties, "properties")
	defs := objectAt(t, schema, "$defs")
	for _, key := range []string{"evidence_summary", "route_trace_result", "streaming_result", "ai_service_result"} {
		if _, ok := defs[key]; !ok {
			t.Fatalf("schema $defs missing %q", key)
		}
	}

	route := objectAt(t, summaryProperties, "route_trace_results")
	routeItems := objectAt(t, route, "items")
	if routeItems["$ref"] != "#/$defs/route_trace_result" {
		t.Fatalf("route_trace_results must reference route_trace_result, got %#v", routeItems["$ref"])
	}

	streaming := objectAt(t, summaryProperties, "streaming_results")
	streamingAdditional := objectAt(t, streaming, "additionalProperties")
	if streamingAdditional["$ref"] != "#/$defs/streaming_result" {
		t.Fatalf("streaming_results must reference streaming_result, got %#v", streamingAdditional["$ref"])
	}

	ai := objectAt(t, summaryProperties, "ai_results")
	aiAdditional := objectAt(t, ai, "additionalProperties")
	if aiAdditional["$ref"] != "#/$defs/ai_service_result" {
		t.Fatalf("ai_results must reference ai_service_result, got %#v", aiAdditional["$ref"])
	}
}

func assertExternalEvidenceSample(t *testing.T, summary map[string]interface{}) {
	t.Helper()

	routeResults := arrayAt(t, summary, "route_trace_results")
	if len(routeResults) == 0 {
		t.Fatalf("expected sample route trace results, got %#v", routeResults)
	}
	firstRoute, ok := routeResults[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first route result object, got %T", routeResults[0])
	}
	if firstRoute["is_real_return_route"] != false {
		t.Fatalf("sample built-in route must not be marked as real return route, got %#v", firstRoute["is_real_return_route"])
	}
	assertEvidenceCategories(t, firstRoute, "evidence_summary", []string{"direction", "visibility", "timeout", "return_route_boundary"})

	ipQuality := objectAt(t, summary, "ip_quality_report")
	assertEvidenceCategories(t, ipQuality, "evidence_summary", []string{"identity", "dnsbl", "mail", "heuristic", "external_api"})

	streamingResults := objectAt(t, summary, "streaming_results")
	netflix := objectAt(t, streamingResults, "Netflix")
	assertEvidenceCategories(t, netflix, "evidence_summary", []string{"availability", "region", "account"})

	aiResults := objectAt(t, summary, "ai_results")
	chatgpt := objectAt(t, aiResults, "ChatGPT")
	assertEvidenceCategories(t, chatgpt, "evidence_summary", []string{"access", "region", "account"})

	modules := objectAt(t, summary, "module_assessments")
	for _, key := range []string{"route", "ip_quality", "streaming", "ai_services"} {
		module := objectAt(t, modules, key)
		if module["status"] == "skipped" {
			t.Fatalf("sample module %q should exercise completed external evidence, got skipped", key)
		}
		evidence := arrayAt(t, module, "evidence")
		if len(evidence) == 0 {
			t.Fatalf("sample module %q missing evidence rows", key)
		}
	}
}

func assertEvidenceCategories(t *testing.T, source map[string]interface{}, key string, categories []string) {
	t.Helper()
	items := arrayAt(t, source, key)
	if len(items) == 0 {
		t.Fatalf("expected %q evidence items, got %#v", key, items)
	}
	seen := map[string]bool{}
	for _, item := range items {
		object, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected evidence item object, got %T", item)
		}
		category, ok := object["category"].(string)
		if !ok || category == "" {
			t.Fatalf("expected evidence category string, got %#v", object["category"])
		}
		for _, required := range []string{"label", "status", "confidence", "impact", "detail", "limitation"} {
			if _, ok := object[required].(string); !ok {
				t.Fatalf("expected evidence %q to contain string %q, got %#v", category, required, object[required])
			}
		}
		seen[category] = true
	}
	for _, category := range categories {
		if !seen[category] {
			t.Fatalf("expected evidence category %q in %q, got %#v", category, key, seen)
		}
	}
}

func arrayAt(t *testing.T, source map[string]interface{}, key string) []interface{} {
	t.Helper()
	value, ok := source[key]
	if !ok {
		t.Fatalf("missing array key %q", key)
	}
	array, ok := value.([]interface{})
	if !ok {
		t.Fatalf("expected %q to be array, got %T", key, value)
	}
	return array
}

func validateJSONSchemaSubset(value interface{}, schema map[string]interface{}, root map[string]interface{}, path string) error {
	if ref, ok := schema["$ref"].(string); ok {
		refSchema, err := resolveLocalSchemaRef(root, ref)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return validateJSONSchemaSubset(value, refSchema, root, path)
	}

	if schemaType, ok := schema["type"].(string); ok {
		if err := validateJSONType(value, schemaType, path); err != nil {
			return err
		}
	}

	if enumValues, ok := schema["enum"].([]interface{}); ok {
		matched := false
		for _, allowed := range enumValues {
			if value == allowed {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s: value %#v is not in enum %#v", path, value, enumValues)
		}
	}

	if objectValue, ok := value.(map[string]interface{}); ok {
		if required, ok := schema["required"].([]interface{}); ok {
			for _, item := range required {
				key, ok := item.(string)
				if !ok {
					return fmt.Errorf("%s: required entry must be string", path)
				}
				if _, exists := objectValue[key]; !exists {
					return fmt.Errorf("%s: missing required key %q", path, key)
				}
			}
		}
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for key, propertySchema := range properties {
				childValue, exists := objectValue[key]
				if !exists {
					continue
				}
				childSchema, ok := propertySchema.(map[string]interface{})
				if !ok {
					return fmt.Errorf("%s.%s: property schema must be object", path, key)
				}
				if err := validateJSONSchemaSubset(childValue, childSchema, root, path+"."+key); err != nil {
					return err
				}
			}
		}
	}

	if arrayValue, ok := value.([]interface{}); ok {
		if itemSchema, ok := schema["items"].(map[string]interface{}); ok {
			for i, item := range arrayValue {
				if err := validateJSONSchemaSubset(item, itemSchema, root, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func validateJSONType(value interface{}, schemaType string, path string) error {
	switch schemaType {
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("%s: expected object, got %T", path, value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("%s: expected array, got %T", path, value)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s: expected string, got %T", path, value)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%s: expected number, got %T", path, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: expected boolean, got %T", path, value)
		}
	default:
		return fmt.Errorf("%s: unsupported schema type %q", path, schemaType)
	}
	return nil
}

func resolveLocalSchemaRef(root map[string]interface{}, ref string) (map[string]interface{}, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("unsupported schema ref %q", ref)
	}
	current := interface{}(root)
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		object, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("schema ref %q crosses non-object", ref)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("schema ref %q not found", ref)
		}
	}
	refSchema, ok := current.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema ref %q target is not object", ref)
	}
	return refSchema, nil
}

func snapshotSystemInfo() *models.SystemInfo {
	return &models.SystemInfo{
		CPU: &models.CPUInfo{
			Model:        "Snapshot CPU",
			Cores:        4,
			Threads:      8,
			FrequencyMHz: 2400,
		},
		Memory: &models.MemoryInfo{
			TotalMB:     8192,
			AvailableMB: 4096,
			MemoryType:  "Unknown",
		},
		Disk: &models.DiskInfo{
			TotalGB:     100,
			AvailableGB: 60,
			DiskType:    "SSD",
		},
		OS: &models.OSInfo{
			Name:         "linux",
			Version:      "snapshot",
			Architecture: "x86_64",
		},
		Virtualization: &models.VirtualizationInfo{
			IsVirtualized: true,
			Type:          "KVM",
			Vendor:        "kvm",
		},
	}
}

func snapshotTestResults() *models.TestResults {
	return &models.TestResults{
		CPUResult: &models.TestResult{
			TestName:        "CPU性能测试",
			Status:          "success",
			DurationSeconds: 30,
			Metrics: map[string]interface{}{
				"backend":                    "sysbench",
				"single_core_score":          80.0,
				"single_core_events_per_sec": 1000.0,
				"multi_core_score":           90.0,
				"multi_core_events_per_sec":  8000.0,
				"total_score":                86.0,
				"cpu_cores":                  8,
			},
		},
		MemoryResult: &models.TestResult{
			TestName:        "内存性能测试",
			Status:          "success",
			DurationSeconds: 12,
			Metrics: map[string]interface{}{
				"backend":            "sysbench",
				"read_speed_mbps":    3200.0,
				"read_speed_source":  "sysbench",
				"write_speed_mbps":   2800.0,
				"write_speed_source": "sysbench",
				"score":              33.5,
			},
		},
		DiskResult: &models.TestResult{
			TestName:        "磁盘性能测试",
			Status:          "success",
			DurationSeconds: 20,
			Metrics: map[string]interface{}{
				"backend":                     "fio",
				"read_speed_mbps":             1000.0,
				"write_speed_mbps":            800.0,
				"random_iops":                 12000,
				"random_read_iops":            7000.0,
				"random_write_iops":           5000.0,
				"random_read_latency_p95_ms":  1.25,
				"random_write_latency_p95_ms": 2.5,
				"score":                       95.0,
			},
		},
		NetworkResult: &models.TestResult{
			TestName:        "网络性能测试",
			Status:          "success",
			DurationSeconds: 8,
			Metrics: map[string]interface{}{
				"backend":                          "iperf3",
				"backend_server":                   "127.0.0.1:5201",
				"average_latency_ms":               10.0,
				"latency_source":                   "tcp_connect",
				"download_speed_mbps":              900.0,
				"download_speed_source":            "iperf3_download",
				"upload_speed_mbps":                850.0,
				"upload_speed_source":              "iperf3_upload",
				"upload_speed_estimated":           false,
				"iperf3_matrix_profile":            "multi_server",
				"iperf3_matrix_server_count":       2,
				"iperf3_matrix_success_count":      2,
				"iperf3_matrix_avg_download_mbps":  850.0,
				"iperf3_matrix_avg_upload_mbps":    800.0,
				"iperf3_matrix_best_download_mbps": 900.0,
				"iperf3_matrix_best_upload_mbps":   850.0,
				"iperf3_matrix_1_server":           "127.0.0.1:5201",
				"iperf3_matrix_1_protocol":         "ipv4",
				"iperf3_matrix_1_latency_ms":       10.0,
				"iperf3_matrix_1_download_mbps":    900.0,
				"iperf3_matrix_1_upload_mbps":      850.0,
				"iperf3_matrix_2_server":           "[2001:db8::1]:5201",
				"iperf3_matrix_2_protocol":         "ipv6",
				"iperf3_matrix_2_latency_ms":       18.0,
				"iperf3_matrix_2_download_mbps":    800.0,
				"iperf3_matrix_2_upload_mbps":      750.0,
				"network_quality_profile":          "tcp_connect_matrix",
				"network_quality_target_count":     2,
				"network_quality_ipv4_available":   true,
				"network_quality_ipv6_available":   true,
				"network_quality_failure_rate":     0.0,
				"network_quality_avg_latency_ms":   8.0,
				"network_quality_jitter_ms":        0.8,
				"network_quality_1_target":         "cloudflare_ipv4_https",
				"network_quality_1_protocol":       "ipv4",
				"network_quality_1_available":      true,
				"network_quality_1_avg_latency_ms": 7.0,
				"network_quality_2_target":         "cloudflare_ipv6_https",
				"network_quality_2_protocol":       "ipv6",
				"network_quality_2_available":      true,
				"network_quality_2_avg_latency_ms": 9.0,
				"score":                            100.0,
			},
		},
	}
}
