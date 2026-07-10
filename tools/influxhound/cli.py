from __future__ import annotations

import argparse
import json
from dataclasses import asdict
from pathlib import Path

from tools.influxhound.anchoring.log2event import anchor_events
from tools.influxhound.bundle import load_bundle
from tools.influxhound.context import compile_context
from tools.influxhound.report import render_report
from tools.influxhound.skills.anomalies import detect_startup_anomalies
from tools.influxhound.skills.parse_logs import parse_influxd_log
from tools.influxhound.skills.timeline import build_startup_timeline
from tools.influxhound.state import build_state


def run(bundle_path: str) -> tuple[str, Path]:
    bundle = load_bundle(bundle_path)
    parsed_lines = parse_influxd_log(bundle.influxd_log_path)
    events = anchor_events(parsed_lines)
    timeline = build_startup_timeline(events)
    anomalies = detect_startup_anomalies(events)
    context = compile_context(bundle, timeline, anomalies)

    known_facts = {
        "event_count": len(events),
        "anomaly_count": len(anomalies),
        "phases_detected": [p["phase"] for p in timeline.get("phases", [])],
    }
    state = build_state(known_facts=known_facts, anomalies=anomalies)

    output_dir = Path("analysis_output")
    output_dir.mkdir(parents=True, exist_ok=True)
    (output_dir / "events.json").write_text(
        json.dumps([asdict(e) for e in events], indent=2, ensure_ascii=False), encoding="utf-8"
    )
    (output_dir / "timeline.json").write_text(json.dumps(timeline, indent=2, ensure_ascii=False), encoding="utf-8")
    (output_dir / "diagnosis.json").write_text(
        json.dumps({"anomalies": anomalies, "state": state, "context": context}, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )

    report = render_report(
        context=context,
        timeline=timeline,
        anomalies=anomalies,
        state=state,
        events=[asdict(e) for e in events],
    )
    return report, output_dir


def main() -> None:
    parser = argparse.ArgumentParser(description="InfluxHound: InfluxDB startup bug analyzer")
    parser.add_argument("bundle_path", help="Path to bug bundle")
    args = parser.parse_args()

    report, _ = run(args.bundle_path)
    print(report)


if __name__ == "__main__":
    main()
