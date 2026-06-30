#!/usr/bin/env bash
set -euo pipefail

workflow=".github/workflows/release.yml"

fail() {
  echo "release workflow smoke failed: $*" >&2
  exit 1
}

require_contains() {
  local pattern="$1"
  grep -Eq -- "$pattern" "$workflow" || fail "$workflow does not contain required pattern: $pattern"
}

[[ -f "$workflow" ]] || fail "missing $workflow"

require_contains 'workflow_dispatch:'
require_contains 'permissions:'
require_contains 'contents: write'
require_contains 'inputs:'
require_contains 'version:'
require_contains 'required: true'
require_contains 'refs/tags/\*'
require_contains 'github\.event\.inputs\.version'
require_contains '\^v\[0-9\]\+'
require_contains 'GITHUB_OUTPUT'
require_contains 'tag_name: \$\{\{ steps\.get_version\.outputs\.VERSION \}\}'
require_contains 'name: \$\{\{ steps\.get_version\.outputs\.VERSION \}\}'
require_contains 'make validate'
require_contains 'scripts/release-checksums-smoke\.sh build'
require_contains 'scripts/release-smoke\.sh \./perfassess_linux_amd64'
require_contains 'checksums\.txt'
require_contains 'perfassess_linux_amd64'
require_contains 'perfassess_linux_arm64'
require_contains 'perfassess_darwin_amd64'
require_contains 'perfassess_darwin_arm64'
require_contains 'perfassess\.exe'

echo "release workflow smoke passed"
