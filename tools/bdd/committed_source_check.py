# SPDX-License-Identifier: GPL-2.0-only
"""Compare an observed build source with the files in one Git commit."""

import stat
import tempfile
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Mapping

from committed_build_source import committed_files, materialize_source


@dataclass(frozen=True)
class CommittedReference:
    """Expected source files; example: CommittedReference(commit, hashes, identities)."""

    commit: str
    hashes: Mapping[str, str]
    identities: Mapping[str, str]


def committed_reference(repository: Path, revision: str) -> CommittedReference:
    """Read the committed tree through a verified export; example: committed_reference(repo, sha)."""
    with tempfile.TemporaryDirectory(prefix="oblikovati-source-reference-") as folder:
        record = materialize_source(repository, revision, Path(folder) / "source")
        hashes = record["files"]
        if not isinstance(hashes, dict):
            raise ValueError(f"source hashes {hashes!r}: expected a file hash mapping")
        return CommittedReference(
            revision, hashes, committed_files(repository, revision)
        )


def expected_directories(files: Mapping[str, str]) -> set[str]:
    """Derive all source directory names; example: expected_directories({'a/b': 'hash'})."""
    directories = {"."}
    for name in files:
        parent = PurePosixPath(name).parent
        while parent != PurePosixPath("."):
            directories.add(parent.as_posix())
            parent = parent.parent
    return directories


def verify_source_file(
    name: str, metadata: object, reference: CommittedReference
) -> None:
    """Check file kind, links, permissions and content; example: verify_source_file(name, metadata, reference)."""
    if not isinstance(metadata, dict) or metadata.get("kind") != "file":
        raise ValueError(f"source {name!r} {metadata!r}: expected a regular file")
    if metadata.get("links") != 1:
        raise ValueError(
            f"source {name!r} links {metadata.get('links')!r}: expected a single link"
        )
    mode = metadata.get("mode")
    expected = 0o755 if reference.identities[name].startswith("100755:") else 0o644
    if not isinstance(mode, int) or stat.S_IMODE(mode) != expected:
        raise ValueError(f"source {name!r} mode {mode!r}: expected {expected:o}")
    if metadata.get("sha256") != reference.hashes[name]:
        raise ValueError(
            f"source {name!r} hash {metadata.get('sha256')!r}: expected committed content"
        )


def verify_source_inventory(
    inventory: Mapping[str, object], reference: CommittedReference
) -> None:
    """Reject source changes or extra paths; example: verify_source_inventory(inventory, reference)."""
    if set(reference.hashes) != set(reference.identities):
        raise ValueError(
            "source reference identities: expected the same committed file paths"
        )
    directories = expected_directories(reference.hashes)
    expected = set(reference.hashes) | directories
    if set(inventory) != expected:
        raise ValueError(
            f"source paths {sorted(set(inventory) ^ expected)[:10]!r}: expected the committed tree"
        )
    for name in directories:
        entry = inventory[name]
        if not isinstance(entry, dict) or entry.get("kind") != "directory":
            raise ValueError(
                f"source directory {name!r} {entry!r}: expected a directory"
            )
    for name in reference.hashes:
        verify_source_file(name, inventory[name], reference)
