package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	LookPath(name string) (string, error)
}

type execCommandRunner struct{}

func (r *execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func (r *execCommandRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

type Iperf3NetworkBackend struct {
	server     string
	servers    []string
	serverFile string
	timeout    time.Duration
	runner     commandRunner
	latencyFn  func([]string) (float64, error)
	matrix     []iperf3ServerResult
	matrixErr  error
}

type Iperf3Config struct {
	Server     string
	Servers    []string
	ServerFile string
	Timeout    time.Duration
	Runner     commandRunner
	LatencyFn  func([]string) (float64, error)
}

type iperf3Endpoint struct {
	Host string
	Port string
}

type iperf3ServerSpec struct {
	Server        string
	Name          string
	Region        string
	Provider      string
	Authorization string
}

func (e iperf3Endpoint) TCPAddress() string {
	port := e.Port
	if port == "" {
		port = "5201"
	}
	return net.JoinHostPort(e.Host, port)
}

func NewIperf3NetworkBackend(cfg Iperf3Config) *Iperf3NetworkBackend {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	runner := cfg.Runner
	if runner == nil {
		runner = &execCommandRunner{}
	}
	servers := normalizeIperf3Servers(cfg.Server, cfg.Servers)

	return &Iperf3NetworkBackend{
		server:     cfg.Server,
		servers:    servers,
		serverFile: cfg.ServerFile,
		timeout:    timeout,
		runner:     runner,
		latencyFn:  cfg.LatencyFn,
	}
}

func (b *Iperf3NetworkBackend) Name() string {
	return models.NetworkBackendIperf3
}

func (b *Iperf3NetworkBackend) Server() string {
	if len(b.servers) > 0 {
		return strings.Join(b.servers, ",")
	}
	if b.serverFile != "" {
		return b.serverFile
	}
	return b.server
}

func (b *Iperf3NetworkBackend) DownloadSource(result NetworkDownloadResult) string {
	return models.NetworkDownloadSourceIperf3
}

func (b *Iperf3NetworkBackend) UploadSource(estimated bool) string {
	return models.NetworkUploadSourceIperf3
}

func (b *Iperf3NetworkBackend) MeasureLatency(hosts []string) (float64, error) {
	if b.latencyFn == nil {
		return 0, fmt.Errorf("iperf3 latency fallback is not configured")
	}
	return b.latencyFn(hosts)
}

func (b *Iperf3NetworkBackend) MeasureDownload() (NetworkDownloadResult, error) {
	servers, err := b.resolveServers()
	if err != nil {
		return NetworkDownloadResult{}, err
	}
	if len(servers) > 1 {
		results, err := b.loadMatrix()
		if err != nil {
			return NetworkDownloadResult{}, err
		}
		speed := averageSuccessfulIperf3Download(results)
		if speed <= 0 {
			return NetworkDownloadResult{}, newBenchmarkError(BenchmarkErrorRuntime, "iperf3_matrix_download", "所有 iperf3 节点下载均失败，请检查节点连通性、防火墙和服务端配置。", fmt.Errorf("all iperf3 matrix download tests failed"))
		}
		return NetworkDownloadResult{SpeedMbps: speed, SourceURL: models.NetworkDownloadSourceIperf3}, nil
	}

	endpoint, err := b.validateReady(servers)
	if err != nil {
		return NetworkDownloadResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, false)...)
	if err != nil {
		return NetworkDownloadResult{}, classifyExternalCommandError("iperf3_download_run", output, err, "确认 iperf3 服务端可达、端口开放，并检查本机出站防火墙。")
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return NetworkDownloadResult{}, newBenchmarkError(BenchmarkErrorParseFailed, "iperf3_download_parse", "iperf3 JSON 输出格式无法识别，请保留 stdout/stderr 用于排查。", err)
	}
	return NetworkDownloadResult{SpeedMbps: speed, SourceURL: models.NetworkDownloadSourceIperf3}, nil
}

func (b *Iperf3NetworkBackend) MeasureUpload(downloadSpeed float64) (float64, bool, error) {
	servers, err := b.resolveServers()
	if err != nil {
		return 0, false, err
	}
	if len(servers) > 1 {
		results, err := b.loadMatrix()
		if err != nil {
			return 0, false, err
		}
		speed := averageSuccessfulIperf3Upload(results)
		if speed <= 0 {
			return 0, false, newBenchmarkError(BenchmarkErrorRuntime, "iperf3_matrix_upload", "所有 iperf3 节点上传均失败，请检查节点反向测试支持、防火墙和服务端配置。", fmt.Errorf("all iperf3 matrix upload tests failed"))
		}
		return speed, false, nil
	}

	endpoint, err := b.validateReady(servers)
	if err != nil {
		return 0, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, true)...)
	if err != nil {
		return 0, false, classifyExternalCommandError("iperf3_upload_run", output, err, "确认 iperf3 服务端支持 reverse 上传测试，并检查服务端/安全组端口。")
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return 0, false, newBenchmarkError(BenchmarkErrorParseFailed, "iperf3_upload_parse", "iperf3 JSON 输出格式无法识别，请保留 stdout/stderr 用于排查。", err)
	}
	return speed, false, nil
}

type iperf3ServerResult struct {
	Server        string
	Name          string
	Region        string
	Provider      string
	Authorization string
	Endpoint      iperf3Endpoint
	Protocol      string
	DownloadMbps  float64
	UploadMbps    float64
	LatencyMs     float64
	Error         string
	ErrorCategory string
	ErrorStage    string
	ErrorHint     string
}

func (b *Iperf3NetworkBackend) AppendMetrics(metrics map[string]interface{}) {
	if len(b.matrix) == 0 {
		return
	}

	metrics["iperf3_matrix_profile"] = "multi_server"
	metrics["iperf3_matrix_server_count"] = len(b.matrix)
	metrics["iperf3_matrix_success_count"] = countSuccessfulIperf3Results(b.matrix)
	if avg := averageSuccessfulIperf3Download(b.matrix); avg > 0 {
		metrics["iperf3_matrix_avg_download_mbps"] = avg
	}
	if avg := averageSuccessfulIperf3Upload(b.matrix); avg > 0 {
		metrics["iperf3_matrix_avg_upload_mbps"] = avg
	}
	if best := bestIperf3Download(b.matrix); best > 0 {
		metrics["iperf3_matrix_best_download_mbps"] = best
	}
	if best := bestIperf3Upload(b.matrix); best > 0 {
		metrics["iperf3_matrix_best_upload_mbps"] = best
	}

	for i, result := range b.matrix {
		prefix := fmt.Sprintf("iperf3_matrix_%d", i+1)
		metrics[prefix+"_server"] = result.Server
		if result.Name != "" {
			metrics[prefix+"_name"] = result.Name
		}
		if result.Region != "" {
			metrics[prefix+"_region"] = result.Region
		}
		if result.Provider != "" {
			metrics[prefix+"_provider"] = result.Provider
		}
		if result.Authorization != "" {
			metrics[prefix+"_authorization"] = result.Authorization
		}
		metrics[prefix+"_host"] = result.Endpoint.Host
		metrics[prefix+"_protocol"] = result.Protocol
		if result.Endpoint.Port != "" {
			metrics[prefix+"_port"] = result.Endpoint.Port
		}
		if result.DownloadMbps > 0 {
			metrics[prefix+"_download_mbps"] = result.DownloadMbps
		}
		if result.UploadMbps > 0 {
			metrics[prefix+"_upload_mbps"] = result.UploadMbps
		}
		if result.LatencyMs > 0 {
			metrics[prefix+"_latency_ms"] = result.LatencyMs
		}
		if result.Error != "" {
			metrics[prefix+"_error"] = result.Error
		}
		if result.ErrorCategory != "" {
			metrics[prefix+"_error_category"] = result.ErrorCategory
		}
		if result.ErrorStage != "" {
			metrics[prefix+"_error_stage"] = result.ErrorStage
		}
		if result.ErrorHint != "" {
			metrics[prefix+"_error_hint"] = result.ErrorHint
		}
	}
}

func (b *Iperf3NetworkBackend) loadMatrix() ([]iperf3ServerResult, error) {
	if b.matrix != nil || b.matrixErr != nil {
		return b.matrix, b.matrixErr
	}
	b.matrix, b.matrixErr = b.runMatrix()
	return b.matrix, b.matrixErr
}

func (b *Iperf3NetworkBackend) runMatrix() ([]iperf3ServerResult, error) {
	if _, err := b.runner.LookPath("iperf3"); err != nil {
		return nil, newBenchmarkError(BenchmarkErrorMissingDependency, "iperf3_lookup", "安装 iperf3 后重试。Ubuntu/Debian: sudo apt install iperf3；RHEL/CentOS: sudo yum install iperf3；macOS: brew install iperf3", err)
	}
	servers, err := b.resolveServers()
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 192.0.2.10:5201 (documentation-only address, replace with your owned or authorized node)")
	}

	specs := b.resolveServerSpecs(servers)
	results := make([]iperf3ServerResult, 0, len(servers))
	for _, spec := range specs {
		result := iperf3ServerResult{
			Server:        spec.Server,
			Name:          spec.Name,
			Region:        spec.Region,
			Provider:      spec.Provider,
			Authorization: spec.Authorization,
		}
		endpoint, err := parseIperf3Endpoint(spec.Server)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		result.Endpoint = endpoint
		result.Protocol = iperf3Protocol(endpoint.Host)
		if b.latencyFn != nil {
			if latency, err := b.latencyFn([]string{endpoint.TCPAddress()}); err == nil {
				result.LatencyMs = latency
			}
		}
		if speed, err := b.runIperf3(endpoint, false); err != nil {
			result.Error = appendIperf3Error(result.Error, "download: "+err.Error())
			setIperf3ServerResultError(&result, err)
		} else {
			result.DownloadMbps = speed
		}
		if speed, err := b.runIperf3(endpoint, true); err != nil {
			result.Error = appendIperf3Error(result.Error, "upload: "+err.Error())
			setIperf3ServerResultError(&result, err)
		} else {
			result.UploadMbps = speed
		}
		results = append(results, result)
	}
	if countSuccessfulIperf3Results(results) == 0 {
		return results, newBenchmarkError(BenchmarkErrorRuntime, "iperf3_matrix_run", "所有 iperf3 节点均失败，请检查节点文件、端口、安全组和服务端进程。", fmt.Errorf("all iperf3 matrix tests failed"))
	}
	return results, nil
}

func (b *Iperf3NetworkBackend) runIperf3(endpoint iperf3Endpoint, reverse bool) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, reverse)...)
	if err != nil {
		return 0, classifyExternalCommandError("iperf3_matrix_run", output, err, "确认 iperf3 节点可达、端口开放，并检查服务端是否允许测试。")
	}
	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return 0, newBenchmarkError(BenchmarkErrorParseFailed, "iperf3_matrix_parse", "iperf3 JSON 输出格式无法识别，请保留 stdout/stderr 用于排查。", err)
	}
	return speed, nil
}

func (b *Iperf3NetworkBackend) validateReady(servers []string) (iperf3Endpoint, error) {
	if len(servers) == 0 {
		return iperf3Endpoint{}, newBenchmarkError(BenchmarkErrorInvalidConfig, "iperf3_config", "提供自有或授权的 --iperf3-server、--iperf3-servers 或 --iperf3-server-file；推荐使用节点文件声明 auth=owned 或 auth=authorized。", fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 192.0.2.10:5201 (documentation-only address, replace with your owned or authorized node)"))
	}
	endpoint, err := parseIperf3Endpoint(servers[0])
	if err != nil {
		return iperf3Endpoint{}, newBenchmarkError(BenchmarkErrorInvalidConfig, "iperf3_config", "服务端格式应为 host、host:port 或 [IPv6]:port。", err)
	}
	if _, err := b.runner.LookPath("iperf3"); err != nil {
		return iperf3Endpoint{}, newBenchmarkError(BenchmarkErrorMissingDependency, "iperf3_lookup", "安装 iperf3 后重试。Ubuntu/Debian: sudo apt install iperf3；RHEL/CentOS: sudo yum install iperf3；macOS: brew install iperf3", err)
	}
	return endpoint, nil
}

func (b *Iperf3NetworkBackend) commandArgs(endpoint iperf3Endpoint, reverse bool) []string {
	args := []string{"-c", endpoint.Host}
	if endpoint.Port != "" {
		args = append(args, "-p", endpoint.Port)
	}
	if reverse {
		args = append(args, "--reverse")
	}
	return append(args, "--json")
}

func parseIperf3Endpoint(server string) (iperf3Endpoint, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return iperf3Endpoint{}, fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 192.0.2.10:5201 (documentation-only address, replace with your owned or authorized node)")
	}

	host, port, err := net.SplitHostPort(server)
	if err == nil {
		if host == "" {
			return iperf3Endpoint{}, fmt.Errorf("iperf3 server host is required")
		}
		if err := validateIperf3Port(port); err != nil {
			return iperf3Endpoint{}, err
		}
		return iperf3Endpoint{Host: host, Port: port}, nil
	}

	if strings.Count(server, ":") > 1 {
		return iperf3Endpoint{}, fmt.Errorf("invalid iperf3 server %q; use [ipv6]:port or host:port", server)
	}
	if strings.Contains(server, ":") {
		lastColon := strings.LastIndex(server, ":")
		host = strings.TrimSpace(server[:lastColon])
		port = strings.TrimSpace(server[lastColon+1:])
		if host == "" {
			return iperf3Endpoint{}, fmt.Errorf("iperf3 server host is required")
		}
		if err := validateIperf3Port(port); err != nil {
			return iperf3Endpoint{}, err
		}
		return iperf3Endpoint{Host: host, Port: port}, nil
	}

	return iperf3Endpoint{Host: server}, nil
}

func validateIperf3Port(port string) error {
	if port == "" {
		return fmt.Errorf("iperf3 server port is required when using host:port")
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("invalid iperf3 server port %q; expected 1-65535", port)
	}
	return nil
}

func normalizeIperf3Servers(single string, servers []string) []string {
	seen := map[string]bool{}
	normalized := []string{}
	add := func(server string) {
		server = strings.TrimSpace(server)
		if server == "" || seen[server] {
			return
		}
		seen[server] = true
		normalized = append(normalized, server)
	}
	add(single)
	for _, server := range servers {
		add(server)
	}
	return normalized
}

func (b *Iperf3NetworkBackend) resolveServers() ([]string, error) {
	fileSpecs, err := loadIperf3ServerSpecsFile(b.serverFile)
	if err != nil {
		return nil, err
	}
	fileServers := make([]string, 0, len(fileSpecs))
	for _, spec := range fileSpecs {
		fileServers = append(fileServers, spec.Server)
	}
	return normalizeIperf3Servers("", append(append([]string{}, b.servers...), fileServers...)), nil
}

func (b *Iperf3NetworkBackend) resolveServerSpecs(servers []string) []iperf3ServerSpec {
	specByServer := map[string]iperf3ServerSpec{}
	fileSpecs, err := loadIperf3ServerSpecsFile(b.serverFile)
	if err == nil {
		for _, spec := range fileSpecs {
			if spec.Server != "" {
				specByServer[spec.Server] = spec
			}
		}
	}

	specs := make([]iperf3ServerSpec, 0, len(servers))
	for _, server := range servers {
		if spec, ok := specByServer[server]; ok {
			specs = append(specs, spec)
			continue
		}
		specs = append(specs, iperf3ServerSpec{Server: server})
	}
	return specs
}

func loadIperf3ServersFile(path string) ([]string, error) {
	specs, err := loadIperf3ServerSpecsFile(path)
	if err != nil {
		return nil, err
	}
	servers := make([]string, 0, len(specs))
	for _, spec := range specs {
		servers = append(servers, spec.Server)
	}
	return servers, nil
}

func loadIperf3ServerSpecsFile(path string) ([]iperf3ServerSpec, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, newBenchmarkError(BenchmarkErrorInvalidConfig, "iperf3_server_file_read", "确认 iperf3 节点文件路径存在且当前用户可读。", err)
	}
	lines := strings.Split(string(content), "\n")
	servers := make([]iperf3ServerSpec, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if index := strings.Index(trimmed, "#"); index >= 0 {
			trimmed = strings.TrimSpace(trimmed[:index])
		}
		if trimmed != "" {
			spec, err := parseIperf3ServerSpecLine(trimmed)
			if err != nil {
				return nil, newBenchmarkError(BenchmarkErrorInvalidConfig, "iperf3_server_file_parse", "节点文件每行应为 host[:port]，并必须追加 auth=owned 或 auth=authorized；可追加 name=、region=、provider= 元数据。", err)
			}
			servers = append(servers, spec)
		}
	}
	return servers, nil
}

func parseIperf3ServerSpecLine(line string) (iperf3ServerSpec, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return iperf3ServerSpec{}, fmt.Errorf("iperf3 server is required")
	}

	spec := iperf3ServerSpec{Server: fields[0]}
	if _, err := parseIperf3Endpoint(spec.Server); err != nil {
		return iperf3ServerSpec{}, err
	}

	for _, field := range fields[1:] {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return iperf3ServerSpec{}, fmt.Errorf("invalid iperf3 server metadata %q; use key=value", field)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if value == "" {
			return iperf3ServerSpec{}, fmt.Errorf("invalid iperf3 server metadata %q; value is required", field)
		}
		switch key {
		case "name":
			spec.Name = value
		case "region":
			spec.Region = value
		case "provider":
			spec.Provider = value
		case "auth", "authorization":
			if value != "owned" && value != "authorized" {
				return iperf3ServerSpec{}, fmt.Errorf("unsupported iperf3 authorization %q; supported values: owned, authorized", value)
			}
			spec.Authorization = value
		default:
			return iperf3ServerSpec{}, fmt.Errorf("unsupported iperf3 server metadata key %q; supported keys: name, region, provider, auth", key)
		}
	}
	if spec.Authorization == "" {
		return iperf3ServerSpec{}, fmt.Errorf("iperf3 server file entries must declare auth=owned or auth=authorized")
	}
	return spec, nil
}

func iperf3Protocol(host string) string {
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		return "ipv6"
	}
	return "ipv4"
}

func appendIperf3Error(current string, next string) string {
	if current == "" {
		return next
	}
	return current + "; " + next
}

func setIperf3ServerResultError(result *iperf3ServerResult, err error) {
	if result == nil || err == nil || result.ErrorCategory != "" {
		return
	}
	category, stage, hint, ok := benchmarkErrorFields(err)
	if !ok {
		return
	}
	result.ErrorCategory = category
	result.ErrorStage = stage
	result.ErrorHint = hint
}

func countSuccessfulIperf3Results(results []iperf3ServerResult) int {
	count := 0
	for _, result := range results {
		if result.DownloadMbps > 0 || result.UploadMbps > 0 {
			count++
		}
	}
	return count
}

func averageSuccessfulIperf3Download(results []iperf3ServerResult) float64 {
	return averageIperf3Value(results, func(result iperf3ServerResult) float64 {
		return result.DownloadMbps
	})
}

func averageSuccessfulIperf3Upload(results []iperf3ServerResult) float64 {
	return averageIperf3Value(results, func(result iperf3ServerResult) float64 {
		return result.UploadMbps
	})
}

func averageIperf3Value(results []iperf3ServerResult, value func(iperf3ServerResult) float64) float64 {
	total := 0.0
	count := 0
	for _, result := range results {
		if current := value(result); current > 0 {
			total += current
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func bestIperf3Download(results []iperf3ServerResult) float64 {
	return bestIperf3Value(results, func(result iperf3ServerResult) float64 {
		return result.DownloadMbps
	})
}

func bestIperf3Upload(results []iperf3ServerResult) float64 {
	return bestIperf3Value(results, func(result iperf3ServerResult) float64 {
		return result.UploadMbps
	})
}

func bestIperf3Value(results []iperf3ServerResult, value func(iperf3ServerResult) float64) float64 {
	best := 0.0
	for _, result := range results {
		if current := value(result); current > best {
			best = current
		}
	}
	return best
}

type iperf3JSONResult struct {
	End struct {
		SumReceived *struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
		SumSent *struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_sent"`
	} `json:"end"`
}

func parseIperf3Mbps(output []byte) (float64, error) {
	var result iperf3JSONResult
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, err
	}

	bitsPerSecond := 0.0
	if result.End.SumReceived != nil && result.End.SumReceived.BitsPerSecond > 0 {
		bitsPerSecond = result.End.SumReceived.BitsPerSecond
	} else if result.End.SumSent != nil && result.End.SumSent.BitsPerSecond > 0 {
		bitsPerSecond = result.End.SumSent.BitsPerSecond
	}
	if bitsPerSecond <= 0 {
		return 0, fmt.Errorf("missing bits_per_second in iperf3 result")
	}

	return bitsPerSecond / 1000000.0, nil
}
