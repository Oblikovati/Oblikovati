# SPDX-License-Identifier: GPL-2.0-only
"""Check that both dependency staging paths use one observed lockfile."""

import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from capture_dependency_stage import stage_capture_dependencies


class FakeDependencyStage:
    """Record the lock and optionally alter the source after registry staging."""

    def __init__(self, source: Path, alter: bool = False) -> None:
        self.source = source
        self.alter = alter
        self.calls: list[tuple[bytes, Path, Path]] = []

    def __call__(self, lock: bytes, cache: Path, destination: Path) -> None:
        self.calls.append((lock, cache, destination))
        if self.alter:
            (self.source / "Cargo.lock").write_bytes(b"changed")


class CaptureDependencyStageTests(unittest.TestCase):
    def test_stages_share_the_same_observed_lock(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "Cargo.lock").write_bytes(b"original")
            registry, git = FakeDependencyStage(root, True), FakeDependencyStage(root)
            env, settings = {"CARGO_HOME": str(root / "host")}, {"CARGO_HOME": str(root / "local")}
            with patch("capture_dependency_stage.stage_registry_cache", registry), patch("capture_dependency_stage.stage_git_cache", git):
                stage_capture_dependencies(root, env, settings)
            self.assertEqual(registry.calls, [(b"original", root / "host", root / "local")])
            self.assertEqual(git.calls, registry.calls)
