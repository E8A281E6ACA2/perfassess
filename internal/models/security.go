package models

// SecurityFinding 表示单个安全检测结果
type SecurityFinding struct {
	Category string `json:"category"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	Advice   string `json:"advice,omitempty"`
}

// SecurityReport 汇总安全检测结果
type SecurityReport struct {
	Findings []*SecurityFinding `json:"findings"`
}
