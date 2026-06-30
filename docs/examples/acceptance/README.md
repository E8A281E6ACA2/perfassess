# VPS 验收样例目录

本目录记录真实 VPS 验收产物的脱敏归档约定，不提交真实机器原始报告。

可使用脱敏工具生成公开归档用的 JSON：

```bash
python3 scripts/redact-report.py /tmp/perfassess-auto/default.json
python3 scripts/redact-report.py /tmp/perfassess-auto
```

脱敏工具会生成 `.redacted.json` 文件，不覆盖原始报告。

建议归档内容：

- `summary.md`: 验收摘要，保留版本、总分、等级、置信度和校准版本
- `summary.json`: 机器可读验收摘要，保留矩阵、报告列表、可选/跳过项和报告覆盖度
- `quick.redacted.json`: `quick.json` 脱敏版本
- `network.redacted.json`: `network.json` 脱敏版本
- `low-basic.redacted.json`: 低配矩阵报告脱敏版本，如已执行 `PERFASSESS_ACCEPTANCE_MATRIX=low`
- `standard-builtin.redacted.json`: 标准矩阵报告脱敏版本，如已执行 `PERFASSESS_ACCEPTANCE_MATRIX=standard`
- `calibration_sample.redacted.json`: 自动测评生成的脱敏校准样本，可用于后续评分阈值回测
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
- `summary.assessment_conclusion`
- `summary.module_assessments`

`summary.md` 中的 `optional_reports` 和 `skipped_reports` 应保留。它们用于区分可选后端成功、可选后端因资源不足或外部环境限制跳过、以及真正的验收失败。

`summary.json` 适合给 CI、发布平台或样本收集脚本消费。它不应包含公网 IP、ISP、ASN、精确地理位置或路由 hop。建议保留：

- `binary_version`
- `acceptance_matrix`
- `generated_reports`
- `optional_reports`
- `skipped_reports`
- `required_reports`
- `report_coverage`
- `report_coverage[].file`
- `report_coverage[].tests_success`
- `report_coverage[].tests_failed`
- `report_coverage[].tests_skipped`
- `report_coverage[].confidence`
- `report_coverage[].confidence_reasons`

`summary.json` 最小示例：

```json
{
  "binary_version": "perfassess 1.0.0",
  "acceptance_matrix": "low,standard",
  "generated_reports": [
    "quick.json",
    "network.json",
    "cpu.json",
    "low-basic.json",
    "standard-builtin.json"
  ],
  "optional_reports": [
    "disk-fio"
  ],
  "skipped_reports": [
    "memory-sysbench: optional backend did not complete but report explains why"
  ],
  "report_coverage": [
    {
      "file": "standard-builtin.json",
      "tests_success": 3,
      "tests_failed": 0,
      "tests_skipped": 1,
      "confidence": "low",
      "confidence_reasons": [
        "内存测试未成功",
        "网络上传为估算值"
      ]
    }
  ]
}
```

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
