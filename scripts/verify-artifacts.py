#!/usr/bin/env python3
import argparse
import hashlib
import json
import sys
import zipfile
from pathlib import Path
from typing import Any


SCHEMA_VERSION = "perfassess-artifact-manifest-v1"


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_manifest(path: Path) -> dict[str, Any]:
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise SystemExit(f"{path}: failed to read artifact manifest: {exc}") from exc
    if not isinstance(manifest, dict) or manifest.get("schema_version") != SCHEMA_VERSION:
        raise SystemExit(f"{path}: unsupported artifact manifest schema")
    artifacts = manifest.get("artifacts")
    if not isinstance(artifacts, list):
        raise SystemExit(f"{path}: artifacts must be a list")
    return manifest


def load_manifest_from_archive(path: Path) -> dict[str, Any]:
    try:
        with zipfile.ZipFile(path) as archive:
            with archive.open("artifact_manifest.json") as handle:
                manifest = json.loads(handle.read().decode("utf-8"))
    except KeyError as exc:
        raise SystemExit(f"{path}: archive does not contain artifact_manifest.json") from exc
    except (OSError, zipfile.BadZipFile, json.JSONDecodeError, UnicodeDecodeError) as exc:
        raise SystemExit(f"{path}: failed to read artifact manifest from archive: {exc}") from exc
    if not isinstance(manifest, dict) or manifest.get("schema_version") != SCHEMA_VERSION:
        raise SystemExit(f"{path}: unsupported artifact manifest schema")
    artifacts = manifest.get("artifacts")
    if not isinstance(artifacts, list):
        raise SystemExit(f"{path}: artifacts must be a list")
    return manifest


def verify_directory(base_dir: Path, manifest: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    for item in manifest.get("artifacts", []):
        if not isinstance(item, dict):
            errors.append("artifact entry is not an object")
            continue
        rel = str(item.get("path") or "")
        if not rel:
            errors.append("artifact entry missing path")
            continue
        path = base_dir / rel
        expected_exists = item.get("exists") is True
        if expected_exists and not path.is_file():
            errors.append(f"{rel}: expected file is missing")
            continue
        if not expected_exists:
            continue
        expected_size = item.get("size_bytes")
        if expected_size is not None and path.stat().st_size != expected_size:
            errors.append(f"{rel}: size mismatch, expected {expected_size}, got {path.stat().st_size}")
        expected_hash = item.get("sha256")
        if expected_hash and sha256_file(path) != expected_hash:
            errors.append(f"{rel}: sha256 mismatch")
    return errors


def verify_archive(base_dir: Path, manifest: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    archive_name = manifest.get("summary", {}).get("archive") if isinstance(manifest.get("summary"), dict) else None
    if not archive_name:
        return errors
    archive_path = base_dir / str(archive_name)
    if not archive_path.is_file():
        errors.append(f"{archive_name}: archive is missing")
        return errors
    manifest_path = base_dir / "artifact_manifest.json"
    if manifest_path.is_file():
        try:
            with zipfile.ZipFile(archive_path) as archive:
                with archive.open("artifact_manifest.json") as handle:
                    archived_manifest = handle.read()
            if archived_manifest != manifest_path.read_bytes():
                errors.append("artifact_manifest.json: archived manifest differs from directory manifest")
        except KeyError:
            errors.append(f"{archive_name}: archive does not contain artifact_manifest.json")
            return errors
        except (OSError, zipfile.BadZipFile) as exc:
            errors.append(f"{archive_name}: failed to compare archived manifest: {exc}")
            return errors
    return verify_archive_file(archive_path, manifest)


def verify_archive_file(archive_path: Path, manifest: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    try:
        with zipfile.ZipFile(archive_path) as archive:
            names = set(archive.namelist())
            for item in manifest.get("artifacts", []):
                if not isinstance(item, dict):
                    errors.append("artifact entry is not an object")
                    continue
                rel = str(item.get("path") or "")
                if not rel:
                    errors.append("artifact entry missing path")
                    continue
                if item.get("included_in_archive") is not True:
                    continue
                if rel not in names:
                    errors.append(f"{archive_path.name}: missing archived artifact {rel}")
                    continue
                info = archive.getinfo(rel)
                expected_size = item.get("size_bytes")
                if expected_size is not None and info.file_size != expected_size:
                    errors.append(f"{rel}: archived size mismatch, expected {expected_size}, got {info.file_size}")
                expected_hash = item.get("sha256")
                if expected_hash:
                    digest = hashlib.sha256()
                    with archive.open(rel) as handle:
                        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                            digest.update(chunk)
                    if digest.hexdigest() != expected_hash:
                        errors.append(f"{rel}: archived sha256 mismatch")
    except zipfile.BadZipFile as exc:
        errors.append(f"{archive_path.name}: invalid zip file: {exc}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Verify a Perfassess artifact manifest.")
    parser.add_argument("path", help="Report output directory, artifact_manifest.json path, or perfassess-report.zip path.")
    parser.add_argument("--skip-archive", action="store_true", help="Only verify files on disk, not zip contents.")
    args = parser.parse_args()

    raw = Path(args.path)
    if raw.is_file() and raw.suffix.lower() == ".zip":
        manifest = load_manifest_from_archive(raw)
        errors = verify_archive_file(raw, manifest)
        if errors:
            for error in errors:
                print(f"[FAIL] {error}", file=sys.stderr)
            return 1
        summary = manifest.get("summary") if isinstance(manifest.get("summary"), dict) else {}
        print(
            "artifact archive verification passed: "
            f"{summary.get('existing_artifacts', 0)}/{summary.get('total_artifacts', 0)} artifacts recorded"
        )
        return 0

    manifest_path = raw / "artifact_manifest.json" if raw.is_dir() else raw
    base_dir = manifest_path.parent
    manifest = load_manifest(manifest_path)
    errors = verify_directory(base_dir, manifest)
    if not args.skip_archive:
        errors.extend(verify_archive(base_dir, manifest))
    if errors:
        for error in errors:
            print(f"[FAIL] {error}", file=sys.stderr)
        return 1

    summary = manifest.get("summary") if isinstance(manifest.get("summary"), dict) else {}
    print(
        "artifact verification passed: "
        f"{summary.get('existing_artifacts', 0)}/{summary.get('total_artifacts', 0)} artifacts present"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
