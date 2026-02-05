from __future__ import annotations


def build_state(known_facts: dict, anomalies: list[dict]) -> dict:
    hypotheses = []
    rejected = []

    for anomaly in anomalies:
        if anomaly["type"] == "PROCESS_PANIC":
            hypotheses.append(
                {
                    "root_cause": "Startup panic likely caused by invalid configuration or corrupted local metadata.",
                    "confidence": 0.8,
                    "evidence_refs": anomaly["evidence_refs"],
                    "next_checks": [
                        "Inspect panic stack trace in influxd.log around evidence line.",
                        "Verify config file compatibility with target InfluxDB version.",
                    ],
                }
            )
        elif anomaly["type"] == "PROCESS_FATAL":
            hypotheses.append(
                {
                    "root_cause": "Fatal startup error indicates dependency/resource initialization failure.",
                    "confidence": 0.75,
                    "evidence_refs": anomaly["evidence_refs"],
                    "next_checks": [
                        "Check file permissions and storage paths configured for meta/WAL/TSM.",
                        "Correlate with system logs for disk or IO errors.",
                    ],
                }
            )
        elif anomaly["type"] == "WAL_REPLAY_INCOMPLETE":
            hypotheses.append(
                {
                    "root_cause": "WAL replay did not complete; potential WAL corruption or restart interruption.",
                    "confidence": 0.7,
                    "evidence_refs": anomaly["evidence_refs"],
                    "next_checks": [
                        "Inspect WAL segment integrity and disk health.",
                        "Check for abrupt process termination in system logs.",
                    ],
                }
            )
        elif anomaly["type"] == "WAL_REPLAY_SLOW":
            hypotheses.append(
                {
                    "root_cause": "WAL replay is slow, likely due to large WAL backlog or IO pressure.",
                    "confidence": 0.65,
                    "evidence_refs": anomaly["evidence_refs"],
                    "next_checks": [
                        "Measure disk throughput and latency during startup.",
                        "Estimate WAL size and number of segments to replay.",
                    ],
                }
            )

    if not hypotheses:
        rejected.append(
            {
                "reason": "No strong startup anomaly signature was detected in current catalog/rules.",
                "next_action": "Expand semantic_event_catalog and matching rules for this log pattern.",
            }
        )

    return {
        "known_facts": known_facts,
        "hypotheses": hypotheses,
        "rejected": rejected,
    }
