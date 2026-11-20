#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Run the experimental committed build and sealed runner as diagnostic evidence."""

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Callable

from local_snapshot_evidence import committed_object, scoped_selection
from sandbox_build_probe import BuildProbe
from sealed_build_chain import run_sealed_build
from sealed_execution import SealedRequest, require_request
from sealed_report_check import check_sealed_report, committed_inventory, require_scenario_source, selected_row
from verify_snapshot import git_revision, read_snapshot_metadata, verify_checkout


def pinned_scenario_revision(checkout: Path) -> str:
    """Verify the pinned scenario checkout; example: pinned_scenario_revision(checkout)."""
    metadata, manifest = read_snapshot_metadata()
    verify_checkout(checkout, metadata, manifest, git_revision)
    return str(metadata["snapshot_commit"])


def parse_arguments(arguments: list[str] | None = None) -> argparse.Namespace:
    """Parse explicit experimental inputs; example: parse_arguments(['--help'])."""
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("source", "vendor", "toolchain", "config", "target", "build-logs", "bwrap", "system", "repository", "runtime", "checkout", "reports", "run-log"):
        parser.add_argument(f"--{name}", required=True, type=Path)
    for name in ("revision", "area", "feature", "scenario"):
        parser.add_argument(f"--{name}", required=True)
    parser.add_argument("--capture", help="one PNG filename when a scenario produced multiple captures")
    parser.add_argument("--local-snapshot", help="committed feature-scoped receipt directory for a diagnostic run")
    parser.add_argument("--registry-archives", type=Path)
    parser.add_argument("--git-repo", nargs=2, action="append", metavar=("SOURCE", "REPOSITORY"))
    args = parser.parse_args(arguments)
    if not re.fullmatch(r"[0-9a-f]{40}", args.revision):
        parser.error("--revision: expected a full lowercase SHA-1 commit")
    if args.git_repo and args.registry_archives is None:
        parser.error("--git-repo requires --registry-archives")
    if args.git_repo and len(dict(args.git_repo)) != len(args.git_repo):
        parser.error("--git-repo: expected each source exactly once")
    return args


def requests(args: argparse.Namespace) -> tuple[BuildProbe, SealedRequest]:
    """Construct the linked build and execution requests; example: requests(args)."""
    probe = BuildProbe(args.source, args.vendor, args.toolchain, args.config, args.target, args.build_logs, args.bwrap, args.system)
    run = SealedRequest(args.runtime, args.target / "build-runner", args.checkout, args.reports, args.run_log, args.bwrap, args.area, args.feature, args.scenario)
    return probe, run


def main(
    arguments: list[str] | None = None,
    invoke: Callable[..., dict[str, object]] = run_sealed_build,
    verify: Callable[[Path], str] = pinned_scenario_revision,
    inventory_reader: Callable[[Path, str], dict[str, object]] = committed_inventory,
    report_checker: Callable[..., dict[str, object]] = check_sealed_report,
) -> None:
    """Run the linked diagnostic; example: main(['--help'])."""
    args = parse_arguments(arguments)
    probe, request = requests(args)
    require_request(request)
    if args.local_snapshot:
        metadata = committed_object(args.repository, args.revision, args.local_snapshot, "snapshot.json")
        local_inventory = committed_object(args.repository, args.revision, args.local_snapshot, "inventory.json")
        row, verify = scoped_selection(request, metadata, local_inventory)
        source_revision = verify(request.checkout)
    else:
        source_revision = verify(request.checkout)
        row = selected_row(inventory_reader(args.repository, args.revision), request, source_revision)
    require_scenario_source(request.checkout, row)
    repositories = None if not args.git_repo else {source: Path(path) for source, path in args.git_repo}
    result = invoke(probe, request, args.repository, args.revision, verify, archives=args.registry_archives, git_repositories=repositories)
    if result.get("reusable") is not False:
        raise ValueError(f"chain reusable {result.get('reusable')!r}: expected false")
    result["report_check"] = report_checker(request, row, source_revision, args.capture)
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"sealed build diagnostic: {error}", file=sys.stderr)
        raise SystemExit(1) from error
