#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Verify a restored MCAD scenario snapshot before using it for BDD runs."""

import argparse
import hashlib
import json
import subprocess
from pathlib import Path
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
    files: dict[str, str] = {}
    for path in sorted(checkout.rglob("*")):
        relative = path.relative_to(checkout)
        if ".git" in relative.parts:
            continue
        if path.is_symlink():
            raise ValueError(
                f"snapshot path {path}: expected a regular file or directory"
            )
        if path.is_file():
            files[relative.as_posix()] = sha256_file(path)
    return files


def git_revision(checkout: Path) -> str:
    """Read a checkout revision; for example, git_revision(Path('/tmp/scenarios'))."""
    if not (checkout / ".git").exists():
        raise ValueError(f"snapshot {checkout}: expected its own Git checkout")
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
    actual = checkout_files(checkout)
    changed = sorted(
        name
        for name in actual.keys() | manifest.keys()
        if actual.get(name) != manifest.get(name)
    )
    if changed:
        raise ValueError(
            f"snapshot {checkout}: expected the pinned files; changed or missing: {changed[:10]}"
        )


def main() -> None:
    """Verify a snapshot checkout; run as `python3 tools/bdd/verify_snapshot.py CHECKOUT`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("checkout", type=Path)
    args = parser.parse_args()
    metadata = json.loads((SNAPSHOT_FOLDER / "current.json").read_text())
    bundle = SNAPSHOT_FOLDER / metadata["bundle"]
    manifest_path = SNAPSHOT_FOLDER / metadata["manifest"]
    verify_artifact(bundle, metadata["bundle_sha256"])
    verify_artifact(manifest_path, metadata["manifest_sha256"])
    manifest = json.loads(manifest_path.read_text())
    verify_checkout(args.checkout, metadata, manifest, git_revision)
    print(
        f"Verified {len(manifest)} files at snapshot commit {metadata['snapshot_commit']}"
    )


if __name__ == "__main__":
    main()
