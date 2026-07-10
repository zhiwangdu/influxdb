from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from tools.influxhound.anchoring.log2event import anchor_events
from tools.influxhound.bundle import load_bundle
from tools.influxhound.skills.anomalies import detect_startup_anomalies
from tools.influxhound.skills.parse_logs import parse_influxd_log
from tools.influxhound.skills.timeline import build_startup_timeline


SAMPLE_LOG = """2025-01-01T00:00:00Z info loading config
2025-01-01T00:00:01Z info opening WAL
2025-01-01T00:00:02Z info replaying WAL
2025-01-01T00:00:40Z info WAL replay complete
2025-01-01T00:00:41Z info starting HTTP service
"""


class InfluxHoundPipelineTest(unittest.TestCase):
    def _create_bundle(self) -> Path:
        td = tempfile.TemporaryDirectory()
        self.addCleanup(td.cleanup)
        root = Path(td.name)
        (root / "logs").mkdir(parents=True, exist_ok=True)
        (root / "bug.yaml").write_text("title: startup slow\n", encoding="utf-8")
        (root / "arch.yaml").write_text("product: InfluxDB\n", encoding="utf-8")
        (root / "runtime.yaml").write_text("os: linux\n", encoding="utf-8")
        (root / "logs" / "influxd.log").write_text(SAMPLE_LOG, encoding="utf-8")
        return root

    def test_parse_anchor_timeline_anomalies(self) -> None:
        root = self._create_bundle()
        bundle = load_bundle(root)
        parsed = parse_influxd_log(bundle.influxd_log_path)
        self.assertEqual(len(parsed), 5)

        events = anchor_events(parsed)
        self.assertTrue(any(evt.event_type == "WAL_REPLAY_START" for evt in events))
        self.assertTrue(any(evt.event_type == "WAL_REPLAY_COMPLETE" for evt in events))

        timeline = build_startup_timeline(events)
        self.assertGreaterEqual(len(timeline["phases"]), 1)

        anomalies = detect_startup_anomalies(events)
        self.assertTrue(any(a["type"] == "WAL_REPLAY_SLOW" for a in anomalies))


if __name__ == "__main__":
    unittest.main()
