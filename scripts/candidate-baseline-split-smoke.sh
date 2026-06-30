#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

fail() {
  echo "candidate baseline split smoke failed: $*" >&2
  exit 1
}

require_file_contains() {
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "missing required file: $file"
  grep -Eq -- "$pattern" "$file" || fail "$file does not contain required pattern: $pattern"
}

plan="docs/candidate-baseline-split-plan.md"

require_file_contains "README.md" 'docs/candidate-baseline-split-plan\.md'
require_file_contains "docs/release-checklist.md" 'candidate-baseline-split-plan\.md'
require_file_contains "docs/release-validation-log.md" 'candidate-baseline-split-plan\.md'

require_file_contains "$plan" 'type\(scope\): 中文描述'
require_file_contains "$plan" 'feat\(report\): 增加外部检测证据可信度摘要'
require_file_contains "$plan" 'fix\(tests\): 细化主流后端错误分类'
require_file_contains "$plan" 'scripts\(calibration\): 增加脱敏校准样本收集链路'
require_file_contains "$plan" 'docs\(calibration\): 补齐真实 VPS 样本矩阵和归档规范'
require_file_contains "$plan" 'docs\(release\): 收紧发布和远程验收门禁'
require_file_contains "$plan" 'test\(acceptance\): 增强 VPS 验收摘要和报告覆盖检查'
require_file_contains "$plan" 'scripts\(bootstrap\): 优化远程启动和端口说明'
require_file_contains "$plan" 'docs\(release\): 增加候选基线拆分提交计划'

require_file_contains "$plan" 'internal/models/evidence\.go'
require_file_contains "$plan" 'scripts/report-contract-smoke\.sh'
require_file_contains "$plan" 'internal/tests/benchmark_error\.go'
require_file_contains "$plan" 'scripts/calibration-collect\.sh'
require_file_contains "$plan" 'scripts/calibration-collect-smoke\.sh'
require_file_contains "$plan" 'docs/calibration-sampling-matrix\.md'
require_file_contains "$plan" 'scripts/release-runbook-smoke\.sh'
require_file_contains "$plan" 'scripts/remote-runbook-smoke\.sh'
require_file_contains "$plan" 'scripts/vps-acceptance-summary-smoke\.sh'
require_file_contains "$plan" 'docs/examples/iperf3-servers\.txt'
require_file_contains "$plan" 'docs/candidate-baseline-split-plan\.md'
require_file_contains "$plan" 'scripts/candidate-baseline-split-smoke\.sh'

require_file_contains "$plan" 'make report-contract-smoke'
require_file_contains "$plan" 'make calibration-collect-smoke'
require_file_contains "$plan" 'make calibration-summary-smoke'
require_file_contains "$plan" 'make remote-runbook-smoke'
require_file_contains "$plan" 'make release-runbook-smoke'
require_file_contains "$plan" 'make vps-acceptance-summary-smoke'
require_file_contains "$plan" 'make bootstrap-prebuilt-smoke'
require_file_contains "$plan" 'make candidate-baseline-split-smoke'
require_file_contains "$plan" 'make pre-commit'
require_file_contains "$plan" 'make release-check'

require_file_contains "$plan" '不提交 `calibration-samples/`'
require_file_contains "$plan" '不把原始 `default\.json`、日志、压缩包或完整报告目录作为校准输入'
require_file_contains "$plan" 'PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1'
require_file_contains "$plan" '--require-score-profiles vps,server,workstation'
require_file_contains "$plan" '--require-policy formal'
require_file_contains "$plan" '文档保留地址'

echo "candidate baseline split smoke passed"
