# SPDX-License-Identifier: GPL-2.0-only
"""Build the inventory tool from the selected commit for native capture."""

import os
import subprocess
import tempfile
from pathlib import Path
from typing import Mapping

from capture_build_environment import (
    capture_build_environment,
    reject_ancestor_cargo_configs,
)
from committed_build_source import materialize_source
from capture_dependency_stage import stage_capture_dependencies


def inventory_command(checkout: Path, revision: str) -> list[str]:
    """Choose the committed inventory tool; example: inventory_command(checkout, revision)."""
    return [
        "cargo", "run", "--locked", "--offline", "-p", "oblikovati-build", "--",
        "scenario-ledger", str(checkout.resolve()), revision,
    ]


def isolated_inventory(
    root: Path, checkout: Path, revision: str, commit: str,
    environment: Mapping[str, str] | None = None,
) -> str:
    """Run inventory in a fresh build tree; example: isolated_inventory(root, checkout, revision, commit)."""
    with tempfile.TemporaryDirectory(prefix="oblikovati-capture-inventory-") as name:
        folder = Path(name)
        source = folder / "source"
        materialize_source(root, commit, source)
        reject_ancestor_cargo_configs(source)
        host_environment = os.environ if environment is None else environment
        settings = capture_build_environment(folder, host_environment)
        stage_capture_dependencies(source, host_environment, settings)
        output = subprocess.check_output(
            inventory_command(checkout, revision), cwd=source, env=settings, text=True,
        )
        reject_ancestor_cargo_configs(source)
        return output
