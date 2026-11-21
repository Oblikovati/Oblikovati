# SPDX-License-Identifier: GPL-2.0-only
"""Exercise inventory preflight with a real Cargo build and hostile settings."""

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from behavior_ledger import application_state
from capture_inventory import isolated_inventory
from git_environment import checkout_git_environment
from verify_snapshot import git_revision


class InventoryRepository:
    """Create a committed inventory tool for the capture preflight."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.git("init", "-q")
        (root / "src").mkdir()
        (root / "Cargo.toml").write_text('[package]\nname="oblikovati-build"\nversion="0.1.0"\nedition="2021"\n')
        (root / "Cargo.lock").write_text('version = 3\n\n[[package]]\nname = "oblikovati-build"\nversion = "0.1.0"\n')
        (root / "src/main.rs").write_text('fn main() { assert!(!cfg!(probe_override)); println!("trusted inventory"); }\n')
        self.git("add", ".")
        self.git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture")
        self.commit = self.git("rev-parse", "HEAD").strip()

    def git(self, *arguments: str) -> str:
        """Run fixture Git; example: fixture.git('status', '--short')."""
        return subprocess.check_output(["git", "-C", str(self.root), *arguments], text=True)


class CaptureInventoryTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("cargo"), "requires Cargo")
    def test_real_preflight_refuses_caller_tool_and_compiler_overrides(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo = root / "repo"
            repo.mkdir()
            fixture = InventoryRepository(repo)
            cache = root / "cargo-home"
            cache.mkdir()
            (cache / "config.toml").write_text('[build]\nrustc-wrapper="/does-not-exist"\n')
            environment = dict(os.environ, CARGO_HOME=str(cache), RUSTFLAGS="--cfg probe_override", RUSTC_WRAPPER="/does-not-exist", OBK_SCENARIO_LEDGER_TOOL="/does-not-exist")
            result = isolated_inventory(repo, repo, "1" * 40, fixture.commit, environment)
            self.assertEqual(result, "trusted inventory\n")

    def test_windows_git_environment_keeps_path_and_native_null_device(self) -> None:
        with patch("os.name", "nt"), patch("os.defpath", "default-path"), patch("os.devnull", "NUL"):
            with patch.dict(os.environ, {"PATH": "git-install-path"}):
                environment = checkout_git_environment()
        self.assertEqual(environment["PATH"], "git-install-path")
        self.assertEqual(environment["GIT_CONFIG_GLOBAL"], "NUL")

    def test_git_environment_cannot_replace_checkout_identity(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            first, other = root / "first", root / "other"
            first.mkdir()
            other.mkdir()
            expected = InventoryRepository(first)
            replacement = InventoryRepository(other)
            (other / "other.txt").write_text("different")
            replacement.git("add", "other.txt")
            replacement.git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "different")
            self.assertNotEqual(replacement.git("rev-parse", "HEAD").strip(), expected.commit)
            with patch.dict(os.environ, {"GIT_DIR": str(other / ".git"), "GIT_WORK_TREE": str(other)}):
                self.assertEqual(git_revision(first), expected.commit)
                self.assertEqual(application_state(first), (expected.commit, True))
