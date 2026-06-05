# VPS 验收样例目录

本目录记录真实 VPS 验收产物的脱敏归档约定，不提交真实机器原始报告。

建议归档内容：

- `summary.md`: 验收摘要，保留版本、总分、等级、置信度和校准版本
- `quick.redacted.json`: `quick.json` 脱敏版本
- `network.redacted.json`: `network.json` 脱敏版本
- `optional-notes.md`: 可选依赖安装情况和失败原因说明

必须脱敏字段：

- `system_info.ip_info.public_ip`
- `system_info.ip_info.isp`
- `system_info.ip_info.geo_location`
- `summary.vps_benchmark_summary.system.public_ip`
- `summary.vps_benchmark_summary.system.isp`
- `summary.vps_benchmark_summary.system.location`
- `test_results.network_result.metrics.speedtest_external_ip`

建议保留字段：

- `summary.benchmark_profile`
- `summary.confidence_level`
- `summary.score_calibration`
- `summary.score_breakdown`
- `summary.vps_benchmark_summary`
- `summary.share_templates`

脱敏示例：

```json
{
  "system_info": {
    "ip_info": {
      "public_ip": "REDACTED",
      "isp": "REDACTED",
      "geo_location": {
        "country": "REDACTED",
        "country_code": "REDACTED",
        "city": "REDACTED"
      }
    }
  },
  "summary": {
    "score_calibration": {
      "version": "2026-06-v1",
      "active_profile": "server"
    }
  }
}
```
