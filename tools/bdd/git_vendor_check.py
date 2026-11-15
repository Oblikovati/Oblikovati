# SPDX-License-Identifier: GPL-2.0-only
"""Compare Git vendor files with locked trees; generated manifests remain unverified."""

import argparse
import json
import re
import tomllib
from pathlib import Path, PurePosixPath
from typing import Mapping

from git_object_inventory import GitBlobReader, git_blob_bytes, locked_git_tree
from git_manifest_declarations import check_git_manifest_declarations
from git_manifest_library import check_git_manifest_library
from git_manifest_targets import check_git_manifest_targets
from registry_vendor_check import (
    ignored_vendor_path,
    read_observed_manifest,
    locked_packages,
    vendor_packages,
    verify_package_namespace,
)
from runtime_snapshot import snapshot_runtime


def package_roots(
    repository: Path, tree: Mapping[str, str], wanted: set[tuple[str, str]]
) -> dict[tuple[str, str], str]:
    """Find locked packages in tracked manifests; example: package_roots(repo, tree, wanted)."""
    found: dict[tuple[str, str], str] = {}
    for path, entry in tree.items():
        if PurePosixPath(path).name != "Cargo.toml" or not entry.startswith(
            "100644:blob:"
        ):
            continue
        document = tomllib.loads(
            git_blob_bytes(repository, entry.split(":")[-1]).decode()
        )
        package = document.get("package", {})
        key = (
            (package.get("name"), package.get("version"))
            if isinstance(package, dict)
            else (None, None)
        )
        if all(isinstance(value, str) for value in key) and key in wanted:
            if key in found:
                raise ValueError(f"Git package {key!r}: expected one tracked manifest")
            found[key] = str(PurePosixPath(path).parent)
    if set(found) != wanted:
        raise ValueError(
            f"Git packages {sorted(wanted - set(found))!r}: expected explicit tracked names and versions"
        )
    return found


def package_entries(tree: Mapping[str, str], root: str) -> dict[str, str]:
    """Select tracked crate files and refuse unsupported objects; example: package_entries(tree, root)."""
    prefix = "" if root == "." else root + "/"
    entries: dict[str, str] = {}
    for path, identity in tree.items():
        if not path.startswith(prefix):
            continue
        relative = path[len(prefix) :]
        if ignored_vendor_path(PurePosixPath(relative)):
            continue
        if not identity.startswith(("100644:blob:", "100755:blob:")):
            raise ValueError(f"Git entry {path!r}: expected a regular tracked blob")
        entries[relative] = identity
    if "Cargo.toml" not in entries:
        raise ValueError(f"Git package {root!r}: expected a tracked Cargo.toml")
    return entries


def check_package(
    folder: str,
    entries: Mapping[str, str],
    inventory: Mapping[str, object],
    reader: GitBlobReader,
    cache: dict[str, str],
) -> int:
    """Check all tracked blobs except the generated manifest; example: check_package(folder, entries, inventory, reader, cache)."""
    manifest = inventory.get(folder + "/Cargo.toml")
    if not isinstance(manifest, dict) or not isinstance(manifest.get("sha256"), str):
        raise ValueError(f"vendor {folder!r}: expected an observed manifest")
    files: dict[str, tuple[str, int]] = {"Cargo.toml": (manifest["sha256"], 0)}
    for path, entry in entries.items():
        if path == "Cargo.toml":
            continue
        mode, _, identity = entry.split(":")
        if identity not in cache:
            cache[identity] = reader.digest(identity)
        files[path] = (cache[identity], 0o111 if mode == "100755" else 0)
    verify_package_namespace(folder, files, None, inventory)
    return len(files) - 1


def verify_git_vendor(
    lock: bytes,
    vendor: Path,
    inventory: Mapping[str, object],
    repositories: Mapping[str, Path],
) -> dict[str, object]:
    """Compare Git crate files and report the unverified manifests; example: verify_git_vendor(lock, vendor, inventory, repositories)."""
    packages = locked_packages(lock)
    folders = vendor_packages(vendor, inventory, packages)
    git_packages = {
        key: item for key, item in packages.items() if item.source.startswith("git+")
    }
    sources = {item.source for item in git_packages.values()}
    if set(repositories) != sources:
        raise ValueError(
            f"Git repositories {sorted(repositories)!r}: expected exactly {sorted(sources)!r}"
        )
    checked = 0
    manifests: list[str] = []
    for source in sorted(sources):
        revision = source.rpartition("#")[2]
        if not re.fullmatch(r"[0-9a-f]{40}", revision):
            raise ValueError(f"Git source {source!r}: expected a full locked SHA-1")
        repository = repositories[source]
        tree = locked_git_tree(repository, revision)
        wanted = {key for key, item in git_packages.items() if item.source == source}
        roots = package_roots(repository, tree, wanted)
        with GitBlobReader(repository) as reader:
            cache: dict[str, str] = {}
            for key in sorted(wanted):
                entries = package_entries(tree, roots[key])
                original = git_blob_bytes(repository, entries["Cargo.toml"].split(":")[-1])
                generated = read_observed_manifest(vendor, folders[key], inventory)
                check_git_manifest_declarations(original, generated, entries, folders[key])
                check_git_manifest_library(original, generated, entries, folders[key])
                check_git_manifest_targets(original, generated, entries, repository, tree, roots[key], folders[key])
                checked += check_package(folders[key], entries, inventory, reader, cache)
                manifests.append(f"{key[0]} {key[1]}")
    return {
        "git_packages_partially_checked": len(manifests),
        "git_files_checked": checked,
        "git_manifest_declarations_checked": len(manifests),
        "git_manifest_check_scope": [
            "dependencies", "dev-dependencies", "build-dependencies",
            "target", "features", "no-build-script", "conventional-library", "target-autodiscovery", "package-links", "bin-example-test-bench-targets",
        ],
        "git_manifests_unverified": manifests,
        "complete": False,
        "reusable": False,
    }


def main() -> None:
    """Run the partial Git check; example: python3 git_vendor_check.py Cargo.lock vendor --git-repo SOURCE REPO."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("lock", type=Path)
    parser.add_argument("vendor", type=Path)
    parser.add_argument(
        "--git-repo",
        nargs=2,
        action="append",
        required=True,
        metavar=("SOURCE", "REPO"),
    )
    args = parser.parse_args()
    repositories = {source: Path(path) for source, path in args.git_repo}
    if len(repositories) != len(args.git_repo):
        parser.error("Git source mapping: expected each source exactly once")
    print(
        json.dumps(
            verify_git_vendor(
                args.lock.read_bytes(),
                args.vendor,
                snapshot_runtime(args.vendor),
                repositories,
            ),
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
