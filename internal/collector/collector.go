// Package collector 提供系统性能数据采集功能
// 负责从操作系统收集 CPU、内存、磁盘、网络等性能指标
package collector

// Collector 接口定义了性能数据采集器的标准行为
// 所有具体的采集器实现都应该实现此接口
type Collector interface {
	// Collect 执行数据采集操作
	// 返回采集到的数据和可能发生的错误
	Collect() (interface{}, error)
}

// 注意：性能测试相关的接口和实现已移至 internal/tests 包
