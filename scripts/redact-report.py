#!/usr/bin/env python3
import argparse
import ipaddress
import json
import re
from pathlib import Path
from typing import Any


REDACTED = "REDACTED"
IPV4_RE = re.compile(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])")
SENSITIVE_KEYS = {
    "public_ip",
    "speedtest_external_ip",
    "external_ip",
    "isp",
    "speedtest_isp",
    "asn",
    "organization",
    "reverse_dns",
    "location",
    "country",
    "country_code",
    "city",
    "latitude",
    "longitude",
}


def is_public_ipv4(value: str) -> bool:
    try:
        ip = ipaddress.ip_address(value)
    except ValueError:
        return False
    return ip.version == 4 and ip.is_global


def redact_public_ipv4s(value: str) -> str:
    def replace(match: re.Match[str]) -> str:
        candidate = match.group(0)
        if is_public_ipv4(candidate):
            return REDACTED
        return candidate

    return IPV4_RE.sub(replace, value)


def redact_value(key: str, value: Any) -> Any:
    if key in {"latitude", "longitude"}:
        return None
    if isinstance(value, list):
        return []
    if isinstance(value, dict):
        return {child_key: redact_value(child_key, child_value) for child_key, child_value in value.items()}
    return REDACTED


def redact_node(value: Any) -> Any:
    if isinstance(value, dict):
        redacted: dict[str, Any] = {}
        for key, child in value.items():
            key_lower = key.lower()
            if key_lower == "hops":
                redacted[key] = []
                continue
            if key_lower in SENSITIVE_KEYS:
                redacted[key] = redact_value(key_lower, child)
                continue
            redacted[key] = redact_node(child)
        return redacted
    if isinstance(value, list):
        return [redact_node(item) for item in value]
    if isinstance(value, str):
        return redact_public_ipv4s(value)
    return value


def redact_file(input_path: Path, output_path: Path) -> None:
    try:
        data = json.loads(input_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise SystemExit(f"{input_path}: failed to read JSON report: {exc}") from exc
    redacted = redact_node(data)
    if isinstance(redacted, dict):
        redacted.setdefault("redacted", True)
        redacted.setdefault("redaction_note", "Sensitive IP, ISP, ASN, reverse DNS, geolocation, and route hop details were redacted by scripts/redact-report.py.")
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(redacted, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def output_for(input_path: Path, output_arg: str = "") -> Path:
    if output_arg:
        return Path(output_arg)
    if input_path.suffix.lower() == ".json":
        return input_path.with_name(input_path.stem + ".redacted.json")
    return input_path.with_name(input_path.name + ".redacted.json")


def iter_json_files(path: Path) -> list[Path]:
    if path.is_file():
        return [path]
    if not path.is_dir():
        raise SystemExit(f"{path}: input path does not exist")
    return sorted(
        item
        for item in path.rglob("*.json")
        if item.is_file() and not item.name.endswith(".redacted.json")
    )


def main() -> int:
    parser = argparse.ArgumentParser(description="Redact sensitive fields from Perfassess JSON reports.")
    parser.add_argument("input", help="Input JSON report file or directory.")
    parser.add_argument("-o", "--output", default="", help="Output file path. Only valid when input is a file.")
    args = parser.parse_args()

    input_path = Path(args.input)
    files = iter_json_files(input_path)
    if not files:
        raise SystemExit(f"{input_path}: no JSON reports found")
    if args.output and len(files) != 1:
        raise SystemExit("--output can only be used with a single input file")
    for file_path in files:
        redact_file(file_path, output_for(file_path, args.output))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
