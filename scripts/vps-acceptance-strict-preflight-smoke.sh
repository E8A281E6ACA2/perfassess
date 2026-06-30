#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
binary="${1:-${PERFASSESS_BINARY:-$repo_root/build/perfassess}}"

if [[ ! -x "$binary" ]]; then
  echo "strict preflight smoke failed: binary is not executable: $binary" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

if PERFASSESS_BINARY="$binary" \
  PERFASSESS_ACCEPTANCE_DIR="$tmpdir/acceptance" \
  PERFASSESS_ACCEPTANCE_MATRIX=low \
  PERFASSESS_ACCEPTANCE_STRICT=1 \
  PERFASSESS_ACCEPTANCE_STRICT_MIN_MEM_MB=999999 \
  PERFASSESS_ACCEPTANCE_OPTIONAL=never \
  "$repo_root/scripts/vps-acceptance.sh" >"$tmpdir/stdout.txt" 2>"$tmpdir/stderr.txt"; then
  echo "strict preflight smoke failed: expected strict acceptance to fail when memory threshold is unreachable" >&2
  exit 1
fi

grep -q "requires at least 999999 MB available memory" "$tmpdir/stderr.txt"
grep -q "run without PERFASSESS_ACCEPTANCE_STRICT=1" "$tmpdir/stderr.txt"

if [[ -e "$tmpdir/acceptance/quick.json" ]]; then
  echo "strict preflight smoke failed: acceptance started reports before resource preflight failed" >&2
  exit 1
fi

echo "strict preflight smoke passed"
