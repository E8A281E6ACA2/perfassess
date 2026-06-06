package models

// IPQualityReport 汇总公网 IP 质量检测结果
type IPQualityReport struct {
	PublicIP        string              `json:"public_ip"`
	IPVersion       string              `json:"ip_version"`
	ISP             string              `json:"isp,omitempty"`
	Country         string              `json:"country,omitempty"`
	City            string              `json:"city,omitempty"`
	IPType          string              `json:"ip_type"`
	RiskLevel       string              `json:"risk_level"`
	RiskScore       int                 `json:"risk_score"`
	BlacklistChecks []*IPBlacklistCheck `json:"blacklist_checks"`
	MailChecks      []*MailPortCheck    `json:"mail_checks"`
	Notes           []string            `json:"notes,omitempty"`
}

// IPBlacklistCheck 表示单个 DNSBL 黑名单查询结果
type IPBlacklistCheck struct {
	Zone   string `json:"zone"`
	Listed bool   `json:"listed"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// MailPortCheck 表示单个邮件端口连通性检测结果
type MailPortCheck struct {
	Target    string `json:"target"`
	Port      int    `json:"port"`
	Reachable bool   `json:"reachable"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
}
