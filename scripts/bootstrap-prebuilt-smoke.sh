#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

make_failing_sha256sum_path() {
  local dir="$1"
  mkdir -p "$dir"
  cat >"$dir/sha256sum" <<'SH'
#!/usr/bin/env bash
exit 127
SH
  chmod +x "$dir/sha256sum"
}

release_dir="$tmpdir/releases/latest/download"
work_dir="$tmpdir/work"
output_dir="$tmpdir/out"
mkdir -p "$release_dir"

asset="perfassess_linux_amd64"
case "$(uname -m)" in
  aarch64|arm64) asset="perfassess_linux_arm64" ;;
esac

cat >"$release_dir/$asset" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  version)
    echo "perfassess prebuilt-smoke"
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
  "session_id": "prebuilt-smoke",
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
  "session_id": "prebuilt-smoke-quick",
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
  *)
    echo "unexpected args: $*" >&2
    exit 2
    ;;
esac
SH
chmod +x "$release_dir/$asset"
(cd "$release_dir" && sha256sum "$asset" > checksums.txt)

(
  cd "$tmpdir"
  PERFASSESS_REPO_URL="$repo_root" \
  PERFASSESS_BOOTSTRAP_DIR="$work_dir" \
  PERFASSESS_AUTO_DIR="$output_dir" \
  PERFASSESS_BOOTSTRAP_BINARY=1 \
  PERFASSESS_BOOTSTRAP_BINARY_BASE_URL="file://$tmpdir/releases" \
  PERFASSESS_AUTO_PROFILE=basic \
  PERFASSESS_QUALITY_PROFILE=builtin \
  PERFASSESS_NETWORK_PROFILE=quick \
  PERFASSESS_AUTO_ACCEPTANCE=0 \
  PERFASSESS_BOOTSTRAP_TESTS=0 \
  PERFASSESS_AUTO_PROGRESS=0 \
  PERFASSESS_BOOTSTRAP_INTERACTIVE=0 \
    "$repo_root/scripts/bootstrap.sh" >/tmp/perfassess-bootstrap-prebuilt-smoke.stdout 2>/tmp/perfassess-bootstrap-prebuilt-smoke.stderr
)

test -x "$work_dir/build/perfassess"
grep -q "using prebuilt binary" /tmp/perfassess-bootstrap-prebuilt-smoke.stdout
grep -q "prebuilt binary checksum verified" /tmp/perfassess-bootstrap-prebuilt-smoke.stdout
grep -q "skipping source build and unit tests" /tmp/perfassess-bootstrap-prebuilt-smoke.stdout
test -f "$output_dir/summary.md"

fake_path="$tmpdir/fake-path"
fallback_work_dir="$tmpdir/work-shasum"
fallback_output_dir="$tmpdir/out-shasum"
make_failing_sha256sum_path "$fake_path"

(
  cd "$tmpdir"
  PATH="$fake_path:$PATH" \
  PERFASSESS_REPO_URL="$repo_root" \
  PERFASSESS_BOOTSTRAP_DIR="$fallback_work_dir" \
  PERFASSESS_AUTO_DIR="$fallback_output_dir" \
  PERFASSESS_BOOTSTRAP_BINARY=1 \
  PERFASSESS_BOOTSTRAP_BINARY_BASE_URL="file://$tmpdir/releases" \
  PERFASSESS_AUTO_PROFILE=basic \
  PERFASSESS_QUALITY_PROFILE=builtin \
  PERFASSESS_NETWORK_PROFILE=quick \
  PERFASSESS_AUTO_ACCEPTANCE=0 \
  PERFASSESS_BOOTSTRAP_TESTS=0 \
  PERFASSESS_AUTO_PROGRESS=0 \
  PERFASSESS_BOOTSTRAP_INTERACTIVE=0 \
    "$repo_root/scripts/bootstrap.sh" >/tmp/perfassess-bootstrap-prebuilt-shasum-smoke.stdout 2>/tmp/perfassess-bootstrap-prebuilt-shasum-smoke.stderr
)

test -x "$fallback_work_dir/build/perfassess"
grep -q "prebuilt binary checksum verified" /tmp/perfassess-bootstrap-prebuilt-shasum-smoke.stdout
test -f "$fallback_output_dir/summary.md"

echo "bootstrap prebuilt smoke passed"
