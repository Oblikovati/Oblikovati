#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Validate scenario acceptance evidence without using diagnostic runs as proof."""

import hashlib
import json
import re
import tomllib
from pathlib import Path
from typing import Callable, Mapping


class StaleEvidence(ValueError):
    """A valid receipt no longer matches the current application inputs."""


def local_file(root: Path, name: str) -> Path:
    """Resolve an artifact inside the repository; for example, local_file(root, 'receipt.json')."""
    path = (root / name).resolve()
    if not path.is_relative_to(root.resolve()) or not path.is_file():
        raise ValueError(f"artifact {name!r}: expected an existing file inside {root}")
    return path


def load_object(path: Path) -> dict[str, object]:
    """Read a JSON object; for example, load_object(Path('receipt.json'))."""
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"JSON {path}: expected an object, got {type(value).__name__}")
    return value


def require_equal(label: str, actual: object, expected: object) -> None:
    """Reject mismatched evidence; for example, require_equal('commit', actual, expected)."""
    if actual != expected:
        raise ValueError(f"{label} {actual!r}: expected {expected!r}")


def check_artifact(root: Path, artifact: object) -> Path:
    """Check an artifact's path and digest; for example, check_artifact(root, record)."""
    if not isinstance(artifact, dict):
        raise ValueError(f"artifact {artifact!r}: expected path and sha256 fields")
    name, expected = artifact.get("path"), artifact.get("sha256")
    if not isinstance(name, str) or not isinstance(expected, str):
        raise ValueError(f"artifact {artifact!r}: expected string path and sha256")
    path = local_file(root, name)
    actual = hashlib.sha256(path.read_bytes()).hexdigest()
    require_equal(f"artifact {name!r} SHA-256", actual, expected)
    return path


def matching_elements(
    report: object, row: Mapping[str, object], kind: str
) -> list[dict[str, object]]:
    """Find scenario or background results; for example, matching_elements(report, row, 'scenario')."""
    if not isinstance(report, list):
        raise ValueError(
            f"Cucumber report {type(report).__name__}: expected a feature list"
        )
    matches: list[dict[str, object]] = []
    path = str(row["path"])
    name = f"{row['rule']} {row['scenario']}" if row["rule"] else row["scenario"]
    for feature in report:
        if not isinstance(feature, dict):
            raise ValueError(f"feature {feature!r}: expected an object")
        uri = str(feature.get("uri", "")).replace("\\", "/")
        if uri != path and not uri.endswith("/" + path):
            continue
        elements = feature.get("elements", [])
        if not isinstance(elements, list):
            raise ValueError(f"feature elements {elements!r}: expected a list")
        matches.extend(
            item
            for item in elements
            if isinstance(item, dict)
            and item.get("type") == kind
            and item.get("name") == name
            and item.get("line") == row["line"]
        )
    return matches


def require_passed_result(item: object, label: str) -> None:
    """Require a Cucumber result to pass; for example, require_passed_result(step, 'step')."""
    if not isinstance(item, dict) or not isinstance(item.get("result"), dict):
        raise ValueError(f"{label} {item!r}: expected a Cucumber result")
    require_equal(f"{label} status", item["result"].get("status"), "passed")


def check_steps(element: Mapping[str, object], expected: object, label: str) -> None:
    """Require every source step and its passing result; for example, check_steps(element, steps, 'scenario')."""
    steps = element.get("steps")
    if not isinstance(steps, list) or not isinstance(expected, list):
        raise ValueError(f"{label} steps {steps!r}: expected source and result lists")
    actual = [
        {"line": step.get("line"), "name": step.get("name")}
        for step in steps
        if isinstance(step, dict)
    ]
    require_equal(f"{label} step sequence", actual, expected)
    for step in steps:
        require_passed_result(step, label)


def check_scenario(report: object, row: Mapping[str, object]) -> None:
    """Require a complete passing ordinary scenario; for example, check_scenario(report, row)."""
    if row["outline"]:
        raise ValueError(
            f"scenario {row['id']!r}: expected expanded outline evidence, which is not supported yet"
        )
    expected = row["scenario_steps"]
    if not isinstance(expected, list) or not expected:
        raise ValueError(
            f"scenario {row['id']!r} steps {expected!r}: expected a nonempty source list"
        )
    matches = matching_elements(report, row, "scenario")
    require_equal(f"scenario {row['id']!r} matches", len(matches), 1)
    scenario = matches[0]
    check_steps(scenario, expected, "scenario")
    check_background(report, row)
    for phase in ("before", "after"):
        hooks = scenario.get(phase, [])
        if not isinstance(hooks, list):
            raise ValueError(
                f"scenario {row['id']!r} {phase} {hooks!r}: expected a list"
            )
        if phase == "after" and not hooks:
            raise ValueError(
                f"scenario {row['id']!r} after hooks {hooks!r}: expected native window cleanup"
            )
        for hook in hooks:
            require_passed_result(hook, phase)


def check_background(report: object, row: Mapping[str, object]) -> None:
    """Check the scenario's background steps; for example, check_background(report, row)."""
    expected = row["background_steps"]
    if not isinstance(expected, list):
        raise ValueError(f"background steps {expected!r}: expected a list")
    matches = matching_elements(report, row, "background")
    require_equal(
        f"scenario {row['id']!r} backgrounds", len(matches), int(bool(expected))
    )
    if expected:
        check_steps(matches[0], expected, "background")


def check_source_revision(root: Path, artifact: object, revision: str) -> None:
    """Require an unchanged clean scenario checkout; for example, check_source_revision(root, artifact, revision)."""
    report = load_object(check_artifact(root, artifact))
    require_equal("source report changed", report.get("source_changed"), False)
    for phase in ("before", "after"):
        state = report.get(phase)
        if not isinstance(state, dict):
            raise ValueError(
                f"source report {phase} {state!r}: expected a checkout state"
            )
        require_equal(f"source report {phase} commit", state.get("commit"), revision)
        require_equal(f"source report {phase} dirty", state.get("dirty"), False)
    before = report["before"]
    after = report["after"]
    if (
        not isinstance(before, dict)
        or not isinstance(after, dict)
        or not isinstance(before.get("root"), str)
        or not before["root"]
    ):
        raise ValueError(
            f"source report checkout {before!r}: expected a recorded checkout root"
        )
    require_equal("source report checkout root", after.get("root"), before["root"])


def check_capture(root: Path, artifact: object) -> None:
    """Require a hashed PNG capture; for example, check_capture(root, artifact)."""
    path = check_artifact(root, artifact)
    if not path.read_bytes().startswith(b"\x89PNG\r\n\x1a\n"):
        raise ValueError(f"capture {path}: expected a PNG image")


def check_runner(receipt: Mapping[str, object], row: Mapping[str, object]) -> None:
    """Require the area's native acceptance runner; for example, check_runner(receipt, row)."""
    runners = {
        "application": "application_scenarios_use_the_native_window",
        "sketch": "sketch_scenarios_use_the_native_window",
    }
    expected = runners.get(str(row["area"]))
    if expected is None:
        raise ValueError(
            f"scenario area {row['area']!r}: expected a supported native acceptance runner"
        )
    require_equal("receipt runner", receipt.get("runner"), expected)
    require_equal("receipt exit code", receipt.get("exit_code"), 0)
    environment = receipt.get("environment")
    if not isinstance(environment, dict) or not all(
        isinstance(environment.get(key), str) and environment[key].strip()
        for key in ("os", "renderer")
    ):
        raise ValueError(
            f"receipt environment {environment!r}: expected nonempty os and renderer"
        )


def locked_revision(root: Path, package: str) -> str:
    """Read a unique Git revision from Cargo.lock; for example, locked_revision(root, 'oblikovati-api')."""
    lock = tomllib.loads((root / "Cargo.lock").read_text(encoding="utf-8"))
    packages = [item for item in lock.get("package", []) if item.get("name") == package]
    if len(packages) != 1:
        raise ValueError(
            f"locked package {package!r}: expected one record, got {len(packages)}"
        )
    source = packages[0].get("source", "")
    revision = source.rpartition("#")[2] if isinstance(source, str) else ""
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError(
            f"locked package {package!r} source {source!r}: expected a Git revision"
        )
    return revision


def check_receipt_integrity(
    root: Path, artifact: object
) -> tuple[Path, dict[str, object], Path]:
    """Validate retained artifact hashes; for example, check_receipt_integrity(root, artifact)."""
    path = check_artifact(root, artifact)
    receipt = load_object(path)
    check_artifact(root, receipt.get("source_report"))
    check_capture(root, receipt.get("capture"))
    report = check_artifact(root, receipt.get("cucumber"))
    return path, receipt, report


def verify_receipt(
    root: Path,
    artifact: object,
    row: Mapping[str, object],
    source_revision: str,
    app_commit: str,
    compatible: Callable[[Path, str, str], bool] | None = None,
) -> None:
    """Verify current native evidence; for example, verify_receipt(root, name, row, source, commit)."""
    receipt_path, receipt, report_path = check_receipt_integrity(root, artifact)
    name = str(receipt_path.relative_to(root.resolve()))
    expected = {
        "kind": "native-acceptance",
        "scenario_id": row["id"],
        "scenario_revision": source_revision,
        "source_sha256": row["source_sha256"],
    }
    for key, value in expected.items():
        require_equal(f"receipt {name!r} {key}", receipt.get(key), value)
    check_runner(receipt, row)
    check_source_revision(root, receipt.get("source_report"), source_revision)
    check_scenario(json.loads(report_path.read_text(encoding="utf-8")), row)
    for field, package in (
        ("api_revision", "oblikovati-api"),
        ("kernel_revision", "martensite-tkernel"),
    ):
        recorded = row.get(field)
        require_equal(f"receipt {name!r} {field}", receipt.get(field), recorded)
        current = locked_revision(root, package)
        if recorded != current:
            raise StaleEvidence(
                f"scenario {row['id']!r} {field} {recorded!r}: expected current revision {current!r}"
            )
    tested = receipt.get("application_commit")
    if not isinstance(tested, str):
        raise ValueError(
            f"receipt {name!r} application_commit {tested!r}: expected a Git commit"
        )
    if compatible is None:
        require_equal(f"receipt {name!r} application_commit", tested, app_commit)
    elif not compatible(root, tested, app_commit):
        raise StaleEvidence(
            f"receipt {name!r} application_commit {tested!r}: expected unchanged application inputs at {app_commit!r}"
        )
