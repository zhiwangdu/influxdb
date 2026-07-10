from __future__ import annotations

from datetime import datetime

from tools.influxhound.anchoring.log2event import AnchoredEvent


WAL_REPLAY_SLOW_SECONDS = 30.0


def _parse_ts(value: str | None) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value)
    except ValueError:
        return None


def detect_startup_anomalies(events: list[AnchoredEvent]) -> list[dict]:
    anomalies: list[dict] = []

    replay_start = next((e for e in events if e.event_type == "WAL_REPLAY_START"), None)
    replay_done = next((e for e in events if e.event_type == "WAL_REPLAY_COMPLETE"), None)

    for event in events:
        if event.event_type in {"PROCESS_PANIC", "PROCESS_FATAL"}:
            anomalies.append(
                {
                    "type": event.event_type,
                    "severity": "high",
                    "summary": event.message,
                    "evidence_refs": [event.evidence_ref],
                }
            )

    if replay_start and not replay_done:
        anomalies.append(
            {
                "type": "WAL_REPLAY_INCOMPLETE",
                "severity": "high",
                "summary": "WAL replay started but no completion event was found.",
                "evidence_refs": [replay_start.evidence_ref],
            }
        )

    if replay_start and replay_done:
        start_ts = _parse_ts(replay_start.ts)
        done_ts = _parse_ts(replay_done.ts)
        if start_ts and done_ts:
            elapsed = (done_ts - start_ts).total_seconds()
            if elapsed >= WAL_REPLAY_SLOW_SECONDS:
                anomalies.append(
                    {
                        "type": "WAL_REPLAY_SLOW",
                        "severity": "medium",
                        "summary": f"WAL replay took {elapsed:.1f}s which exceeds {WAL_REPLAY_SLOW_SECONDS:.1f}s.",
                        "evidence_refs": [replay_start.evidence_ref, replay_done.evidence_ref],
                    }
                )

    return anomalies
