package models

// IPQualityReport 汇总公网 IP 质量检测结果
type IPQualityReport struct {
	PublicIP         string               `json:"public_ip"`
	IPVersion        string               `json:"ip_version"`
	ISP              string               `json:"isp,omitempty"`
	Country          string               `json:"country,omitempty"`
	City             string               `json:"city,omitempty"`
	ASN              string               `json:"asn,omitempty"`
	Organization     string               `json:"organization,omitempty"`
	ReverseDNS       []string             `json:"reverse_dns,omitempty"`
	IPType           string               `json:"ip_type"`
	RiskLevel        string               `json:"risk_level"`
	RiskScore        int                  `json:"risk_score"`
	RiskFactors      []*IPRiskFactor      `json:"risk_factors"`
	RiskSources      []*IPRiskSource      `json:"risk_sources,omitempty"`
	BlacklistSummary *IPBlacklistSummary  `json:"blacklist_summary,omitempty"`
	BlacklistChecks  []*IPBlacklistCheck  `json:"blacklist_checks"`
	MailSummary      *MailPortSummary     `json:"mail_summary,omitempty"`
	MailChecks       []*MailPortCheck     `json:"mail_checks"`
	NetworkStack     *IPNetworkStack      `json:"network_stack,omitempty"`
	Verdict          *IPQualityVerdict    `json:"verdict,omitempty"`
	Evidence         []*IPQualityEvidence `json:"evidence,omitempty"`
	EvidenceSummary  []*EvidenceSummary   `json:"evidence_summary,omitempty"`
	Recommendations  []string             `json:"recommendations,omitempty"`
	Notes            []string             `json:"notes,omitempty"`
}

// IPQualityVerdict 表示面向人工阅读的 IP 质量结论
type IPQualityVerdict struct {
	Grade       string `json:"grade"`
	Summary     string `json:"summary"`
	IPTypeLabel string `json:"ip_type_label"`
	RiskLabel   string `json:"risk_label"`
	MailUsable  bool   `json:"mail_usable"`
	HostingHint bool   `json:"hosting_hint"`
	ProxyHint   bool   `json:"proxy_hint"`
}

// IPQualityEvidence 表示支持 IP 质量结论的证据项
type IPQualityEvidence struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// IPRiskFactor 表示单个风险因子的推断结果
type IPRiskFactor struct {
	Name       string `json:"name"`
	Detected   bool   `json:"detected"`
	Confidence string `json:"confidence"`
	Source     string `json:"source"`
	Detail     string `json:"detail,omitempty"`
}

// IPRiskSource 表示一个风险数据来源或启发式来源的可用性。
type IPRiskSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Signal  string `json:"signal,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Weight  int    `json:"weight,omitempty"`
	Enabled bool   `json:"enabled"`
}

// IPBlacklistSummary 汇总 DNSBL 查询结果
type IPBlacklistSummary struct {
	Total   int `json:"total"`
	Clean   int `json:"clean"`
	Listed  int `json:"listed"`
	Timeout int `json:"timeout"`
	Skipped int `json:"skipped"`
	Other   int `json:"other"`
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
	Service   string `json:"service,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Target    string `json:"target"`
	Port      int    `json:"port"`
	Reachable bool   `json:"reachable"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
}

// MailPortSummary 汇总邮件服务商连通性。
type MailPortSummary struct {
	Total        int `json:"total"`
	Reachable    int `json:"reachable"`
	Blocked      int `json:"blocked"`
	Timeout      int `json:"timeout"`
	Providers    int `json:"providers"`
	ProviderOpen int `json:"provider_open"`
}

// IPNetworkStack 表示本次 IP 质量检测的网络栈视角。
type IPNetworkStack struct {
	DetectedVersion string `json:"detected_version"`
	IPv4Available   bool   `json:"ipv4_available"`
	IPv6Available   bool   `json:"ipv6_available"`
	DualStack       bool   `json:"dual_stack"`
	Note            string `json:"note,omitempty"`
}
