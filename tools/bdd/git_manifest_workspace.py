# SPDX-License-Identifier: GPL-2.0-only
"""Read the implicit Cargo workspace from independently checked Git blobs."""

import tomllib
from pathlib import Path, PurePosixPath
from typing import Mapping

from git_object_inventory import git_blob_bytes


def locked_workspace(repository: Path, tree: Mapping[str, str], root: str, label: str) -> Mapping[str, object]:
    """Read the closest tracked workspace; example: locked_workspace(repo, tree, root, label)."""
    parent = PurePosixPath(root).parent
    while True:
        path = (parent / "Cargo.toml").as_posix()
        identity = tree.get(path)
        if identity is not None:
            if not identity.startswith(("100644:blob:", "100755:blob:")):
                raise ValueError(f"workspace manifest {path}: expected a regular tracked file")
            document = tomllib.loads(git_blob_bytes(repository, identity.split(":")[-1]).decode("utf-8"))
            workspace = document.get("workspace")
            if isinstance(workspace, dict):
                return workspace
        if parent == PurePosixPath("."):
            raise ValueError(f"manifest {label} workspace: expected a tracked workspace root")
        parent = parent.parent
