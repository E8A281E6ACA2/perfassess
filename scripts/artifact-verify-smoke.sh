#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
binary="${1:-${PERFASSESS_BINARY:-$repo_root/build/perfassess}}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

PERFASSESS_AUTO_DIR="$tmpdir/auto" \
PERFASSESS_AUTO_PROFILE=basic \
PERFASSESS_AUTO_ACCEPTANCE=0 \
PERFASSESS_AUTO_PROGRESS=0 \
PERFASSESS_SKIP_BUILD=1 \
PERFASSESS_BINARY="$binary" \
"$repo_root/scripts/perfassess-auto.sh" >/dev/null

python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto" >/dev/null
python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto/perfassess-report.zip" >/dev/null

python3 - "$tmpdir/auto/artifact_manifest.json" "$tmpdir/auto/perfassess-report.zip" <<'PY'
import json
import sys
import zipfile
from pathlib import Path

manifest = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
entries = {
    item.get("path"): item
    for item in manifest.get("artifacts", [])
    if isinstance(item, dict)
}

for path in ["build.stdout.txt", "build.stderr.log"]:
    item = entries.get(path)
    if not item or not item.get("exists") or not item.get("included_in_archive"):
        raise SystemExit(f"{path} missing from artifact manifest or archive metadata")

with zipfile.ZipFile(sys.argv[2], "r") as archive:
    names = set(archive.namelist())
    for path in ["build.stdout.txt", "build.stderr.log"]:
        if path not in names:
            raise SystemExit(f"{path} missing from report archive")
PY

cp "$tmpdir/auto/perfassess-report.zip" "$tmpdir/original-report.zip"
cp "$tmpdir/auto/default.json" "$tmpdir/original-default.json"

printf '\n# tampered\n' >>"$tmpdir/auto/default.json"
if python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto" >"$tmpdir/tamper.out" 2>"$tmpdir/tamper.err"; then
  echo "expected artifact verification to fail after tampering with default.json" >&2
  exit 1
fi
grep -q "default.json: sha256 mismatch" "$tmpdir/tamper.err"
cp "$tmpdir/original-report.zip" "$tmpdir/auto/perfassess-report.zip"
cp "$tmpdir/original-default.json" "$tmpdir/auto/default.json"

python3 - "$tmpdir/auto/perfassess-report.zip" "$tmpdir/tampered-report.zip" <<'PY'
import sys
import zipfile
from pathlib import Path

source = Path(sys.argv[1])
target = Path(sys.argv[2])

with zipfile.ZipFile(source, "r") as src, zipfile.ZipFile(target, "w", zipfile.ZIP_DEFLATED) as dst:
    for info in src.infolist():
        if info.filename == "default.json":
            data = src.read(info.filename) + b"\n# tampered\n"
        else:
            data = src.read(info.filename)
        dst.writestr(info, data)
PY

if python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/tampered-report.zip" >"$tmpdir/tamper-zip.out" 2>"$tmpdir/tamper-zip.err"; then
  echo "expected artifact verification to fail after tampering with archived default.json" >&2
  exit 1
fi
grep -q "default.json: archived sha256 mismatch" "$tmpdir/tamper-zip.err"

cp "$tmpdir/tampered-report.zip" "$tmpdir/auto/perfassess-report.zip"
if python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto" >"$tmpdir/tamper-dir-zip.out" 2>"$tmpdir/tamper-dir-zip.err"; then
  echo "expected artifact verification to fail after tampering with archived default.json in report directory" >&2
  exit 1
fi
grep -q "default.json: archived sha256 mismatch" "$tmpdir/tamper-dir-zip.err"

python3 - "$tmpdir/auto/artifact_manifest.json" "$tmpdir/original-report.zip" "$tmpdir/manifest-drift-report.zip" <<'PY'
import json
import sys
import zipfile
from pathlib import Path

manifest_path = Path(sys.argv[1])
source = Path(sys.argv[2])
target = Path(sys.argv[3])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
manifest["summary"]["existing_artifacts"] = -1
manifest_bytes = json.dumps(manifest, ensure_ascii=False, indent=2).encode("utf-8") + b"\n"

with zipfile.ZipFile(source, "r") as src, zipfile.ZipFile(target, "w", zipfile.ZIP_DEFLATED) as dst:
    for info in src.infolist():
        data = manifest_bytes if info.filename == "artifact_manifest.json" else src.read(info.filename)
        dst.writestr(info, data)
PY

cp "$tmpdir/manifest-drift-report.zip" "$tmpdir/auto/perfassess-report.zip"
if python3 "$repo_root/scripts/verify-artifacts.py" "$tmpdir/auto" >"$tmpdir/manifest-drift.out" 2>"$tmpdir/manifest-drift.err"; then
  echo "expected artifact verification to fail after archived manifest drift" >&2
  exit 1
fi
grep -Eq "artifact_manifest.json: archived manifest differs from directory manifest|perfassess-report.zip: (size|sha256) mismatch" "$tmpdir/manifest-drift.err"

echo "artifact verification smoke passed"
