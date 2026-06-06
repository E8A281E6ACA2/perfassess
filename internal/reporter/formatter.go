// Package reporter 提供输出格式化功能
package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/E8A281E6ACA2/perfassess/internal/models"
)

// OutputFormatter 输出格式化器
// 负责将报告输出到控制台或文件
type OutputFormatter struct{}

// NewOutputFormatter 创建新的输出格式化器
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{}
}

// OutputToConsole 将报告输出到控制台
// 使用彩色文本增强可读性
func (of *OutputFormatter) OutputToConsole(report *models.Report) error {
	return of.OutputToConsoleWithFormat(report, "text")
}

func (of *OutputFormatter) OutputToConsoleWithFormat(report *models.Report, format string) error {
	if report == nil {
		return fmt.Errorf("报告不能为空")
	}

	if format == "json" {
		content, err := of.OutputJSON(report)
		if err != nil {
			return err
		}
		fmt.Println(content)
		return nil
	}

	// 获取格式化内容
	content := report.FormattedContent
	if content == "" {
		return fmt.Errorf("报告内容为空")
	}

	// 为关键信息添加颜色
	coloredContent := of.colorizeContent(content)

	// 输出到控制台
	fmt.Println(coloredContent)

	return nil
}

// OutputToFile 将报告输出到文件
// 保存为纯文本格式
func (of *OutputFormatter) OutputToFile(report *models.Report, filepath string) error {
	return of.OutputToFileWithFormat(report, filepath, "text")
}

func (of *OutputFormatter) OutputToFileWithFormat(report *models.Report, filepath string, format string) error {
	if report == nil {
		return fmt.Errorf("报告不能为空")
	}

	if filepath == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	var content string
	if format == "json" {
		jsonContent, err := of.OutputJSON(report)
		if err != nil {
			return err
		}
		content = jsonContent + "\n"
	} else {
		content = report.FormattedContent
		if content == "" {
			return fmt.Errorf("报告内容为空")
		}
	}

	// 写入文件
	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// FormatTable 格式化表格数据
// 将map数据格式化为对齐的表格文本
func (of *OutputFormatter) FormatTable(data map[string]interface{}) string {
	if data == nil || len(data) == 0 {
		return ""
	}

	var sb strings.Builder

	// 计算最大键长度用于对齐
	maxKeyLen := 0
	for key := range data {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	// 格式化每一行
	for key, value := range data {
		padding := strings.Repeat(" ", maxKeyLen-len(key))
		sb.WriteString(fmt.Sprintf("%s%s: %v\n", key, padding, value))
	}

	return sb.String()
}

// Colorize 为文本添加颜色
// 使用ANSI转义码实现终端颜色
func (of *OutputFormatter) Colorize(text string, color string) string {
	// ANSI颜色代码
	colors := map[string]string{
		"reset":   "\033[0m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"bold":    "\033[1m",
	}

	colorCode, ok := colors[color]
	if !ok {
		return text
	}

	return colorCode + text + colors["reset"]
}

// colorizeContent 为报告内容添加颜色
func (of *OutputFormatter) colorizeContent(content string) string {
	lines := strings.Split(content, "\n")
	var coloredLines []string

	for _, line := range lines {
		coloredLine := line

		// 标题行（包含 === 或 --- 或 ╔ ╚ ║）
		if strings.Contains(line, "===") || strings.Contains(line, "---") ||
			strings.Contains(line, "╔") || strings.Contains(line, "╚") ||
			strings.Contains(line, "║") || strings.Contains(line, "════") {
			coloredLine = of.Colorize(line, "cyan")
		} else if strings.Contains(line, "性能等级") {
			// 性能等级行
			if strings.Contains(line, "优秀") {
				coloredLine = strings.Replace(line, "优秀", of.Colorize("优秀", "green"), 1)
			} else if strings.Contains(line, "良好") {
				coloredLine = strings.Replace(line, "良好", of.Colorize("良好", "blue"), 1)
			} else if strings.Contains(line, "一般") {
				coloredLine = strings.Replace(line, "一般", of.Colorize("一般", "yellow"), 1)
			} else if strings.Contains(line, "较差") {
				coloredLine = strings.Replace(line, "较差", of.Colorize("较差", "red"), 1)
			}
		} else if strings.Contains(line, "状态:") {
			// 状态行
			if strings.Contains(line, "success") {
				coloredLine = strings.Replace(line, "success", of.Colorize("success", "green"), 1)
			} else if strings.Contains(line, "failed") {
				coloredLine = strings.Replace(line, "failed", of.Colorize("failed", "red"), 1)
			} else if strings.Contains(line, "skipped") {
				coloredLine = strings.Replace(line, "skipped", of.Colorize("skipped", "yellow"), 1)
			}
		} else if strings.Contains(line, "总体评分:") || strings.Contains(line, "CPU评分:") ||
			strings.Contains(line, "内存评分:") || strings.Contains(line, "磁盘评分:") ||
			strings.Contains(line, "网络评分:") {
			// 评分行 - 高亮显示分数
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				coloredLine = parts[0] + ":" + of.Colorize(parts[1], "bold")
			}
		} else if strings.Contains(line, "错误信息:") {
			// 错误信息行
			coloredLine = of.Colorize(line, "red")
		}

		coloredLines = append(coloredLines, coloredLine)
	}

	return strings.Join(coloredLines, "\n")
}

// OutputJSON 将报告输出为JSON格式
// 用于机器可读的输出
func (of *OutputFormatter) OutputJSON(report *models.Report) (string, error) {
	if report == nil {
		return "", fmt.Errorf("报告不能为空")
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化 JSON 报告失败: %w", err)
	}
	return string(data), nil
}

// OutputSummary 输出简化的摘要信息
// 只显示关键指标，适合快速查看
func (of *OutputFormatter) OutputSummary(report *models.Report) error {
	if report == nil {
		return fmt.Errorf("报告不能为空")
	}

	var sb strings.Builder

	sb.WriteString(of.Colorize("=== 性能评估摘要 ===\n", "cyan"))
	sb.WriteString("\n")

	// 总体评分
	if report.Summary != nil {
		if totalScore, ok := report.Summary["total_score"].(float64); ok {
			sb.WriteString(fmt.Sprintf("总体评分: %s\n",
				of.Colorize(fmt.Sprintf("%.2f / 100", totalScore), "bold")))
		}

		if grade, ok := report.Summary["grade"].(string); ok {
			gradeColor := "white"
			switch grade {
			case "优秀":
				gradeColor = "green"
			case "良好":
				gradeColor = "blue"
			case "一般":
				gradeColor = "yellow"
			case "较差":
				gradeColor = "red"
			}
			sb.WriteString(fmt.Sprintf("性能等级: %s\n", of.Colorize(grade, gradeColor)))
		}

		sb.WriteString("\n")

		// 各项评分
		if cpuScore, ok := report.Summary["cpu_score"].(float64); ok {
			sb.WriteString(fmt.Sprintf("CPU:    %.2f\n", cpuScore))
		}
		if memScore, ok := report.Summary["memory_score"].(float64); ok {
			sb.WriteString(fmt.Sprintf("内存:   %.2f\n", memScore))
		}
		if diskScore, ok := report.Summary["disk_score"].(float64); ok {
			sb.WriteString(fmt.Sprintf("磁盘:   %.2f\n", diskScore))
		}
		if netScore, ok := report.Summary["network_score"].(float64); ok {
			sb.WriteString(fmt.Sprintf("网络:   %.2f\n", netScore))
		}
	}

	fmt.Print(sb.String())
	return nil
}
