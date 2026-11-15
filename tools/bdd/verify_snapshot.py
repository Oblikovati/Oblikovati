#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Verify a restored MCAD scenario snapshot before using it for BDD runs."""

import argparse
import hashlib
import json
import stat
import subprocess
from pathlib import Path, PurePosixPath
from typing import Callable

SNAPSHOT_FOLDER = (
    Path(__file__).resolve().parents[2] / "test-utilities/benchmarks/scenario-snapshots"
)


def sha256_file(path: Path) -> str:
    """Return a file's SHA-256; for example, sha256_file(Path('current.json'))."""
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify_artifact(path: Path, expected: str) -> None:
    """Reject an artifact whose hash differs; for example, verify_artifact(bundle, digest)."""
    actual = sha256_file(path)
    if actual != expected:
        raise ValueError(f"artifact {path}: expected SHA-256 {expected}, got {actual}")


def checkout_files(checkout: Path) -> dict[str, str]:
    """Hash the checkout files; for example, checkout_files(Path('/tmp/scenarios'))."""
    files, _ = checkout_namespace(checkout)
    return files


def checkout_namespace(checkout: Path) -> tuple[dict[str, str], set[str]]:
    """Read files and directories; for example, checkout_namespace(Path('/tmp/scenarios'))."""
    files: dict[str, str] = {}
    directories: set[str] = set()
    for path in sorted(checkout.rglob("*")):
        relative = path.relative_to(checkout)
        if relative.parts[0] == ".git":
            continue
        mode = path.lstat().st_mode
        if stat.S_ISREG(mode):
            files[relative.as_posix()] = sha256_file(path)
        elif stat.S_ISDIR(mode):
            directories.add(relative.as_posix())
        else:
            raise ValueError(
                f"snapshot path {path}: expected a regular file or directory"
            )
    return files, directories


def manifest_directories(manifest: dict[str, str]) -> set[str]:
    """Derive tracked parents; for example, manifest_directories({'a/b': 'hash'})."""
    directories: set[str] = set()
    for name in manifest:
        source = PurePosixPath(name)
        if (
            not source.parts
            or source.is_absolute()
            or ".." in source.parts
            or "\\" in name
            or source.as_posix() != name
        ):
            raise ValueError(
                f"manifest path {name!r}: expected a normalized relative file path"
            )
        parent = source.parent
        while parent != PurePosixPath("."):
            directories.add(parent.as_posix())
            parent = parent.parent
    return directories


def git_revision(checkout: Path) -> str:
    """Read a checkout revision; for example, git_revision(Path('/tmp/scenarios'))."""
    if not (checkout / ".git").exists():
        raise ValueError(f"snapshot {checkout}: expected its own Git checkout")
    status = subprocess.check_output(
        ["git", "-C", str(checkout), "status", "--porcelain", "--untracked-files=all"],
        text=True,
    )
    if status:
        raise ValueError(f"snapshot {checkout}: expected clean Git status, got {status!r}")
    return subprocess.check_output(
        ["git", "-C", str(checkout), "rev-parse", "HEAD"], text=True
    ).strip()


def verify_checkout(
    checkout: Path,
    metadata: dict[str, object],
    manifest: dict[str, str],
    read_revision: Callable[[Path], str],
) -> None:
    """Check revision and content; for example, verify_checkout(path, metadata, manifest, git_revision)."""
    revision = read_revision(checkout)
    if revision != metadata["snapshot_commit"]:
        raise ValueError(
            f"snapshot {checkout}: expected commit {metadata['snapshot_commit']}, got {revision}"
        )
    actual, directories = checkout_namespace(checkout)
    changed = sorted(
        name
        for name in actual.keys() | manifest.keys()
        if actual.get(name) != manifest.get(name)
    )
    if changed:
        raise ValueError(
            f"snapshot {checkout}: expected the pinned files; changed or missing: {changed[:10]}"
        )
    unexpected = sorted(directories - manifest_directories(manifest))
    if unexpected:
        raise ValueError(
            f"snapshot {checkout}: expected only pinned directories; unexpected: {unexpected[:10]}"
        )


def read_snapshot_metadata() -> tuple[dict[str, object], dict[str, str]]:
    """Verify the pinned artifacts and return their manifest; example: read_snapshot_metadata()."""
    metadata = json.loads((SNAPSHOT_FOLDER / "current.json").read_text())
    verify_artifact(SNAPSHOT_FOLDER / metadata["bundle"], metadata["bundle_sha256"])
    manifest_path = SNAPSHOT_FOLDER / metadata["manifest"]
    verify_artifact(manifest_path, metadata["manifest_sha256"])
    return metadata, json.loads(manifest_path.read_text())


def restore_snapshot(destination: Path) -> None:
    """Restore and verify the pinned source; example: restore_snapshot(Path('/tmp/source'))."""
    metadata, manifest = read_snapshot_metadata()
    subprocess.run(
        ["git", "clone", "--quiet", "-c", "core.autocrlf=false", "--branch",
         metadata["branch"], str(SNAPSHOT_FOLDER / metadata["bundle"]), str(destination)],
        check=True,
    )
    verify_checkout(destination, metadata, manifest, git_revision)


def main() -> None:
    """Verify or restore a snapshot; run as `python3 tools/bdd/verify_snapshot.py CHECKOUT`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("checkout", type=Path)
    parser.add_argument("--restore", action="store_true")
    args = parser.parse_args()
    if args.restore:
        restore_snapshot(args.checkout)
        return
    metadata, manifest = read_snapshot_metadata()
    verify_checkout(args.checkout, metadata, manifest, git_revision)
    print(f"Verified {len(manifest)} files at snapshot commit {metadata['snapshot_commit']}")


if __name__ == "__main__":
    main()
