# SPDX-License-Identifier: GPL-2.0-only
"""Materialize a verified Git tree for experimental bounded native builds."""

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import tarfile
import tempfile
from pathlib import Path, PurePosixPath
from typing import BinaryIO, Mapping

from scenario_dependencies import parse_tree_entry


def git_environment() -> dict[str, str]:
    """Keep Git configuration out of the export; example: git_environment()."""
    return {
        "PATH": "/usr/bin:/bin",
        "LC_ALL": "C",
        "GIT_CONFIG_NOSYSTEM": "1",
        "GIT_CONFIG_GLOBAL": "/dev/null",
        "GIT_NO_REPLACE_OBJECTS": "1",
    }


def validate_path(name: str) -> None:
    """Reject paths outside the source tree; example: validate_path('src/lib.rs')."""
    path = PurePosixPath(name)
    if (
        not name
        or path.is_absolute()
        or path.as_posix() != name
        or any(part in (".", "..", ".git") for part in path.parts)
        or "\\" in name
    ):
        raise ValueError(
            f"source path {name!r}: expected a normalized relative Git path"
        )


def committed_files(repository: Path, revision: str) -> dict[str, str]:
    """Read supported committed files; example: committed_files(repo, full_sha)."""
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError(f"revision {revision!r}: expected a full Git SHA-1")
    result = subprocess.check_output(
        ["git", "-C", str(repository), "ls-tree", "-rz", "--full-tree", revision],
        env=git_environment(),
    )
    files: dict[str, str] = {}
    for entry in result.split(b"\0"):
        if not entry:
            continue
        name, identity = parse_tree_entry(entry)
        validate_path(name)
        if not identity.startswith(("100644:blob:", "100755:blob:")):
            raise ValueError(
                f"source entry {name!r} {identity!r}: expected a regular Git blob"
            )
        files[name] = identity
    if not files:
        raise ValueError(f"revision {revision!r}: expected a nonempty Git tree")
    return files


def write_blob(stream: BinaryIO, target: Path, size: int, identity: str) -> str:
    """Write and check one Git blob; example: write_blob(stream, path, size, identity)."""
    git_hash = hashlib.sha1(f"blob {size}\0".encode())
    content_hash = hashlib.sha256()
    remaining = size
    with target.open("xb") as output:
        while remaining:
            chunk = stream.read(min(1024 * 1024, remaining))
            if not chunk:
                raise ValueError(
                    f"source file {target}: expected {remaining} more bytes"
                )
            output.write(chunk)
            git_hash.update(chunk)
            content_hash.update(chunk)
            remaining -= len(chunk)
    if git_hash.hexdigest() != identity.split(":")[-1]:
        raise ValueError(f"source file {target}: expected Git blob {identity!r}")
    os.chmod(target, 0o755 if identity.startswith("100755:") else 0o644)
    return content_hash.hexdigest()


def unpack_member(
    archive: tarfile.TarFile,
    member: tarfile.TarInfo,
    destination: Path,
    files: Mapping[str, str],
    seen: dict[str, str],
) -> None:
    """Extract one expected regular blob; example: unpack_member(archive, member, root, files, seen)."""
    name = member.name.rstrip("/") if member.isdir() else member.name
    validate_path(name)
    if member.isdir():
        return
    if not member.isfile() or name not in files or name in seen:
        raise ValueError(f"archive member {name!r}: expected one tracked regular file")
    stream = archive.extractfile(member)
    if stream is None:
        raise ValueError(f"archive member {name!r}: expected readable file content")
    target = destination.joinpath(*PurePosixPath(name).parts)
    target.parent.mkdir(parents=True, exist_ok=True)
    with stream:
        seen[name] = write_blob(stream, target, member.size, files[name])


def unpack_archive(
    source: BinaryIO, destination: Path, files: Mapping[str, str]
) -> dict[str, str]:
    """Extract and verify every tracked file; example: unpack_archive(tar, output, files)."""
    seen: dict[str, str] = {}
    with tarfile.open(fileobj=source, mode="r:") as archive:
        for member in archive:
            unpack_member(archive, member, destination, files, seen)
    missing = sorted(files.keys() - seen.keys())
    if missing:
        raise ValueError(
            f"archive missing {missing[:10]!r}: expected every tracked file"
        )
    return seen


def materialize_source(
    repository: Path, revision: str, destination: Path
) -> dict[str, object]:
    """Create a verified source tree; example: materialize_source(repo, sha, new_dir)."""
    if os.name != "posix":
        raise ValueError(
            f"platform {os.name!r}: expected a POSIX host for source export"
        )
    repo = repository.resolve(strict=True)
    parent = destination.parent.resolve(strict=True)
    output = parent / destination.name
    if not destination.is_absolute() or output.is_relative_to(repo):
        raise ValueError(
            f"destination {destination}: expected an absolute path outside {repo}"
        )
    files = committed_files(repo, revision)
    output.mkdir()
    try:
        with tempfile.TemporaryFile(dir=parent) as archive:
            subprocess.run(
                ["git", "-C", str(repo), "archive", "--format=tar", revision],
                env=git_environment(),
                stdout=archive,
                check=True,
            )
            archive.flush()
            archive.seek(0)
            seen = unpack_archive(archive, output, files)
        return {"commit": revision, "files": seen, "reusable": False}
    except BaseException:
        shutil.rmtree(output)
        raise


def main() -> None:
    """Export one experimental source tree; run `python3 committed_build_source.py --help`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("repository", type=Path)
    parser.add_argument("revision")
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    print(
        json.dumps(
            materialize_source(args.repository, args.revision, args.destination),
            sort_keys=True,
        )
    )


if __name__ == "__main__":
    main()
