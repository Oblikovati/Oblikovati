# SPDX-License-Identifier: GPL-2.0-only
"""Diagnose Cargo package selection against one locked Git source."""

import argparse
import hashlib
import json
import re
import subprocess
import tempfile
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from pathlib import Path, PurePosixPath

from committed_build_source import materialize_source
from git_manifest_declarations import check_git_manifest_declarations
from git_manifest_library import check_git_manifest_library
from git_manifest_metadata import check_git_manifest_metadata
from git_manifest_path_metadata import check_git_manifest_path_metadata
from git_manifest_targets import check_git_manifest_targets
from git_object_inventory import GitBlobReader, git_blob_bytes, locked_git_tree
from git_vendor_check import check_package, package_entries, package_roots
from registry_vendor_check import read_observed_manifest
from runtime_snapshot import snapshot_runtime

SANDBOX_FLAGS = (
    "--unshare-all",
    "--unshare-user",
    "--disable-userns",
    "--die-with-parent",
    "--clearenv",
)
SANDBOX_FILES = (
    "--symlink",
    "usr/bin",
    "/bin",
    "--symlink",
    "usr/lib",
    "/lib",
    "--symlink",
    "usr/lib64",
    "/lib64",
    "--tmpfs",
    "/tmp",
    "--dev",
    "/dev",
    "--proc",
    "/proc",
    "--dir",
    "/tmp/cargo-home",
    "--symlink",
    "/cache/registry",
    "/tmp/cargo-home/registry",
    "--symlink",
    "/cache/git",
    "/tmp/cargo-home/git",
)
SANDBOX_ENVIRONMENT = (
    ("PATH", "/toolchain/bin:/usr/bin:/bin"),
    ("HOME", "/tmp"),
    ("CARGO_HOME", "/tmp/cargo-home"),
    ("CARGO_NET_OFFLINE", "true"),
    ("CARGO_TARGET_DIR", "/tmp/target"),
    ("GIT_CONFIG_NOSYSTEM", "1"),
    ("GIT_CONFIG_GLOBAL", "/dev/null"),
)
CARGO_PACKAGE_LIST = (
    "--chdir",
    "/source",
    "/toolchain/bin/cargo",
    "package",
    "--list",
    "--allow-dirty",
    "--offline",
    "-p",
)


@dataclass(frozen=True)
class SandboxPaths:
    """Locate isolated Cargo inputs; example: SandboxPaths(bwrap, toolchain, cache)."""

    bwrap: Path
    toolchain: Path
    cache: Path


@dataclass(frozen=True)
class SelectionRequest:
    """Identify one locked package; example: SelectionRequest(repo, sha, name, version, vendor, paths)."""

    repository: Path
    revision: str
    package: str
    version: str
    vendor: Path
    sandbox: SandboxPaths


def package_list_command(
    sandbox: SandboxPaths, source: Path, package: str
) -> list[str]:
    """Build an offline package-list sandbox; example: package_list_command(paths, source, 'crate')."""
    command = [str(sandbox.bwrap), *SANDBOX_FLAGS]
    mounts = (
        (Path("/usr"), "/usr"),
        (sandbox.toolchain, "/toolchain"),
        (source, "/source"),
        (sandbox.cache / "registry", "/cache/registry"),
        (sandbox.cache / "git", "/cache/git"),
    )
    for host, guest in mounts:
        command += ["--ro-bind", str(host), guest]
    command.extend(SANDBOX_FILES)
    for name, value in SANDBOX_ENVIRONMENT:
        command += ["--setenv", name, value]
    return command + [*CARGO_PACKAGE_LIST, package]


def validate_package_path(name: str) -> None:
    """Reject paths that cannot name a portable tracked file; example: validate_package_path('src/lib.rs')."""
    path = PurePosixPath(name)
    if (
        path.is_absolute()
        or path.as_posix() != name
        or ".." in path.parts
        or any(c in name for c in ("\\", "\x00", "\r"))
    ):
        raise ValueError(f"package path {name!r}: expected a normalized relative file")


def parse_package_listing(listing: str) -> list[str]:
    """Read unique Cargo output paths; example: parse_package_listing('Cargo.toml\\n')."""
    if not listing or not listing.endswith("\n"):
        raise ValueError(
            f"Cargo package list {listing!r}: expected newline-terminated paths"
        )
    names = listing[:-1].split("\n")
    if len(names) != len(set(names)):
        raise ValueError(f"Cargo package list {names!r}: expected unique file paths")
    for name in names:
        validate_package_path(name)
    return names


def selected_entries(listing: str, entries: Mapping[str, str]) -> dict[str, str]:
    """Match Cargo's listing to tracked files; example: selected_entries('Cargo.toml\\n', entries)."""
    names = parse_package_listing(listing)
    unknown = set(names) - entries.keys() - {"Cargo.toml.orig"}
    if unknown:
        raise ValueError(
            f"package files {sorted(unknown)[:10]!r}: expected tracked files or Cargo.toml.orig"
        )
    selected = {name: entries[name] for name in names if name in entries}
    if "Cargo.toml" not in selected or "src/lib.rs" not in selected:
        raise ValueError(
            "Cargo package list: expected tracked Cargo.toml and src/lib.rs"
        )
    return selected


def check_manifests(
    original: bytes,
    generated: bytes,
    entries: Mapping[str, str],
    repository: Path,
    tree: Mapping[str, str],
    root: str,
    label: str,
) -> None:
    """Compare supported manifest fields; example: check_manifests(original, generated, entries, repo, tree, root, label)."""
    check_git_manifest_declarations(original, generated, entries, label)
    check_git_manifest_library(original, generated, entries, label)
    check_git_manifest_targets(
        original, generated, entries, repository, tree, root, label
    )
    check_git_manifest_metadata(original, generated, repository, tree, root, label)
    check_git_manifest_path_metadata(original, generated, repository, tree, root, label)


def execute_package_list(command: list[str]) -> str:
    """List files with the isolated Cargo process; example: execute_package_list(command)."""
    return subprocess.run(
        command, capture_output=True, text=True, check=True, env={}
    ).stdout


def list_locked_source(
    request: SelectionRequest,
    runner: Callable[[list[str]], str],
    vendor_before: Mapping[str, object],
) -> str:
    """List a committed source without accepting changes; example: list_locked_source(request, run, before)."""
    with tempfile.TemporaryDirectory(prefix="obk-cargo-selection-") as folder:
        source = Path(folder) / "source"
        materialize_source(request.repository, request.revision, source)
        source_before = snapshot_runtime(source)
        listing = runner(package_list_command(request.sandbox, source, request.package))
        if (
            snapshot_runtime(source) != source_before
            or snapshot_runtime(request.vendor) != vendor_before
        ):
            raise ValueError(
                "package inputs: expected unchanged source and vendor trees"
            )
        return listing


def validate_identity(request: SelectionRequest) -> None:
    """Reject invalid package identifiers; example: validate_identity(request)."""
    if not re.fullmatch(r"[a-zA-Z0-9_-]+", request.package) or not re.fullmatch(
        r"[a-zA-Z0-9.+_-]+", request.version
    ):
        raise ValueError(
            f"package {request.package!r} version {request.version!r}: expected Cargo package identifiers"
        )


def verify_selection(
    request: SelectionRequest,
    tree: Mapping[str, str],
    root: str,
    selected: Mapping[str, str],
    vendor_before: Mapping[str, object],
) -> int:
    """Compare the selected locked files and manifest fields; example: verify_selection(request, tree, root, files, before)."""
    label = f"{request.package}-{request.version}"
    original = git_blob_bytes(request.repository, selected["Cargo.toml"].split(":")[-1])
    generated = read_observed_manifest(request.vendor, label, vendor_before)
    check_manifests(
        original, generated, selected, request.repository, tree, root, label
    )
    with GitBlobReader(request.repository) as reader:
        files = check_package(label, selected, vendor_before, reader, {})
    if snapshot_runtime(request.vendor) != vendor_before:
        raise ValueError("vendor tree: expected unchanged files during the check")
    return files


def diagnostic_result(
    request: SelectionRequest, listing: str, files: int
) -> dict[str, object]:
    """Report a non-reusable diagnostic; example: diagnostic_result(request, listing, 3)."""
    return {
        "package": f"{request.package}-{request.version}",
        "source_commit": request.revision,
        "selected_files_checked": files,
        "selection_sha256": hashlib.sha256(listing.encode()).hexdigest(),
        "manifest_complete": False,
        "complete": False,
        "reusable": False,
        "limitations": "Cargo, host system, cache, kernel and process observations are not independently authenticated.",
    }


def diagnose(
    request: SelectionRequest, runner: Callable[[list[str]], str] = execute_package_list
) -> dict[str, object]:
    """Check one Cargo-selected vendor package; example: diagnose(request)."""
    validate_identity(request)
    tree = locked_git_tree(request.repository, request.revision)
    root = package_roots(
        request.repository, tree, {(request.package, request.version)}
    )[(request.package, request.version)]
    entries = package_entries(tree, root)
    before = snapshot_runtime(request.vendor)
    listing = list_locked_source(request, runner, before)
    selected = selected_entries(listing, entries)
    files = verify_selection(request, tree, root, selected, before)
    return diagnostic_result(request, listing, files)


def main() -> None:
    """Run the package-selection diagnostic; example: git_package_selection.py --help."""
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("repository", "vendor", "toolchain", "cache", "bwrap"):
        parser.add_argument(f"--{name}", required=True, type=Path)
    for name in ("revision", "package", "version"):
        parser.add_argument(f"--{name}", required=True)
    args = parser.parse_args()
    sandbox = SandboxPaths(args.bwrap, args.toolchain, args.cache)
    request = SelectionRequest(
        args.repository, args.revision, args.package, args.version, args.vendor, sandbox
    )
    print(json.dumps(diagnose(request), indent=2))


if __name__ == "__main__":
    main()
