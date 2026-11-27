#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Generate and inspect conservative scenario dependency maps."""

import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path

from behavior_ledger import (
    INVENTORY,
    ROOT,
    read_json,
    require_object,
    verify_inputs,
    write_ledger,
)
from scenario_dependencies import (
    build_map,
    changed_dependencies,
    changed_runtime_dependencies,
    dependency_checkout_state,
    require_checkout,
    reuse_decision,
    validate_map,
    worktree_clean,
)
from verify_snapshot import restore_snapshot


def generate(
    checkout: Path,
    output: Path,
    scenario: str | None,
    runtime_root: Path | None,
) -> None:
    """Map a verified snapshot; example: generate(checkout, output)."""
    inventory = require_object(read_json(INVENTORY), "inventory")
    verify_inputs(ROOT, checkout, inventory)
    commit, clean = dependency_checkout_state(ROOT)
    mapping = build_map(ROOT, inventory, commit, scenario, runtime_root)
    validate_map(mapping, inventory)
    if dependency_checkout_state(ROOT) != (commit, clean):
        raise ValueError("application inputs: expected unchanged state while mapping")
    write_ledger(output, mapping)
    print_summary(mapping)


def print_summary(mapping: dict[str, object]) -> None:
    """Report map scope; example: print_summary(mapping)."""
    summary = {
        "scenarios": len(mapping["scenarios"]),
        "tracked_inputs": len(mapping["application_inputs"]),
        "runtime_coverage": mapping["runtime_coverage"],
        "worktree_clean": mapping["worktree_clean"],
    }
    print(json.dumps(summary, indent=2))


def compare(mapping_path: Path, scenario: str, runtime_root: Path | None) -> None:
    """Explain tracked changes for a scenario; example: compare(map_path, scenario_id)."""
    inventory = require_object(read_json(INVENTORY), "inventory")
    mapping = require_object(read_json(mapping_path), "dependency map")
    current, _ = dependency_checkout_state(ROOT)
    changes = changed_dependencies(ROOT, mapping, inventory, scenario, current)
    runtime_changes = runtime_changes_for(mapping, scenario, runtime_root)
    require_checkout(ROOT, current)
    if not worktree_clean(ROOT):
        raise ValueError("application worktree: expected clean state after comparison")
    print(
        json.dumps(
            {
                "scenario": scenario,
                "changed_inputs": changes,
                "runtime_coverage": mapping["runtime_coverage"],
                "changed_runtime_inputs": runtime_changes,
                "receipt_reuse": reuse_decision(mapping, changes, runtime_changes),
            },
            indent=2,
        )
    )


def runtime_changes_for(
    mapping: dict[str, object],
    scenario: str,
    runtime_root: Path | None,
) -> list[str] | None:
    """Compare a runtime only for its own scenario; example: runtime_changes_for(mapping, id, root)."""
    if mapping.get("runtime_scenario") != scenario:
        if runtime_root is not None:
            raise ValueError(
                f"runtime root {runtime_root!r}: expected the recorded scenario "
                f"{mapping.get('runtime_scenario')!r}, got {scenario!r}"
            )
        return None
    if runtime_root is None:
        raise ValueError("runtime root None: expected the recorded scenario runtime")
    return changed_runtime_dependencies(mapping, scenario, runtime_root)


def main() -> None:
    """Run the mapping CLI; example: dependency_cli.py generate --output /tmp/map.json."""
    args = parse_arguments()
    if args.action == "compare":
        compare(args.map, args.scenario, args.runtime_root)
        return
    if args.checkout is not None:
        generate(args.checkout, args.output, args.scenario, args.runtime_root)
        return
    with tempfile.TemporaryDirectory(prefix="oblikovati-dependencies-") as folder:
        checkout = Path(folder) / "checkout"
        restore_snapshot(checkout)
        generate(checkout, args.output, args.scenario, args.runtime_root)


def parse_arguments() -> argparse.Namespace:
    """Read a generate or compare request; example: parse_arguments()."""
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="action", required=True)
    create = subparsers.add_parser("generate")
    create.add_argument("--checkout", type=Path)
    create.add_argument("--output", type=Path, required=True)
    create.add_argument("--scenario")
    create.add_argument("--runtime-root", type=Path)
    inspect = subparsers.add_parser("compare")
    inspect.add_argument("--map", type=Path, required=True)
    inspect.add_argument("--scenario", required=True)
    inspect.add_argument("--runtime-root", type=Path)
    return parser.parse_args()


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"scenario dependencies: {error}", file=sys.stderr)
        raise SystemExit(1) from error
