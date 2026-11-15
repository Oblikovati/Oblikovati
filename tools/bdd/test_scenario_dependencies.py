# SPDX-License-Identifier: GPL-2.0-only
"""Check conservative, per-scenario dependency records."""

import copy
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from dependency_cli import runtime_changes_for
from scenario_dependencies import (
    build_map,
    changed_dependencies,
    changed_runtime_dependencies,
    reuse_decision,
    validate_map,
)


class GitDependencyFixture:
    """A named Git fixture with two scenario sources and shared application files."""

    def __init__(self, root: Path) -> None:
        self.root = root
        self.git("init", "-q")
        self.git("config", "user.name", "Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        self.write("crates/head/src/lib.rs", "shared")
        self.write("assets/theme.xml", "theme")
        self.git("add", ".")
        self.git("commit", "-qm", "baseline")
        self.baseline = self.git("rev-parse", "HEAD")

    def git(self, *args: str) -> str:
        """Run Git in the fixture; example: fixture.git('rev-parse', 'HEAD')."""
        return subprocess.check_output(
            ["git", "-C", str(self.root), *args], text=True
        ).strip()

    def write(self, name: str, value: str) -> None:
        """Write a fixture file; example: fixture.write('src/a', 'value')."""
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(value, encoding="utf-8")

    def commit(self, message: str) -> str:
        """Commit fixture changes; example: fixture.commit('change')."""
        self.git("add", "-A")
        self.git("commit", "-qm", message)
        return self.git("rev-parse", "HEAD")


def source_row(identifier: str, feature: str) -> dict[str, object]:
    """Create a scenario declaration; example: source_row('one', 'first')."""
    return {
        "id": identifier,
        "area": "application",
        "path": f"features/application/{feature}/scenario.feature",
        "source_sha256": "a" * 64,
    }


class ScenarioDependencyTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.fixture = GitDependencyFixture(Path(self.temporary.name))
        self.inventory = {
            "source_revision": "b" * 40,
            "scenarios": [source_row("one", "first"), source_row("two", "second")],
        }

    def manifest(self) -> dict[str, object]:
        """Build the baseline map; example: self.manifest()."""
        return build_map(self.fixture.root, self.inventory, self.fixture.baseline)

    def test_each_scenario_has_its_own_source_and_shared_complete_git_tree(
        self,
    ) -> None:
        mapping = self.manifest()
        validate_map(mapping, self.inventory)
        self.assertEqual(set(mapping["scenarios"]), {"one", "two"})
        self.assertNotEqual(
            mapping["scenarios"]["one"]["source"], mapping["scenarios"]["two"]["source"]
        )
        self.assertEqual(mapping["runtime_coverage"], "unbounded")
        self.assertEqual(
            set(mapping["application_inputs"]),
            {"assets/theme.xml", "crates/head/src/lib.rs"},
        )

    def test_changed_added_and_deleted_application_inputs_are_reported(self) -> None:
        mapping = self.manifest()
        self.fixture.write("crates/head/src/lib.rs", "changed")
        self.fixture.write("new/file", "new")
        (self.fixture.root / "assets/theme.xml").unlink()
        current = self.fixture.commit("change")
        self.assertEqual(
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "one", current
            ),
            ["assets/theme.xml", "crates/head/src/lib.rs", "new/file"],
        )

    def test_missing_unknown_and_forged_scope_is_rejected(self) -> None:
        mapping = self.manifest()
        forged = copy.deepcopy(mapping)
        del forged["application_inputs"]["assets/theme.xml"]
        with self.assertRaisesRegex(ValueError, "application input"):
            changed_dependencies(
                self.fixture.root, forged, self.inventory, "one", self.fixture.baseline
            )
        with self.assertRaisesRegex(ValueError, "scenario"):
            changed_dependencies(
                self.fixture.root,
                mapping,
                self.inventory,
                "missing",
                self.fixture.baseline,
            )
        forged = copy.deepcopy(mapping)
        forged["runtime_coverage"] = "complete"
        with self.assertRaisesRegex(ValueError, "runtime coverage"):
            validate_map(forged, self.inventory)

    def test_unbounded_runtime_cannot_authorize_receipt_reuse(self) -> None:
        mapping = self.manifest()
        self.assertEqual(
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "one", self.fixture.baseline
            ),
            [],
        )
        self.assertEqual(
            reuse_decision(mapping, []),
            {"reusable": False, "reason": "runtime inputs are not bounded"},
        )
        mapping["runtime_coverage"] = "hermetic-v1"
        with self.assertRaisesRegex(ValueError, "runtime coverage"):
            reuse_decision(mapping, [])

    def test_runtime_snapshot_is_bound_to_one_scenario(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "runtime").write_bytes(b"library")
            mapping = build_map(
                self.fixture.root,
                self.inventory,
                self.fixture.baseline,
                runtime_scenario="one",
                runtime_root=root,
            )
            validate_map(mapping, self.inventory)
            self.assertEqual(mapping["scenarios"]["one"]["runtime"], "runtime_inputs")
            self.assertNotIn("runtime", mapping["scenarios"]["two"])
            self.assertIn("runtime", mapping["runtime_inputs"])
            self.assertFalse(reuse_decision(mapping, [])["reusable"])

    def test_runtime_changes_are_reported_and_cannot_be_used_by_another_scenario(
        self,
    ) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root / "runtime").write_bytes(b"library")
            mapping = build_map(
                self.fixture.root,
                self.inventory,
                self.fixture.baseline,
                runtime_scenario="one",
                runtime_root=root,
            )
            self.assertEqual(changed_runtime_dependencies(mapping, "one", root), [])
            (root / "runtime").write_bytes(b"changed")
            self.assertEqual(
                changed_runtime_dependencies(mapping, "one", root), ["runtime"]
            )
            self.assertEqual(
                reuse_decision(mapping, [], ["runtime"])["reason"],
                "runtime inputs changed",
            )
            with self.assertRaisesRegex(ValueError, "recorded runtime"):
                changed_runtime_dependencies(mapping, "two", root)

    def test_runtime_cli_only_compares_the_scenarios_own_capsule(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            mapping = build_map(
                self.fixture.root,
                self.inventory,
                self.fixture.baseline,
                runtime_scenario="one",
                runtime_root=root,
            )
            self.assertIsNone(runtime_changes_for(mapping, "two", None))
            with self.assertRaisesRegex(ValueError, "runtime root None"):
                runtime_changes_for(mapping, "one", None)
            self.assertEqual(runtime_changes_for(mapping, "one", root), [])

    @unittest.skipUnless(os.name == "posix", "requires Git file modes")
    def test_file_mode_change_is_reported(self) -> None:
        mapping = self.manifest()
        (self.fixture.root / "crates/head/src/lib.rs").chmod(0o755)
        current = self.fixture.commit("change mode")
        self.assertEqual(
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "one", current
            ),
            ["crates/head/src/lib.rs"],
        )

    def test_runtime_snapshot_cannot_be_assigned_to_an_unknown_scenario(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            with self.assertRaisesRegex(ValueError, "mapped scenario"):
                build_map(
                    self.fixture.root,
                    self.inventory,
                    self.fixture.baseline,
                    runtime_scenario="missing",
                    runtime_root=Path(folder),
                )

    def test_changed_scenario_source_and_unknown_policy_are_refused(self) -> None:
        mapping = self.manifest()
        changed = copy.deepcopy(self.inventory)
        changed["scenarios"][0]["source_sha256"] = "c" * 64
        with self.assertRaisesRegex(ValueError, "scenario dependencies"):
            validate_map(mapping, changed)
        forged = copy.deepcopy(mapping)
        forged["version"] = 99
        with self.assertRaisesRegex(ValueError, "dependency version"):
            changed_dependencies(
                self.fixture.root, forged, self.inventory, "one", self.fixture.baseline
            )

    def test_renames_are_treated_as_remove_and_add(self) -> None:
        mapping = self.manifest()
        self.fixture.git("mv", "assets/theme.xml", "assets/renamed.xml")
        current = self.fixture.commit("rename")
        self.assertEqual(
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "two", current
            ),
            ["assets/renamed.xml", "assets/theme.xml"],
        )

    def test_dirty_tracked_and_untracked_inputs_are_refused(self) -> None:
        mapping = self.manifest()
        self.fixture.write("crates/head/src/lib.rs", "dirty")
        with self.assertRaisesRegex(ValueError, "clean"):
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "one", self.fixture.baseline
            )
        self.fixture.git("checkout", "--", "crates/head/src/lib.rs")
        self.fixture.write("untracked", "unknown")
        with self.assertRaisesRegex(ValueError, "clean"):
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "one", self.fixture.baseline
            )

    def test_index_flags_cannot_hide_changed_tracked_inputs(self) -> None:
        mapping = self.manifest()
        path = "crates/head/src/lib.rs"
        for flag in ("--assume-unchanged", "--skip-worktree"):
            with self.subTest(flag=flag):
                self.fixture.git("update-index", flag, path)
                self.fixture.write(path, "hidden change")
                with self.assertRaisesRegex(ValueError, "clean"):
                    changed_dependencies(
                        self.fixture.root,
                        mapping,
                        self.inventory,
                        "one",
                        self.fixture.baseline,
                    )
                self.fixture.git("update-index", "--no-assume-unchanged", path)
                self.fixture.git("update-index", "--no-skip-worktree", path)
                self.fixture.git("checkout", "--", path)

    def test_git_environment_cannot_redirect_or_replace_history(self) -> None:
        mapping = self.manifest()
        self.fixture.write("crates/head/src/lib.rs", "changed")
        current = self.fixture.commit("changed")
        self.fixture.git("replace", current, self.fixture.baseline)
        with tempfile.TemporaryDirectory() as other:
            GitDependencyFixture(Path(other))
            with patch.dict(
                os.environ, {"GIT_DIR": str(Path(other) / ".git")}, clear=False
            ):
                changes = changed_dependencies(
                    self.fixture.root, mapping, self.inventory, "one", current
                )
        self.assertEqual(changes, ["crates/head/src/lib.rs"])

    def test_divergent_history_is_not_accepted(self) -> None:
        mapping = self.manifest()
        self.fixture.git("checkout", "--orphan", "other")
        self.fixture.write("other", "unrelated")
        current = self.fixture.commit("unrelated")
        with self.assertRaisesRegex(ValueError, "ancestor"):
            changed_dependencies(
                self.fixture.root, mapping, self.inventory, "one", current
            )


if __name__ == "__main__":
    unittest.main()
