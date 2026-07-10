from __future__ import annotations

DEFAULT_PHASES = [
    "CONFIG",
    "META_OPEN",
    "WAL_OPEN",
    "WAL_REPLAY",
    "TSM_OPEN",
    "INDEX_LOAD",
    "COMPACTION_INIT",
    "SERVICE_START",
]

DEFAULT_EVENT_TYPES = [
    "CONFIG_LOADED",
    "META_OPEN_START",
    "META_OPEN_COMPLETE",
    "WAL_OPEN_START",
    "WAL_OPEN_COMPLETE",
    "WAL_REPLAY_START",
    "WAL_REPLAY_COMPLETE",
    "TSM_OPEN_START",
    "TSM_OPEN_COMPLETE",
    "INDEX_LOAD_START",
    "INDEX_LOAD_COMPLETE",
    "COMPACTION_INIT_START",
    "COMPACTION_INIT_COMPLETE",
    "SERVICE_START",
    "SERVICE_READY",
    "PROCESS_PANIC",
    "PROCESS_FATAL",
    "ERROR_LOG",
    "WARN_LOG",
    "UNKNOWN",
]


def merge_catalog(user_catalog: dict | None) -> dict:
    user_catalog = user_catalog or {}
    return {
        "phases": user_catalog.get("phases", DEFAULT_PHASES),
        "event_types": user_catalog.get("event_types", DEFAULT_EVENT_TYPES),
    }
