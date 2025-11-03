// Package analyzer 提供性能数据分析功能
// 负责对采集到的原始性能数据进行分析、计算和评分
package analyzer

// Analyzer 接口定义了性能分析器的标准行为
// 所有具体的分析器实现都应该实现此接口
type Analyzer interface {
	// Analyze 对输入数据进行分析
	// 参数 data 是待分析的原始数据
	// 返回分析结果和可能发生的错误
	Analyze(data interface{}) (interface{}, error)
}
