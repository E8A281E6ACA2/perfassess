#!/usr/bin/env bash
set -euo pipefail

repo="${PERFASSESS_RELEASE_REPO:-E8A281E6ACA2/perfassess}"
version="${PERFASSESS_RELEASE_VERSION:-latest}"
base_url="${PERFASSESS_RELEASE_BASE_URL:-https://github.com/${repo}/releases}"
tmpdir=""
run_smoke="${PERFASSESS_RELEASE_SMOKE:-1}"

usage() {
  cat <<EOF
Usage: scripts/verify-release-assets.sh [options]

Options:
  --repo OWNER/REPO       GitHub repository. Default: $repo
  --version VERSION       Release tag or latest. Default: $version
  --base-url URL          Release base URL. Default: $base_url
  --skip-smoke            Only download assets and verify checksums; skip binary smoke.
  -h, --help              Show this help.

Environment:
  PERFASSESS_RELEASE_REPO=E8A281E6ACA2/perfassess
  PERFASSESS_RELEASE_VERSION=latest
  PERFASSESS_RELEASE_BASE_URL=https://github.com/E8A281E6ACA2/perfassess/releases
  PERFASSESS_RELEASE_SMOKE=0
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      [[ $# -ge 2 ]] || { echo "--repo requires a value" >&2; exit 1; }
      repo="$2"
      base_url="https://github.com/${repo}/releases"
      shift 2
      ;;
    --version)
      [[ $# -ge 2 ]] || { echo "--version requires a value" >&2; exit 1; }
      version="$2"
      shift 2
      ;;
    --base-url)
      [[ $# -ge 2 ]] || { echo "--base-url requires a value" >&2; exit 1; }
      base_url="$2"
      shift 2
      ;;
    --skip-smoke)
      run_smoke="0"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

asset_url() {
  local asset="$1"
  if [[ "$version" == "latest" ]]; then
    printf '%s/latest/download/%s' "$base_url" "$asset"
  else
    printf '%s/download/%s/%s' "$base_url" "$version" "$asset"
  fi
}

smoke_asset_name() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) return 1 ;;
  esac

  case "$os" in
    linux|darwin)
      printf 'perfassess_%s_%s' "$os" "$arch"
      ;;
    *)
      return 1
      ;;
  esac
}

verify_checksums() {
  local checksums_file="${1:-checksums.txt}"
  if command -v sha256sum >/dev/null 2>&1; then
    if sha256sum -c "$checksums_file"; then
      return
    fi
    if ! command -v shasum >/dev/null 2>&1; then
      return 1
    fi
    echo "sha256sum verification failed; retrying with shasum -a 256" >&2
  fi
  if ! command -v shasum >/dev/null 2>&1; then
    echo "sha256sum or shasum is required" >&2
    exit 1
  fi

  local expected file actual failed=0
  while read -r expected file _rest; do
    [[ -n "${expected:-}" && -n "${file:-}" ]] || continue
    if [[ ! -f "$file" ]]; then
      echo "$file: FAILED open or read" >&2
      failed=1
      continue
    fi
    actual="$(shasum -a 256 "$file" | awk '{ print $1 }')"
    if [[ "$actual" == "$expected" ]]; then
      echo "$file: OK"
    else
      echo "$file: FAILED" >&2
      failed=1
    fi
  done <"$checksums_file"
  [[ "$failed" -eq 0 ]]
}

cleanup() {
  [[ -n "$tmpdir" ]] && rm -rf "$tmpdir"
}
trap cleanup EXIT

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }
command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1 || { echo "sha256sum or shasum is required" >&2; exit 1; }

assets=(
  perfassess_linux_amd64
  perfassess_linux_arm64
  perfassess_darwin_amd64
  perfassess_darwin_arm64
  perfassess.exe
  checksums.txt
)

tmpdir="$(mktemp -d)"

echo "Verifying Perfassess release assets"
echo "Repository: $repo"
echo "Version: $version"
echo "Base URL: $base_url"
echo "Work dir: $tmpdir"

for asset in "${assets[@]}"; do
  echo "Downloading $asset"
  curl -fsSL --retry 2 --connect-timeout 10 --max-time 180 "$(asset_url "$asset")" -o "$tmpdir/$asset"
  [[ -s "$tmpdir/$asset" ]] || { echo "$asset is empty" >&2; exit 1; }
done

(
  cd "$tmpdir"
  verify_checksums checksums.txt
)

smoke_asset=""
if smoke_asset="$(smoke_asset_name)"; then
  chmod +x "$tmpdir/$smoke_asset"
  "$tmpdir/$smoke_asset" version
else
  echo "no executable release asset for $(uname -s)/$(uname -m); binary smoke will be skipped"
  run_smoke="0"
fi

case "$run_smoke" in
  0|false|no)
    echo "release binary smoke skipped"
    ;;
  *)
    repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    PERFASSESS_AUTO_DIR="$tmpdir/auto" \
    PERFASSESS_AUTO_PROFILE=basic \
    PERFASSESS_AUTO_ACCEPTANCE=0 \
    PERFASSESS_AUTO_PROGRESS=0 \
    PERFASSESS_SKIP_BUILD=1 \
    PERFASSESS_BINARY="$tmpdir/$smoke_asset" \
      "$repo_root/scripts/perfassess-auto.sh" >/dev/null
    python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto" >/dev/null
    python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto/perfassess-report.zip" >/dev/null
    echo "release binary smoke passed"
    ;;
esac

echo "release assets verified"
