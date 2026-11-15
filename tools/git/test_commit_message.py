# SPDX-License-Identifier: GPL-2.0-only
"""Test commit message validation and the installed Git hook."""

import subprocess
import tempfile
import unittest
from pathlib import Path

from commit_message import message_error

HOOK = Path(__file__).resolve().parents[2] / ".githooks" / "commit-msg"


class FakeGitRepository:
    """Make a disposable repository with the real commit-message hook."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.git("init", "-q")
        self.git("config", "user.name", "Test User")
        self.git("config", "user.email", "test@example.invalid")
        self.git("config", "core.hooksPath", str(HOOK.parent))
        (root / "tracked.txt").write_text("test\n")
        self.git("add", "tracked.txt")

    def git(self, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["git", *args], cwd=self.root, text=True, capture_output=True, check=check
        )

    def commit(self, *paragraphs: str) -> subprocess.CompletedProcess[str]:
        arguments = [item for text in paragraphs for item in ("-m", text)]
        return self.git("commit", *arguments, check=False)


class CommitMessageTests(unittest.TestCase):
    def test_accepts_body_and_issue_reference(self) -> None:
        self.assertIsNone(message_error("Fix selection\n\nKeep the selected identity.\n\nRefs #3979"))

    def test_rejects_subject_only_and_unexplained_references(self) -> None:
        for message in ("Fix selection", "Fix selection\n\nRefs #3979", "Fix selection\n\nSigned-off-by: A\nRefs #3979"):
            with self.subTest(message=message):
                self.assertIsNotNone(message_error(message))

    def test_requires_a_separate_positive_issue_reference(self) -> None:
        for reference in ("Refs #0", "Refs #abc", "Refs #12 later", "# Refs #12", "Ref #12"):
            with self.subTest(reference=reference):
                self.assertIsNotNone(message_error(f"Fix selection\n\nKeep identity.\n\n{reference}"))

    def test_requires_blank_line_after_subject(self) -> None:
        self.assertIsNotNone(message_error("Fix selection\nKeep identity.\nRefs #12"))

    def test_real_git_hook_rejects_bad_message_and_accepts_valid_message(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repository = FakeGitRepository(Path(folder))
            rejected = repository.commit("Fix selection")
            self.assertNotEqual(rejected.returncode, 0)
            self.assertIn("expected a blank line", rejected.stderr)
            accepted = repository.commit("Fix selection", "Keep the selected identity.", "Refs #3979")
            self.assertEqual(accepted.returncode, 0, accepted.stderr)

    def test_merge_commit_requires_the_same_message_fields(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repository = FakeGitRepository(Path(folder))
            repository.commit("Initial file", "Set up a test file.", "Refs #1")
            repository.git("checkout", "-qb", "side")
            (repository.root / "side.txt").write_text("side\n")
            repository.git("add", "side.txt")
            repository.commit("Add side file", "Prepare a branch.", "Refs #1")
            repository.git("checkout", "-q", "-")
            rejected = repository.git("merge", "--no-ff", "side", "-m", "Merge side", check=False)
            self.assertNotEqual(rejected.returncode, 0)
            self.assertIn("expected a blank line", rejected.stderr)
            repository.git("merge", "--abort")
            accepted = repository.git(
                "merge", "--no-ff", "side", "-m", "Merge side",
                "-m", "Include the tested change.", "-m", "Refs #3979", check=False,
            )
            self.assertEqual(accepted.returncode, 0, accepted.stderr)

    def test_respects_git_comment_character(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repository = FakeGitRepository(Path(folder))
            repository.git("config", "core.commentChar", ";")
            result = repository.commit("Fix selection", "Keep identity.\n\n; Refs #3979")
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("expected a separate Refs", result.stderr)

    def test_real_git_hook_does_not_count_commented_reference(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repository = FakeGitRepository(Path(folder))
            result = repository.commit("Fix selection", "Keep identity.\n\n# Refs #3979")
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("expected a separate Refs", result.stderr)


if __name__ == "__main__":
    unittest.main()
