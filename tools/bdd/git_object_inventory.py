# SPDX-License-Identifier: GPL-2.0-only
"""Read locked Git tree entries and hash their actual blob contents."""

import hashlib
import re
import subprocess
from pathlib import Path
from typing import BinaryIO

from committed_build_source import git_environment, validate_path
from scenario_dependencies import parse_tree_entry


def locked_git_tree(repository: Path, revision: str) -> dict[str, str]:
    """Read a full locked Git tree without worktree files; example: locked_git_tree(repo, sha)."""
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError(f"Git revision {revision!r}: expected a full SHA-1")
    output = subprocess.check_output(
        ["git", "-C", str(repository), "ls-tree", "-rz", "--full-tree", revision],
        env=git_environment(),
    )
    result: dict[str, str] = {}
    for entry in output.split(b"\0"):
        if entry:
            path, identity = parse_tree_entry(entry)
            validate_path(path)
            result[path] = identity
    return result


def sha1_git_blob(content: bytes) -> str:
    """Compute a Git blob identity; example: sha1_git_blob(b'content')."""
    return hashlib.sha1(f"blob {len(content)}\0".encode() + content).hexdigest()


def git_blob_bytes(repository: Path, identity: str) -> bytes:
    """Read and validate a small Git blob; example: git_blob_bytes(repo, object_id)."""
    if not re.fullmatch(r"[0-9a-f]{40}", identity):
        raise ValueError(f"Git blob {identity!r}: expected a full SHA-1")
    content = subprocess.check_output(
        ["git", "-C", str(repository), "cat-file", "blob", identity],
        env=git_environment(),
    )
    if sha1_git_blob(content) != identity:
        raise ValueError(f"Git blob {identity!r}: expected matching object bytes")
    return content


def hash_blob_stream(stream: BinaryIO, identity: str, size: int) -> str:
    """Hash exactly one Git blob from the batch stream; example: hash_blob_stream(stream, id, size)."""
    git_hash = hashlib.sha1(f"blob {size}\0".encode())
    content_hash = hashlib.sha256()
    left = size
    while left:
        chunk = stream.read(min(left, 1024 * 1024))
        if not chunk:
            raise ValueError(f"Git blob {identity!r}: expected {left} more bytes")
        git_hash.update(chunk)
        content_hash.update(chunk)
        left -= len(chunk)
    if git_hash.hexdigest() != identity or stream.read(1) != b"\n":
        raise ValueError(
            f"Git blob {identity!r}: expected matching bytes and terminator"
        )
    return content_hash.hexdigest()


class GitBlobReader:
    """Hash many blobs through one Git process; example: with GitBlobReader(repo) as reader: reader.digest(oid)."""

    def __init__(self, repository: Path) -> None:
        self.process = subprocess.Popen(
            ["git", "-C", str(repository), "cat-file", "--batch"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            env=git_environment(),
        )

    def __enter__(self) -> "GitBlobReader":
        """Return the active reader; example: with GitBlobReader(repo) as reader: pass."""
        return self

    def digest(self, identity: str) -> str:
        """Hash one locked blob; example: reader.digest(object_id)."""
        if not re.fullmatch(r"[0-9a-f]{40}", identity):
            raise ValueError(f"Git blob {identity!r}: expected a full SHA-1")
        if self.process.stdin is None or self.process.stdout is None:
            raise ValueError("Git batch streams: expected open input and output")
        self.process.stdin.write((identity + "\n").encode())
        self.process.stdin.flush()
        header = self.process.stdout.readline().decode("ascii").strip().split()
        if (
            len(header) != 3
            or header[0] != identity
            or header[1] != "blob"
            or not header[2].isdigit()
        ):
            raise ValueError(
                f"Git batch header {header!r}: expected the requested blob and byte size"
            )
        return hash_blob_stream(self.process.stdout, identity, int(header[2]))

    def __exit__(self, kind: object, value: object, traceback: object) -> None:
        """Close the Git process, including failed reads; example: reader.__exit__(None, None, None)."""
        if kind is not None:
            self.process.kill()
        if self.process.stdin is not None:
            self.process.stdin.close()
        if self.process.stdout is not None:
            self.process.stdout.close()
        code = self.process.wait()
        if kind is None and code != 0:
            raise ValueError(f"Git batch exit {code}: expected success")
