#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Combine the pinned scenario inventory, classifications, and verified receipts."""

import json
import os
import re
import tempfile
import subprocess
from collections import Counter, defaultdict
from pathlib import Path
from typing import Callable, Mapping

from behavior_evidence import StaleEvidence, verify_receipt
from verify_snapshot import (
    SNAPSHOT_FOLDER,
    git_revision,
    verify_artifact,
    verify_checkout,
)

ROOT = Path(__file__).resolve().parents[2]
INVENTORY = ROOT / "architecture/rust-port/behavior-source-inventory.json"
CLASSIFICATIONS = ROOT / "architecture/rust-port/behavior-classifications.json"
STATES = frozenset({"inventoried", "blocked", "implemented", "verified", "excluded"})


def read_json(path: Path) -> object:
    """Read JSON from a file; for example, read_json(Path('inventory.json'))."""
    return json.loads(path.read_text(encoding="utf-8"))


def require_object(value: object, label: str) -> dict[str, object]:
    """Require a JSON object; for example, require_object(value, 'inventory')."""
    if not isinstance(value, dict):
        raise ValueError(f"{label} {value!r}: expected a JSON object")
    return value


def require_list(value: object, label: str) -> list[object]:
    """Require a JSON list; for example, require_list(value, 'scenarios')."""
    if not isinstance(value, list):
        raise ValueError(f"{label} {value!r}: expected a JSON list")
    return value


def index_rows(value: object, label: str) -> dict[str, dict[str, object]]:
    """Index unique scenario IDs; for example, index_rows(rows, 'inventory')."""
    result: dict[str, dict[str, object]] = {}
    for raw in require_list(value, label):
        row = require_object(raw, label)
        identifier = row.get("id")
        if not isinstance(identifier, str) or not identifier or identifier in result:
            raise ValueError(
                f"{label} ID {identifier!r}: expected a unique nonempty string"
            )
        result[identifier] = row
    return result


def text_field(row: Mapping[str, object], field: str, label: str) -> str:
    """Read a required nonempty string; for example, text_field(row, 'owner', 'classification')."""
    value = row.get(field)
    if not isinstance(value, str) or not value.strip():
        raise ValueError(f"{label} {field} {value!r}: expected a nonempty string")
    return value


def validate_classification(row: Mapping[str, object]) -> None:
    """Check classification fields; for example, validate_classification(classification)."""
    label = f"classification {row.get('id')!r}"
    for field in (
        "source_revision",
        "source_sha256",
        "milestone",
        "owner",
        "target_module",
        "acceptance_test",
    ):
        text_field(row, field, label)
    milestone = str(row["milestone"])
    if not re.fullmatch(r"R[0-9]+[A-Z]?", milestone):
        raise ValueError(
            f"{label} milestone {milestone!r}: expected an R-number milestone"
        )
    if row.get("state") not in STATES:
        raise ValueError(
            f"{label} state {row.get('state')!r}: expected one of {sorted(STATES)}"
        )
    issues = row.get("issues")
    if (
        not isinstance(issues, list)
        or not issues
        or not all(
            isinstance(issue, str)
            and re.fullmatch(r"https://github\.com/[^/]+/[^/]+/issues/[0-9]+", issue)
            for issue in issues
        )
    ):
        raise ValueError(
            f"{label} issues {issues!r}: expected at least one GitHub issue link"
        )
    if row.get("state") in ("blocked", "excluded"):
        text_field(row, "reason", label)


def reconcile_row(
    source: dict[str, object],
    classification: dict[str, object] | None,
    revision: str,
    app_commit: str,
    app_clean: bool,
    root: Path,
    receipt_check: Callable[[Path, object, Mapping[str, object], str, str], None],
    integrity_check: Callable[[Path, object], object],
) -> dict[str, object]:
    """Reconcile one source row; for example, reconcile_row(source, None, revision, commit, True, root, verify_receipt)."""
    result = dict(source)
    result.update(
        {
            "classified": False,
            "stale": False,
            "stale_reason": None,
            "historical_receipt": None,
            "source_revision": revision,
            "issues": [],
            "target_module": None,
            "acceptance_test": None,
            "go_reference": None,
            "go_test_premise": None,
            "api_revision": None,
            "kernel_revision": None,
            "kernel_capabilities": None,
            "platforms": None,
            "kernel_integration": None,
            "rust_tests": None,
            "oracle": None,
            "expected_result": None,
        }
    )
    if classification is None:
        return result
    validate_classification(classification)
    if source.get("capture_only") and classification["state"] != "excluded":
        raise ValueError(
            f"scenario {source.get('id')!r}: expected excluded classification for @capture_only"
        )
    if (
        source.get("capture_host_only")
        and not source.get("capture_only")
        and classification["state"] != "blocked"
    ):
        raise ValueError(
            f"scenario {source.get('id')!r}: expected blocked classification for @capture_host_only"
        )
    if classification["state"] == "verified":
        integrity_check(root, classification.get("last_receipt"))
    for field in (
        "milestone",
        "owner",
        "target_module",
        "acceptance_test",
        "issues",
        "go_reference",
        "go_test_premise",
        "api_revision",
        "kernel_revision",
        "kernel_capabilities",
        "platforms",
        "kernel_integration",
        "rust_tests",
        "oracle",
        "expected_result",
    ):
        result[field] = classification.get(field)
    result["classified"] = True
    if classification.get("historical_receipts"):
        result["historical_receipts"] = classification["historical_receipts"]
    stale = (
        classification["source_revision"] != revision
        or classification["source_sha256"] != source["source_sha256"]
    )
    result["stale"] = stale
    if stale or source["source_error"] is not None:
        result["state"] = (
            "blocked" if source["source_error"] is not None else "inventoried"
        )
        result["stale"] = stale or classification["state"] == "verified"
        result["stale_reason"] = (
            "source changed or is invalid" if result["stale"] else None
        )
        result["historical_receipt"] = classification.get("last_receipt")
        result["last_receipt"] = None
        return result
    result["state"] = classification["state"]
    result["reason"] = classification.get("reason")
    result["last_receipt"] = classification.get("last_receipt")
    if result["state"] == "verified":
        try:
            receipt_check(root, result["last_receipt"], result, revision, app_commit)
            if not app_clean:
                raise StaleEvidence("uncommitted application inputs")
        except StaleEvidence as error:
            result.update(
                {
                    "state": "inventoried",
                    "stale": True,
                    "stale_reason": str(error),
                    "historical_receipt": result["last_receipt"],
                    "last_receipt": None,
                }
            )

    return result


def reconcile(
    inventory: object,
    classifications: object,
    app_commit: str,
    app_clean: bool,
    root: Path,
    receipt_check: Callable[[Path, object, Mapping[str, object], str, str], None],
    integrity_check: Callable[[Path, object], object],
) -> dict[str, object]:
    """Build a ledger without losing orphaned classifications; for example, reconcile(inventory, classes, commit, True, root, verify_receipt)."""
    source = require_object(inventory, "inventory")
    revision = text_field(source, "source_revision", "inventory")
    classes = require_object(classifications, "classifications")
    if classes.get("version") != 1:
        raise ValueError(
            f"classification version {classes.get('version')!r}: expected 1"
        )
    inputs = index_rows(source.get("scenarios"), "inventory scenarios")
    assignments = index_rows(classes.get("scenarios"), "classifications")
    rows = [
        reconcile_row(
            row,
            assignments.get(key),
            revision,
            app_commit,
            app_clean,
            root,
            receipt_check,
            integrity_check,
        )
        for key, row in inputs.items()
    ]
    return {
        "source_revision": revision,
        "scenarios": rows,
        "orphaned_classifications": sorted(assignments.keys() - inputs.keys()),
    }


def summarize(rows: list[object], key: str) -> dict[str, dict[str, int]]:
    """Count verified and total rows by area or milestone; for example, summarize(rows, 'area')."""
    groups: defaultdict[str, Counter[str]] = defaultdict(Counter)
    for value in rows:
        row = require_object(value, "ledger scenario")
        group = str(row.get(key) or "unclassified")
        groups[group]["total"] += 1
        groups[group]["verified"] += int(row.get("state") == "verified")
        groups[group]["blocked"] += int(row.get("state") == "blocked")
        groups[group]["excluded"] += int(row.get("state") == "excluded")
        groups[group]["unclassified"] += int(not row.get("classified"))
        groups[group]["stale"] += int(bool(row.get("stale")))
    return {
        name: dict(sorted(counts.items())) for name, counts in sorted(groups.items())
    }


BASELINE_FIELDS = (
    "go_reference",
    "go_test_premise",
    "api_revision",
    "kernel_revision",
    "kernel_capabilities",
    "platforms",
    "kernel_integration",
    "rust_tests",
    "oracle",
    "expected_result",
)


def missing_baseline_fields(row: Mapping[str, object]) -> list[str]:
    """List R0 evidence fields still missing; for example, missing_baseline_fields(row)."""
    return [
        field
        for field in BASELINE_FIELDS
        if not isinstance(row.get(field), str) or not str(row[field]).strip()
    ]


def require_classified(ledger: Mapping[str, object]) -> None:
    """Fail on missing, stale, or orphaned classifications; for example, require_classified(ledger)."""
    rows = require_list(ledger.get("scenarios"), "ledger scenarios")
    missing = sum(not require_object(row, "scenario").get("classified") for row in rows)
    stale = sum(bool(require_object(row, "scenario").get("stale")) for row in rows)
    orphans = require_list(
        ledger.get("orphaned_classifications"), "orphaned classifications"
    )
    if missing or stale or orphans:
        raise ValueError(
            f"ledger classifications: expected all current; {missing} missing, {stale} stale, {len(orphans)} orphaned"
        )
    incomplete = sum(
        bool(missing_baseline_fields(require_object(row, "scenario"))) for row in rows
    )
    if incomplete:
        raise ValueError(
            f"ledger baseline: expected R0 evidence fields; {incomplete} scenarios incomplete"
        )


def verify_inputs(root: Path, checkout: Path, inventory: Mapping[str, object]) -> None:
    """Verify the pinned checkout and inventory hashes; for example, verify_inputs(root, checkout, inventory)."""
    metadata = require_object(
        read_json(SNAPSHOT_FOLDER / "current.json"), "snapshot metadata"
    )
    verify_artifact(
        SNAPSHOT_FOLDER / str(metadata["bundle"]), str(metadata["bundle_sha256"])
    )
    manifest_path = SNAPSHOT_FOLDER / str(metadata["manifest"])
    verify_artifact(manifest_path, str(metadata["manifest_sha256"]))
    manifest = require_object(read_json(manifest_path), "snapshot manifest")
    verify_checkout(checkout, metadata, manifest, git_revision)
    if inventory.get("source_revision") != metadata["snapshot_commit"]:
        raise ValueError(
            f"inventory revision {inventory.get('source_revision')!r}: expected {metadata['snapshot_commit']!r}"
        )
    for row in index_rows(inventory.get("scenarios"), "inventory scenarios").values():
        if manifest.get(row.get("path")) != row.get("source_sha256"):
            raise ValueError(
                f"inventory source {row.get('path')!r}: expected its pinned file hash"
            )
    tool = os.environ.get("OBK_SCENARIO_LEDGER_TOOL")
    command = ([tool] if tool else ["cargo", "xtask"]) + [
        "scenario-ledger",
        str(checkout.resolve()),
        str(metadata["snapshot_commit"]),
    ]
    generated = read_inventory_command(root, command)
    if json.loads(generated) != inventory:
        raise ValueError(
            "scenario inventory: expected all declarations from the current pinned source"
        )


def read_inventory_command(root: Path, command: list[str]) -> str:
    """Run inventory without Git index writes; example: read_inventory_command(root, command)."""
    environment = os.environ.copy()
    environment["GIT_OPTIONAL_LOCKS"] = "0"
    return subprocess.check_output(command, cwd=root, text=True, env=environment)


APPLICATION_PATHS = [
    ".",
    ":(exclude)README.md",
    ":(exclude)architecture/rust-port/**",
    ":(exclude)tools/bdd/**",
    ":(exclude)tools/build/**",
]


def application_state(root: Path) -> tuple[str, bool]:
    """Read application commit and relevant dirty state; for example, application_state(Path('.'))."""
    commit = subprocess.check_output(
        ["git", "-C", str(root), "rev-parse", "HEAD"], text=True
    ).strip()
    status = subprocess.check_output(
        [
            "git",
            "-C",
            str(root),
            "status",
            "--porcelain",
            "--untracked-files=all",
            "--",
            *APPLICATION_PATHS,
        ],
        text=True,
    )
    return commit, not bool(status.strip())


def application_inputs_match(root: Path, tested: str, current: str) -> bool:
    """Check that an ancestor receipt has the same application inputs; for example, application_inputs_match(root, old, head)."""
    if not re.fullmatch(r"[0-9a-f]{40}", tested) or not re.fullmatch(
        r"[0-9a-f]{40}", current
    ):
        return False
    ancestor = subprocess.run(
        ["git", "-C", str(root), "merge-base", "--is-ancestor", tested, current],
        capture_output=True,
    )
    if ancestor.returncode != 0:
        return False
    difference = subprocess.run(
        [
            "git",
            "-C",
            str(root),
            "diff",
            "--quiet",
            tested,
            current,
            "--",
            *APPLICATION_PATHS,
        ],
        capture_output=True,
    )
    if difference.returncode not in (0, 1):
        raise ValueError(
            f"application diff {tested!r}..{current!r}: expected a valid Git comparison"
        )
    return difference.returncode == 0


def verify_current_receipt(
    root: Path, artifact: object, row: Mapping[str, object], source: str, commit: str
) -> None:
    """Validate a receipt against current inputs; for example, verify_current_receipt(root, artifact, row, source, commit)."""
    verify_receipt(root, artifact, row, source, commit, application_inputs_match)


def write_ledger(path: Path, ledger: Mapping[str, object]) -> None:
    """Atomically write a ledger; for example, write_ledger(Path('ledger.json'), ledger)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        "w", encoding="utf-8", dir=path.parent, delete=False
    ) as stream:
        temporary = Path(stream.name)
        json.dump(ledger, stream, separators=(",", ":"))
        stream.write("\n")
    try:
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)
