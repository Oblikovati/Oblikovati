# SPDX-License-Identifier: GPL-2.0-only
"""Stage isolated Git databases for locked Cargo dependencies and submodules."""

import re
import shutil
import stat
import tempfile
import subprocess
from pathlib import Path

from git_environment import git_environment
from git_object_inventory import locked_git_tree
from registry_vendor_check import locked_packages


def locked_git_commits(lock: bytes) -> set[str]:
    """Select full Git commits from the lockfile; example: locked_git_commits(lock)."""
    commits: set[str] = set()
    for package in locked_packages(lock).values():
        if not package.source.startswith("git+"):
            continue
        revision = package.source.rpartition("#")[2]
        if not re.fullmatch(r"[0-9a-f]{40}", revision):
            raise ValueError(f"Git revision {revision!r}: expected a full SHA-1 commit")
        commits.add(revision)
    return commits


def cached_git_repositories(cargo_home: Path) -> list[Path]:
    """Enumerate real cached Git databases; example: cached_git_repositories(home)."""
    root = cargo_home / "git" / "db"
    if not root.is_dir():
        return []
    return sorted(path for path in root.iterdir() if path.is_dir() and not path.is_symlink())


def repository_for_commit(repositories: list[Path], revision: str) -> Path:
    """Find one cached database with the commit; example: repository_for_commit(repos, sha)."""
    matches = []
    for repository in repositories:
        status = subprocess.run(
            ["git", f"--git-dir={repository}", "cat-file", "-e", f"{revision}^{{commit}}"],
            env=git_environment(), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
        if status.returncode == 0:
            matches.append(repository)
    if len(matches) != 1:
        raise ValueError(f"Git commit {revision}: expected one cached database, got {matches!r}")
    return matches[0]


def require_regular_objects(objects: Path) -> None:
    """Reject alternate or special object paths; example: require_regular_objects(objects)."""
    if objects.is_symlink() or not objects.is_dir():
        raise ValueError(f"Git object store {objects}: expected a real directory")
    alternate = objects / "info" / "alternates"
    if alternate.exists() or alternate.is_symlink():
        raise ValueError(f"Git alternates {alternate}: expected no external object store")
    for entry in objects.rglob("*"):
        mode = entry.lstat().st_mode
        if not (stat.S_ISDIR(mode) or stat.S_ISREG(mode)):
            raise ValueError(f"Git object path {entry}: expected a regular file or directory")


def isolated_git_source(source: Path, destination: Path) -> Path:
    """Copy objects without caller Git config; example: isolated_git_source(source, destination)."""
    if (source / "shallow").exists() or (source / "shallow").is_symlink():
        raise ValueError(f"Git database {source}: expected a complete non-shallow source")
    objects = source / "objects"
    require_regular_objects(objects)
    subprocess.run(["git", "init", "--bare", "-q", str(destination)], env=git_environment(), check=True)
    shutil.copytree(objects, destination / "objects", dirs_exist_ok=True)
    return destination


def copy_git_commit(source: Path, destination: Path, revision: str) -> None:
    """Fetch one shallow commit into an isolated database; example: copy_git_commit(src, dst, sha)."""
    if not destination.exists():
        subprocess.run(["git", "init", "--bare", "-q", str(destination)], env=git_environment(), check=True)
    subprocess.run([
        "git", f"--git-dir={destination}", "-c", "protocol.file.allow=always", "fetch",
        "-q", "--no-tags", "--depth=1", "--no-write-fetch-head",
        source.resolve().as_uri(), f"{revision}:refs/commit/{revision}",
    ], env=git_environment(), check=True)


def git_submodule_commits(repository: Path, revision: str) -> set[str]:
    """Read every tracked submodule commit; example: git_submodule_commits(repo, sha)."""
    tree = locked_git_tree(repository, revision)
    return {entry.split(":")[-1] for entry in tree.values() if entry.startswith("160000:commit:")}


def verify_staged_database(repository: Path) -> None:
    """Check staged Git object integrity; example: verify_staged_database(repo)."""
    subprocess.run([
        "git", f"--git-dir={repository}", "fsck", "--full", "--strict", "--no-reflogs", "--no-dangling",
    ], env=git_environment(), check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def stage_git_cache(lock: bytes, cargo_home: Path, destination: Path) -> None:
    """Stage locked commits and recursive gitlinks; example: stage_git_cache(lock, home, isolated)."""
    pending = locked_git_commits(lock)
    repositories = cached_git_repositories(cargo_home)
    staged: set[Path] = set()
    seen: set[str] = set()
    with tempfile.TemporaryDirectory(prefix="git-objects-", dir=destination) as directory:
        scratch = Path(directory)
        while pending:
            revision = min(pending)
            pending.remove(revision)
            if revision in seen:
                continue
            seen.add(revision)
            source = repository_for_commit(repositories, revision)
            mirror = scratch / source.name
            if not mirror.exists():
                isolated_git_source(source, mirror)
            target = destination / "git" / "db" / source.name
            target.parent.mkdir(parents=True, exist_ok=True)
            copy_git_commit(mirror, target, revision)
            staged.add(target)
            pending.update(git_submodule_commits(target, revision) - seen)
    for repository in sorted(staged):
        verify_staged_database(repository)
