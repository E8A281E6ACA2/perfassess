#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

fail() {
  echo "project invariants smoke failed: $*" >&2
  exit 1
}

require_file_contains() {
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "missing required file: $file"
  grep -Eq "$pattern" "$file" || fail "$file does not contain required pattern: $pattern"
}

forbidden_hits() {
  local pattern="$1"
  shift
  rg -n "$pattern" "$@" \
    -g '!study/**' \
    -g '!build/**' \
    -g '!logs/**' \
    -g '!scripts/project-invariants-smoke.sh' \
    -g '!coverage.out' \
    -g '!coverage.html' || true
}

require_file_contains "go.mod" '^module github\.com/E8A281E6ACA2/perfassess$'
require_file_contains "README.md" 'raw\.githubusercontent\.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap\.sh'
require_file_contains "README.md" '0\.0\.0\.0:8080.*不是浏览器访问地址|0\.0\.0\.0:8080'
require_file_contains "README.md" '国内方向参考.*不是真实回程|不等同真实回程'
require_file_contains "AGENTS.md" 'type\(scope\): 中文描述'
require_file_contains "docs/mainstream-vps-benchmark-analysis.md" '不默认上传报告'
require_file_contains "docs/mainstream-vps-benchmark-analysis.md" '不默认长期追踪本地历史'
require_file_contains "docs/project-analysis-and-roadmap.md" '先可信，再丰富'
require_file_contains "docs/project-analysis-and-roadmap.md" 'make pre-commit'
require_file_contains "docs/release-checklist.md" 'make validate'
require_file_contains "docs/release-checklist.md" 'make pre-commit'
require_file_contains "docs/release-checklist.md" 'make vps-acceptance-low-standard'
require_file_contains "docs/vps-acceptance.md" 'PERFASSESS_ACCEPTANCE_MATRIX=low,standard'
require_file_contains "docs/vps-acceptance.md" 'make vps-acceptance-low-standard'
require_file_contains "docs/vps-acceptance.md" 'summary\.json'
require_file_contains "docs/vps-acceptance.md" 'report_coverage'
require_file_contains "docs/vps-acceptance.md" 'confidence_reasons'
require_file_contains "docs/examples/acceptance/README.md" 'summary\.json'
require_file_contains "docs/examples/acceptance/README.md" 'report_coverage'
require_file_contains "scripts/vps-acceptance.sh" 'summary\.json'
require_file_contains "scripts/vps-acceptance.sh" 'report_coverage'
require_file_contains "scripts/vps-acceptance.sh" 'confidence_reasons'
require_file_contains "Makefile" '^pre-commit:'
require_file_contains "Makefile" '^vps-acceptance-low-standard:'
require_file_contains ".github/workflows/ci.yml" 'make validate'
require_file_contains ".github/workflows/release.yml" 'make validate'

old_name_hits="$(forbidden_hits 'high-performance-multi-terminal-automated-performance-evaluation-system|github\.com/E8A281E6ACA2/high-performance|High Performance Multi Terminal' README.md docs scripts internal cmd pkg go.mod .github AGENTS.md)"
if [[ -n "$old_name_hits" ]]; then
  echo "$old_name_hits" >&2
  fail "old project name or module path is still referenced"
fi

remote_probe_hits="$(forbidden_hits 'probe-route|return-route-file|return_route_file|return-route-probe-design|is_real_return_route=true|已实现.*真实回程|真实回程.*已实现|远端探针.*已实现|公共探针池.*已内置' README.md docs scripts internal cmd pkg .github)"
if [[ -n "$remote_probe_hits" ]]; then
  echo "$remote_probe_hits" >&2
  fail "removed real return-route probe design is still referenced as an implemented path"
fi

history_command_hits="$(forbidden_hits 'perfassess history|--store|~/.perfassess/history\.jsonl|新增 `perfassess history|historyCmd|newHistory|HistoryStore' README.md docs scripts internal cmd pkg .github)"
if [[ -n "$history_command_hits" ]]; then
  echo "$history_command_hits" >&2
  fail "removed default local history commands are still referenced"
fi

if [[ -e internal/history/history.go || -e internal/history/history_test.go || -e docs/return-route-probe-design.md ]]; then
  fail "removed history or return-route probe files still exist"
fi

if ! rg -n 'func \(c \*CLI\) newCompareDirCommand' internal/cli/cli.go >/dev/null; then
  fail "compare-dir command is missing"
fi

if rg -n 'newHistory|historyCmd|HistoryStore|history add|history list|history trend' internal cmd pkg >/dev/null; then
  fail "default local history implementation is still present"
fi

echo "project invariants smoke passed"
