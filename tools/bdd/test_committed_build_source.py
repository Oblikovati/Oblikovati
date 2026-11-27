# SPDX-License-Identifier: GPL-2.0-only
"""Check that experimental build sources contain exactly committed Git blobs."""

import io
import os
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch
from pathlib import Path

from committed_build_source import (
    committed_files,
    git_environment,
    materialize_source,
    unpack_archive,
    validate_path,
)


class SourceRepository:
    """A small repository with an ignored input and one executable file."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.git("init", "-q")
        (root / ".gitignore").write_text("ignored.txt\n")
        (root / "source.rs").write_text("committed")
        (root / "build.sh").write_text("build")
        (root / "build.sh").chmod(0o755)
        self.git("add", ".")
        self.git("commit", "-qm", "source")
        self.commit = self.git("rev-parse", "HEAD")

    def git(self, *args: str) -> str:
        """Run a Git command in this fixture; example: fixture.git('status')."""
        return subprocess.check_output(
            [
                "git",
                "-C",
                str(self.root),
                "-c",
                "user.name=Fixture",
                "-c",
                "user.email=fixture@example.invalid",
                *args,
            ],
            text=True,
        ).strip()


class CommittedBuildSourceTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        self.fixture = SourceRepository(self.repo)
        self.output = self.root / "snapshot"

    def test_dirty_and_ignored_inputs_are_excluded_and_modes_preserved(self) -> None:
        (self.repo / "source.rs").write_text("changed")
        (self.repo / "ignored.txt").write_text("secret")
        (self.repo / "untracked").write_text("outside")
        result = materialize_source(self.repo, self.fixture.commit, self.output)
        self.assertEqual((self.output / "source.rs").read_text(), "committed")
        self.assertFalse((self.output / "ignored.txt").exists())
        self.assertFalse((self.output / "untracked").exists())
        self.assertEqual((self.output / "build.sh").stat().st_mode & 0o777, 0o755)
        self.assertEqual(set(result["files"]), {".gitignore", "source.rs", "build.sh"})
        self.assertFalse(result["reusable"])

    def test_existing_or_internal_destination_is_refused(self) -> None:
        self.output.mkdir()
        with self.assertRaises(FileExistsError):
            materialize_source(self.repo, self.fixture.commit, self.output)
        with self.assertRaisesRegex(ValueError, "outside"):
            materialize_source(self.repo, self.fixture.commit, self.repo / "output")

    def test_unsupported_symlink_is_refused(self) -> None:
        (self.repo / "link").symlink_to("source.rs")
        self.fixture.git("add", "link")
        self.fixture.git("commit", "-qm", "link")
        with self.assertRaisesRegex(ValueError, "regular Git blob"):
            materialize_source(
                self.repo, self.fixture.git("rev-parse", "HEAD"), self.output
            )
        self.assertFalse(self.output.exists())

    def test_export_ignore_is_detected_and_partial_output_removed(self) -> None:
        (self.repo / ".gitattributes").write_text("source.rs export-ignore\n")
        self.fixture.git("add", ".gitattributes")
        self.fixture.git("commit", "-qm", "ignore")
        with self.assertRaisesRegex(ValueError, "archive missing"):
            materialize_source(
                self.repo, self.fixture.git("rev-parse", "HEAD"), self.output
            )
        self.assertFalse(self.output.exists())

    def test_bad_archive_content_and_traversal_are_rejected(self) -> None:
        files = committed_files(self.repo, self.fixture.commit)
        self.output.mkdir()
        for name, content in (("../outside", b"x"), ("source.rs", b"altered")):
            with self.subTest(name=name), tempfile.TemporaryFile() as archive:
                with tarfile.open(fileobj=archive, mode="w") as tar:
                    member = tarfile.TarInfo(name)
                    member.size = len(content)
                    tar.addfile(member, io.BytesIO(content))
                archive.seek(0)
                with self.assertRaises(ValueError):
                    unpack_archive(archive, self.output, files)
        self.assertFalse((self.root / "outside").exists())

    def test_archive_rejects_duplicate_and_symbolic_link_entries(self) -> None:
        files = committed_files(self.repo, self.fixture.commit)
        for kind in ("duplicate", "symlink"):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as folder:
                with tempfile.TemporaryFile() as archive:
                    with tarfile.open(fileobj=archive, mode="w") as tar:
                        member = tarfile.TarInfo("source.rs")
                        if kind == "symlink":
                            member.type = tarfile.SYMTYPE
                            member.linkname = "../outside"
                            tar.addfile(member)
                        else:
                            member.size = len(b"committed")
                            for _ in range(2):
                                tar.addfile(member, io.BytesIO(b"committed"))
                    archive.seek(0)
                    with self.assertRaisesRegex(ValueError, "tracked regular file"):
                        unpack_archive(archive, Path(folder), files)

    def test_unsupported_host_is_refused(self) -> None:
        with patch("committed_build_source.os.name", "nt"):
            with self.assertRaisesRegex(ValueError, "expected a POSIX host"):
                materialize_source(self.repo, self.fixture.commit, self.output)

    def test_hostile_git_environment_does_not_change_the_export(self) -> None:
        poison = {
            "GIT_DIR": "/missing/repository",
            "GIT_CONFIG_COUNT": "1",
            "GIT_CONFIG_KEY_0": "core.repositoryformatversion",
            "GIT_CONFIG_VALUE_0": "99",
            "RUSTFLAGS": "--cfg poison",
        }
        with patch.dict(os.environ, poison):
            result = materialize_source(self.repo, self.fixture.commit, self.output)
        self.assertEqual((self.output / "source.rs").read_text(), "committed")
        self.assertFalse(result["reusable"])

    def test_git_configuration_is_not_inherited(self) -> None:
        self.assertNotIn("RUSTFLAGS", git_environment())
        self.assertEqual(git_environment()["GIT_CONFIG_GLOBAL"], "/dev/null")
        for name in ("../x", "/x", "x/../y", ".git/config", "x\\y"):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_path(name)
        with self.assertRaisesRegex(ValueError, "full Git SHA"):
            committed_files(self.repo, "HEAD")


if __name__ == "__main__":
    unittest.main()
