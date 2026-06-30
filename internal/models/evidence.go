package models

// EvidenceSummary 表示一类证据的状态、可信度、影响范围和限制。
type EvidenceSummary struct {
	Category   string `json:"category"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Confidence string `json:"confidence"`
	Impact     string `json:"impact"`
	Detail     string `json:"detail"`
	Limitation string `json:"limitation"`
}
