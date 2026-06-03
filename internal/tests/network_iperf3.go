package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"performance-assessment-system/internal/models"
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
	server    string
	servers   []string
	timeout   time.Duration
	runner    commandRunner
	latencyFn func([]string) (float64, error)
	matrix    []iperf3ServerResult
	matrixErr error
}

type Iperf3Config struct {
	Server    string
	Servers   []string
	Timeout   time.Duration
	Runner    commandRunner
	LatencyFn func([]string) (float64, error)
}

type iperf3Endpoint struct {
	Host string
	Port string
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
		server:    cfg.Server,
		servers:   servers,
		timeout:   timeout,
		runner:    runner,
		latencyFn: cfg.LatencyFn,
	}
}

func (b *Iperf3NetworkBackend) Name() string {
	return models.NetworkBackendIperf3
}

func (b *Iperf3NetworkBackend) Server() string {
	if len(b.servers) > 0 {
		return strings.Join(b.servers, ",")
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
	if len(b.servers) > 1 {
		results, err := b.loadMatrix()
		if err != nil {
			return NetworkDownloadResult{}, err
		}
		speed := averageSuccessfulIperf3Download(results)
		if speed <= 0 {
			return NetworkDownloadResult{}, fmt.Errorf("all iperf3 matrix download tests failed")
		}
		return NetworkDownloadResult{SpeedMbps: speed, SourceURL: models.NetworkDownloadSourceIperf3}, nil
	}

	endpoint, err := b.validateReady()
	if err != nil {
		return NetworkDownloadResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, false)...)
	if err != nil {
		return NetworkDownloadResult{}, fmt.Errorf("iperf3 download failed: %w", err)
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return NetworkDownloadResult{}, fmt.Errorf("parse iperf3 download result: %w", err)
	}
	return NetworkDownloadResult{SpeedMbps: speed, SourceURL: models.NetworkDownloadSourceIperf3}, nil
}

func (b *Iperf3NetworkBackend) MeasureUpload(downloadSpeed float64) (float64, bool, error) {
	if len(b.servers) > 1 {
		results, err := b.loadMatrix()
		if err != nil {
			return 0, false, err
		}
		speed := averageSuccessfulIperf3Upload(results)
		if speed <= 0 {
			return 0, false, fmt.Errorf("all iperf3 matrix upload tests failed")
		}
		return speed, false, nil
	}

	endpoint, err := b.validateReady()
	if err != nil {
		return 0, false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, true)...)
	if err != nil {
		return 0, false, fmt.Errorf("iperf3 upload failed: %w", err)
	}

	speed, err := parseIperf3Mbps(output)
	if err != nil {
		return 0, false, fmt.Errorf("parse iperf3 upload result: %w", err)
	}
	return speed, false, nil
}

type iperf3ServerResult struct {
	Server       string
	Endpoint     iperf3Endpoint
	Protocol     string
	DownloadMbps float64
	UploadMbps   float64
	LatencyMs    float64
	Error        string
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
		return nil, fmt.Errorf("iperf3 is not installed; install it manually before using --network-backend iperf3. Ubuntu/Debian: sudo apt install iperf3; RHEL/CentOS: sudo yum install iperf3; macOS: brew install iperf3")
	}
	if len(b.servers) == 0 {
		return nil, fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 1.2.3.4:5201")
	}

	results := make([]iperf3ServerResult, 0, len(b.servers))
	for _, server := range b.servers {
		result := iperf3ServerResult{Server: server}
		endpoint, err := parseIperf3Endpoint(server)
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
		} else {
			result.DownloadMbps = speed
		}
		if speed, err := b.runIperf3(endpoint, true); err != nil {
			result.Error = appendIperf3Error(result.Error, "upload: "+err.Error())
		} else {
			result.UploadMbps = speed
		}
		results = append(results, result)
	}
	if countSuccessfulIperf3Results(results) == 0 {
		return results, fmt.Errorf("all iperf3 matrix tests failed")
	}
	return results, nil
}

func (b *Iperf3NetworkBackend) runIperf3(endpoint iperf3Endpoint, reverse bool) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	output, err := b.runner.Run(ctx, "iperf3", b.commandArgs(endpoint, reverse)...)
	if err != nil {
		return 0, err
	}
	return parseIperf3Mbps(output)
}

func (b *Iperf3NetworkBackend) validateReady() (iperf3Endpoint, error) {
	endpoint, err := parseIperf3Endpoint(b.server)
	if err != nil {
		return iperf3Endpoint{}, err
	}
	if _, err := b.runner.LookPath("iperf3"); err != nil {
		return iperf3Endpoint{}, fmt.Errorf("iperf3 is not installed; install it manually before using --network-backend iperf3. Ubuntu/Debian: sudo apt install iperf3; RHEL/CentOS: sudo yum install iperf3; macOS: brew install iperf3")
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
		return iperf3Endpoint{}, fmt.Errorf("iperf3 server is required; example: --network-backend iperf3 --iperf3-server 1.2.3.4:5201")
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
