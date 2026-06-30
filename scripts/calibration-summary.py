#!/usr/bin/env python3
import argparse
import hashlib
import ipaddress
import json
import re
import statistics
import sys
from pathlib import Path
from typing import Any


SAMPLE_SCHEMA = "perfassess-calibration-sample-v1"
COMPONENTS = ["cpu", "memory", "disk", "network"]
METRICS = {
    "overall": ["total_score"],
    "cpu": ["score", "single_core_score", "multi_core_score", "total_score"],
    "memory": ["score", "read_mbps", "write_mbps"],
    "disk": ["score", "sequential_read_mbps", "sequential_write_mbps", "random_iops"],
    "network": ["score", "latency_ms", "download_mbps", "upload_mbps", "quality_jitter_ms", "quality_failure_rate"],
}
BASELINE_METRICS = {
    "memory_read_base_mbps": ("memory", "read_mbps", "higher"),
    "memory_write_base_mbps": ("memory", "write_mbps", "higher"),
    "disk_sequential_read_mbps": ("disk", "sequential_read_mbps", "higher"),
    "disk_sequential_write_mbps": ("disk", "sequential_write_mbps", "higher"),
    "disk_random_iops": ("disk", "random_iops", "higher"),
    "network_latency_base_ms": ("network", "latency_ms", "lower"),
    "network_download_base_mbps": ("network", "download_mbps", "higher"),
    "network_upload_base_mbps": ("network", "upload_mbps", "higher"),
}
CONFIDENCE_RANK = {"low": 0, "medium": 1, "high": 2}
POLICY_RANK = {"exploratory": 0, "candidate": 1, "formal": 2}
FORBIDDEN_SAMPLE_KEYS = [
    "public_ip",
    "isp",
    "asn",
    "organization",
    "reverse_dns",
    "route_trace_results",
    "hops",
    "session_id",
]
FORBIDDEN_TOP_LEVEL = ["system_info", "test_results", "summary", "formatted_content", "session_id"]
ALLOWED_ENVIRONMENT_KEYS = {
    "cpu_model",
    "cpu_cores",
    "cpu_threads",
    "memory_total_mb",
    "disk_total_gb",
    "os",
    "architecture",
    "virtualization",
}
IPV4_PATTERN = re.compile(r"\b(?:\d{1,3}\.){3}\d{1,3}\b")


def text(value: Any, default: str = "-") -> str:
    if value is None:
        return default
    if isinstance(value, str):
        value = value.strip()
        return value if value else default
    return str(value)


def data_get(data: dict[str, Any], *keys: str, default: Any = None) -> Any:
    current: Any = data
    for key in keys:
        if not isinstance(current, dict):
            return default
        current = current.get(key)
        if current is None:
            return default
    return current


def number(value: Any) -> float | None:
    if value is None or isinstance(value, bool):
        return None
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def percentile(values: list[float], pct: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * pct
    lower = int(rank)
    upper = min(lower + 1, len(ordered) - 1)
    fraction = rank - lower
    return ordered[lower] + (ordered[upper] - ordered[lower]) * fraction


def stats(values: list[float]) -> dict[str, Any]:
    clean = [value for value in values if value is not None]
    if not clean:
        return {"count": 0}
    return {
        "count": len(clean),
        "min": min(clean),
        "max": max(clean),
        "avg": statistics.fmean(clean),
        "p50": percentile(clean, 0.50),
        "p75": percentile(clean, 0.75),
        "p90": percentile(clean, 0.90),
    }


def find_sample_paths(inputs: list[str]) -> list[Path]:
    paths: list[Path] = []
    for raw in inputs:
        path = Path(raw)
        if path.is_file():
            paths.append(path)
            continue
        if path.is_dir():
            direct = path / "calibration_sample.json"
            if direct.is_file():
                paths.append(direct)
            paths.extend(path.rglob("calibration_sample.json"))
    deduped: list[Path] = []
    seen: set[Path] = set()
    for path in paths:
        resolved = path.resolve()
        if resolved in seen:
            continue
        seen.add(resolved)
        deduped.append(resolved)
    return deduped


def load_sample(path: Path) -> dict[str, Any]:
    try:
        sample = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise SystemExit(f"{path}: failed to read calibration sample: {exc}") from exc
    if not isinstance(sample, dict) or sample.get("schema_version") != SAMPLE_SCHEMA:
        raise SystemExit(f"{path}: unsupported calibration sample schema")
    if sample.get("redacted") is not True:
        raise SystemExit(f"{path}: calibration sample is not marked redacted")
    validate_sample_privacy(sample, path)
    sample["_path"] = str(path)
    return sample


def validate_sample_manifest(path: Path) -> None:
    manifest_path = path.parent / "manifest.json"
    if not manifest_path.is_file():
        raise SystemExit(f"{path}: manifest.json is required but missing")
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise SystemExit(f"{path}: failed to read sample manifest: {exc}") from exc
    if not isinstance(manifest, dict) or manifest.get("schema_version") != "perfassess-calibration-sample-manifest-v1":
        raise SystemExit(f"{path}: unsupported sample manifest schema")
    files = manifest.get("files")
    if not isinstance(files, list):
        raise SystemExit(f"{path}: sample manifest files must be a list")
    sample_entry = None
    for item in files:
        if isinstance(item, dict) and item.get("path") == "calibration_sample.json":
            sample_entry = item
            break
    if not sample_entry:
        raise SystemExit(f"{path}: sample manifest does not reference calibration_sample.json")
    expected = sample_entry.get("sha256")
    if not isinstance(expected, str) or not re.fullmatch(r"[0-9a-fA-F]{64}", expected):
        raise SystemExit(f"{path}: sample manifest has invalid calibration_sample.json sha256")
    actual = hashlib.sha256(path.read_bytes()).hexdigest()
    if actual.lower() != expected.lower():
        raise SystemExit(f"{path}: calibration_sample.json sha256 does not match manifest")


def validate_sample_privacy(sample: dict[str, Any], path: Path) -> None:
    privacy_note = sample.get("privacy_note")
    if not isinstance(privacy_note, str) or "已排除" not in privacy_note or "原始日志" not in privacy_note:
        raise SystemExit(f"{path}: calibration sample privacy_note does not document redaction boundary")
    for key in FORBIDDEN_TOP_LEVEL:
        if key in sample:
            raise SystemExit(f"{path}: calibration sample embeds raw report field: {key}")
    environment = sample.get("environment")
    if not isinstance(environment, dict):
        raise SystemExit(f"{path}: calibration sample environment must be an object")
    unexpected_environment_keys = sorted(set(environment) - ALLOWED_ENVIRONMENT_KEYS)
    if unexpected_environment_keys:
        raise SystemExit(f"{path}: calibration sample environment has unexpected keys: {unexpected_environment_keys}")

    def walk(value: Any, location: str = "$") -> None:
        if isinstance(value, dict):
            for key, child in value.items():
                key_lower = str(key).lower()
                for forbidden in FORBIDDEN_SAMPLE_KEYS:
                    if forbidden in key_lower:
                        raise SystemExit(f"{path}: calibration sample leaks forbidden key at {location}.{key}")
                walk(child, f"{location}.{key}")
        elif isinstance(value, list):
            for index, child in enumerate(value):
                walk(child, f"{location}[{index}]")
        elif isinstance(value, str):
            for candidate in IPV4_PATTERN.findall(value):
                try:
                    address = ipaddress.ip_address(candidate)
                except ValueError:
                    continue
                if address.is_global:
                    raise SystemExit(f"{path}: calibration sample leaks public IPv4 value at {location}: {candidate}")

    walk(sample)


def sample_scores(sample: dict[str, Any]) -> dict[str, Any]:
    return sample.get("scores") if isinstance(sample.get("scores"), dict) else {}


def sample_benchmark_profile(sample: dict[str, Any]) -> dict[str, Any]:
    profile = sample.get("benchmark_profile")
    return profile if isinstance(profile, dict) else {}


def backend_signature(sample: dict[str, Any]) -> str:
    components = sample.get("component_samples") if isinstance(sample.get("component_samples"), dict) else {}
    parts = []
    for key in COMPONENTS:
        item = components.get(key) if isinstance(components.get(key), dict) else {}
        parts.append(f"{key}:{text(item.get('backend'))}")
    return ",".join(parts)


def sample_row(sample: dict[str, Any]) -> dict[str, Any]:
    scores = sample_scores(sample)
    env = sample.get("environment") if isinstance(sample.get("environment"), dict) else {}
    benchmark = sample_benchmark_profile(sample)
    return {
        "path": sample.get("_path"),
        "score_profile": scores.get("score_profile"),
        "calibration_version": scores.get("calibration_version"),
        "grade": scores.get("grade"),
        "confidence_level": scores.get("confidence_level"),
        "total_score": scores.get("total_score"),
        "cpu_model": env.get("cpu_model"),
        "cpu_threads": env.get("cpu_threads"),
        "memory_total_mb": env.get("memory_total_mb"),
        "disk_total_gb": env.get("disk_total_gb"),
        "virtualization": env.get("virtualization"),
        "benchmark_profile": benchmark.get("name"),
        "mainstream_count": benchmark.get("mainstream_count"),
        "backend_signature": backend_signature(sample),
    }


def sample_matches_filters(
    sample: dict[str, Any],
    score_profile: str = "",
    min_confidence: str = "",
    require_mainstream: bool = False,
    min_mainstream_count: int | None = None,
) -> tuple[bool, str]:
    scores = sample_scores(sample)
    benchmark = sample_benchmark_profile(sample)
    if score_profile and scores.get("score_profile") != score_profile:
        return False, f"score_profile={scores.get('score_profile')}"
    if min_confidence:
        current = CONFIDENCE_RANK.get(text(scores.get("confidence_level"), "").lower(), -1)
        required = CONFIDENCE_RANK.get(min_confidence, 0)
        if current < required:
            return False, f"confidence={scores.get('confidence_level')}"
    mainstream_count = int(number(benchmark.get("mainstream_count")) or 0)
    if require_mainstream and mainstream_count < 4:
        return False, f"mainstream_count={mainstream_count}"
    if min_mainstream_count is not None and mainstream_count < min_mainstream_count:
        return False, f"mainstream_count={mainstream_count}"
    return True, ""


def filter_samples(
    samples: list[dict[str, Any]],
    score_profile: str = "",
    min_confidence: str = "",
    require_mainstream: bool = False,
    min_mainstream_count: int | None = None,
) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    kept = []
    rejected: dict[str, int] = {}
    for sample in samples:
        ok, reason = sample_matches_filters(sample, score_profile, min_confidence, require_mainstream, min_mainstream_count)
        if ok:
            kept.append(sample)
        else:
            rejected[reason] = rejected.get(reason, 0) + 1
    filters = {
        "score_profile": score_profile or None,
        "min_confidence": min_confidence or None,
        "require_mainstream": require_mainstream,
        "min_mainstream_count": min_mainstream_count,
    }
    return kept, {"filters": filters, "input_count": len(samples), "kept_count": len(kept), "rejected": rejected}


def metric_values(samples: list[dict[str, Any]], component: str, metric: str) -> list[float]:
    values: list[float] = []
    for sample in samples:
        if component == "overall":
            scores = sample.get("scores") if isinstance(sample.get("scores"), dict) else {}
            value = number(scores.get(metric))
        else:
            components = sample.get("component_samples") if isinstance(sample.get("component_samples"), dict) else {}
            item = components.get(component) if isinstance(components.get(component), dict) else {}
            module = item.get("module") if isinstance(item.get("module"), dict) else {}
            if module.get("status") and module.get("status") != "success":
                continue
            value = number(item.get(metric))
        if value is not None:
            values.append(value)
    return values


def summarize_group(samples: list[dict[str, Any]]) -> dict[str, Any]:
    metrics: dict[str, Any] = {}
    for component, names in METRICS.items():
        metrics[component] = {}
        for name in names:
            metrics[component][name] = stats(metric_values(samples, component, name))
    grades: dict[str, int] = {}
    confidences: dict[str, int] = {}
    backends: dict[str, int] = {}
    for sample in samples:
        row = sample_row(sample)
        grades[text(row.get("grade"))] = grades.get(text(row.get("grade")), 0) + 1
        confidences[text(row.get("confidence_level"))] = confidences.get(text(row.get("confidence_level")), 0) + 1
        backends[text(row.get("backend_signature"))] = backends.get(text(row.get("backend_signature")), 0) + 1
    return {
        "count": len(samples),
        "metrics": metrics,
        "grades": grades,
        "confidence_levels": confidences,
        "backend_signatures": backends,
        "policy": dataset_policy(samples),
    }


def summarize(
    samples: list[dict[str, Any]],
    filter_info: dict[str, Any] | None = None,
    manifest_policy: dict[str, Any] | None = None,
) -> dict[str, Any]:
    by_profile: dict[str, list[dict[str, Any]]] = {}
    by_calibration: dict[str, list[dict[str, Any]]] = {}
    for sample in samples:
        scores = sample.get("scores") if isinstance(sample.get("scores"), dict) else {}
        by_profile.setdefault(text(scores.get("score_profile")), []).append(sample)
        by_calibration.setdefault(text(scores.get("calibration_version")), []).append(sample)
    return {
        "schema_version": "perfassess-calibration-summary-v1",
        "sample_count": len(samples),
        "filter_info": filter_info or {"input_count": len(samples), "kept_count": len(samples), "rejected": {}},
        "manifest_policy": manifest_policy or {"required": False, "checked_count": 0},
        "samples": [sample_row(sample) for sample in samples],
        "overall": summarize_group(samples),
        "by_score_profile": {key: summarize_group(value) for key, value in sorted(by_profile.items())},
        "profile_policies": {key: dataset_policy(value) for key, value in sorted(by_profile.items())},
        "by_calibration_version": {key: summarize_group(value) for key, value in sorted(by_calibration.items())},
        "recommendations": calibration_recommendations(by_profile),
        "policy": dataset_policy(samples),
    }


def count_by(samples: list[dict[str, Any]], getter) -> dict[str, int]:
    counts: dict[str, int] = {}
    for sample in samples:
        value = text(getter(sample))
        counts[value] = counts.get(value, 0) + 1
    return counts


def share(count: int, total: int) -> float:
    return float(count) / float(total) if total > 0 else 0.0


def dataset_policy(samples: list[dict[str, Any]]) -> dict[str, Any]:
    total = len(samples)
    blockers: list[str] = []
    warnings: list[str] = []
    plan: list[dict[str, Any]] = []
    if total < 8:
        missing = 8 - total
        blockers.append(f"样本数 {total} 小于候选校准最低要求 8。")
        plan.append({
            "target": "candidate",
            "priority": "high",
            "action": "collect_more_samples",
            "missing": missing,
            "detail": f"至少还需要 {missing} 份满足 candidate 质量门槛的脱敏样本。",
        })
    elif total < 20:
        missing = 20 - total
        warnings.append(f"样本数 {total} 小于正式校准建议 20，只适合作为候选校准观察。")
        plan.append({
            "target": "formal",
            "priority": "medium",
            "action": "collect_more_samples",
            "missing": missing,
            "detail": f"正式校准建议至少再补 {missing} 份同口径高置信样本。",
        })

    calibration_versions = count_by(samples, lambda sample: sample_scores(sample).get("calibration_version"))
    if len(calibration_versions) > 1:
        blockers.append("样本混用了多个 score_calibration.version。")

    confidence_counts = count_by(samples, lambda sample: sample_scores(sample).get("confidence_level"))
    high_count = confidence_counts.get("high", 0)
    medium_plus = high_count + confidence_counts.get("medium", 0)
    if high_count < total:
        warnings.append("正式校准要求全部样本 confidence_level=high。")
        plan.append({
            "target": "formal",
            "priority": "medium",
            "action": "raise_confidence",
            "missing": total - high_count,
            "detail": f"仍有 {total - high_count} 份样本低于 high，正式校准前应使用主流后端或补测提升置信度。",
        })
    if medium_plus < total:
        blockers.append("候选校准要求全部样本 confidence_level 至少为 medium。")
        plan.append({
            "target": "candidate",
            "priority": "high",
            "action": "raise_confidence",
            "missing": total - medium_plus,
            "detail": f"仍有 {total - medium_plus} 份样本低于 medium，应重新采集或排除。",
        })

    mainstream_counts = [
        int(number(sample_benchmark_profile(sample).get("mainstream_count")) or 0)
        for sample in samples
    ]
    mainstream_all = sum(1 for value in mainstream_counts if value >= 4)
    mainstream_three = sum(1 for value in mainstream_counts if value >= 3)
    if mainstream_all < total:
        warnings.append("正式校准要求全部样本 mainstream_count=4。")
        plan.append({
            "target": "formal",
            "priority": "medium",
            "action": "increase_mainstream_coverage",
            "missing": total - mainstream_all,
            "detail": f"仍有 {total - mainstream_all} 份样本未覆盖四项主流后端。",
        })
    if mainstream_three < total:
        blockers.append("候选校准要求全部样本 mainstream_count 至少为 3。")
        plan.append({
            "target": "candidate",
            "priority": "high",
            "action": "increase_mainstream_coverage",
            "missing": total - mainstream_three,
            "detail": f"仍有 {total - mainstream_three} 份样本 mainstream_count 低于 3，应优先补 sysbench/fio/speedtest 或授权 iperf3。",
        })

    virtualization_counts = count_by(samples, lambda sample: (sample.get("environment") or {}).get("virtualization") if isinstance(sample.get("environment"), dict) else None)
    cpu_counts = count_by(samples, lambda sample: (sample.get("environment") or {}).get("cpu_model") if isinstance(sample.get("environment"), dict) else None)
    architecture_counts = count_by(samples, lambda sample: (sample.get("environment") or {}).get("architecture") if isinstance(sample.get("environment"), dict) else None)
    if virtualization_counts:
        top_virtualization, top_count = max(virtualization_counts.items(), key=lambda item: item[1])
        if share(top_count, total) > 0.70:
            warnings.append(f"虚拟化类型 {top_virtualization} 占比超过 70%。")
            plan.append({
                "target": "formal",
                "priority": "medium",
                "action": "diversify_virtualization",
                "missing": 0,
                "detail": f"虚拟化类型 {top_virtualization} 过于集中，应补充其他虚拟化或云厂商样本。",
            })
    if cpu_counts:
        top_cpu, top_count = max(cpu_counts.items(), key=lambda item: item[1])
        if share(top_count, total) > 0.40:
            warnings.append(f"CPU 型号 {top_cpu} 占比超过 40%。")
            plan.append({
                "target": "formal",
                "priority": "medium",
                "action": "diversify_cpu_model",
                "missing": 0,
                "detail": f"CPU 型号 {top_cpu} 过于集中，应补充不同代际、架构或供应商样本。",
            })

    if blockers:
        level = "exploratory"
        passed = False
    elif warnings:
        level = "candidate"
        passed = False
    else:
        level = "formal"
        passed = True

    return {
        "level": level,
        "formal_ready": passed,
        "sample_count": total,
        "blockers": blockers,
        "warnings": warnings,
        "confidence_counts": confidence_counts,
        "mainstream": {
            "all_four": mainstream_all,
            "at_least_three": mainstream_three,
        },
        "collection_plan": plan,
        "coverage": {
            "virtualization": virtualization_counts,
            "cpu_model": cpu_counts,
            "architecture": architecture_counts,
            "calibration_version": calibration_versions,
        },
    }


def active_baselines(samples: list[dict[str, Any]], profile: str) -> dict[str, Any]:
    for sample in samples:
        calibration = sample.get("score_calibration") if isinstance(sample.get("score_calibration"), dict) else {}
        profiles = calibration.get("profiles") if isinstance(calibration.get("profiles"), dict) else {}
        profile_baselines = profiles.get(profile)
        if isinstance(profile_baselines, dict):
            return profile_baselines
        active = calibration.get("active_baselines")
        scores = sample.get("scores") if isinstance(sample.get("scores"), dict) else {}
        if scores.get("score_profile") == profile and isinstance(active, dict):
            return active
    return {}


def round_baseline(value: float, metric: str) -> float:
    if metric == "disk_random_iops":
        return round(value / 100) * 100
    if metric == "network_latency_base_ms":
        return round(value, 1)
    if value >= 1000:
        return round(value / 50) * 50
    if value >= 100:
        return round(value / 10) * 10
    return round(value, 1)


def recommendation_action(current: float | None, proposed: float | None, minimum_change: float) -> str:
    if current is None or proposed is None or current <= 0:
        return "collect_more"
    ratio = (proposed - current) / current
    if abs(ratio) < minimum_change:
        return "keep"
    if ratio > 0:
        return "raise"
    return "lower"


def calibration_recommendations(by_profile: dict[str, list[dict[str, Any]]], minimum_samples: int = 5, minimum_change: float = 0.10) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for profile, samples in sorted(by_profile.items()):
        baselines = active_baselines(samples, profile)
        summary = summarize_group(samples)
        profile_result = {
            "sample_count": len(samples),
            "minimum_samples": minimum_samples,
            "ready": len(samples) >= minimum_samples,
            "baseline_suggestions": {},
            "notes": [],
        }
        if len(samples) < minimum_samples:
            profile_result["notes"].append(f"样本数 {len(samples)} 小于建议最小值 {minimum_samples}，仅展示候选值，不建议直接调整阈值。")
        for baseline_key, (component, metric, direction) in BASELINE_METRICS.items():
            metric_stats = summary.get("metrics", {}).get(component, {}).get(metric, {})
            current = number(baselines.get(baseline_key))
            if direction == "lower":
                candidate = number(metric_stats.get("p75") or metric_stats.get("p50"))
                basis = "p75"
            else:
                candidate = number(metric_stats.get("p75") or metric_stats.get("p50"))
                basis = "p75"
            proposed = round_baseline(candidate, baseline_key) if candidate is not None else None
            action = recommendation_action(current, proposed, minimum_change)
            if len(samples) < minimum_samples:
                action = "collect_more"
            profile_result["baseline_suggestions"][baseline_key] = {
                "component": component,
                "metric": metric,
                "direction": direction,
                "sample_count": metric_stats.get("count", 0),
                "current": current,
                "candidate": candidate,
                "suggested": proposed,
                "basis": basis,
                "action": action,
                "delta_percent": ((proposed - current) / current * 100) if current and proposed is not None else None,
            }
        result[profile] = profile_result
    return result


def fmt(value: Any, digits: int = 2) -> str:
    value = number(value)
    if value is None:
        return "-"
    return f"{value:.{digits}f}"


def metric_markdown_rows(summary: dict[str, Any], component: str, metric_names: list[str]) -> list[str]:
    rows = []
    metrics = summary.get("metrics", {}).get(component, {})
    for name in metric_names:
        item = metrics.get(name, {})
        rows.append(
            f"| {component}.{name} | {item.get('count', 0)} | {fmt(item.get('avg'))} | {fmt(item.get('p50'))} | {fmt(item.get('p75'))} | {fmt(item.get('p90'))} | {fmt(item.get('min'))} | {fmt(item.get('max'))} |"
        )
    return rows


def render_markdown(result: dict[str, Any]) -> str:
    filter_info = result.get("filter_info") if isinstance(result.get("filter_info"), dict) else {}
    rejected = filter_info.get("rejected") if isinstance(filter_info.get("rejected"), dict) else {}
    lines = [
        "# Perfassess Calibration Summary",
        "",
        f"- sample_count: {result.get('sample_count', 0)}",
        f"- input_count: {filter_info.get('input_count', result.get('sample_count', 0))}",
        f"- rejected_count: {sum(rejected.values()) if rejected else 0}",
        f"- manifest_required: {text(data_get(result, 'manifest_policy', 'required'))}",
        f"- manifest_checked_count: {text(data_get(result, 'manifest_policy', 'checked_count'))}",
        f"- policy_level: {text(data_get(result, 'policy', 'level'))}",
        f"- formal_ready: {text(data_get(result, 'policy', 'formal_ready'))}",
        "",
        "## Dataset Policy",
        "",
    ]
    policy = result.get("policy") if isinstance(result.get("policy"), dict) else {}
    blockers = policy.get("blockers") if isinstance(policy.get("blockers"), list) else []
    warnings = policy.get("warnings") if isinstance(policy.get("warnings"), list) else []
    if blockers:
        lines.append("### Blockers")
        lines.extend(f"- {text(item)}" for item in blockers)
        lines.append("")
    if warnings:
        lines.append("### Warnings")
        lines.extend(f"- {text(item)}" for item in warnings)
        lines.append("")
    if not blockers and not warnings:
        lines.extend(["- 数据集满足正式校准最低规则。", ""])
    plan = policy.get("collection_plan") if isinstance(policy.get("collection_plan"), list) else []
    if plan:
        lines.extend([
            "### Collection Plan",
            "",
            "| target | priority | action | missing | detail |",
            "|--------|----------|--------|---------|--------|",
        ])
        for item in plan:
            if not isinstance(item, dict):
                continue
            lines.append(
                f"| {text(item.get('target'))} | {text(item.get('priority'))} | {text(item.get('action'))} | {text(item.get('missing'))} | {text(item.get('detail'))} |"
            )
        lines.append("")
    lines.extend([
        "## Samples",
        "",
        "| # | score_profile | calibration | grade | confidence | total_score | backend_signature | path |",
        "|---|---------------|-------------|-------|------------|-------------|-------------------|------|",
    ])
    for idx, row in enumerate(result.get("samples", []), start=1):
        lines.append(
            f"| {idx} | {text(row.get('score_profile'))} | {text(row.get('calibration_version'))} | {text(row.get('grade'))} | {text(row.get('confidence_level'))} | {fmt(row.get('total_score'))} | {text(row.get('backend_signature'))} | {text(row.get('path'))} |"
        )
    lines.extend([
        "",
        "## Overall Metrics",
        "",
        "| metric | count | avg | p50 | p75 | p90 | min | max |",
        "|--------|-------|-----|-----|-----|-----|-----|-----|",
    ])
    for component, names in METRICS.items():
        lines.extend(metric_markdown_rows(result.get("overall", {}), component, names))
    lines.extend(["", "## Score Profiles", ""])
    for profile, summary in result.get("by_score_profile", {}).items():
        total = summary.get("metrics", {}).get("overall", {}).get("total_score", {})
        lines.append(f"- {profile}: count {summary.get('count', 0)}, total_score p50 {fmt(total.get('p50'))}, p75 {fmt(total.get('p75'))}, p90 {fmt(total.get('p90'))}")
    profile_policies = result.get("profile_policies") if isinstance(result.get("profile_policies"), dict) else {}
    if profile_policies:
        lines.extend([
            "",
            "## Score Profile Policies",
            "",
            "| score_profile | level | formal_ready | blockers | warnings | next_actions |",
            "|---------------|-------|--------------|----------|----------|--------------|",
        ])
        for profile, policy in profile_policies.items():
            if not isinstance(policy, dict):
                continue
            blockers = policy.get("blockers") if isinstance(policy.get("blockers"), list) else []
            warnings = policy.get("warnings") if isinstance(policy.get("warnings"), list) else []
            plan = policy.get("collection_plan") if isinstance(policy.get("collection_plan"), list) else []
            actions = []
            for item in plan:
                if isinstance(item, dict) and item.get("action"):
                    actions.append(text(item.get("action")))
            lines.append(
                f"| {text(profile)} | {text(policy.get('level'))} | {text(policy.get('formal_ready'))} | {len(blockers)} | {len(warnings)} | {', '.join(actions[:4]) if actions else '-'} |"
            )
    lines.extend(["", "## Calibration Recommendations", ""])
    action_label = {
        "keep": "保持",
        "raise": "上调",
        "lower": "下调",
        "collect_more": "继续收集",
    }
    for profile, recommendation in result.get("recommendations", {}).items():
        ready = "yes" if recommendation.get("ready") else "no"
        lines.extend([
            f"### {profile}",
            "",
            f"- sample_count: {recommendation.get('sample_count', 0)}",
            f"- ready: {ready}",
        ])
        notes = recommendation.get("notes") if isinstance(recommendation.get("notes"), list) else []
        for note in notes:
            lines.append(f"- note: {text(note)}")
        lines.extend([
            "",
            "| baseline | current | suggested | action | basis | samples | delta |",
            "|----------|---------|-----------|--------|-------|---------|-------|",
        ])
        suggestions = recommendation.get("baseline_suggestions") if isinstance(recommendation.get("baseline_suggestions"), dict) else {}
        for key, item in suggestions.items():
            lines.append(
                f"| {key} | {fmt(item.get('current'))} | {fmt(item.get('suggested'))} | {action_label.get(text(item.get('action')), text(item.get('action')))} | {text(item.get('basis'))} {text(item.get('component'))}.{text(item.get('metric'))} | {item.get('sample_count', 0)} | {fmt(item.get('delta_percent'))}% |"
            )
    return "\n".join(lines) + "\n"


def policy_level(result: dict[str, Any]) -> str:
    return text(data_get(result, "policy", "level"), "exploratory")


def policy_satisfies(result: dict[str, Any], required: str) -> bool:
    if not required:
        return True
    current = policy_level(result)
    required_rank = POLICY_RANK.get(required, 0)
    if POLICY_RANK.get(current, -1) < required_rank:
        return False
    profile_policies = result.get("profile_policies")
    if isinstance(profile_policies, dict):
        for policy in profile_policies.values():
            if not isinstance(policy, dict):
                return False
            level = text(policy.get("level"), "exploratory")
            if POLICY_RANK.get(level, -1) < required_rank:
                return False
    return True


def required_profiles_present(result: dict[str, Any], required_profiles: list[str]) -> tuple[bool, list[str]]:
    if not required_profiles:
        return True, []
    profile_policies = result.get("profile_policies")
    available = set(profile_policies.keys()) if isinstance(profile_policies, dict) else set()
    missing = [profile for profile in required_profiles if profile not in available]
    return not missing, missing


def policy_failure_message(result: dict[str, Any], required: str) -> str:
    current = policy_level(result)
    policy = result.get("policy") if isinstance(result.get("policy"), dict) else {}
    blockers = policy.get("blockers") if isinstance(policy.get("blockers"), list) else []
    warnings = policy.get("warnings") if isinstance(policy.get("warnings"), list) else []
    details = blockers if blockers else warnings
    profile_failures = []
    required_rank = POLICY_RANK.get(required, 0)
    profile_policies = result.get("profile_policies")
    if isinstance(profile_policies, dict):
        for profile, profile_policy in sorted(profile_policies.items()):
            if not isinstance(profile_policy, dict):
                profile_failures.append(f"{profile}: invalid policy")
                continue
            level = text(profile_policy.get("level"), "exploratory")
            if POLICY_RANK.get(level, -1) < required_rank:
                profile_blockers = profile_policy.get("blockers") if isinstance(profile_policy.get("blockers"), list) else []
                profile_warnings = profile_policy.get("warnings") if isinstance(profile_policy.get("warnings"), list) else []
                profile_details = profile_blockers if profile_blockers else profile_warnings
                detail = text(profile_details[0]) if profile_details else f"policy={level}"
                profile_failures.append(f"{profile}: {detail}")
    suffix = ""
    if details:
        suffix = "；" + "；".join(text(item) for item in details[:3])
    if profile_failures:
        suffix += "；profile policies: " + "；".join(profile_failures[:3])
    return f"dataset policy {current} does not satisfy required level {required}{suffix}"


def main() -> int:
    parser = argparse.ArgumentParser(description="Summarize Perfassess redacted calibration samples.")
    parser.add_argument("inputs", nargs="+", help="calibration_sample.json files or directories containing them.")
    parser.add_argument("--format", choices=["markdown", "json", "jsonl"], default="markdown", help="Output format.")
    parser.add_argument("--score-profile", choices=["vps", "server", "workstation"], default="", help="Only include samples with this score profile.")
    parser.add_argument("--min-confidence", choices=["low", "medium", "high"], default="", help="Only include samples at or above this confidence level.")
    parser.add_argument("--require-mainstream", action="store_true", help="Only include samples where all four core backends are mainstream.")
    parser.add_argument("--min-mainstream-count", type=int, default=None, help="Only include samples with at least this many mainstream core backends.")
    parser.add_argument("--require-policy", choices=["exploratory", "candidate", "formal"], default="", help="Exit non-zero unless the filtered dataset reaches this policy level.")
    parser.add_argument("--require-score-profiles", default="", help="Comma-separated score profiles that must be present, for example: vps,server,workstation.")
    parser.add_argument("--require-manifest", action="store_true", help="Require each sample directory to include manifest.json with a matching SHA256.")
    parser.add_argument("-o", "--output", default="", help="Output file path. Defaults to stdout.")
    args = parser.parse_args()
    required_score_profiles = [item.strip() for item in args.require_score_profiles.split(",") if item.strip()]
    invalid_profiles = [profile for profile in required_score_profiles if profile not in {"vps", "server", "workstation"}]
    if invalid_profiles:
        raise SystemExit(f"unsupported required score profiles: {', '.join(invalid_profiles)}")

    paths = find_sample_paths(args.inputs)
    if not paths:
        raise SystemExit("no calibration_sample.json files found")
    if args.require_manifest:
        for path in paths:
            validate_sample_manifest(path)
    samples = [load_sample(path) for path in paths]
    filtered_samples, filter_info = filter_samples(
        samples,
        score_profile=args.score_profile,
        min_confidence=args.min_confidence,
        require_mainstream=args.require_mainstream,
        min_mainstream_count=args.min_mainstream_count,
    )
    if not filtered_samples:
        raise SystemExit("no calibration samples matched filters")
    manifest_policy = {
        "required": bool(args.require_manifest),
        "checked_count": len(paths) if args.require_manifest else 0,
    }
    result = summarize(filtered_samples, filter_info, manifest_policy)

    if args.format == "json":
        output = json.dumps(result, ensure_ascii=False, indent=2) + "\n"
    elif args.format == "jsonl":
        output = "".join(json.dumps(row, ensure_ascii=False) + "\n" for row in result["samples"])
    else:
        output = render_markdown(result)

    if args.output:
        Path(args.output).write_text(output, encoding="utf-8")
    else:
        sys.stdout.write(output)

    profiles_ok, missing_profiles = required_profiles_present(result, required_score_profiles)
    if not profiles_ok:
        print(f"dataset missing required score profiles: {', '.join(missing_profiles)}", file=sys.stderr)
        return 2

    if args.require_policy and not policy_satisfies(result, args.require_policy):
        print(policy_failure_message(result, args.require_policy), file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
