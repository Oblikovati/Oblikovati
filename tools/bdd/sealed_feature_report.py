# SPDX-License-Identifier: GPL-2.0-only
"""Check one complete application's feature report without authorizing reuse."""

import hashlib
import json
from pathlib import Path
from typing import Mapping

from behavior_capture import artifact
from behavior_evidence import check_capture, check_scenario, check_source_revision, example_capture_name
from sealed_execution import SealedRequest
from sealed_report_check import regular_artifact, require_junit


def selected_feature_rows(inventory: Mapping[str, object], request: SealedRequest, revision: str) -> list[dict[str, object]]:
    """Select all scenarios of one feature; example: selected_feature_rows(inventory, request, revision)."""
    if inventory.get("source_revision") != revision or not isinstance(inventory.get("scenarios"), list):
        raise ValueError(f"inventory revision {inventory.get('source_revision')!r}: expected {revision!r}")
    rows = [row for row in inventory["scenarios"] if isinstance(row, dict)
            and (row.get("area"), row.get("feature")) == (request.area, request.feature)]
    identifiers = [row.get("id") for row in rows]
    if not rows or any(not isinstance(value, str) for value in identifiers) or len(set(identifiers)) != len(rows):
        raise ValueError(f"feature {request.feature!r}: expected nonempty, uniquely identified scenario rows")
    identities = [(row.get("path"), row.get("line"), row.get("scenario")) for row in rows]
    if len(set(identities)) != len(rows):
        raise ValueError(f"feature {request.feature!r}: expected unique scenario source identities")
    if len({row.get("path") for row in rows}) != 1:
        raise ValueError(f"feature {request.feature!r}: expected scenarios from exactly one feature file")
    for row in rows:
        capture_namespace(row)
    case_count(rows)
    return rows


def case_count(rows: list[dict[str, object]]) -> int:
    """Count scenario cases including outline examples; example: case_count(rows)."""
    total = 0
    for row in rows:
        if row.get("outline"):
            examples = row.get("examples")
            if not isinstance(examples, list) or not examples:
                raise ValueError(f"outline {row.get('id')!r}: expected examples")
            total += len(examples)
        else:
            total += 1
    return total


def require_feature_cases(report: object, count: int) -> None:
    """Reject missing or extra report cases; example: require_feature_cases(report, 10)."""
    if not isinstance(report, list):
        raise ValueError(f"Cucumber report {report!r}: expected a feature list")
    elements = []
    for feature in report:
        if not isinstance(feature, dict) or not isinstance(feature.get("elements"), list):
            raise ValueError(f"Cucumber feature {feature!r}: expected elements")
        for item in feature["elements"]:
            if not isinstance(item, dict) or item.get("type") not in ("scenario", "background"):
                raise ValueError(f"Cucumber element {item!r}: expected a scenario or background")
            if item["type"] == "scenario":
                elements.append(item)
    if len(elements) != count:
        raise ValueError(f"Cucumber scenario count {len(elements)}: expected {count}")


def capture_namespace(row: Mapping[str, object]) -> str:
    """Derive the native capture namespace; example: capture_namespace(row)."""
    path, line, name = row.get("path"), row.get("line"), row.get("scenario")
    if not isinstance(path, str) or type(line) is not int or line <= 0 or not isinstance(name, str):
        raise ValueError(f"scenario identity {(path, line, name)!r}: expected a path, positive line and name")
    return hashlib.sha256(f"{path}:{line}:{name}".encode()).hexdigest()


def require_feature_captures(root: Path, row: Mapping[str, object]) -> None:
    """Check screenshots for one scenario namespace; example: require_feature_captures(root, row)."""
    scenarios = root / "scenarios"
    namespace = scenarios / capture_namespace(row)
    folder = namespace / "final"
    if any(path.is_symlink() for path in (scenarios, namespace, folder)) or not folder.is_dir() or not folder.resolve().is_relative_to(root.resolve()):
        raise ValueError(f"capture directory {folder}: expected the selected scenario's final directory")
    images = sorted(path for path in folder.glob("*.png") if path.is_file() and not path.is_symlink())
    expected = [example_capture_name(row, example) for example in row["examples"]] if row.get("outline") else None
    if expected is None and len(images) != 1:
        raise ValueError(f"scenario captures {images!r}: expected one final PNG")
    if expected is not None and sorted(path.name for path in images) != sorted(expected):
        raise ValueError(f"outline captures {images!r}: expected {expected!r}")
    for path in images:
        check_capture(root, artifact(root, path))


def check_feature_report(request: SealedRequest, rows: list[dict[str, object]], revision: str) -> dict[str, object]:
    """Validate every selected scenario and capture; example: check_feature_report(request, rows, revision)."""
    root = request.reports
    report = json.loads((root / regular_artifact(root, "cucumber.json")["path"]).read_text())
    count = case_count(rows)
    require_feature_cases(report, count)
    for row in rows:
        check_scenario(report, row)
        require_feature_captures(root, row)
    source = regular_artifact(root, "source-revision.json")
    check_source_revision(root, source, revision)
    state = json.loads((root / source["path"]).read_text())
    if state["before"].get("root") != "/scenario":
        raise ValueError(f"source report root {state['before'].get('root')!r}: expected /scenario")
    require_junit(root, count)
    return {"scenario_ids": [row["id"] for row in rows], "content_checked": True, "reusable": False}
