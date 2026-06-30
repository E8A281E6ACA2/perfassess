#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

source_path="${1:-}"
dataset_dir="${2:-${PERFASSESS_CALIBRATION_DATASET_DIR:-$repo_root/calibration-samples}}"
require_policy="${PERFASSESS_CALIBRATION_REQUIRE_POLICY:-}"
require_score_profiles="${PERFASSESS_CALIBRATION_REQUIRE_SCORE_PROFILES:-}"
require_manifest="${PERFASSESS_CALIBRATION_REQUIRE_MANIFEST:-0}"

usage() {
  cat <<'EOF'
Usage:
  scripts/calibration-collect.sh /tmp/perfassess-auto/calibration_sample.json [dataset-dir]
  scripts/calibration-collect.sh /tmp/perfassess-auto [dataset-dir]

Environment:
  PERFASSESS_CALIBRATION_DATASET_DIR              Default dataset directory.
  PERFASSESS_CALIBRATION_LABEL                    Optional sample folder name, letters/numbers/dot/underscore/dash only.
  PERFASSESS_CALIBRATION_REQUIRE_MANIFEST         Set to 1 to require manifest.json SHA256 validation for the dataset.
  PERFASSESS_CALIBRATION_REQUIRE_POLICY           Optional gate: exploratory, candidate, formal.
  PERFASSESS_CALIBRATION_REQUIRE_SCORE_PROFILES   Optional comma-separated gate, for example: vps,server,workstation.

This script only accepts redacted calibration_sample.json files. It does not copy raw reports, logs, IP data, or archives.
EOF
}

fail() {
  echo "calibration collect failed: $*" >&2
  exit 1
}

if [[ -z "$source_path" || "$source_path" == "-h" || "$source_path" == "--help" ]]; then
  usage
  [[ -n "$source_path" ]] && exit 0
  exit 1
fi

if [[ -d "$source_path" ]]; then
  sample_path="$source_path/calibration_sample.json"
elif [[ -f "$source_path" ]]; then
  sample_path="$source_path"
else
  fail "source does not exist: $source_path"
fi

[[ -f "$sample_path" ]] || fail "calibration_sample.json not found at: $sample_path"

label="${PERFASSESS_CALIBRATION_LABEL:-}"
if [[ -z "$label" ]]; then
  label="$(date -u +%Y%m%dT%H%M%SZ)"
fi
if [[ ! "$label" =~ ^[A-Za-z0-9._-]+$ ]]; then
  fail "sample label must only contain letters, numbers, dot, underscore, or dash: $label"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
mkdir -p "$tmpdir/input"
cp "$sample_path" "$tmpdir/input/calibration_sample.json"

"$repo_root/scripts/calibration-summary.py" "$tmpdir/input" --format json -o "$tmpdir/validated.json" >/dev/null

mkdir -p "$dataset_dir"
target_dir="$dataset_dir/$label"
if [[ -e "$target_dir" ]]; then
  fail "target sample directory already exists: $target_dir"
fi
mkdir -p "$target_dir"
cp "$tmpdir/input/calibration_sample.json" "$target_dir/calibration_sample.json"
sample_sha256="$(python3 - "$target_dir/calibration_sample.json" <<'PY'
import hashlib
import sys
from pathlib import Path

print(hashlib.sha256(Path(sys.argv[1]).read_bytes()).hexdigest())
PY
)"
collected_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
python3 - "$target_dir/manifest.json" "$target_dir/calibration_sample.json" "$label" "$collected_at" "$sample_sha256" <<'PY'
import json
import sys
from pathlib import Path

manifest_path = Path(sys.argv[1])
sample_path = Path(sys.argv[2])
label = sys.argv[3]
collected_at = sys.argv[4]
sample_sha256 = sys.argv[5]
sample = json.loads(sample_path.read_text(encoding="utf-8"))
scores = sample.get("scores") if isinstance(sample.get("scores"), dict) else {}
benchmark = sample.get("benchmark_profile") if isinstance(sample.get("benchmark_profile"), dict) else {}
manifest = {
    "schema_version": "perfassess-calibration-sample-manifest-v1",
    "label": label,
    "collected_at": collected_at,
    "sample": {
        "schema_version": sample.get("schema_version"),
        "score_profile": scores.get("score_profile"),
        "calibration_version": scores.get("calibration_version"),
        "confidence_level": scores.get("confidence_level"),
        "benchmark_profile": benchmark.get("name"),
        "mainstream_count": benchmark.get("mainstream_count"),
    },
    "files": [
        {
            "path": "calibration_sample.json",
            "sha256": sample_sha256,
            "privacy": "redacted_calibration_sample",
        }
    ],
    "privacy_note": "Manifest intentionally excludes source paths, public IPs, provider account data, raw reports, logs, and archives.",
}
manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

"$repo_root/scripts/calibration-summary.py" "$dataset_dir" --format markdown -o "$dataset_dir/summary.md"
"$repo_root/scripts/calibration-summary.py" "$dataset_dir" --format json -o "$dataset_dir/summary.json"
"$repo_root/scripts/calibration-summary.py" "$dataset_dir" --format jsonl -o "$dataset_dir/samples.jsonl"

gate_args=("$dataset_dir")
case "$require_manifest" in
  1|true|yes) gate_args+=("--require-manifest") ;;
  0|false|no|"") ;;
  *) fail "PERFASSESS_CALIBRATION_REQUIRE_MANIFEST must be 0/1, true/false, or yes/no" ;;
esac
if [[ -n "$require_score_profiles" ]]; then
  gate_args+=("--require-score-profiles" "$require_score_profiles")
fi
if [[ -n "$require_policy" ]]; then
  gate_args+=("--require-policy" "$require_policy")
fi
if [[ "${#gate_args[@]}" -gt 1 ]]; then
  if ! "$repo_root/scripts/calibration-summary.py" "${gate_args[@]}" --format json -o "$tmpdir/gate-summary.json"; then
    echo "calibration collect failed: dataset gate did not pass; summary remains available at $dataset_dir/summary.md" >&2
    exit 2
  fi
fi

echo "calibration sample collected"
echo "Sample: $target_dir/calibration_sample.json"
echo "Manifest: $target_dir/manifest.json"
echo "Summary: $dataset_dir/summary.md"
echo "JSON: $dataset_dir/summary.json"
