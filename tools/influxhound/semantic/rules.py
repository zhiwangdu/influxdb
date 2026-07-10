from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Pattern


@dataclass(frozen=True)
class EventRule:
    event_type: str
    phase: str
    pattern: Pattern[str]


RULES = [
    EventRule("CONFIG_LOADED", "CONFIG", re.compile(r"(?i)config")),
    EventRule("META_OPEN_START", "META_OPEN", re.compile(r"(?i)(opening|open) meta")),
    EventRule("META_OPEN_COMPLETE", "META_OPEN", re.compile(r"(?i)opened meta")),
    EventRule("WAL_OPEN_START", "WAL_OPEN", re.compile(r"(?i)(opening|open) wal")),
    EventRule("WAL_OPEN_COMPLETE", "WAL_OPEN", re.compile(r"(?i)opened wal")),
    EventRule("WAL_REPLAY_START", "WAL_REPLAY", re.compile(r"(?i)(replaying|replay) wal")),
    EventRule("WAL_REPLAY_COMPLETE", "WAL_REPLAY", re.compile(r"(?i)(wal replay complete|replayed wal|replay took)")),
    EventRule("TSM_OPEN_START", "TSM_OPEN", re.compile(r"(?i)(opening|open) tsm")),
    EventRule("TSM_OPEN_COMPLETE", "TSM_OPEN", re.compile(r"(?i)opened tsm")),
    EventRule("INDEX_LOAD_START", "INDEX_LOAD", re.compile(r"(?i)(loading|load) index")),
    EventRule("INDEX_LOAD_COMPLETE", "INDEX_LOAD", re.compile(r"(?i)index (loaded|opened)")),
    EventRule("COMPACTION_INIT_START", "COMPACTION_INIT", re.compile(r"(?i)starting compaction")),
    EventRule("COMPACTION_INIT_COMPLETE", "COMPACTION_INIT", re.compile(r"(?i)compaction.+(ready|initialized)")),
    EventRule("SERVICE_START", "SERVICE_START", re.compile(r"(?i)starting .*service|starting http service")),
    EventRule("SERVICE_READY", "SERVICE_START", re.compile(r"(?i)listening on|ready for queries|server startup complete")),
    EventRule("PROCESS_PANIC", "SERVICE_START", re.compile(r"(?i)panic:")),
    EventRule("PROCESS_FATAL", "SERVICE_START", re.compile(r"(?i)fatal")),
]
