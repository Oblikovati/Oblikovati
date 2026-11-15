# SPDX-License-Identifier: GPL-2.0-only
"""Tests for truthful scenario classification and coverage reporting."""

import copy
import unittest
from pathlib import Path

from behavior_ledger import reconcile, require_classified, summarize
from behavior_evidence import StaleEvidence


class FakeReceiptVerifier:
    def __init__(self) -> None:
        self.calls: list[str] = []

    def __call__(
        self,
        root: Path,
        artifact: object,
        row: dict[str, object],
        source: str,
        commit: str,
    ) -> None:
        self.calls.append(str(row["id"]))
        if artifact != {"path": "valid", "sha256": "digest"}:
            raise ValueError(f"receipt {artifact!r}: expected valid evidence")


class FakeReceiptIntegrity:
    def __call__(self, root: Path, artifact: object) -> None:
        if artifact != {"path": "valid", "sha256": "digest"}:
            raise ValueError(f"receipt {artifact!r}: expected valid evidence")


class LedgerFixture:
    def __init__(self) -> None:
        self.inventory = {
            "source_revision": "source-1",
            "scenarios": [
                {
                    "id": "first",
                    "area": "application",
                    "path": "features/application/a/a.feature",
                    "scenario": "first",
                    "line": 1,
                    "outline": False,
                    "source_sha256": "hash-1",
                    "source_error": None,
                    "state": "inventoried",
                    "milestone": None,
                    "owner": None,
                    "last_receipt": None,
                    "background_steps": [],
                    "scenario_steps": [],
                },
                {
                    "id": "second",
                    "area": "sketch",
                    "path": "features/sketch/b/b.feature",
                    "scenario": "second",
                    "line": 1,
                    "outline": False,
                    "source_sha256": "hash-2",
                    "source_error": "invalid Gherkin",
                    "state": "blocked",
                    "milestone": None,
                    "owner": None,
                    "last_receipt": None,
                    "background_steps": [],
                    "scenario_steps": [],
                },
            ],
        }
        self.entry = {
            "id": "first",
            "source_revision": "source-1",
            "source_sha256": "hash-1",
            "milestone": "R3",
            "owner": "session",
            "target_module": "crates/session",
            "acceptance_test": "native scenario first",
            "issues": ["https://github.com/Oblikovati/Oblikovati/issues/1"],
            "state": "verified",
            "last_receipt": {"path": "valid", "sha256": "digest"},
        }
        self.verifier = FakeReceiptVerifier()

    def run(self, entries: list[object], clean: bool = True) -> dict[str, object]:
        return reconcile(
            self.inventory,
            {"version": 1, "scenarios": entries},
            "app",
            clean,
            Path("."),
            self.verifier,
            FakeReceiptIntegrity(),
        )


class LedgerTests(unittest.TestCase):
    def test_capture_host_only_is_blocked_without_host_attestation(self) -> None:
        fixture = LedgerFixture()
        fixture.inventory["scenarios"][0]["capture_host_only"] = True
        with self.assertRaisesRegex(ValueError, "blocked classification"):
            fixture.run([fixture.entry])
        blocked = dict(
            fixture.entry,
            state="blocked",
            reason="Capture host fixture required",
            last_receipt=None,
        )
        row = fixture.run([blocked])["scenarios"][0]
        self.assertEqual(row["state"], "blocked")
        self.assertEqual(fixture.verifier.calls, [])

    def test_capture_only_cannot_be_reported_verified(self) -> None:
        fixture = LedgerFixture()
        fixture.inventory["scenarios"][0]["capture_only"] = True
        with self.assertRaisesRegex(ValueError, "excluded classification"):
            fixture.run([fixture.entry])
        excluded = dict(
            fixture.entry,
            state="excluded",
            reason="Source requires the captured fixture",
            last_receipt=None,
        )
        excluded["historical_receipts"] = [
            {"source_revision": "old", "path": "prior", "sha256": "digest"}
        ]
        row = fixture.run([excluded])["scenarios"][0]
        self.assertEqual(row["state"], "excluded")
        self.assertEqual(row["historical_receipts"], excluded["historical_receipts"])
        self.assertEqual(summarize([row], "area")["application"]["excluded"], 1)
        self.assertEqual(fixture.verifier.calls, [])

    def test_unclassified_rows_are_not_verified_and_strict_check_fails(self) -> None:
        fixture = LedgerFixture()
        ledger = fixture.run([])
        rows = ledger["scenarios"]
        areas = summarize(rows, "area")
        self.assertEqual(areas["application"]["verified"], 0)
        self.assertEqual(areas["sketch"]["blocked"], 1)
        self.assertEqual(summarize(rows, "milestone")["unclassified"]["total"], 2)
        with self.assertRaisesRegex(ValueError, "2 missing"):
            require_classified(ledger)

    def test_verified_claim_requires_current_receipt_and_clean_application(
        self,
    ) -> None:
        fixture = LedgerFixture()
        ledger = fixture.run([fixture.entry])
        self.assertEqual(ledger["scenarios"][0]["state"], "verified")
        self.assertEqual(fixture.verifier.calls, ["first"])
        dirty = fixture.run([fixture.entry], clean=False)["scenarios"][0]
        self.assertTrue(dirty["stale"])
        self.assertEqual(dirty["state"], "inventoried")
        self.assertEqual(dirty["historical_receipt"], fixture.entry["last_receipt"])
        self.assertIsNone(dirty["last_receipt"])
        invalid = dict(fixture.entry, last_receipt=None)
        with self.assertRaisesRegex(ValueError, "expected valid evidence"):
            fixture.run([invalid])

    def test_changed_source_invalidates_receipt_and_keeps_classification(self) -> None:
        fixture = LedgerFixture()
        stale = dict(fixture.entry, source_sha256="old")
        ledger = fixture.run([stale])
        row = ledger["scenarios"][0]
        self.assertTrue(row["stale"])
        self.assertEqual(row["state"], "inventoried")
        self.assertEqual(row["milestone"], "R3")
        self.assertIsNone(row["last_receipt"])
        self.assertEqual(row["historical_receipt"], fixture.entry["last_receipt"])
        self.assertEqual(fixture.verifier.calls, [])
        with self.assertRaisesRegex(ValueError, "1 stale"):
            require_classified(ledger)

    def test_invalid_source_and_orphaned_classification_remain_visible(self) -> None:
        fixture = LedgerFixture()
        blocked = dict(fixture.entry, id="second", source_sha256="hash-2")
        orphan = dict(fixture.entry, id="removed")
        ledger = fixture.run([blocked, orphan])
        self.assertEqual(ledger["scenarios"][1]["state"], "blocked")
        self.assertEqual(ledger["orphaned_classifications"], ["removed"])
        self.assertEqual(fixture.verifier.calls, [])
        with self.assertRaisesRegex(ValueError, "1 orphaned"):
            require_classified(ledger)

    def test_duplicate_and_incomplete_classifications_are_rejected(self) -> None:
        fixture = LedgerFixture()
        with self.assertRaisesRegex(ValueError, "unique"):
            fixture.run([fixture.entry, copy.deepcopy(fixture.entry)])
        invalid = dict(fixture.entry, owner="")
        with self.assertRaisesRegex(ValueError, "owner"):
            fixture.run([invalid])


class GitRepositoryFixture:
    def __init__(self, root: Path) -> None:
        import subprocess

        self.root = root
        self.subprocess = subprocess
        self.git("init", "-q")
        self.git("config", "user.name", "Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        (root / "crates").mkdir()
        (root / "crates/code.rs").write_text("original")
        (root / "architecture/rust-port").mkdir(parents=True)
        (root / "architecture/rust-port/ledger.json").write_text("original")
        (root / "tools/bdd").mkdir(parents=True)
        (root / "tools/bdd/checker.py").write_text("original")
        (root / "tools/build").mkdir(parents=True)
        (root / "tools/build/inventory.rs").write_text("original")
        self.git("add", ".")
        self.git("commit", "-qm", "initial")

    def git(self, *arguments: str) -> str:
        return self.subprocess.check_output(
            ["git", "-C", str(self.root), *arguments], text=True
        ).strip()


class ApplicationInputsTests(unittest.TestCase):
    def test_inventory_command_does_not_write_git_index(self) -> None:
        import tempfile
        from behavior_ledger import read_inventory_command
        from runtime_snapshot import snapshot_runtime

        with tempfile.TemporaryDirectory() as folder:
            fixture = GitRepositoryFixture(Path(folder))
            before = snapshot_runtime(fixture.root / ".git")
            result = read_inventory_command(
                fixture.root, ["git", "status", "--porcelain"]
            )
            self.assertEqual(result, "")
            self.assertEqual(snapshot_runtime(fixture.root / ".git"), before)

    def test_only_unchanged_ancestor_application_inputs_can_reuse_receipts(
        self,
    ) -> None:
        import tempfile
        from behavior_ledger import application_inputs_match, application_state

        with tempfile.TemporaryDirectory() as folder:
            fixture = GitRepositoryFixture(Path(folder))
            initial, clean = application_state(fixture.root)
            self.assertTrue(clean)
            (fixture.root / "architecture/rust-port/ledger.json").write_text("updated")
            fixture.git("add", ".")
            fixture.git("commit", "-qm", "ledger only")
            ledger_commit, clean = application_state(fixture.root)
            self.assertTrue(clean)
            self.assertTrue(
                application_inputs_match(fixture.root, initial, ledger_commit)
            )
            self.assertFalse(
                application_inputs_match(fixture.root, ledger_commit, initial)
            )
            (fixture.root / "tools/bdd/checker.py").write_text("updated checker")
            (fixture.root / "tools/build/inventory.rs").write_text("updated inventory")
            self.assertTrue(application_state(fixture.root)[1])
            fixture.git("add", ".")
            fixture.git("commit", "-qm", "update ledger tools")
            tools_commit, clean = application_state(fixture.root)
            self.assertTrue(clean)
            self.assertTrue(
                application_inputs_match(fixture.root, initial, tools_commit)
            )
            (fixture.root / "crates/code.rs").write_text("changed")
            self.assertFalse(application_state(fixture.root)[1])
            fixture.git("add", ".")
            fixture.git("commit", "-qm", "change code")
            changed, clean = application_state(fixture.root)
            self.assertTrue(clean)
            self.assertFalse(application_inputs_match(fixture.root, initial, changed))


class BaselineCompletenessTests(unittest.TestCase):
    def test_full_classification_still_requires_r0_evidence_fields(self) -> None:
        fixture = LedgerFixture()
        first = dict(fixture.entry, state="inventoried", last_receipt=None)
        second = dict(first, id="second", source_sha256="hash-2")
        ledger = fixture.run([first, second])
        with self.assertRaisesRegex(
            ValueError, "R0 evidence fields; 2 scenarios incomplete"
        ):
            require_classified(ledger)


class LedgerOutputTests(unittest.TestCase):
    def test_atomic_output_round_trips_without_partial_content(self) -> None:
        import json
        import tempfile
        from behavior_ledger import write_ledger

        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "ledger.json"
            path.write_text("old content")
            write_ledger(path, {"source_revision": "revision", "scenarios": []})
            self.assertEqual(
                json.loads(path.read_text()),
                {"source_revision": "revision", "scenarios": []},
            )
            self.assertEqual(
                [entry.name for entry in path.parent.iterdir()], ["ledger.json"]
            )


class LedgerDriftTests(unittest.TestCase):
    def test_stale_generated_artifact_is_reported_before_missing_classifications(
        self,
    ) -> None:
        from behavior_cli import check_generated

        ledger = LedgerFixture().run([])
        with self.assertRaisesRegex(ValueError, "expected current generated rows"):
            check_generated(ledger, {"scenarios": []})
        with self.assertRaisesRegex(ValueError, "2 missing"):
            check_generated(ledger, ledger)


class StaleReceiptVerifier:
    def __call__(
        self,
        root: Path,
        artifact: object,
        row: dict[str, object],
        source: str,
        commit: str,
    ) -> None:
        raise StaleEvidence("application inputs changed")


class StaleEvidenceTests(unittest.TestCase):
    def test_changed_application_inputs_downgrade_the_claim_without_stopping_report(
        self,
    ) -> None:
        fixture = LedgerFixture()
        ledger = reconcile(
            fixture.inventory,
            {"version": 1, "scenarios": [fixture.entry]},
            "app",
            True,
            Path("."),
            StaleReceiptVerifier(),
            FakeReceiptIntegrity(),
        )
        row = ledger["scenarios"][0]
        self.assertEqual(row["state"], "inventoried")
        self.assertTrue(row["stale"])
        self.assertIn("application inputs changed", row["stale_reason"])
        self.assertEqual(row["historical_receipt"], fixture.entry["last_receipt"])


class HistoricalIntegrityTests(unittest.TestCase):
    def test_stale_source_does_not_hide_a_missing_receipt(self) -> None:
        fixture = LedgerFixture()
        stale = dict(fixture.entry, source_sha256="old", last_receipt=None)
        with self.assertRaisesRegex(ValueError, "expected valid evidence"):
            fixture.run([stale])
