#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

fake_bin="$tmpdir/perfassess"
output_dir="$tmpdir/out"

cat >"$fake_bin" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  version)
    echo "perfassess smoke"
    ;;
  check-deps)
    echo "dependencies ok"
    ;;
  --output-format)
    out=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        -o)
          out="$2"
          shift 2
          ;;
        *)
          shift
          ;;
      esac
    done
    cat >"$out" <<'JSON'
{
  "session_id": "smoke",
  "timestamp": "2026-06-30T00:00:00Z",
  "system_info": {},
  "test_results": {},
  "summary": {
    "total_score": 0,
    "grade": "未完成",
    "tests_success": 0,
    "tests_failed": 0,
    "tests_skipped": 0,
    "vps_benchmark_summary": {},
    "confidence_level": {"level": "low"},
    "score_calibration": {"version": "smoke"},
    "share_templates": {"plain_text": "smoke"}
  }
}
JSON
    ;;
  --quick)
    echo "quick failed for smoke"
    exit 23
    ;;
  *)
    echo "unexpected args: $*" >&2
    exit 2
    ;;
esac
SH
chmod +x "$fake_bin"

set +e
PERFASSESS_SKIP_BUILD=1 \
PERFASSESS_BINARY="$fake_bin" \
PERFASSESS_AUTO_DIR="$output_dir" \
PERFASSESS_AUTO_ACCEPTANCE=0 \
PERFASSESS_AUTO_HEARTBEAT=1 \
  "$repo_root/scripts/perfassess-auto.sh" >/tmp/perfassess-auto-running-summary-smoke.stdout 2>/tmp/perfassess-auto-running-summary-smoke.stderr
code="$?"
set -e

if [[ "$code" -eq 0 ]]; then
  echo "auto running summary smoke failed: expected script failure" >&2
  exit 1
fi

test -f "$output_dir/summary.md"
test -f "$output_dir/console.txt"
grep -q "Perfassess 自动测评失败" "$output_dir/summary.md"
grep -q "quick" "$output_dir/summary.md"
grep -q "默认 JSON 报告" "$output_dir/summary.md"
grep -q "快速测评错误输出" "$output_dir/summary.md"

fake_path="$tmpdir/fake-path"
mkdir -p "$fake_path"
cat >"$fake_path/go" <<'SH'
#!/usr/bin/env bash
echo "go build failed for smoke" >&2
exit 42
SH
chmod +x "$fake_path/go"

build_fail_dir="$tmpdir/build-fail-out"
set +e
PATH="$fake_path:$PATH" \
PERFASSESS_AUTO_DIR="$build_fail_dir" \
PERFASSESS_AUTO_ACCEPTANCE=0 \
  "$repo_root/scripts/perfassess-auto.sh" >/tmp/perfassess-auto-build-fail-smoke.stdout 2>/tmp/perfassess-auto-build-fail-smoke.stderr
build_code="$?"
set -e

if [[ "$build_code" -eq 0 ]]; then
  echo "auto running summary smoke failed: expected build failure" >&2
  exit 1
fi

test -f "$build_fail_dir/summary.md"
test -f "$build_fail_dir/build.stdout.txt"
test -f "$build_fail_dir/build.stderr.log"
grep -q "Perfassess 自动测评失败" "$build_fail_dir/summary.md"
grep -q "| 失败步骤 | build |" "$build_fail_dir/summary.md"
grep -q "二进制构建失败" "$build_fail_dir/summary.md"
grep -q "构建错误输出" "$build_fail_dir/summary.md"
grep -q "go build failed for smoke" "$build_fail_dir/summary.md"
grep -q "go build failed for smoke" "$build_fail_dir/build.stderr.log"

echo "auto running summary smoke passed"
