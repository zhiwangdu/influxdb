#!/usr/bin/env python3
"""Lightweight validator for SDD spec YAML files.

This validator avoids external dependencies and performs structure checks
using indentation-aware key scanning.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

REQUIRED_TOP_KEYS = {
    "id",
    "title",
    "status",
    "context",
    "scope",
    "contracts",
    "implementation_plan",
    "validation",
    "release_and_rollback",
}

ID_PATTERN = re.compile(r"^SPEC-\d{4}-\d{3}$")


def parse_top_keys(text: str) -> dict[str, str]:
    out: dict[str, str] = {}
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if line.startswith(" ") or line.startswith("\t"):
            continue
        if ":" not in line:
            continue
        k, v = line.split(":", 1)
        out[k.strip()] = v.strip().strip('"')
    return out


def ensure_contains(text: str, fragment: str) -> bool:
    return fragment in text


def validate(path: Path) -> list[str]:
    errors: list[str] = []
    text = path.read_text(encoding="utf-8")
    top = parse_top_keys(text)

    missing = sorted(REQUIRED_TOP_KEYS - set(top.keys()))
    if missing:
        errors.append(f"missing top-level keys: {', '.join(missing)}")

    sid = top.get("id", "")
    if sid and not ID_PATTERN.match(sid):
        errors.append(f"invalid id format: {sid}")

    for key_path in [
        "goals:",
        "in_scope:",
        "out_of_scope:",
        "milestones:",
        "unit_tests:",
        "integration_tests:",
        "rollout:",
        "rollback:",
    ]:
        if not ensure_contains(text, key_path):
            errors.append(f"missing required section marker: {key_path}")

    return errors


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print("Usage: python tools/sdd/validate_spec.py <spec.yaml> [more specs]")
        return 2

    code = 0
    for name in argv[1:]:
        path = Path(name)
        if not path.exists():
            print(f"[FAIL] {name}: file not found")
            code = 1
            continue

        errs = validate(path)
        if errs:
            print(f"[FAIL] {name}")
            for e in errs:
                print(f"  - {e}")
            code = 1
        else:
            print(f"[PASS] {name}")

    return code


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
