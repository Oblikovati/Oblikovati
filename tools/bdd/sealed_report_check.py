# SPDX-License-Identifier: GPL-2.0-only
"""Check a sealed diagnostic report against the selected committed inventory."""

import hashlib
import json
import subprocess
import xml.etree.ElementTree as ET
from pathlib import Path, PurePosixPath
from typing import Mapping

from behavior_capture import artifact, selected_capture
from behavior_evidence import check_capture, check_scenario, check_source_revision, example_capture_name
from git_environment import git_environment
from sealed_execution import SealedRequest

INVENTORY_PATH = "architecture/rust-port/behavior-source-inventory.json"


def committed_inventory(repository: Path, revision: str) -> dict[str, object]:
    """Read the inventory from the tested commit; example: committed_inventory(repo, sha)."""
    raw = subprocess.check_output(["git", "-C", str(repository), "show", f"{revision}:{INVENTORY_PATH}"], env=git_environment())
    value = json.loads(raw)
    if not isinstance(value, dict):
        raise ValueError(f"committed inventory {value!r}: expected an object")
    return value


def selected_row(inventory: Mapping[str, object], request: SealedRequest, revision: str) -> dict[str, object]:
    """Select exactly one source scenario; example: selected_row(inventory, request, sha)."""
    if inventory.get("source_revision") != revision or not isinstance(inventory.get("scenarios"), list):
        raise ValueError(f"inventory revision {inventory.get('source_revision')!r}: expected {revision!r}")
    rows = [row for row in inventory["scenarios"] if isinstance(row, dict) and (row.get("area"), row.get("feature"), row.get("scenario")) == (request.area, request.feature, request.scenario)]
    if len(rows) != 1:
        raise ValueError(f"scenario selector {request.scenario!r}: expected one exact committed inventory row, got {len(rows)}")
    return rows[0]


def require_scenario_source(checkout: Path, row: Mapping[str, object]) -> None:
    """Match the selected source file to the committed inventory; example: require_scenario_source(checkout, row)."""
    name, expected = row.get("path"), row.get("source_sha256")
    if not isinstance(name, str) or not isinstance(expected, str):
        raise ValueError(f"scenario path {name!r}: expected a path and source hash")
    relative = PurePosixPath(name)
    if relative.is_absolute() or any(part in ("", ".", "..") for part in relative.parts) or not name.startswith("features/"):
        raise ValueError(f"scenario path {name!r}: expected a relative feature path")
    path = checkout.joinpath(*relative.parts)
    if path.is_symlink() or not path.resolve().is_relative_to(checkout.resolve()) or hashlib.sha256(path.read_bytes()).hexdigest() != expected:
        raise ValueError(f"scenario source {name!r}: expected the committed inventory hash")


def regular_artifact(root: Path, name: str) -> dict[str, str]:
    """Describe an existing non-symlink output; example: regular_artifact(reports, 'junit.xml')."""
    path = root / name
    if path.is_symlink() or not path.is_file():
        raise ValueError(f"report artifact {path}: expected a regular file")
    return artifact(root, path)


def require_single_case(report: object, row: Mapping[str, object]) -> None:
    """Reject extra scenario executions; example: require_single_case(report, row)."""
    if not isinstance(report, list):
        raise ValueError(f"Cucumber report {report!r}: expected a feature list")
    elements: list[object] = []
    for feature in report:
        if not isinstance(feature, dict) or not isinstance(feature.get("elements"), list):
            raise ValueError(f"Cucumber feature {feature!r}: expected an object with elements")
        for item in feature["elements"]:
            if not isinstance(item, dict) or item.get("type") not in ("scenario", "background"):
                raise ValueError(f"Cucumber element {item!r}: expected a scenario or background")
            if item["type"] == "scenario":
                elements.append(item)
    expected = len(row["examples"]) if row.get("outline") and isinstance(row.get("examples"), list) else 1
    if len(elements) != expected:
        raise ValueError(f"Cucumber scenario count {len(elements)}: expected {expected}")


def require_junit(root: Path, count: int) -> None:
    """Reject malformed or failed JUnit output; example: require_junit(reports)."""
    regular_artifact(root, "junit.xml")
    try:
        parsed = ET.parse(root / "junit.xml").getroot()
    except ET.ParseError as error:
        raise ValueError(f"JUnit report {root / 'junit.xml'}: expected valid XML") from error
    cases = list(parsed.iter("testcase"))
    suites = list(parsed.iter("testsuite"))
    if parsed.tag not in ("testsuite", "testsuites") or not suites:
        raise ValueError(f"JUnit root {parsed.tag!r}: expected at least one testsuite")
    if any(suite.get(name, "0") != "0" for suite in suites for name in ("failures", "errors", "skipped")):
        raise ValueError("JUnit report: expected no failed or skipped cases")
    if len(cases) != count or any(child.tag in ("failure", "error", "skipped") for case in cases for child in case):
        raise ValueError("JUnit report: expected the selected case count and no failed or skipped cases")


def require_report_captures(root: Path, row: Mapping[str, object], capture: str | None) -> None:
    """Check the selected captures using the receipt checker; example: require_report_captures(root, row, None)."""
    if row.get("outline"):
        if capture is not None or not isinstance(row.get("examples"), list) or not row["examples"]:
            raise ValueError("outline capture: expected a capture for each source example")
        names = [example_capture_name(row, example) for example in row["examples"]]
    else:
        names = [selected_capture(root, capture).name]
    for name in names:
        path = selected_capture(root, name)
        check_capture(root, artifact(root, path))


def check_sealed_report(request: SealedRequest, row: Mapping[str, object], revision: str, capture: str | None) -> dict[str, object]:
    """Check scenario artifacts without claiming their origin; example: check_sealed_report(request, row, sha, None)."""
    root = request.reports
    report = json.loads((root / regular_artifact(root, "cucumber.json")["path"]).read_text(encoding="utf-8"))
    require_single_case(report, row)
    check_scenario(report, row)
    source_artifact = regular_artifact(root, "source-revision.json")
    check_source_revision(root, source_artifact, revision)
    source_report = json.loads((root / source_artifact["path"]).read_text(encoding="utf-8"))
    if source_report["before"].get("root") != "/scenario":
        raise ValueError(f"source report root {source_report['before'].get('root')!r}: expected /scenario")
    count = len(row["examples"]) if row.get("outline") and isinstance(row.get("examples"), list) else 1
    require_junit(root, count)
    require_report_captures(root, row, capture)
    return {"scenario_id": row["id"], "content_checked": True, "reusable": False}
