from __future__ import annotations

import re
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Iterable, Optional


LOG_LINE_PATTERN = re.compile(
    r"^(?P<ts>\d{4}-\d{2}-\d{2}T[^\s]+)\s+(?P<level>[a-zA-Z]+)\s+(?P<msg>.*)$"
)


@dataclass(frozen=True)
class ParsedLogLine:
    line_no: int
    ts: Optional[datetime]
    level: str
    message: str
    raw: str


def _parse_ts(ts_raw: str) -> Optional[datetime]:
    if not ts_raw:
        return None
    try:
        return datetime.fromisoformat(ts_raw.replace("Z", "+00:00"))
    except ValueError:
        return None


def parse_influxd_log(path: str | Path) -> list[ParsedLogLine]:
    path = Path(path)
    lines: list[ParsedLogLine] = []
    for idx, raw_line in enumerate(path.read_text(encoding="utf-8", errors="replace").splitlines(), start=1):
        match = LOG_LINE_PATTERN.match(raw_line)
        if match:
            ts = _parse_ts(match.group("ts"))
            level = match.group("level").lower()
            msg = match.group("msg")
        else:
            ts = None
            level = "unknown"
            msg = raw_line
        lines.append(ParsedLogLine(line_no=idx, ts=ts, level=level, message=msg, raw=raw_line))
    return lines
