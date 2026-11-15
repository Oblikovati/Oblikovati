#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Run and verify a real native receipt; use Xvfb or an active desktop display."""

import argparse
import subprocess
import sys
import tempfile
from pathlib import Path

from behavior_capture import artifact, receipt_path
from behavior_evidence import verify_receipt
from behavior_ledger import ROOT
from capture_cli import prepare_request
from verify_snapshot import restore_snapshot

SCENARIO = "scenario:233169e6ac19c7ddb757045b4193b135178c2e3e5089bebbc1fc574f8784e861"


def run_native_receipt(checkout: Path, output: Path, renderer: str, scenario: str) -> None:
    """Run the real capture CLI; example: run_native_receipt(source, dest, 'Vulkan', SCENARIO)."""
    request = prepare_request(checkout, scenario, output, renderer, None)
    subprocess.run(
        [sys.executable, "-B", "tools/bdd/capture_cli.py", "--checkout", str(checkout),
         "--scenario-id", scenario, "--output", str(output), "--renderer", renderer],
        cwd=ROOT, check=True,
    )
    receipt = receipt_path(request, request.destination)
    verify_receipt(ROOT, artifact(ROOT, receipt), request.row, request.source_revision,
                   request.application_commit)
    print(f"Verified native capture receipt: {receipt}")


def main() -> None:
    """Exercise the native capture pipeline; run with `python3 tools/bdd/live_capture_check.py --help`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--renderer", required=True)
    parser.add_argument("--scenario-id", default=SCENARIO)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="oblikovati-live-source-") as directory:
        checkout = Path(directory) / "checkout"
        restore_snapshot(checkout)
        run_native_receipt(checkout, args.output, args.renderer, args.scenario_id)


if __name__ == "__main__":
    main()
