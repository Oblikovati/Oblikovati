# SPDX-License-Identifier: GPL-2.0-only
"""Compare vendored registry packages with their locked crate archives."""

import argparse
import hashlib
import json
import os
import re
import stat
import tarfile
import tomllib
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import BinaryIO, Mapping
from urllib.parse import parse_qsl, urlsplit, urlunsplit

from committed_source_check import expected_directories
from runtime_snapshot import (
    file_digest,
    metadata,
    open_flags,
    require_unchanged,
    snapshot_runtime,
)

REGISTRY = "registry+https://github.com/rust-lang/crates.io-index"


@dataclass(frozen=True)
class LockedPackage:
    """One external lock entry; example: LockedPackage('name', '1.0.0', source, checksum)."""

    name: str
    version: str
    source: str
    checksum: str | None


def locked_packages(lock: bytes) -> dict[tuple[str, str], LockedPackage]:
    """Read unique external package identities; example: locked_packages(lock_bytes)."""
    document = tomllib.loads(lock.decode("utf-8"))
    packages: dict[tuple[str, str], LockedPackage] = {}
    entries = document.get("package")
    if not isinstance(entries, list):
        raise ValueError(f"lock packages {entries!r}: expected a list")
    for item in entries:
        if not isinstance(item, dict):
            raise ValueError(f"lock package {item!r}: expected an object")
        source = item.get("source")
        if source is None:
            continue
        package = parse_locked_package(item)
        key = (package.name, package.version)
        if key in packages:
            raise ValueError(f"locked package {key!r}: expected one external source")
        packages[key] = package
    return packages


def parse_locked_package(item: Mapping[str, object]) -> LockedPackage:
    """Validate a supported locked package; example: parse_locked_package(entry)."""
    name, version, source, checksum = (
        item.get(key) for key in ("name", "version", "source", "checksum")
    )
    if not isinstance(name, str) or not re.fullmatch(r"[A-Za-z0-9_-]+", name):
        raise ValueError(f"package name {name!r}: expected a Cargo package name")
    if not isinstance(version, str) or not re.fullmatch(r"[0-9A-Za-z.+-]+", version):
        raise ValueError(
            f"package version {version!r}: expected a Cargo package version"
        )
    if source == REGISTRY:
        if not isinstance(checksum, str) or not re.fullmatch(r"[0-9a-f]{64}", checksum):
            raise ValueError(
                f"package {name!r} checksum {checksum!r}: expected SHA-256"
            )
    elif not isinstance(source, str) or not source.startswith("git+"):
        raise ValueError(
            f"package {name!r} source {source!r}: expected crates.io or Git"
        )
    return LockedPackage(
        name, version, source, checksum if isinstance(checksum, str) else None
    )


def git_source_table(source: str) -> tuple[str, dict[str, str]]:
    """Derive a Cargo Git source replacement; example: git_source_table(source)."""
    key, separator, commit = source.partition("#")
    if not separator or not re.fullmatch(r"[0-9a-f]{7,64}", commit):
        raise ValueError(f"Git source {source!r}: expected a locked commit")
    parsed = urlsplit(key.removeprefix("git+"))
    pairs = parse_qsl(parsed.query, keep_blank_values=True, strict_parsing=True)
    if len(pairs) > 1 or any(
        name not in ("rev", "tag", "branch") or not value for name, value in pairs
    ):
        raise ValueError(
            f"Git source {source!r}: expected at most one revision selector"
        )
    url = urlunsplit((parsed.scheme, parsed.netloc, parsed.path, "", ""))
    if not parsed.scheme or not parsed.netloc:
        raise ValueError(f"Git source {source!r}: expected an absolute Git URL")
    return key, {"git": url, "replace-with": "vendored-sources", **dict(pairs)}


def verify_vendor_config(lock: bytes, content: bytes) -> None:
    """Require locked sources to resolve to /vendor; example: verify_vendor_config(lock, content)."""
    packages = locked_packages(lock)
    expected = {"vendored-sources": {"directory": "/vendor"}}
    if any(package.source == REGISTRY for package in packages.values()):
        expected["crates-io"] = {"replace-with": "vendored-sources"}
    for package in packages.values():
        if package.source.startswith("git+"):
            name, table = git_source_table(package.source)
            expected[name] = table
    config = tomllib.loads(content.decode("utf-8"))
    if config != {"source": expected}:
        raise ValueError(
            f"vendor configuration {config!r}: expected only locked sources replaced by /vendor"
        )


def read_observed_manifest(
    root: Path, folder: str, inventory: Mapping[str, object]
) -> bytes:
    """Read a manifest only if it matches the observed file; example: read_observed_manifest(root, folder, inventory)."""
    root_fd = os.open(root, open_flags(True))
    try:
        folder_fd = os.open(folder, open_flags(True), dir_fd=root_fd)
        try:
            return read_observed_file(
                folder_fd, "Cargo.toml", inventory[f"{folder}/Cargo.toml"]
            )
        finally:
            os.close(folder_fd)
    finally:
        os.close(root_fd)


def read_observed_file(parent: int, name: str, observed: object) -> bytes:
    """Read one unchanged no-follow file; example: read_observed_file(parent, name, observed)."""
    if not isinstance(observed, dict) or observed.get("kind") != "file":
        raise ValueError(
            f"observed file {name!r} {observed!r}: expected a regular file"
        )
    fd = os.open(name, open_flags(False), dir_fd=parent)
    try:
        before = os.fstat(fd)
        expected = {k: v for k, v in observed.items() if k != "sha256"}
        if metadata(before, "file") != expected:
            raise ValueError(f"observed file {name!r}: expected the recorded metadata")
        chunks: list[bytes] = []
        while chunk := os.read(fd, 1024 * 1024):
            chunks.append(chunk)
        content = b"".join(chunks)
        require_unchanged(before, os.fstat(fd), name)
        if hashlib.sha256(content).hexdigest() != observed.get("sha256"):
            raise ValueError(f"observed file {name!r}: expected the recorded SHA-256")
        return content
    finally:
        os.close(fd)


def vendor_packages(
    root: Path,
    inventory: Mapping[str, object],
    packages: Mapping[tuple[str, str], LockedPackage],
) -> dict[tuple[str, str], str]:
    """Map every vendor directory to one locked identity; example: vendor_packages(root, inventory, packages)."""
    entry = inventory.get(".")
    if not isinstance(entry, dict) or not isinstance(entry.get("children"), list):
        raise ValueError("vendor root: expected an observed directory with children")
    found: dict[tuple[str, str], str] = {}
    for folder in entry["children"]:
        if (
            not isinstance(folder, str)
            or folder in (".", "..")
            or "/" in folder
            or "\\" in folder
        ):
            raise ValueError(f"vendor folder {folder!r}: expected one directory name")
        key = vendor_identity(root, folder, inventory)
        if key not in packages or key in found:
            raise ValueError(
                f"vendor package {key!r}: expected one unique locked package"
            )
        found[key] = folder
    if set(found) != set(packages):
        raise ValueError(
            f"vendor packages {sorted(set(packages) - set(found))[:10]!r}: expected every locked external package"
        )
    return found


def vendor_identity(
    root: Path, folder: str, inventory: Mapping[str, object]
) -> tuple[str, str]:
    """Read a vendored package identity; example: vendor_identity(root, folder, inventory)."""
    entry = inventory.get(folder)
    if not isinstance(entry, dict) or entry.get("kind") != "directory":
        raise ValueError(f"vendor entry {folder!r}: expected a package directory")
    try:
        manifest = tomllib.loads(
            read_observed_manifest(root, folder, inventory).decode("utf-8")
        )
        package = manifest["package"]
        name, version = package["name"], package["version"]
    except (KeyError, TypeError, UnicodeError) as error:
        raise ValueError(
            f"vendor package {folder!r}: expected a Cargo package manifest"
        ) from error
    if not isinstance(name, str) or not isinstance(version, str):
        raise ValueError(f"vendor package {folder!r}: expected string name and version")
    return name, version


def ignored_vendor_path(path: PurePosixPath) -> bool:
    """Match Cargo's VCS metadata exclusions; example: ignored_vendor_path(PurePosixPath('.gitignore'))."""
    # Cargo vendor drops these VCS-control paths (#3978); all other files remain required.
    return (
        ".git" in path.parts
        or path.name in (".gitignore", ".gitattributes")
        or path.as_posix() == ".cargo-ok"
    )


def archive_member_name(name: str, package: LockedPackage) -> str:
    """Reject unsafe or wrongly rooted archive paths; example: archive_member_name(name, package)."""
    path = PurePosixPath(name)
    root = f"{package.name}-{package.version}"
    if (
        path.is_absolute()
        or path.as_posix() != name
        or "\\" in name
        or ".." in path.parts
        or len(path.parts) < 2
        or path.parts[0] != root
    ):
        raise ValueError(
            f"archive path {name!r}: expected a normalized path inside {root!r}"
        )
    return PurePosixPath(*path.parts[1:]).as_posix()


def digest_stream(stream: BinaryIO) -> str:
    """Hash a stream without retaining the full file; example: digest_stream(stream)."""
    digest = hashlib.sha256()
    while chunk := stream.read(1024 * 1024):
        digest.update(chunk)
    return digest.hexdigest()


def archive_entries(
    archive: tarfile.TarFile, package: LockedPackage
) -> dict[str, tuple[str, int]]:
    """Read allowed package file hashes and executable bits; example: archive_entries(tar, package)."""
    files: dict[str, tuple[str, int]] = {}
    for member in archive:
        name = archive_member_name(
            member.name.rstrip("/") if member.isdir() else member.name, package
        )
        if member.isdir() or ignored_vendor_path(PurePosixPath(name)):
            continue
        if not member.isfile() or name in files or name == ".cargo-checksum.json":
            raise ValueError(
                f"archive member {name!r}: expected a unique regular package file"
            )
        stream = archive.extractfile(member)
        if stream is None:
            raise ValueError(f"archive member {name!r}: expected readable content")
        with stream:
            files[name] = (digest_stream(stream), member.mode & 0o111)
    return files


def locked_archive(
    archives: Path, package: LockedPackage
) -> dict[str, tuple[str, int]]:
    """Verify a crate archive against the lockfile; example: locked_archive(cache, package)."""
    parent = os.open(archives, open_flags(True))
    name = f"{package.name}-{package.version}.crate"
    try:
        fd = os.open(name, open_flags(False), dir_fd=parent)
        try:
            before = os.fstat(fd)
            if not stat.S_ISREG(before.st_mode) or file_digest(fd) != package.checksum:
                raise ValueError(
                    f"archive {name!r}: expected the locked SHA-256 {package.checksum!r}"
                )
            os.lseek(fd, 0, os.SEEK_SET)
            with os.fdopen(os.dup(fd), "rb") as stream, tarfile.open(
                fileobj=stream, mode="r:gz"
            ) as archive:
                files = archive_entries(archive, package)
            require_unchanged(before, os.fstat(fd), name)
            return files
        finally:
            os.close(fd)
    finally:
        os.close(parent)


def checksum_metadata(files: Mapping[str, tuple[str, int]], checksum: str) -> str:
    """Derive Cargo's canonical metadata hash; example: checksum_metadata(files, checksum)."""
    # Cargo 1.99 writes this exact warning; a different encoding is refused, not trusted.
    record = {
        "$comment": "This file only protects against accidental modifications. It is not a security mechanism and does not protect against malicious changes.",
        "files": {name: value[0] for name, value in files.items()},
        "package": checksum,
    }
    return hashlib.sha256(
        json.dumps(
            record, sort_keys=True, separators=(",", ":"), ensure_ascii=False
        ).encode()
    ).hexdigest()


def verify_registry_package(
    folder: str,
    package: LockedPackage,
    files: Mapping[str, tuple[str, int]],
    inventory: Mapping[str, object],
) -> None:
    """Check the complete vendored package namespace; example: verify_registry_package(folder, package, files, inventory)."""
    expected = dict(files)
    expected[".cargo-checksum.json"] = (
        checksum_metadata(files, str(package.checksum)),
        0,
    )
    directories = expected_directories(expected)
    actual = {
        "." if key == folder else key[len(folder) + 1 :]: value
        for key, value in inventory.items()
        if key == folder or key.startswith(folder + "/")
    }
    paths = set(expected) | directories
    if set(actual) != paths:
        raise ValueError(
            f"vendor {folder!r} paths {sorted(set(actual) ^ paths)[:10]!r}: expected archive files and directories"
        )
    for name in directories:
        if (
            not isinstance(actual[name], dict)
            or actual[name].get("kind") != "directory"
        ):
            raise ValueError(
                f"vendor {folder!r} directory {name!r}: expected a directory"
            )
    for name, value in expected.items():
        verify_vendor_file(folder, name, actual[name], value)


def verify_vendor_file(
    folder: str, name: str, observed: object, expected: tuple[str, int]
) -> None:
    """Compare one file with its archive-derived identity; example: verify_vendor_file(folder, name, observed, expected)."""
    if (
        not isinstance(observed, dict)
        or observed.get("kind") != "file"
        or observed.get("links") != 1
    ):
        raise ValueError(
            f"vendor {folder!r} file {name!r}: expected a single-link regular file"
        )
    mode = observed.get("mode")
    if (
        observed.get("sha256") != expected[0]
        or not isinstance(mode, int)
        or mode & 0o111 != expected[1]
    ):
        raise ValueError(
            f"vendor {folder!r} file {name!r}: expected archive content and executable bits"
        )


def verify_registry_vendor(
    lock: bytes, root: Path, archives: Path, inventory: Mapping[str, object]
) -> dict[str, object]:
    """Check registry packages and report unverified Git sources; example: verify_registry_vendor(lock, vendor, archives, snapshot)."""
    packages = locked_packages(lock)
    folders = vendor_packages(root, inventory, packages)
    registry = [p for p in packages.values() if p.source == REGISTRY]
    for package in registry:
        verify_registry_package(
            folders[(package.name, package.version)],
            package,
            locked_archive(archives, package),
            inventory,
        )
    git = sorted(
        f"{p.name} {p.version}"
        for p in packages.values()
        if p.source.startswith("git+")
    )
    return {
        "registry_packages_checked": len(registry),
        "git_packages_unverified": git,
        "complete": False,
    }


def main() -> None:
    """Check a vendor tree without claiming full provenance; run `python3 registry_vendor_check.py --help`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("lock", type=Path)
    parser.add_argument("vendor", type=Path)
    parser.add_argument("archives", type=Path)
    args = parser.parse_args()
    print(
        json.dumps(
            verify_registry_vendor(
                args.lock.read_bytes(),
                args.vendor,
                args.archives,
                snapshot_runtime(args.vendor),
            ),
            sort_keys=True,
        )
    )


if __name__ == "__main__":
    main()
