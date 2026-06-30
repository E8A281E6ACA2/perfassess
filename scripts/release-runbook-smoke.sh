#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

fail() {
  echo "release runbook smoke failed: $*" >&2
  exit 1
}

require_file_contains() {
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "missing required file: $file"
  grep -Eq -- "$pattern" "$file" || fail "$file does not contain required pattern: $pattern"
}

runbook="docs/release-runbook.md"

require_file_contains "README.md" 'docs/release-runbook\.md'
require_file_contains "docs/release-checklist.md" 'release-runbook\.md'
require_file_contains "$runbook" 'make release-check'
require_file_contains "$runbook" 'git tag -a v1\.0\.0'
require_file_contains "$runbook" 'git push origin v1\.0\.0'
require_file_contains "$runbook" '\.github/workflows/release\.yml'
require_file_contains "$runbook" 'perfassess_linux_amd64'
require_file_contains "$runbook" 'perfassess_linux_arm64'
require_file_contains "$runbook" 'perfassess_darwin_amd64'
require_file_contains "$runbook" 'perfassess_darwin_arm64'
require_file_contains "$runbook" 'perfassess\.exe'
require_file_contains "$runbook" 'checksums\.txt'
require_file_contains "$runbook" 'sha256sum -c'
require_file_contains "$runbook" 'scripts/verify-release-assets\.sh --version latest'
require_file_contains "$runbook" 'scripts/verify-release-assets\.sh --version v1\.0\.0'
require_file_contains "$runbook" '--skip-smoke'
require_file_contains "$runbook" 'basic 自动测评'
require_file_contains "$runbook" '当前平台'
require_file_contains "$runbook" '不支持执行 Release 资产'
require_file_contains "$runbook" 'shasum -a 256'
require_file_contains "$runbook" '填写 `version` 输入'
require_file_contains "$runbook" 'vMAJOR\.MINOR\.PATCH'
require_file_contains "$runbook" 'PERFASSESS_BOOTSTRAP_BINARY=1'
require_file_contains "$runbook" 'PERFASSESS_BOOTSTRAP_BINARY=auto'
require_file_contains "$runbook" 'git push origin :refs/tags/v1\.0\.0'
require_file_contains "$runbook" 'git status --short --branch'

echo "release runbook smoke passed"
