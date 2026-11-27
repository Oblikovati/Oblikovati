#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Capture a stable local scenario checkout for diagnostic runs."""

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

from git_environment import checkout_git_environment

FEATURE_SUFFIXES = frozenset({".feature", ".md", ".svg", ".json"})
FIXTURE_SUFFIXES = frozenset({".opd", ".json", ".md"})


def selected_files(source: Path, feature: str | None = None, fixtures: tuple[str, ...] = ()) -> list[Path]:
    """List scenario inputs; example: selected_files(Path('/scenarios'))."""
    if not (source / "features").is_dir():
        raise ValueError(f"scenario source {source}: expected a features directory")
    if feature is not None:
        return selected_feature_files(source, feature, fixtures)
    if fixtures:
        raise ValueError(f"fixtures {fixtures!r}: expected a feature-scoped snapshot")
    roots = [(source, {".md"}), (source / "features", FEATURE_SUFFIXES)]
    if (source / "fixtures").is_dir():
        roots.append((source / "fixtures", FIXTURE_SUFFIXES))
    paths: list[Path] = []
    for root, suffixes in roots:
        candidates = root.iterdir() if root == source else root.rglob("*")
        for path in candidates:
            if path.suffix.lower() not in suffixes or path.name == "BLOCKERS.md":
                continue
            if path.is_symlink():
                raise ValueError(f"scenario input {path}: expected a regular file, not a symlink")
            if path.is_file():
                paths.append(path.relative_to(source))
    return sorted(paths)


def selected_feature_files(source: Path, feature: str, fixtures: tuple[str, ...]) -> list[Path]:
    """Select one feature and named native fixtures; example: selected_feature_files(root, 'application/docking-system', ())."""
    feature_path = Path(feature)
    if feature_path.is_absolute() or feature_path.as_posix() != feature or len(feature_path.parts) != 2 or ".." in feature_path.parts:
        raise ValueError(f"feature {feature!r}: expected an area/feature path")
    folder = source / "features" / feature_path
    if folder.is_symlink() or folder.parent.is_symlink() or not folder.is_dir() or not folder.resolve().is_relative_to(source.resolve()):
        raise ValueError(f"feature {feature!r}: expected a directory inside the source")
    paths = []
    for path in folder.rglob("*"):
        if path.suffix.lower() not in FEATURE_SUFFIXES or path.name == "BLOCKERS.md":
            continue
        if path.is_symlink():
            raise ValueError(f"scenario input {path}: expected a regular file, not a symlink")
        if path.is_file():
            paths.append(path.relative_to(source))
    if not any(path.suffix == ".feature" for path in paths):
        raise ValueError(f"feature {feature!r}: expected at least one .feature file")
    for name in fixtures:
        path = Path(name)
        target = source / path
        if path.is_absolute() or path.as_posix() != name or ".." in path.parts or not path.parts or path.parts[0] != "fixtures" or path.suffix.lower() != ".opd":
            raise ValueError(f"fixture {name!r}: expected a relative native .opd fixture")
        if target.is_symlink() or not target.is_file() or not target.resolve().is_relative_to(source.resolve()):
            raise ValueError(f"fixture {name!r}: expected a regular file inside the source")
        paths.append(path)
    return sorted(set(paths))


def inventory(source: Path, feature: str | None = None, fixtures: tuple[str, ...] = ()) -> dict[str, str]:
    """Hash selected source files; example: inventory(Path('/scenarios'))."""
    return {name.as_posix(): hashlib.sha256((source / name).read_bytes()).hexdigest()
            for name in selected_files(source, feature, fixtures)}


def copy_inputs(source: Path, checkout: Path, manifest: dict[str, str]) -> None:
    """Copy the selected inputs; example: copy_inputs(source, checkout, manifest)."""
    for name in manifest:
        target = checkout / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source / name, target)


def commit_snapshot(checkout: Path) -> str:
    """Commit local snapshot bytes; example: commit_snapshot(Path('/tmp/checkout'))."""
    environment = checkout_git_environment() | {
        "GIT_AUTHOR_NAME": "Oblikovati Snapshot", "GIT_AUTHOR_EMAIL": "snapshot@oblikovati.org",
        "GIT_COMMITTER_NAME": "Oblikovati Snapshot", "GIT_COMMITTER_EMAIL": "snapshot@oblikovati.org",
    }
    for command in (["init", "-q", "-b", "snapshot"], ["add", "--all"],
                    ["-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "Capture local scenario inputs"]):
        subprocess.run(["git", "-C", str(checkout), *command], env=environment, check=True)
    return subprocess.check_output(["git", "-C", str(checkout), "rev-parse", "HEAD"],
                                   env=environment, text=True).strip()


def create_snapshot(source: Path, destination: Path, feature: str | None = None, fixtures: tuple[str, ...] = ()) -> dict[str, object]:
    """Capture stable local input bytes; example: create_snapshot(source, destination)."""
    source = source.resolve(strict=True)
    destination = destination.absolute()
    if destination.exists() or destination.is_symlink() or destination.is_relative_to(source):
        raise ValueError(f"snapshot destination {destination}: expected a new path outside {source}")
    before = inventory(source, feature, fixtures)
    with tempfile.TemporaryDirectory(prefix="scenario-snapshot-", dir=destination.parent) as name:
        staging = Path(name)
        checkout = staging / "checkout"
        checkout.mkdir()
        copy_inputs(source, checkout, before)
        copied, after = inventory(checkout, feature, fixtures), inventory(source, feature, fixtures)
        if copied != before or after != before:
            changed = sorted(name for name in before.keys() | copied.keys() | after.keys()
                             if before.get(name) != copied.get(name) or before.get(name) != after.get(name))
            raise ValueError(f"scenario source {source}: expected unchanged selected inputs; changed {changed[:10]!r}")
        revision = commit_snapshot(checkout)
        result: dict[str, object] = {"revision": revision, "files": before,
                                    "provenance": "local unversioned source; not an upstream commit", "feature_scope": feature,
                                    "excluded": "reference images, BLOCKERS.md and non-native CAD fixtures"}
        (staging / "snapshot.json").write_text(json.dumps(result, indent=2) + "\n")
        staging.rename(destination)
    return result


def main() -> None:
    """Capture a snapshot; example: local_scenario_snapshot.py SOURCE OUTPUT."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--feature", help="capture one area/feature folder")
    parser.add_argument("--fixture", action="append", default=[], help="native fixture relative to the source")
    args = parser.parse_args()
    if args.fixture and not args.feature:
        parser.error("--fixture requires --feature")
    result = create_snapshot(args.source, args.output, args.feature, tuple(args.fixture))
    print(json.dumps({"revision": result["revision"], "files": len(result["files"])}))


if __name__ == "__main__":
    main()
