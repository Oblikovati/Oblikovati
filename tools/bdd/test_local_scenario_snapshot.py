# SPDX-License-Identifier: GPL-2.0-only
"""Test local scenario snapshot selection and stability."""
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import local_scenario_snapshot as snapshot


class ChangingCopy:
    """Change a source file after the copy to model a concurrent scenario update."""
    def __init__(self) -> None:
        self.copy = snapshot.copy_inputs

    def __call__(self, source: Path, checkout: Path, manifest: dict[str, str]) -> None:
        self.copy(source, checkout, manifest)
        (source / "features/example/example.feature").write_text("Feature: changed\n")


class SnapshotTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = self.root / "source"
        (self.source / "features/example").mkdir(parents=True)
        (self.source / "features/example/example.feature").write_text("Feature: example\n")
        (self.source / "features/example/BLOCKERS.md").write_text("private notes\n")
        (self.source / "features/example/reference.png").write_bytes(b"image")
        (self.source / "fixtures/cad").mkdir(parents=True)
        (self.source / "fixtures/cad/part.opd").write_text("native fixture\n")

    def test_captures_a_clean_local_checkout_and_native_fixture(self) -> None:
        output = self.root / "snapshot"
        with patch.dict("os.environ", {"GIT_DIR": str(self.root / "unrelated-git")}):
            result = snapshot.create_snapshot(self.source, output)
        self.assertEqual(len(result["revision"]), 40)
        self.assertTrue((output / "checkout/fixtures/cad/part.opd").is_file())
        self.assertFalse((output / "checkout/features/example/BLOCKERS.md").exists())
        self.assertFalse((output / "checkout/features/example/reference.png").exists())
        self.assertEqual(snapshot.inventory(self.source), snapshot.inventory(output / "checkout"))

    def test_changed_source_is_rejected_without_publishing(self) -> None:
        output = self.root / "snapshot"
        with patch("local_scenario_snapshot.copy_inputs", ChangingCopy()):
            with self.assertRaisesRegex(ValueError, "unchanged selected inputs"):
                snapshot.create_snapshot(self.source, output)
        self.assertFalse(output.exists())

    def test_feature_scope_captures_named_fixture_and_rejects_invalid_paths(self) -> None:
        output = self.root / "scoped"
        feature = "example/example"
        # The fixture covers selection even when unrelated feature files change.
        folder = self.source / "features/example/example"
        folder.mkdir()
        (folder / "example.feature").write_text("Feature: scoped\n")
        result = snapshot.create_snapshot(self.source, output, feature, ("fixtures/cad/part.opd",))
        self.assertEqual(result["feature_scope"], feature)
        self.assertTrue((output / "checkout/fixtures/cad/part.opd").is_file())
        self.assertFalse((output / "checkout/features/example/example.feature").exists())
        with self.assertRaisesRegex(ValueError, "area/feature"):
            snapshot.selected_feature_files(self.source, "../outside", ())
        with self.assertRaisesRegex(ValueError, "native .opd"):
            snapshot.selected_feature_files(self.source, feature, ("fixtures/../outside.opd",))

    def test_rejects_symlink_and_existing_destination(self) -> None:
        output = self.root / "snapshot"
        output.mkdir()
        with self.assertRaisesRegex(ValueError, "expected a new path"):
            snapshot.create_snapshot(self.source, output)
        link = self.source / "features/example/link.feature"
        link.symlink_to(self.source / "features/example/example.feature")
        with self.assertRaisesRegex(ValueError, "not a symlink"):
            snapshot.inventory(self.source)


if __name__ == "__main__":
    unittest.main()
