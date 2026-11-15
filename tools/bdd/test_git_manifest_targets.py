# SPDX-License-Identifier: GPL-2.0-only
"""Test conservative normalized target checks against locked source declarations."""

import tempfile
import unittest
from pathlib import Path

from git_manifest_targets import (
    check_git_manifest_targets,
    discovery_enabled,
    explicit_targets,
    generated_targets,
    inferred_target,
    inferred_targets,
    target_edition,
    workspace_edition,
)
from git_object_inventory import locked_git_tree
from test_committed_build_source import SourceRepository


class GitManifestTargetTests(unittest.TestCase):
    def test_inferred_target_conventions_and_collisions(self) -> None:
        self.assertEqual(inferred_target("tests/a.rs", "tests"), ("a", "tests/a.rs"))
        self.assertEqual(inferred_target("tests/b/main.rs", "tests"), ("b", "tests/b/main.rs"))
        for path in ("tests/.hidden.rs", "tests/deep/helper.rs", "examples/a.rs"):
            self.assertIsNone(inferred_target(path, "tests"))
        with self.assertRaisesRegex(ValueError, "expected one conventional"):
            inferred_targets({"tests/a.rs": "blob", "tests/a/main.rs": "blob"}, "test", "pkg")

    def test_explicit_target_normalization_and_rejections(self) -> None:
        files = {"tests/a.rs": "blob", "checks/custom.rs": "blob"}
        inferred = {"a": "tests/a.rs"}
        self.assertEqual(explicit_targets([{"name": "a"}], inferred, files, "x"), {"a": {"name": "a", "path": "tests/a.rs"}})
        self.assertEqual(explicit_targets([{"name": "custom", "path": "./checks/custom.rs"}], inferred, files, "x")["custom"]["path"], "checks/custom.rs")
        for rows in ([{}], [{"name": "a"}, {"name": "a"}], [{"name": "x", "path": "../outside"}]):
            with self.subTest(rows=rows), self.assertRaisesRegex(ValueError, "expected"):
                explicit_targets(rows, inferred, files, "x")

    def test_edition_dependent_discovery(self) -> None:
        self.assertFalse(discovery_enabled({}, "autotests", True, "2015", "x"))
        self.assertTrue(discovery_enabled({}, "autotests", False, "2015", "x"))
        self.assertTrue(discovery_enabled({}, "autotests", True, "2024", "x"))
        self.assertFalse(discovery_enabled({"autotests": False}, "autotests", False, "2024", "x"))
        with self.assertRaisesRegex(ValueError, "expected a boolean"):
            discovery_enabled({"autotests": 0}, "autotests", False, "2024", "x")

    def test_workspace_edition_uses_locked_git_content(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            repo = Path(folder)
            fixture = SourceRepository(repo)
            (repo / "Cargo.toml").write_text('[workspace.package]\nedition="2024"\n')
            fixture.git("add", "Cargo.toml")
            fixture.git("commit", "-qm", "workspace edition")
            tree = locked_git_tree(repo, fixture.git("rev-parse", "HEAD"))
            self.assertEqual(workspace_edition(repo, tree, "crates/demo", "demo"), "2024")
            self.assertEqual(target_edition({"package": {"edition": {"workspace": True}}}, repo, tree, "crates/demo", "demo"), "2024")
            with self.assertRaisesRegex(ValueError, "expected a regular tracked file"):
                workspace_edition(repo, {"Cargo.toml": "120000:blob:bad"}, "crates/demo", "demo")

    def test_generated_target_set_and_flags_match(self) -> None:
        source = b'[package]\nname="demo"\nedition="2024"\n[[example]]\nname="manual"\npath="examples/manual.rs"\ntest=true\n'
        generated = source + b'[[example]]\nname="extra"\npath="examples/extra.rs"\n'
        files = {"examples/manual.rs": "blob", "examples/extra.rs": "blob"}
        check_git_manifest_targets(source, generated, files, Path("."), {}, ".", "demo")
        with self.assertRaisesRegex(ValueError, "expected locked edition"):
            check_git_manifest_targets(source, generated.replace(b'edition="2024"', b'edition="2015"'), files, Path("."), {}, ".", "demo")
        for forged in (generated.replace(b'test=true', b'test=false'), generated.replace(b'examples/extra.rs', b'examples/evil.rs'), source):
            with self.subTest(forged=forged), self.assertRaisesRegex(ValueError, "expected locked targets"):
                check_git_manifest_targets(source, forged, files, Path("."), {}, ".", "demo")

    def test_duplicate_generated_target_is_declined(self) -> None:
        with self.assertRaisesRegex(ValueError, "expected a unique named target"):
            generated_targets([{"name": "a"}, {"name": "a"}], "demo")
