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
mkdir -p "$release_dir"

cat >"$release_dir/perfassess_linux_amd64" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  version)
    echo "perfassess verify-release-smoke"
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
  "session_id": "verify-release-smoke",
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
  "session_id": "verify-release-smoke-quick",
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
chmod +x "$release_dir/perfassess_linux_amd64"

printf 'linux-arm64\n' >"$release_dir/perfassess_linux_arm64"
printf 'darwin-amd64\n' >"$release_dir/perfassess_darwin_amd64"
printf 'darwin-arm64\n' >"$release_dir/perfassess_darwin_arm64"
printf 'windows-amd64\n' >"$release_dir/perfassess.exe"

(
  cd "$release_dir"
  sha256sum perfassess_linux_amd64 perfassess_linux_arm64 perfassess_darwin_amd64 perfassess_darwin_arm64 perfassess.exe > checksums.txt
)

PERFASSESS_RELEASE_BASE_URL="file://$tmpdir/releases" \
PERFASSESS_RELEASE_VERSION=latest \
  bash "$repo_root/scripts/verify-release-assets.sh" >/tmp/perfassess-verify-release-assets-smoke.stdout 2>/tmp/perfassess-verify-release-assets-smoke.stderr

grep -q "release assets verified" /tmp/perfassess-verify-release-assets-smoke.stdout
grep -q "release binary smoke passed" /tmp/perfassess-verify-release-assets-smoke.stdout
grep -q "perfassess verify-release-smoke" /tmp/perfassess-verify-release-assets-smoke.stdout

PERFASSESS_RELEASE_BASE_URL="file://$tmpdir/releases" \
PERFASSESS_RELEASE_VERSION=latest \
  bash "$repo_root/scripts/verify-release-assets.sh" --skip-smoke >/tmp/perfassess-verify-release-assets-skip-smoke.stdout 2>/tmp/perfassess-verify-release-assets-skip-smoke.stderr
grep -q "release binary smoke skipped" /tmp/perfassess-verify-release-assets-skip-smoke.stdout
grep -q "release assets verified" /tmp/perfassess-verify-release-assets-skip-smoke.stdout

fake_path="$tmpdir/fake-path"
make_failing_sha256sum_path "$fake_path"

PATH="$fake_path:$PATH" \
PERFASSESS_RELEASE_BASE_URL="file://$tmpdir/releases" \
PERFASSESS_RELEASE_VERSION=latest \
  bash "$repo_root/scripts/verify-release-assets.sh" --skip-smoke >/tmp/perfassess-verify-release-assets-shasum.stdout 2>/tmp/perfassess-verify-release-assets-shasum.stderr
grep -q "release assets verified" /tmp/perfassess-verify-release-assets-shasum.stdout
grep -q "retrying with shasum -a 256" /tmp/perfassess-verify-release-assets-shasum.stderr

echo "verify release assets smoke passed"
