#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Report full-feature test runs against the current scenario source hashes."""

import argparse
import hashlib
import json
import re
from pathlib import Path
from functools import lru_cache

from behavior_evidence import check_scenario
from feature_arguments import arguments_match, capture_scenarios_match, load_capture, plain_steps
from inventory import collect_features

ROOT = Path(__file__).resolve().parents[2]
RESULTS = ROOT / "architecture/rust-port/feature-run-results.json"
SCENARIO = re.compile(r"^\s*Scenario:\s*(.+?)\s*$")
STEP = re.compile(r"^\s*(?:Given|When|Then|And|But|\*)\s+(.+?)\s*$")


def digest(path: Path) -> str:
    """Return a file's SHA-256; for example, digest(Path('run.json'))."""
    return hashlib.sha256(path.read_bytes()).hexdigest()


def passed(element: dict[str, object]) -> bool:
    """Check every executed step and cleanup hook; for example, passed(scenario)."""
    steps = element.get("steps")
    cleanup = element.get("after")
    if (
        not isinstance(steps, list)
        or not steps
        or not isinstance(cleanup, list)
        or not cleanup
    ):
        return False
    hooks = element.get("before", [])
    return all(
        item.get("result", {}).get("status") == "passed"
        for item in steps + hooks + cleanup
    )


def result_status(
    source: Path, path: str, info: dict[str, object], record: object
) -> tuple[str, int]:
    """Compare one complete result with its source; for example, result_status(root, path, info, row)."""
    if not isinstance(record, dict):
        return "not_run", 0
    if record.get("feature_sha256") != info["sha256"]:
        return "source_changed", 0
    if not supporting_files_match(source, record.get("supporting_files", {})):
        return "support_changed", 0
    blocked = isinstance(record.get("blocker"), str) and bool(record["blocker"].strip())
    report = ROOT / str(record.get("cucumber", ""))
    if not report.is_file() or digest(report) != record.get("cucumber_sha256"):
        return ("blocked" if blocked else "invalid_result"), 0
    if info["outlines"]:
        result = compare_inventoried_report(path, info, report, record)
    else:
        result = compare_report(source, path, report)
        if result[0] == "needs_source_review":
            result = compare_argument_report(path, info, report, record)
    if digest(source / path) != info["sha256"]:
        return "source_changed", 0
    if not supporting_files_match(source, record.get("supporting_files", {})):
        return "support_changed", 0
    if blocked and result[0] in ("green", "incomplete"):
        return "blocked", result[1]
    return result


def supporting_files_match(source: Path, values: object) -> bool:
    """Check recorded feature notes; for example, supporting_files_match(root, hashes)."""
    if not isinstance(values, dict):
        return False
    for name, expected in values.items():
        path = (source / name).resolve()
        if not path.is_relative_to(source.resolve()) or not path.is_file():
            return False
        if digest(path) != expected:
            return False
    return True


@lru_cache(maxsize=1)
def source_inventory() -> list[dict[str, object]]:
    """Read the parsed scenario snapshot; for example, source_inventory() for an outline."""
    path = ROOT / "architecture/rust-port/behavior-source-inventory.json"
    return json.loads(path.read_text(encoding="utf-8"))["scenarios"]


def compare_inventoried_report(
    path: str, info: dict[str, object], report: Path, record: dict[str, object]
) -> tuple[str, int]:
    """Check every parsed declaration and example; for example, compare_inventoried_report(path, info, report, record)."""
    inventory = recorded_inventory(record)
    if inventory is None:
        return "invalid_result", 0
    rows = [row for row in inventory if row["path"] == path]
    if not rows or len(rows) != info["scenario_declarations"] or any(
        row["source_sha256"] != info["sha256"] for row in rows
    ):
        return "needs_outline_review", 0
    results = json.loads(report.read_text(encoding="utf-8"))
    return count_checked_scenarios(results, rows)


def recorded_inventory(record: dict[str, object]) -> list[dict[str, object]] | None:
    """Load a hashed parsed inventory, or the standard snapshot; for example, recorded_inventory(record)."""
    name = record.get("scenario_inventory")
    if name is None:
        return source_inventory()
    path = (ROOT / str(name)).resolve()
    if not path.is_relative_to(ROOT.resolve()) or not path.is_file():
        return None
    if digest(path) != record.get("scenario_inventory_sha256"):
        return None
    return json.loads(path.read_text(encoding="utf-8"))["scenarios"]


def count_checked_scenarios(
    report: object, rows: list[dict[str, object]]
) -> tuple[str, int]:
    """Count declarations with all examples passing; for example, count_checked_scenarios(report, rows)."""
    count = 0
    for row in rows:
        try:
            check_scenario(report, row)
        except ValueError:
            continue
        count += 1
    return ("green" if count == len(rows) else "incomplete", count)


def compare_argument_report(
    path: str, info: dict[str, object], report: Path, record: dict[str, object]
) -> tuple[str, int]:
    """Verify parsed step arguments; for example, compare_argument_report(path, info, report, record)."""
    rows = argument_inventory(path, info, record)
    entries = load_capture(ROOT, record)
    if rows is None or entries is None:
        return "needs_source_review", 0
    if not capture_scenarios_match(path, rows, entries):
        return "incomplete", 0
    try:
        results = json.loads(report.read_text(encoding="utf-8"))
    except (ValueError, UnicodeDecodeError):
        return "invalid_result", 0
    if not report_scenarios_match(results, path, rows):
        return "incomplete", 0
    count = count_argument_scenarios(results, rows, entries)
    return ("green" if count == len(rows) else "incomplete", count)


def argument_inventory(
    path: str, info: dict[str, object], record: dict[str, object]
) -> list[dict[str, object]] | None:
    """Validate the parsed inventory; for example, argument_inventory(path, info, record)."""
    rows = recorded_inventory(record)
    if rows is None or not all(isinstance(row, dict) for row in rows):
        return None
    rows = [row for row in rows if row.get("path") == path]
    if not rows or len(rows) != info["scenario_declarations"] or any(
        row.get("source_sha256") != info["sha256"]
        or row.get("arguments_recorded") is not True
        or row.get("source_error") is not None
        or type(row.get("line")) is not int
        or not isinstance(row.get("scenario"), str)
        or not isinstance(row.get("background_steps"), list)
        or not isinstance(row.get("scenario_steps"), list)
        for row in rows
    ):
        return None
    identities = {(row["line"], row["scenario"]) for row in rows}
    if len(identities) != len(rows):
        return None
    return rows


def count_argument_scenarios(
    results: object, rows: list[dict[str, object]], entries: list[dict[str, object]]
) -> int:
    """Count scenarios with matching arguments and results; for example, count_argument_scenarios(report, rows, entries)."""
    count = 0
    for row in rows:
        if not arguments_match(row, entries):
            continue
        try:
            check_scenario(results, plain_steps(row))
        except (ValueError, KeyError, TypeError):
            continue
        count += 1
    return count


def report_scenarios_match(report: object, path: str, rows: list[dict[str, object]]) -> bool:
    """Require exactly the declared cases; for example, report_scenarios_match(report, path, rows)."""
    if not isinstance(report, list):
        return False
    features = [item for item in report if isinstance(item, dict)
                and str(item.get("uri", "")).replace("\\", "/").endswith(path)]
    if len(features) != 1 or not isinstance(features[0].get("elements"), list):
        return False
    actual = [(item.get("line"), item.get("name")) for item in features[0]["elements"]
              if isinstance(item, dict) and item.get("type") == "scenario"]
    expected = [(row["line"], f"{row['rule']} {row['scenario']}" if row.get("rule")
                 else row["scenario"]) for row in rows]
    if not all(type(line) is int and isinstance(name, str) for line, name in actual + expected):
        return False
    return len(actual) == len(expected) and set(actual) == set(expected)


def ordinary_scenarios(feature: Path) -> dict[str, dict[str, object]] | None:
    """Read ordinary scenarios and their steps; for example, ordinary_scenarios(Path('a.feature'))."""
    scenarios: dict[str, dict[str, object]] = {}
    current: dict[str, object] | None = None
    for line_number, line in enumerate(
        feature.read_text(encoding="utf-8-sig").splitlines(), 1
    ):
        stripped = line.strip()
        if stripped.startswith(("Background:", "Rule:", "|", '"""', "```")):
            return None
        match = SCENARIO.match(line)
        if match:
            name = match.group(1)
            if name in scenarios:
                return None
            current = {"line": line_number, "steps": []}
            scenarios[name] = current
            continue
        step = STEP.match(line)
        if step and current is not None:
            current["steps"].append({"line": line_number, "name": step.group(1)})
    return scenarios


def compare_report(source: Path, path: str, report: Path) -> tuple[str, int]:
    """Check source steps and outcomes; for example, compare_report(root, path, report)."""
    scenarios = ordinary_scenarios(source / path)
    if not scenarios or any(not item["steps"] for item in scenarios.values()):
        return "needs_source_review", 0
    features = json.loads(report.read_text(encoding="utf-8"))
    matches = [item for item in features if str(item.get("uri", "")).endswith(path)]
    if len(matches) != 1:
        return "invalid_result", 0
    elements = [
        item
        for item in matches[0].get("elements", [])
        if item.get("type") == "scenario"
    ]
    actual = [item.get("name") for item in elements]
    if sorted(actual) != sorted(scenarios):
        return "incomplete", 0
    count = sum(
        passed(item) and matches_source(item, scenarios[item["name"]])
        for item in elements
    )
    return ("green" if count == len(scenarios) else "incomplete", count)


def matches_source(element: dict[str, object], scenario: dict[str, object]) -> bool:
    """Check that the report contains the source steps; for example, matches_source(result, source)."""
    actual = [
        {"line": item.get("line"), "name": item.get("name")}
        for item in element["steps"]
    ]
    return element.get("line") == scenario["line"] and actual == scenario["steps"]


def report_progress(source: Path, records: dict[str, object]) -> dict[str, object]:
    """Summarize source-matched runs; for example, report_progress(Path('/scenarios'), {})."""
    features = collect_features(source)
    rows = {}
    for path, info in features.items():
        status, count = result_status(source, path, info, records.get(path))
        record = records.get(path)
        commit = record.get("application_commit") if isinstance(record, dict) else None
        rows[path] = {
            **info,
            "status": status,
            "passed_scenarios": count,
            "tested_application_commit": commit,
            "blocker": record.get("blocker") if isinstance(record, dict) else None,
        }
    return {
        "features": rows,
        "removed_features": sorted(set(records) - set(features)),
        "total_features": len(rows),
        "green_features": sum(row["status"] == "green" for row in rows.values()),
        "blocked_features": sum(row["status"] == "blocked" for row in rows.values()),
        "scenario_declarations": sum(
            row["scenario_declarations"] for row in rows.values()
        ),
        "passed_scenarios": sum(row["passed_scenarios"] for row in rows.values()),
    }


def main() -> None:
    """Report the live source; for example, `feature_progress.py /path/to/MCAD-Scenarios`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    args = parser.parse_args()
    records = json.loads(RESULTS.read_text(encoding="utf-8"))
    print(json.dumps(report_progress(args.source, records), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
