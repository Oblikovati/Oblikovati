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


def inventory_command(checkout: Path, revision: str) -> list[str]:
    """Choose the committed inventory tool; example: inventory_command(checkout, revision)."""
    return [
        "cargo", "run", "--locked", "--offline", "-p", "oblikovati-build", "--",
        "scenario-ledger", str(checkout.resolve()), revision,
    ]


def inventory_environment(folder: Path, environment: Mapping[str, str]) -> dict[str, str]:
    """Restrict Cargo and Git settings; example: inventory_environment(folder, env)."""
    settings = capture_build_environment(folder, environment)
    settings.update({
        "GIT_OPTIONAL_LOCKS": "0",
        "GIT_CONFIG_NOSYSTEM": "1",
        "GIT_CONFIG_GLOBAL": os.devnull,
        "GIT_NO_REPLACE_OBJECTS": "1",
    })
    return settings


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
        settings = inventory_environment(folder, os.environ if environment is None else environment)
        output = subprocess.check_output(
            inventory_command(checkout, revision), cwd=source, env=settings, text=True,
        )
        reject_ancestor_cargo_configs(source)
        return output
