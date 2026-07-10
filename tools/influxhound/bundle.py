from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict


@dataclass(frozen=True)
class BugBundle:
    root: Path
    bug: Dict[str, Any]
    arch: Dict[str, Any]
    runtime: Dict[str, Any]
    semantic_event_catalog: Dict[str, Any]
    influxd_log_path: Path


class BundleLoadError(RuntimeError):
    pass


def _coerce(value: str) -> Any:
    value = value.strip()
    if value in {"", "null", "~"}:
        return ""
    lowered = value.lower()
    if lowered in {"true", "false"}:
        return lowered == "true"
    try:
        if "." in value:
            return float(value)
        return int(value)
    except ValueError:
        return value.strip('"\'')


def _simple_yaml_load(text: str) -> Dict[str, Any]:
    result: Dict[str, Any] = {}
    current_key: str | None = None
    for raw in text.splitlines():
        line = raw.rstrip()
        if not line or line.lstrip().startswith("#"):
            continue
        if line.startswith("  - ") and current_key:
            result.setdefault(current_key, []).append(_coerce(line[4:]))
            continue
        if ":" in line:
            key, value = line.split(":", 1)
            key = key.strip()
            value = value.strip()
            if value == "":
                result[key] = []
                current_key = key
            else:
                result[key] = _coerce(value)
                current_key = key
    return result


def _read_yaml(path: Path, required: bool = True) -> Dict[str, Any]:
    if not path.exists():
        if required:
            raise BundleLoadError(f"required file does not exist: {path}")
        return {}
    return _simple_yaml_load(path.read_text(encoding="utf-8"))


def load_bundle(bundle_path: str | Path) -> BugBundle:
    root = Path(bundle_path).expanduser().resolve()
    if not root.exists() or not root.is_dir():
        raise BundleLoadError(f"bundle path is invalid: {root}")

    bug = _read_yaml(root / "bug.yaml")
    arch = _read_yaml(root / "arch.yaml")
    runtime = _read_yaml(root / "runtime.yaml")
    catalog = _read_yaml(root / "semantic_event_catalog.yaml", required=False)

    influxd_log_path = root / "logs" / "influxd.log"
    if not influxd_log_path.exists():
        raise BundleLoadError(f"required log file does not exist: {influxd_log_path}")

    return BugBundle(
        root=root,
        bug=bug,
        arch=arch,
        runtime=runtime,
        semantic_event_catalog=catalog,
        influxd_log_path=influxd_log_path,
    )
