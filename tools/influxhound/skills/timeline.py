from __future__ import annotations

from collections import defaultdict

from tools.influxhound.anchoring.log2event import AnchoredEvent


def build_startup_timeline(events: list[AnchoredEvent]) -> dict:
    phase_events = defaultdict(list)
    for event in events:
        phase_events[event.phase].append(
            {
                "event_id": event.event_id,
                "event_type": event.event_type,
                "ts": event.ts,
                "level": event.level,
                "message": event.message,
                "evidence_ref": event.evidence_ref,
            }
        )

    ordered = []
    for phase, items in phase_events.items():
        ordered.append({"phase": phase, "events": items})
    return {"phases": ordered}
