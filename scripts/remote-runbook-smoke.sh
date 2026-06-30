#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

fail() {
  echo "remote runbook smoke failed: $*" >&2
  exit 1
}

require_file_contains() {
  local file="$1"
  local pattern="$2"
  [[ -f "$file" ]] || fail "missing required file: $file"
  grep -Eq -- "$pattern" "$file" || fail "$file does not contain required pattern: $pattern"
}

runbook="docs/remote-test-runbook.md"

require_file_contains "README.md" 'docs/remote-test-runbook\.md'
require_file_contains "$runbook" 'raw\.githubusercontent\.com/E8A281E6ACA2/perfassess/main/scripts/bootstrap\.sh'
require_file_contains "$runbook" '--profile standard --web --port 8080'
require_file_contains "$runbook" '--profile full --web --skip-acceptance'
require_file_contains "$runbook" '--profile full --quality mainstream --web --skip-acceptance'
require_file_contains "$runbook" 'PERFASSESS_BOOTSTRAP_BINARY=1 bash'
require_file_contains "$runbook" 'PERFASSESS_BOOTSTRAP_BINARY=0 bash'
require_file_contains "$runbook" 'checksums\.txt'
require_file_contains "$runbook" 'PERFASSESS_BOOTSTRAP_SWAP=0 bash'
require_file_contains "$runbook" '不要访问 `0\.0\.0\.0:8080`'
require_file_contains "$runbook" '服务器公网 IP'
require_file_contains "$runbook" 'ssh -L 8080:localhost:8080'
require_file_contains "$runbook" '/tmp/perfassess-auto/summary\.md'
require_file_contains "$runbook" '/tmp/perfassess-auto/console\.txt'
require_file_contains "$runbook" '/tmp/perfassess-auto/default\.json'
require_file_contains "$runbook" '/tmp/perfassess-auto/artifact_manifest\.json'
require_file_contains "$runbook" '/tmp/perfassess-auto/perfassess-report\.zip'
require_file_contains "$runbook" '/tmp/perfassess-auto/calibration_sample\.json'
require_file_contains "$runbook" 'python3 scripts/verify-artifacts\.py /tmp/perfassess-auto'
require_file_contains "$runbook" 'python3 scripts/redact-report\.py /tmp/perfassess-auto/default\.json'
require_file_contains "$runbook" 'python3 scripts/calibration-summary\.py /path/to/samples'
require_file_contains "$runbook" 'docs/calibration-dataset-policy\.md'
require_file_contains "$runbook" 'calibration_sample\.redacted\.json'
require_file_contains "$runbook" '/tmp/perfassess-auto/default\.stdout\.txt\.stderr\.log'
require_file_contains "$runbook" 'PERFASSESS_ACCEPTANCE_MATRIX=low,standard'
require_file_contains "$runbook" 'PERFASSESS_ACCEPTANCE_STRICT=1'
require_file_contains "$runbook" 'MemAvailable >= 768 MB'
require_file_contains "$runbook" 'PERFASSESS_IPERF3_SERVER=1\.2\.3\.4:5201 bash'
require_file_contains "$runbook" 'auth=owned.*auth=authorized|auth=authorized.*auth=owned'
require_file_contains "$runbook" 'scripts/bootstrap\.sh --clean'
require_file_contains "$runbook" 'scripts/bootstrap\.sh --clean-all'
require_file_contains "$runbook" '--destroy-after-web --web-ttl 600'

if grep -Eq 'PERFASSESS_[A-Z0-9_]+=.*curl -fsSL' "$runbook"; then
  fail "environment variable is applied to curl instead of bash in $runbook"
fi

echo "remote runbook smoke passed"
