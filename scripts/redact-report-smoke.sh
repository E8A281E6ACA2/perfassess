#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cat >"$tmpdir/report.json" <<'JSON'
{
  "session_id": "session_123",
  "system_info": {
    "ip_info": {
      "public_ip": "8.8.8.8",
      "isp": "Example ISP",
      "geo_location": {
        "country": "United States",
        "country_code": "US",
        "city": "Mountain View",
        "latitude": 37.386,
        "longitude": -122.0838
      }
    }
  },
  "test_results": {
    "network_result": {
      "metrics": {
        "speedtest_external_ip": "1.1.1.1",
        "speedtest_isp": "Example Speedtest ISP",
        "download_speed_mbps": 100.0
      }
    }
  },
  "summary": {
    "vps_benchmark_summary": {
      "system": {
        "public_ip": "203.0.113.10",
        "isp": "Example ISP",
        "location": "Example City"
      }
    },
    "ip_quality_report": {
      "public_ip": "8.8.4.4",
      "asn": "AS15169",
      "organization": "Example Org",
      "reverse_dns": ["dns.google"],
      "risk_score": 10
    },
    "route_trace_results": [
      {
        "target": "1.1.1.1",
        "hops": [{"ip": "8.8.8.8", "hostname": "dns.google"}],
        "success": true
      }
    ],
    "score_calibration": {
      "version": "2026-06-v1"
    }
  }
}
JSON

python3 "$repo_root/scripts/redact-report.py" "$tmpdir/report.json" -o "$tmpdir/report.redacted.json"
python3 -m json.tool "$tmpdir/report.redacted.json" >/dev/null

python3 - "$tmpdir/report.redacted.json" <<'PY'
import json
import pathlib
import sys

data = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
if data.get("redacted") is not True:
    raise SystemExit("redacted marker missing")
if data.get("system_info", {}).get("ip_info", {}).get("public_ip") != "REDACTED":
    raise SystemExit("system public_ip was not redacted")
if data.get("test_results", {}).get("network_result", {}).get("metrics", {}).get("speedtest_external_ip") != "REDACTED":
    raise SystemExit("speedtest external IP was not redacted")
if data.get("test_results", {}).get("network_result", {}).get("metrics", {}).get("speedtest_isp") != "REDACTED":
    raise SystemExit("speedtest ISP was not redacted")
system = data.get("summary", {}).get("vps_benchmark_summary", {}).get("system", {})
if system.get("public_ip") != "REDACTED" or system.get("isp") != "REDACTED" or system.get("location") != "REDACTED":
    raise SystemExit("vps system identity fields were not redacted")
ip_quality = data.get("summary", {}).get("ip_quality_report", {})
if ip_quality.get("asn") != "REDACTED" or ip_quality.get("organization") != "REDACTED" or ip_quality.get("reverse_dns") != []:
    raise SystemExit("ip quality identity fields were not redacted")
route = data.get("summary", {}).get("route_trace_results", [{}])[0]
if route.get("hops") != []:
    raise SystemExit("route hops were not removed")
if data.get("summary", {}).get("score_calibration", {}).get("version") != "2026-06-v1":
    raise SystemExit("non-sensitive score calibration was not preserved")

for forbidden in ["8.8.8.8", "8.8.4.4", "1.1.1.1", "Example ISP", "Example Speedtest ISP", "Example Org", "dns.google", "Mountain View"]:
    if forbidden in json.dumps(data, ensure_ascii=False):
        raise SystemExit(f"sensitive value remained after redaction: {forbidden}")
PY

mkdir -p "$tmpdir/reports"
cp "$tmpdir/report.json" "$tmpdir/reports/quick.json"
python3 "$repo_root/scripts/redact-report.py" "$tmpdir/reports"
test -f "$tmpdir/reports/quick.redacted.json"

echo "redact report smoke passed"
