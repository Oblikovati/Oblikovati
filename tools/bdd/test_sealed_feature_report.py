# SPDX-License-Identifier: GPL-2.0-only
"""Check full-feature diagnostic reports and scenario capture identity."""
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from dataclasses import replace

from behavior_evidence import example_capture_name
from sealed_feature_report import (
    capture_namespace,
    check_feature_report,
    require_feature_captures,
    selected_feature_rows,
)
from test_behavior_outline import OutlineFixture
from test_sealed_execution import SealedFixture


class FeatureReportTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        root = Path(self.temporary.name)
        self.fixture = SealedFixture(root)
        self.request = replace(self.fixture.request(), feature="docking-system", scenario="", all_scenarios=True)
        repository = Path(__file__).resolve().parents[2]
        receipt = repository / "architecture/rust-port/receipts/2026-11-20-docking-current-4133"
        source = repository / "architecture/rust-port/receipts/2026-11-20-docking-local-snapshot-3978/inventory.json"
        inventory = json.loads(source.read_text())
        self.revision = inventory["source_revision"]
        self.rows = selected_feature_rows(inventory, self.request, self.revision)
        for name in ("cucumber.json", "junit.xml"):
            shutil.copyfile(receipt / name, self.request.reports / name)
        shutil.copytree(receipt / "scenarios", self.request.reports / "scenarios")
        state = {"commit": self.revision, "dirty": False, "root": "/scenario"}
        (self.request.reports / "source-revision.json").write_text(json.dumps({"before": state, "after": state, "source_changed": False}))

    def test_existing_full_report_matches_each_scenario_and_capture(self) -> None:
        result = check_feature_report(self.request, self.rows, self.revision)
        self.assertTrue(result["content_checked"])
        self.assertFalse(result["reusable"])
        self.assertEqual(len(result["scenario_ids"]), 10)

    def test_missing_case_or_failed_step_is_rejected(self) -> None:
        path = self.request.reports / "cucumber.json"
        original = json.loads(path.read_text())
        report = json.loads(json.dumps(original))
        report[0]["elements"].pop()
        path.write_text(json.dumps(report))
        with self.assertRaisesRegex(ValueError, "scenario count"):
            check_feature_report(self.request, self.rows, self.revision)
        original[0]["elements"][0]["steps"][0]["result"]["status"] = "failed"
        path.write_text(json.dumps(original))
        with self.assertRaises(ValueError):
            check_feature_report(self.request, self.rows, self.revision)

    def test_wrong_namespace_or_invalid_capture_is_rejected(self) -> None:
        folder = self.request.reports / "scenarios" / capture_namespace(self.rows[0]) / "final"
        image = next(folder.glob("*.png"))
        image.write_bytes(b"not a PNG")
        with self.assertRaisesRegex(ValueError, "PNG"):
            check_feature_report(self.request, self.rows, self.revision)
        image.unlink()
        with self.assertRaisesRegex(ValueError, "one final PNG"):
            check_feature_report(self.request, self.rows, self.revision)

    def test_duplicate_or_other_feature_rows_are_rejected(self) -> None:
        inventory = {"source_revision": self.revision, "scenarios": self.rows + [self.rows[0]]}
        with self.assertRaisesRegex(ValueError, "uniquely"):
            selected_feature_rows(inventory, self.request, self.revision)
        duplicate = dict(self.rows[0], id="scenario:other")
        inventory["scenarios"] = self.rows + [duplicate]
        with self.assertRaisesRegex(ValueError, "unique scenario source identities"):
            selected_feature_rows(inventory, self.request, self.revision)
        inventory["scenarios"] = []
        with self.assertRaisesRegex(ValueError, "nonempty"):
            selected_feature_rows(inventory, self.request, self.revision)


class OutlineCaptureTests(unittest.TestCase):
    def test_expanded_examples_require_distinct_native_capture_namespaces(self) -> None:
        row = OutlineFixture().row
        with tempfile.TemporaryDirectory() as directory:
            reports = Path(directory)
            images = []
            for example, name in zip(row["examples"], ("Value one", "Value two")):
                identity = f"{row['path']}:{example['line']}:{name}"
                namespace = hashlib.sha256(identity.encode()).hexdigest()
                folder = reports / "scenarios" / namespace / "final"
                folder.mkdir(parents=True)
                image = folder / example_capture_name(row, example)
                image.write_bytes(b"\x89PNG\r\n\x1a\n" + name.encode())
                images.append(image)
            require_feature_captures(reports, row)
            images[1].rename(images[0].parent / images[1].name)
            with self.assertRaises(ValueError):
                require_feature_captures(reports, row)


if __name__ == "__main__":
    unittest.main()
