#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Report, generate, or enforce the scenario behavior ledger."""

import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path

from behavior_ledger import (
    CLASSIFICATIONS,
    INVENTORY,
    ROOT,
    application_state,
    incomplete_baseline_backlog,
    read_json,
    reconcile,
    require_classified,
    require_list,
    require_object,
    summarize,
    verify_current_receipt,
    verify_inputs,
    write_ledger,
)
from behavior_evidence import check_receipt_integrity
from behavior_backlog import feature_backlog
from behavior_queue import scenario_queue
from verify_snapshot import restore_snapshot


def execute(action: str, checkout: Path, output: Path | None) -> None:
    """Run one ledger operation; for example, execute('report', checkout, None)."""
    inventory = require_object(read_json(INVENTORY), "inventory")
    verify_inputs(ROOT, checkout, inventory)
    commit, clean = application_state(ROOT)
    ledger = reconcile(
        inventory,
        read_json(CLASSIFICATIONS),
        commit,
        clean,
        ROOT,
        verify_current_receipt,
        check_receipt_integrity,
    )
    if application_state(ROOT) != (commit, clean):
        raise ValueError(
            "application inputs: expected unchanged Git state during ledger generation"
        )
    if action == "check":
        check_generated(
            ledger, read_json(ROOT / "architecture/rust-port/behavior-ledger.json")
        )
    if action == "generate":
        if output is None:
            raise ValueError("ledger output None: expected --output for generate")
        write_ledger(output, ledger)
        return
    rows = require_list(ledger["scenarios"], "ledger scenarios")
    if action == "backlog":
        print(json.dumps(feature_backlog(rows), indent=2))
        return
    if action == "queue":
        queue = scenario_queue(rows)
        queue.update(source_revision=ledger["source_revision"], application_commit=commit)
        print(json.dumps(queue, indent=2))
        return
    print(
        json.dumps(
            {
                "source_revision": ledger["source_revision"],
                "application_commit": commit,
                "areas": summarize(rows, "area"),
                "milestones": summarize(rows, "milestone"),
                "incomplete_baseline_backlog": incomplete_baseline_backlog(rows),
                "orphaned_classifications": ledger["orphaned_classifications"],
            },
            indent=2,
        )
    )


def check_generated(ledger: dict[str, object], expected: object) -> None:
    """Check generated rows and classifications; for example, check_generated(ledger, saved)."""
    if expected != ledger:
        raise ValueError("checked-in behavior ledger: expected current generated rows")
    require_classified(ledger)


def main() -> None:
    """Run the CLI; for example, `python3 -B tools/bdd/behavior_cli.py check`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("report", "check", "generate", "backlog", "queue"))
    parser.add_argument("checkout", type=Path, nargs="?")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if args.checkout is not None:
        execute(args.action, args.checkout, args.output)
        return
    with tempfile.TemporaryDirectory(prefix="oblikovati-scenarios-") as folder:
        checkout = Path(folder) / "checkout"
        restore_snapshot(checkout)
        execute(args.action, checkout, args.output)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"behavior ledger: {error}", file=sys.stderr)
        raise SystemExit(1) from error
