from __future__ import annotations

from dataclasses import asdict, dataclass

from tools.influxhound.semantic.rules import RULES
from tools.influxhound.skills.parse_logs import ParsedLogLine


@dataclass(frozen=True)
class AnchoredEvent:
    event_id: str
    event_type: str
    phase: str
    ts: str | None
    level: str
    message: str
    raw: str
    evidence_ref: str



def anchor_events(parsed_lines: list[ParsedLogLine], log_rel_path: str = "logs/influxd.log") -> list[AnchoredEvent]:
    events: list[AnchoredEvent] = []
    for parsed in parsed_lines:
        selected = None
        for rule in RULES:
            if rule.pattern.search(parsed.message):
                selected = rule
                break

        if selected is None:
            if parsed.level in {"error", "warn", "warning"}:
                event_type = "ERROR_LOG" if parsed.level == "error" else "WARN_LOG"
                phase = "SERVICE_START"
            else:
                event_type = "UNKNOWN"
                phase = "SERVICE_START"
        else:
            event_type = selected.event_type
            phase = selected.phase

        events.append(
            AnchoredEvent(
                event_id=f"evt-{parsed.line_no}",
                event_type=event_type,
                phase=phase,
                ts=parsed.ts.isoformat() if parsed.ts else None,
                level=parsed.level,
                message=parsed.message,
                raw=parsed.raw,
                evidence_ref=f"{log_rel_path}:{parsed.line_no}",
            )
        )
    return events
