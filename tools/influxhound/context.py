from __future__ import annotations

from tools.influxhound.bundle import BugBundle


def compile_context(bundle: BugBundle, timeline: dict, anomalies: list[dict]) -> dict:
    return {
        "bug": {
            "title": bundle.bug.get("title", ""),
            "expected": bundle.bug.get("expected", ""),
            "actual": bundle.bug.get("actual", ""),
            "repro_steps": bundle.bug.get("repro_steps", []),
            "first_seen_version": bundle.bug.get("first_seen_version", ""),
        },
        "architecture": {
            "product": bundle.arch.get("product", "InfluxDB"),
            "version": bundle.arch.get("version", ""),
            "mode": bundle.arch.get("mode", "single"),
            "startup_phases": bundle.arch.get("startup_phases", []),
        },
        "runtime": bundle.runtime,
        "timeline_phase_count": len(timeline.get("phases", [])),
        "anomaly_count": len(anomalies),
    }
