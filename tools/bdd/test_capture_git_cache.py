# SPDX-License-Identifier: GPL-2.0-only
"""Test isolated Git commit and submodule staging."""

import subprocess
import tempfile
import unittest
from pathlib import Path

from capture_git_cache import isolated_git_source, stage_git_cache


class GitCacheFixture:
    """Create a parent commit and a child gitlink in separate cached databases."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.home = root / "cargo"
        (self.home / "git/db").mkdir(parents=True)
        self.child, self.child_commit = self.repository("child")
        self.parent, self.parent_commit = self.repository("parent", self.child_commit)
        self.lock = (
            'version=3\n[[package]]\nname="fixture"\nversion="1.0.0"\n'
            f'source="git+https://example.invalid/parent?rev={self.parent_commit}#{self.parent_commit}"\n'
        ).encode()
        self.destination = root / "isolated"
        self.destination.mkdir()

    def git(self, repo: Path, *args: str) -> str:
        """Run Git for a fixture repository; example: fixture.git(repo, 'status')."""
        return subprocess.check_output(["git", "-C", str(repo), *args], text=True).strip()

    def repository(self, name: str, child: str | None = None) -> tuple[Path, str]:
        """Create a tracked commit and its cached bare copy; example: fixture.repository('item')."""
        repo = self.root / name
        repo.mkdir()
        self.git(repo, "init", "-q")
        (repo / "source.txt").write_text(f"{name} committed")
        self.git(repo, "add", "source.txt")
        if child is not None:
            self.git(repo, "update-index", "--add", "--cacheinfo", f"160000,{child},external/child")
        self.git(repo, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture")
        revision = self.git(repo, "rev-parse", "HEAD")
        subprocess.run(["git", "clone", "--bare", "-q", str(repo), str(self.home / "git/db" / (name + "-cache"))], check=True)
        return repo, revision


class CaptureGitCacheTests(unittest.TestCase):
    def test_stages_locked_commit_and_gitlink_without_shared_checkout(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = GitCacheFixture(Path(folder))
            (fixture.parent / "source.txt").write_text("uncommitted forgery")
            stage_git_cache(fixture.lock, fixture.home, fixture.destination)
            db = fixture.destination / "git/db"
            self.assertEqual({path.name for path in db.iterdir()}, {"parent-cache", "child-cache"})
            content = subprocess.check_output(["git", f"--git-dir={db / 'parent-cache'}", "show", f"{fixture.parent_commit}:source.txt"], text=True)
            self.assertEqual(content, "parent committed")
            self.assertFalse((fixture.destination / "git/checkouts").exists())

    def test_missing_or_ambiguous_commits_are_rejected(self) -> None:
        for fault in ("missing", "ambiguous"):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as folder:
                fixture = GitCacheFixture(Path(folder))
                if fault == "missing":
                    import shutil
                    shutil.rmtree(fixture.home / "git/db/child-cache")
                else:
                    import shutil
                    shutil.copytree(fixture.home / "git/db/parent-cache", fixture.home / "git/db/duplicate-cache")
                with self.assertRaisesRegex(ValueError, "expected one cached database"):
                    stage_git_cache(fixture.lock, fixture.home, fixture.destination)

    def test_caller_git_config_is_not_copied(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = GitCacheFixture(Path(folder))
            source = fixture.home / "git/db/parent-cache"
            with (source / "config").open("a") as config:
                config.write('\n[alias]\n  injected = !false\n')
            mirror = isolated_git_source(source, fixture.destination / "mirror")
            self.assertNotIn("injected", (mirror / "config").read_text())

    def test_alternates_and_special_objects_are_refused(self) -> None:
        for fault in ("alternates", "symlink"):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory() as folder:
                fixture = GitCacheFixture(Path(folder))
                objects = fixture.home / "git/db/parent-cache/objects"
                if fault == "alternates":
                    (objects / "info/alternates").write_text("/outside\n")
                else:
                    (objects / "info/link").symlink_to(Path(folder) / "outside")
                with self.assertRaisesRegex(ValueError, "expected"):
                    isolated_git_source(objects.parent, fixture.destination / "mirror")

    def test_corrupt_git_blob_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = GitCacheFixture(Path(folder))
            identity = fixture.git(fixture.parent, "rev-parse", f"{fixture.parent_commit}:source.txt")
            blob = fixture.home / "git/db/parent-cache/objects" / identity[:2] / identity[2:]
            blob.unlink()
            blob.write_bytes(b"corrupt object")
            with self.assertRaises((ValueError, subprocess.CalledProcessError)):
                stage_git_cache(fixture.lock, fixture.home, fixture.destination)

    def test_unlocked_short_revision_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            fixture = GitCacheFixture(Path(folder))
            short = fixture.lock.replace(fixture.parent_commit.encode(), b"abcd")
            with self.assertRaisesRegex(ValueError, "expected a full SHA-1"):
                stage_git_cache(short, fixture.home, fixture.destination)
