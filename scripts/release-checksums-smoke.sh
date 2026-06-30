#!/usr/bin/env bash
set -euo pipefail

build_dir="${1:-build}"
checksums="$build_dir/checksums.txt"

required=(
  perfassess_linux_amd64
  perfassess_linux_arm64
  perfassess_darwin_amd64
  perfassess_darwin_arm64
  perfassess.exe
)

[[ -f "$checksums" ]] || {
  echo "release checksums smoke failed: missing $checksums" >&2
  exit 1
}

for asset in "${required[@]}"; do
  [[ -f "$build_dir/$asset" ]] || {
    echo "release checksums smoke failed: missing asset $build_dir/$asset" >&2
    exit 1
  }
  grep -Eq "^[0-9a-f]{64}[[:space:]]+$asset$" "$checksums" || {
    echo "release checksums smoke failed: missing checksum entry for $asset" >&2
    exit 1
  }
done

(cd "$build_dir" && sha256sum -c checksums.txt >/dev/null)

echo "release checksums smoke passed"
