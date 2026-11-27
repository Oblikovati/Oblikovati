# SPDX-License-Identifier: GPL-2.0-only
"""Read a committed, feature-scoped local snapshot for diagnostic execution."""

import json
from pathlib import Path, PurePosixPath
import re
import subprocess
from typing import Callable

from git_environment import git_environment
from sealed_execution import SealedRequest
from sealed_report_check import selected_row
from verify_snapshot import git_revision, verify_checkout


def committed_object(repository: Path, revision: str, directory: str, name: str) -> dict[str, object]:
    """Read snapshot metadata from the tested commit; example: committed_object(repo, sha, directory, 'snapshot.json')."""
    path = PurePosixPath(directory)
    if path.is_absolute() or path.as_posix() != directory or ".." in path.parts or not directory.startswith("architecture/rust-port/receipts/"):
        raise ValueError(f"snapshot directory {directory!r}: expected a relative receipt directory")
    raw = subprocess.check_output(["git", "-C", str(repository), "show", f"{revision}:{directory}/{name}"], env=git_environment())
    value = json.loads(raw)
    if not isinstance(value, dict):
        raise ValueError(f"snapshot {name!r}: expected a JSON object")
    return value


def scoped_selection(request: SealedRequest, metadata: dict[str, object], inventory: dict[str, object], read_revision: Callable[[Path], str] = git_revision) -> tuple[dict[str, object], Callable[[Path], str]]:
    """Check a local feature snapshot and select one scenario; example: scoped_selection(request, metadata, inventory)."""
    revision, files = metadata.get("revision"), metadata.get("files")
    if not isinstance(revision, str) or not re.fullmatch(r"[0-9a-f]{40}", revision) or not isinstance(files, dict):
        raise ValueError(f"snapshot revision {revision!r} or manifest {files!r}: expected a Git SHA and file map")
    scope = f"{request.area}/{request.feature}"
    if metadata.get("feature_scope") != scope:
        raise ValueError(f"snapshot scope {metadata.get('feature_scope')!r}: expected {scope!r}")
    if not all(isinstance(name, str) and isinstance(digest, str) and re.fullmatch(r"[0-9a-f]{64}", digest) for name, digest in files.items()):
        raise ValueError("snapshot manifest: expected relative paths and SHA-256 values")
    row = selected_row(inventory, request, revision)
    def verify(checkout: Path) -> str:
        verify_checkout(checkout, {"snapshot_commit": revision}, files, read_revision)
        return revision
    verify(request.checkout)
    return row, verify
