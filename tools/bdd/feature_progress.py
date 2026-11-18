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
    if isinstance(record.get("blocker"), str) and record["blocker"].strip():
        return "blocked", 0
    report = ROOT / str(record.get("cucumber", ""))
    if not report.is_file() or digest(report) != record.get("cucumber_sha256"):
        return "invalid_result", 0
    if info["outlines"]:
        result = compare_inventoried_report(path, info, report, record)
    else:
        result = compare_report(source, path, report)
    return result if digest(source / path) == info["sha256"] else ("source_changed", 0)


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
    if len(rows) != info["scenario_declarations"] or any(
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
