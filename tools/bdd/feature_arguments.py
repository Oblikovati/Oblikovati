# SPDX-License-Identifier: GPL-2.0-only
"""Compare parsed scenario arguments with the native runner's capture."""

import hashlib
import json
from pathlib import Path


def load_capture(root: Path, record: dict[str, object]) -> list[dict[str, object]] | None:
    """Load a hashed runner capture; for example, load_capture(root, record)."""
    name = record.get("scenario_arguments")
    if not isinstance(name, str):
        return None
    path = (root / name).resolve()
    if not path.is_relative_to(root.resolve()) or not path.is_file():
        return None
    content = path.read_bytes()
    if hashlib.sha256(content).hexdigest() != record.get("scenario_arguments_sha256"):
        return None
    try:
        capture = json.loads(content)
    except (ValueError, UnicodeDecodeError):
        return None
    if not isinstance(capture, dict) or capture.get("cucumber_sha256") != record.get("cucumber_sha256"):
        return None
    entries = capture.get("scenarios")
    if not isinstance(entries, list) or not all(isinstance(item, dict) for item in entries):
        return None
    return entries


def arguments_match(row: dict[str, object], entries: list[dict[str, object]]) -> bool:
    """Match one scenario's parsed arguments; for example, arguments_match(row, entries)."""
    matches = [item for item in entries if item.get("path") == row["path"]
               and item.get("line") == row["line"] and item.get("name") == row["scenario"]]
    if len(matches) != 1:
        return False
    captured = matches[0]
    return all(captured.get(field) == row.get(field) for field in ("background_steps", "scenario_steps"))


def capture_scenarios_match(
    path: str, rows: list[dict[str, object]], entries: list[dict[str, object]]
) -> bool:
    """Require exactly the captured source cases; for example, capture_scenarios_match(path, rows, entries)."""
    actual = [(item.get("line"), item.get("name")) for item in entries if item.get("path") == path]
    expected = [(row.get("line"), row.get("scenario")) for row in rows]
    if not all(type(line) is int and isinstance(name, str) for line, name in actual + expected):
        return False
    return len(actual) == len(expected) and set(actual) == set(expected)


def plain_steps(row: dict[str, object]) -> dict[str, object]:
    """Remove argument fields for Cucumber result checks; for example, plain_steps(row)."""
    result = row.copy()
    for field in ("background_steps", "scenario_steps"):
        result[field] = [{"line": step["line"], "name": step["name"]} for step in row[field]]
    return result
