from __future__ import annotations


def _section(title: str) -> str:
    return f"\n## {title}\n"


def render_report(context: dict, timeline: dict, anomalies: list[dict], state: dict, events: list[dict]) -> str:
    lines = ["# InfluxHound Startup Analysis Report"]

    lines.append(_section("Bug Summary"))
    bug = context.get("bug", {})
    lines.append(f"- Title: {bug.get('title', '')}")
    lines.append(f"- Expected: {bug.get('expected', '')}")
    lines.append(f"- Actual: {bug.get('actual', '')}")
    lines.append(f"- First Seen Version: {bug.get('first_seen_version', '')}")

    lines.append(_section("Startup Timeline"))
    for phase in timeline.get("phases", []):
        lines.append(f"### {phase['phase']}")
        for event in phase["events"][:8]:
            lines.append(f"- [{event['event_type']}] {event['message']} ({event['evidence_ref']})")

    lines.append(_section("Anomaly Signals"))
    if not anomalies:
        lines.append("- None detected")
    else:
        for anomaly in anomalies:
            refs = ", ".join(anomaly["evidence_refs"])
            lines.append(f"- {anomaly['type']} ({anomaly['severity']}): {anomaly['summary']} | evidence: {refs}")

    lines.append(_section("Candidate Root Causes"))
    if not state.get("hypotheses"):
        lines.append("- No high-confidence hypothesis yet.")
    else:
        for idx, hyp in enumerate(state["hypotheses"], start=1):
            lines.append(f"{idx}. {hyp['root_cause']} (confidence={hyp['confidence']})")
            lines.append(f"   - Evidence: {', '.join(hyp['evidence_refs'])}")
            for check in hyp["next_checks"]:
                lines.append(f"   - Next check: {check}")

    lines.append(_section("Anchored Event Samples"))
    for event in events[:10]:
        lines.append(f"- {event['event_id']} {event['event_type']} {event['evidence_ref']} :: {event['message']}")

    return "\n".join(lines).strip() + "\n"
