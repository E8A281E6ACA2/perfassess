#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
binary="${1:-${PERFASSESS_BINARY:-$repo_root/build/perfassess}}"

if [[ ! -x "$binary" ]]; then
  echo "calibration collect smoke failed: binary is not executable: $binary" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

PERFASSESS_AUTO_DIR="$tmpdir/auto" \
PERFASSESS_AUTO_PROFILE=basic \
PERFASSESS_AUTO_ACCEPTANCE=0 \
PERFASSESS_AUTO_PROGRESS=0 \
PERFASSESS_SKIP_BUILD=1 \
PERFASSESS_BINARY="$binary" \
"$repo_root/scripts/perfassess-auto.sh" >/dev/null

PERFASSESS_CALIBRATION_LABEL=sample-a \
"$repo_root/scripts/calibration-collect.sh" "$tmpdir/auto" "$tmpdir/dataset" >"$tmpdir/collect-a.out"

test -f "$tmpdir/dataset/sample-a/calibration_sample.json"
test -f "$tmpdir/dataset/sample-a/manifest.json"
test -f "$tmpdir/dataset/summary.md"
test -f "$tmpdir/dataset/summary.json"
test -f "$tmpdir/dataset/samples.jsonl"
grep -q "calibration sample collected" "$tmpdir/collect-a.out"
grep -q "Manifest:" "$tmpdir/collect-a.out"
grep -q "Perfassess Calibration Summary" "$tmpdir/dataset/summary.md"
python3 -m json.tool "$tmpdir/dataset/summary.json" >/dev/null
"$repo_root/scripts/calibration-summary.py" "$tmpdir/dataset" --require-manifest --format json -o "$tmpdir/manifest-required.json"
python3 -m json.tool "$tmpdir/manifest-required.json" >/dev/null
python3 - "$tmpdir/manifest-required.json" <<'PY'
import json
import pathlib
import sys

summary = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
policy = summary.get("manifest_policy")
if not isinstance(policy, dict) or policy.get("required") is not True or policy.get("checked_count") != 1:
    raise SystemExit("manifest policy summary mismatch")
PY
"$repo_root/scripts/calibration-summary.py" "$tmpdir/dataset" --require-manifest --format markdown -o "$tmpdir/manifest-required.md"
grep -q "manifest_required: True" "$tmpdir/manifest-required.md"
grep -q "manifest_checked_count: 1" "$tmpdir/manifest-required.md"
python3 - "$tmpdir/dataset/sample-a/manifest.json" "$tmpdir/dataset/sample-a/calibration_sample.json" "$tmpdir" <<'PY'
import hashlib
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
sample_path = pathlib.Path(sys.argv[2])
tmpdir = pathlib.Path(sys.argv[3])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
if manifest.get("schema_version") != "perfassess-calibration-sample-manifest-v1":
    raise SystemExit("unexpected manifest schema")
if manifest.get("label") != "sample-a":
    raise SystemExit("manifest label mismatch")
sample = manifest.get("sample")
if not isinstance(sample, dict):
    raise SystemExit("manifest sample summary missing")
if sample.get("schema_version") != "perfassess-calibration-sample-v1":
    raise SystemExit("manifest sample schema mismatch")
if not sample.get("score_profile") or not sample.get("calibration_version") or not sample.get("confidence_level"):
    raise SystemExit("manifest sample scoring summary missing")
if "mainstream_count" not in sample:
    raise SystemExit("manifest sample mainstream_count missing")
files = manifest.get("files")
if not isinstance(files, list) or len(files) != 1:
    raise SystemExit("manifest files missing")
expected = hashlib.sha256(sample_path.read_bytes()).hexdigest()
if files[0].get("path") != "calibration_sample.json" or files[0].get("sha256") != expected:
    raise SystemExit("manifest sha256 mismatch")
serialized = json.dumps(manifest, ensure_ascii=False)
if str(tmpdir) in serialized:
    raise SystemExit("manifest leaked source path")
for forbidden in ["public_ip", "isp", "asn", str(sample_path)]:
    if forbidden.lower() in serialized.lower():
        raise SystemExit(f"manifest leaked forbidden term: {forbidden}")
PY

cp -R "$tmpdir/dataset" "$tmpdir/tampered-dataset"
python3 - "$tmpdir/tampered-dataset/sample-a/calibration_sample.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
sample = json.loads(path.read_text(encoding="utf-8"))
sample.setdefault("scores", {})["grade"] = "tampered"
path.write_text(json.dumps(sample, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
if "$repo_root/scripts/calibration-summary.py" "$tmpdir/tampered-dataset" --require-manifest --format json -o "$tmpdir/tampered-summary.json" 2>"$tmpdir/tampered.err"; then
  echo "expected require-manifest to reject tampered calibration samples" >&2
  exit 1
fi
grep -q "sha256 does not match manifest" "$tmpdir/tampered.err"

PERFASSESS_CALIBRATION_LABEL=sample-manifest \
PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=1 \
"$repo_root/scripts/calibration-collect.sh" "$tmpdir/auto/calibration_sample.json" "$tmpdir/manifest-gated-dataset" >"$tmpdir/manifest-gated.out"
test -f "$tmpdir/manifest-gated-dataset/sample-manifest/manifest.json"
grep -q "calibration sample collected" "$tmpdir/manifest-gated.out"

if PERFASSESS_CALIBRATION_LABEL=sample-bad-manifest \
  PERFASSESS_CALIBRATION_REQUIRE_MANIFEST=maybe \
  "$repo_root/scripts/calibration-collect.sh" "$tmpdir/auto/calibration_sample.json" "$tmpdir/bad-manifest-gate" 2>"$tmpdir/bad-manifest-gate.err"; then
  echo "expected invalid manifest gate value to fail" >&2
  exit 1
fi
grep -q "PERFASSESS_CALIBRATION_REQUIRE_MANIFEST must be" "$tmpdir/bad-manifest-gate.err"

if PERFASSESS_CALIBRATION_LABEL=sample-a "$repo_root/scripts/calibration-collect.sh" "$tmpdir/auto/calibration_sample.json" "$tmpdir/dataset" 2>"$tmpdir/duplicate.err"; then
  echo "expected calibration collect to reject duplicate labels" >&2
  exit 1
fi
grep -q "target sample directory already exists" "$tmpdir/duplicate.err"

if PERFASSESS_CALIBRATION_LABEL='bad/label' "$repo_root/scripts/calibration-collect.sh" "$tmpdir/auto/calibration_sample.json" "$tmpdir/dataset" 2>"$tmpdir/bad-label.err"; then
  echo "expected calibration collect to reject unsafe labels" >&2
  exit 1
fi
grep -q "sample label must only contain" "$tmpdir/bad-label.err"

cp "$tmpdir/auto/calibration_sample.json" "$tmpdir/unredacted.json"
python3 - "$tmpdir/unredacted.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
sample = json.loads(path.read_text(encoding="utf-8"))
sample["redacted"] = False
path.write_text(json.dumps(sample, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

if PERFASSESS_CALIBRATION_LABEL=unredacted "$repo_root/scripts/calibration-collect.sh" "$tmpdir/unredacted.json" "$tmpdir/dataset" 2>"$tmpdir/unredacted.err"; then
  echo "expected calibration collect to reject unredacted samples" >&2
  exit 1
fi
grep -q "not marked redacted" "$tmpdir/unredacted.err"
test ! -e "$tmpdir/dataset/unredacted"

if PERFASSESS_CALIBRATION_LABEL=sample-b \
  PERFASSESS_CALIBRATION_REQUIRE_POLICY=candidate \
  "$repo_root/scripts/calibration-collect.sh" "$tmpdir/auto/calibration_sample.json" "$tmpdir/gated-dataset" >"$tmpdir/gated.out" 2>"$tmpdir/gated.err"; then
  echo "expected calibration collect candidate gate to fail for a tiny builtin dataset" >&2
  exit 1
fi
grep -q "dataset gate did not pass" "$tmpdir/gated.err"
test -f "$tmpdir/gated-dataset/sample-b/calibration_sample.json"
test -f "$tmpdir/gated-dataset/sample-b/manifest.json"
test -f "$tmpdir/gated-dataset/summary.md"
test -f "$tmpdir/gated-dataset/summary.json"
test -f "$tmpdir/gated-dataset/samples.jsonl"
grep -q "Perfassess Calibration Summary" "$tmpdir/gated-dataset/summary.md"
python3 -m json.tool "$tmpdir/gated-dataset/summary.json" >/dev/null

echo "calibration collect smoke passed"
