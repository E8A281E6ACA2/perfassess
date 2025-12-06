package tests

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"performance-assessment-system/internal/models"
	"performance-assessment-system/pkg/logger"
)

// StressTest 提供长时间压力测试
type StressTest struct {
	logger         *logger.Logger
	cpuDuration    time.Duration
	memDuration    time.Duration
	diskDuration   time.Duration
	sampleInterval time.Duration
	workDir        string
}

// NewStressTest 创建压力测试实例
func NewStressTest(logger *logger.Logger) *StressTest {
	return &StressTest{
		logger:         logger,
		cpuDuration:    60 * time.Second,
		memDuration:    60 * time.Second,
		diskDuration:   60 * time.Second,
		sampleInterval: 3 * time.Second,
		workDir:        os.TempDir(),
	}
}

// Run 执行完整压力测试
func (st *StressTest) Run(ctx context.Context) (*models.StressTestReport, error) {
	st.logger.Info("开始长时间压力测试")

	results := []*models.StressComponentResult{}
	start := time.Now()

	if r := st.runCPU(ctx); r != nil {
		results = append(results, r)
	}

	if r := st.runMemory(ctx); r != nil {
		results = append(results, r)
	}

	if r := st.runDisk(ctx); r != nil {
		results = append(results, r)
	}

	report := &models.StressTestReport{
		TotalDurationSeconds: time.Since(start).Seconds(),
		Components:           results,
		TemperatureAvailable: st.hasTemperatureSupport(),
	}

	st.logger.Info("压力测试完成")
	return report, nil
}

func (st *StressTest) runCPU(ctx context.Context) *models.StressComponentResult {
	result := st.newComponentResult("CPU", st.cpuDuration)
	st.logger.Info(fmt.Sprintf("CPU 压测（%ds）开始", int(st.cpuDuration.Seconds())))

	stop := make(chan struct{})
	var wg sync.WaitGroup
	workers := runtime.NumCPU()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(seed float64) {
			defer wg.Done()
			val := seed
			for {
				select {
				case <-stop:
					return
				default:
					val = math.Sin(val+0.0001) * math.Cos(val+0.001)
					if val > 1e6 {
						val = seed
					}
				}
			}
		}(float64(i + 1))
	}

	tempCh := st.collectTemperatures(st.cpuDuration)
	st.blockWithContext(ctx, st.cpuDuration)
	close(stop)
	wg.Wait()

	stats := <-tempCh
	st.applyTemperatureStats(result, stats)
	result.Metrics["worker_count"] = workers

	return result
}

func (st *StressTest) runMemory(ctx context.Context) *models.StressComponentResult {
	result := st.newComponentResult("Memory", st.memDuration)
	st.logger.Info(fmt.Sprintf("内存压测（%ds）开始", int(st.memDuration.Seconds())))

	sizeMB := st.determineMemoryLoadSize()
	data := make([]byte, sizeMB*1024*1024)

	stop := make(chan struct{})
	var bytesTouched int64

	go func() {
		pattern := byte(0)
		for {
			select {
			case <-stop:
				return
			default:
				for i := range data {
					data[i] = pattern
				}
				bytesTouched += int64(len(data))
				pattern++
			}
		}
	}()

	tempCh := st.collectTemperatures(st.memDuration)
	st.blockWithContext(ctx, st.memDuration)
	close(stop)
	stats := <-tempCh
	st.applyTemperatureStats(result, stats)

	if st.memDuration.Seconds() > 0 {
		result.Metrics["memory_mb"] = sizeMB
		result.Metrics["processed_mb_per_sec"] = float64(bytesTouched)/1024/1024/st.memDuration.Seconds()
	}

	return result
}

func (st *StressTest) runDisk(ctx context.Context) *models.StressComponentResult {
	result := st.newComponentResult("Disk", st.diskDuration)
	st.logger.Info(fmt.Sprintf("磁盘压测（%ds）开始", int(st.diskDuration.Seconds())))

	file, err := os.CreateTemp(st.workDir, "stress-disk-*.dat")
	if err != nil {
		result.Status = "failed"
		result.Notes = fmt.Sprintf("创建临时文件失败: %v", err)
		return result
	}
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	buffer := make([]byte, 4*1024*1024)
	rand.Read(buffer)

	stop := make(chan struct{})
	var bytesWritten int64

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				if _, err := file.Write(buffer); err != nil {
					result.Status = "failed"
					result.Notes = fmt.Sprintf("写入失败: %v", err)
					return
				}
				if err := file.Sync(); err != nil {
					result.Status = "failed"
					result.Notes = fmt.Sprintf("同步失败: %v", err)
					return
				}
				if _, err := file.Seek(0, 0); err != nil {
					result.Status = "failed"
					result.Notes = fmt.Sprintf("Seek失败: %v", err)
					return
				}
				bytesWritten += int64(len(buffer))
			}
		}
	}()

	tempCh := st.collectTemperatures(st.diskDuration)
	st.blockWithContext(ctx, st.diskDuration)
	close(stop)
	stats := <-tempCh
	st.applyTemperatureStats(result, stats)

	if result.Status == "failed" && result.Notes != "" {
		return result
	}

	if st.diskDuration.Seconds() > 0 {
		result.Metrics["write_mb_per_sec"] = float64(bytesWritten)/1024/1024/st.diskDuration.Seconds()
	}

	return result
}

func (st *StressTest) newComponentResult(name string, duration time.Duration) *models.StressComponentResult {
	return &models.StressComponentResult{
		Name:            name,
		DurationSeconds: duration.Seconds(),
		Status:          "success",
		Metrics:         make(map[string]interface{}),
	}
}

type tempStats struct {
	avg float64
	max float64
	has bool
}

func (st *StressTest) collectTemperatures(duration time.Duration) <-chan tempStats {
	ch := make(chan tempStats, 1)
	go func() {
		defer close(ch)
		stats := tempStats{}
		ticker := time.NewTicker(st.sampleInterval)
		defer ticker.Stop()
		timeout := time.NewTimer(duration)
		defer timeout.Stop()

		var sum float64
		var count int

		for {
			select {
			case <-ticker.C:
				if temp, ok := readAverageTemperature(); ok {
					stats.has = true
					sum += temp
					count++
					if temp > stats.max {
						stats.max = temp
					}
				}
			case <-timeout.C:
				if count > 0 {
					stats.avg = sum / float64(count)
				}
				ch <- stats
				return
			}
		}
	}()
	return ch
}

func (st *StressTest) applyTemperatureStats(result *models.StressComponentResult, stats tempStats) {
	if stats.has {
		result.AverageTemperature = stats.avg
		result.PeakTemperature = stats.max
	} else {
		result.Notes = st.appendNote(result.Notes, "无法获取温度数据")
	}
}

func (st *StressTest) appendNote(existing, add string) string {
	if existing == "" {
		return add
	}
	return existing + "; " + add
}

func (st *StressTest) hasTemperatureSupport() bool {
	_, err := host.SensorsTemperatures()
	return err == nil
}

func readAverageTemperature() (float64, bool) {
	temps, err := host.SensorsTemperatures()
	if err != nil || len(temps) == 0 {
		return 0, false
	}
	var sum float64
	var count float64
	for _, t := range temps {
		if t.Temperature > 1 {
			sum += t.Temperature
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return sum / count, true
}

func (st *StressTest) determineMemoryLoadSize() int {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 256
	}
	availableMB := int(vmStat.Available / 1024 / 1024)
	switch {
	case availableMB > 4096:
		return 512
	case availableMB > 2048:
		return 384
	case availableMB > 1024:
		return 256
	default:
		return 128
	}
}

func (st *StressTest) blockWithContext(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
