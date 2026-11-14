#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Run one pinned native scenario and prepare evidence for visual review."""

import argparse
import json
import platform
import subprocess
import sys
from pathlib import Path
from typing import Mapping

from behavior_capture import CaptureRequest, capture_receipt
from behavior_evidence import locked_revision
from behavior_ledger import (
    CLASSIFICATIONS,
    INVENTORY,
    ROOT,
    application_state,
    index_rows,
    read_json,
    require_object,
    validate_classification,
    verify_inputs,
)


class NativeCargoRun:
    """Run the real native Cucumber test and retain its output; example: NativeCargoRun(root, 'application')."""

    def __init__(self, root: Path, area: str) -> None:
        self.root = root
        self.area = area

    def __call__(self, reports: Path, environment: dict[str, str]) -> int:
        command = native_command(self.area)
        with (reports / "native-test.log").open("w", encoding="utf-8") as output:
            with subprocess.Popen(
                command,
                cwd=self.root,
                env=environment,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
            ) as child:
                if child.stdout is None:
                    raise ValueError("native runner stdout: expected a readable pipe")
                for line in child.stdout:
                    output.write(line)
                    print(line, end="", flush=True)
                return child.wait()


def native_command(area: str) -> list[str]:
    """Choose the runner for the selected area; example: native_command('sketch')."""
    if area not in ("application", "sketch", "solid-model"):
        raise ValueError(
            f"scenario area {area!r}: expected application, sketch, or solid-model"
        )
    runner = (
        "sketch_scenarios_use_the_native_window"
        if area == "sketch"
        else "application_scenarios_use_the_native_window"
    )
    command = ["cargo", "test", "--locked", "-p", "oblikovati-head", "--lib"]
    return command + [runner, "--", "--nocapture"]


def find_scenario(
    inventory: Mapping[str, object],
    classifications: Mapping[str, object],
    identifier: str,
) -> dict[str, object]:
    """Select a reviewed scenario with unchanged source; example: find_scenario(inventory, classes, 'scenario:...')."""
    if classifications.get("version") != 1:
        raise ValueError(
            f"classification version {classifications.get('version')!r}: expected 1"
        )
    scenarios = index_rows(inventory.get("scenarios"), "inventory scenarios")
    reviewed = index_rows(classifications.get("scenarios"), "classifications")
    if identifier not in scenarios or identifier not in reviewed:
        raise ValueError(
            f"scenario {identifier!r}: expected an inventoried and reviewed classification"
        )
    row, classification = scenarios[identifier], reviewed[identifier]
    require_current_classification(inventory, row, classification, identifier)
    return dict(row)


def require_current_classification(
    inventory: Mapping[str, object],
    row: Mapping[str, object],
    classification: Mapping[str, object],
    identifier: str,
) -> None:
    """Refuse blocked or stale reviewed source; example: require_current_classification(inventory, row, classification, id)."""
    validate_classification(classification)
    if row.get("source_error") or classification["state"] in ("blocked", "excluded"):
        raise ValueError(f"scenario {identifier!r}: expected valid, non-blocked source")
    expected = (inventory.get("source_revision"), row.get("source_sha256"))
    actual = (
        classification.get("source_revision"),
        classification.get("source_sha256"),
    )
    if actual != expected:
        raise ValueError(
            f"scenario {identifier!r} source {actual!r}: expected current {expected!r}"
        )


def receipt_destination(root: Path, value: Path) -> Path:
    """Keep published receipts within their tracked directory; example: receipt_destination(root, Path('architecture/rust-port/receipts/run'))."""
    parent = (root / "architecture/rust-port/receipts").resolve()
    destination = (root / value).resolve()
    if destination == parent or not destination.is_relative_to(parent):
        raise ValueError(
            f"receipt destination {destination}: expected a new directory under {parent}"
        )
    return destination


def prepare_request(
    checkout: Path,
    identifier: str,
    destination: Path,
    renderer: str,
    capture: str | None,
) -> CaptureRequest:
    """Validate the pinned inputs and construct one native run; example: prepare_request(checkout, scenario_id, dest, 'Vulkan', None)."""
    require_renderer(renderer)
    verified = verified_run_inputs(checkout, identifier)
    return configured_request(checkout, destination, renderer, capture, verified)


def configured_request(
    checkout: Path,
    destination: Path,
    renderer: str,
    capture: str | None,
    verified: tuple[dict[str, object], dict[str, object], str],
) -> CaptureRequest:
    """Assemble verified run settings; example: configured_request(checkout, dest, renderer, None, verified)."""
    inventory, row, commit = verified
    return CaptureRequest(
        ROOT,
        checkout,
        receipt_destination(ROOT, destination),
        row,
        str(inventory["source_revision"]),
        commit,
        platform.system(),
        renderer,
        capture,
    )


def require_renderer(renderer: str) -> None:
    """Require a recorded rendering environment; example: require_renderer('Vulkan Lavapipe')."""
    if not renderer.strip():
        raise ValueError(
            f"renderer {renderer!r}: expected a nonempty renderer description"
        )


def verified_run_inputs(
    checkout: Path, identifier: str
) -> tuple[dict[str, object], dict[str, object], str]:
    """Pin source, lockfile and clean application state; example: verified_run_inputs(checkout, scenario_id)."""
    inventory = require_object(read_json(INVENTORY), "inventory")
    verify_inputs(ROOT, checkout, inventory)
    classifications = require_object(read_json(CLASSIFICATIONS), "classifications")
    row = find_scenario(inventory, classifications, identifier)
    row["api_revision"] = locked_revision(ROOT, "oblikovati-api")
    row["kernel_revision"] = locked_revision(ROOT, "martensite-tkernel")
    commit, clean = application_state(ROOT)
    if not clean:
        raise ValueError(
            f"application commit {commit}: expected clean application inputs before a native run"
        )
    return inventory, row, commit


def main() -> None:
    """Run the command-line workflow; example: capture_cli.py --checkout path --scenario-id id --output path --renderer Vulkan."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--checkout", required=True, type=Path)
    parser.add_argument("--scenario-id", required=True)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--renderer", required=True)
    parser.add_argument(
        "--capture", help="PNG filename for an ordinary scenario when the run creates multiple captures"
    )
    args = parser.parse_args()
    request = prepare_request(
        args.checkout, args.scenario_id, args.output, args.renderer, args.capture
    )
    result = capture_receipt(
        request, NativeCargoRun(ROOT, str(request.row["area"])), application_state
    )
    print(json.dumps({"receipt": result, "review_required": True}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"native receipt: {error}", file=sys.stderr)
        raise SystemExit(1) from error
