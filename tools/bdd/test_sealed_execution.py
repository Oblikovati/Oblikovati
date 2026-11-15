# SPDX-License-Identifier: GPL-2.0-only
"""Test the closed-filesystem runner and its fail-closed checks."""

import os
import sys
import tempfile
import unittest
from pathlib import Path

from sealed_execution import SealedRequest, run_process, run_sealed


class FakeSourceVerifier:
    """Record source checks; example: FakeSourceVerifier()."""

    def __init__(self) -> None:
        self.calls: list[Path] = []
        self.identity = "a" * 40
        self.change_after_first = False

    def __call__(self, checkout: Path) -> str:
        self.calls.append(checkout)
        if self.change_after_first and len(self.calls) > 1:
            return "b" * 40
        return self.identity


class FakeSealedProcess:
    """Record the sandbox command and optionally change a mounted input."""

    def __init__(self) -> None:
        self.command: list[str] = []
        self.change: Path | None = None
        self.status = 0

    def __call__(self, command: list[str], log: Path) -> int:
        self.command = command
        log.write_text("fixture run", encoding="utf-8")
        if self.change is not None:
            self.change.write_bytes(b"changed")
        return self.status


class SealedFixture:
    """Provide an isolated runtime, runner and source directory."""

    def __init__(self, parent: Path) -> None:
        self.runtime = parent / "runtime"
        self.runtime.mkdir()
        for name in ("scenario", "reports", "tmp", "dev"):
            (self.runtime / name).mkdir()
        (self.runtime / "runner").write_bytes(b"")
        self.binary = parent / "runner-bin"
        self.binary.write_bytes(b"fixture binary")
        self.checkout = parent / "checkout"
        self.checkout.mkdir()
        (self.checkout / "input").write_bytes(b"source")
        self.reports = parent / "reports"
        self.reports.mkdir()
        self.log = parent / "native.log"
        self.bwrap = parent / "bwrap"
        self.bwrap.write_bytes(b"fixture sandbox")
        self.process = FakeSealedProcess()
        self.verify = FakeSourceVerifier()

    def request(self) -> SealedRequest:
        """Make one fixed request; example: fixture.request()."""
        return SealedRequest(
            self.runtime,
            self.binary,
            self.checkout,
            self.reports,
            self.log,
            self.bwrap,
            "application",
            "document-opening",
            "Open a part",
        )


@unittest.skipUnless(
    sys.platform == "linux" and hasattr(os, "O_NOATIME"),
    "requires Linux file descriptors",
)
class SealedExecutionTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.fixture = SealedFixture(Path(self.temporary.name))

    def test_command_closes_the_environment_and_bounds_visible_inputs(self) -> None:
        request = self.fixture.request()
        proof = run_sealed(request, self.fixture.process, self.fixture.verify)
        command = self.fixture.process.command
        self.assertIn("--unshare-all", command)
        self.assertIn("--unshare-user", command)
        self.assertIn("--disable-userns", command)
        self.assertIn("--clearenv", command)
        self.assertNotIn("--proc", command)
        self.assertEqual(
            self.fixture.verify.calls, [request.checkout, request.checkout]
        )
        self.assertEqual(proof["runtime_coverage"], "unbounded")
        self.assertFalse(proof["reusable"])
        self.assertIn("input", proof["source_namespace"])

    def test_runtime_mutation_is_refused(self) -> None:
        request = self.fixture.request()
        self.fixture.process.change = self.fixture.runtime / "runner"
        with self.assertRaisesRegex(ValueError, "runtime inputs"):
            run_sealed(request, self.fixture.process, self.fixture.verify)

    def test_source_change_is_refused(self) -> None:
        self.fixture.verify.change_after_first = True
        with self.assertRaisesRegex(ValueError, "source identity"):
            run_sealed(
                self.fixture.request(), self.fixture.process, self.fixture.verify
            )

    def test_source_namespace_mutation_is_refused_even_with_same_revision(self) -> None:
        self.fixture.process.change = self.fixture.checkout / "input"
        with self.assertRaisesRegex(ValueError, "source namespace"):
            run_sealed(
                self.fixture.request(), self.fixture.process, self.fixture.verify
            )

    def test_binary_mutation_is_refused(self) -> None:
        request = self.fixture.request()
        self.fixture.process.change = self.fixture.binary
        with self.assertRaisesRegex(ValueError, "runner input"):
            run_sealed(request, self.fixture.process, self.fixture.verify)

    def test_relative_mount_and_invalid_filter_are_refused(self) -> None:
        from dataclasses import replace

        request = replace(self.fixture.request(), binary=Path("relative"))
        with self.assertRaisesRegex(ValueError, "absolute path"):
            run_sealed(request, self.fixture.process, self.fixture.verify)
        request = replace(self.fixture.request(), feature="../wrong")
        with self.assertRaisesRegex(ValueError, "lowercase feature folder"):
            run_sealed(request, self.fixture.process, self.fixture.verify)
        self.assertEqual(self.fixture.process.command, [])

    def test_existing_report_is_never_overwritten(self) -> None:
        (self.fixture.reports / "old").write_bytes(b"evidence")
        with self.assertRaisesRegex(ValueError, "empty report"):
            run_sealed(
                self.fixture.request(), self.fixture.process, self.fixture.verify
            )
        self.assertEqual((self.fixture.reports / "old").read_bytes(), b"evidence")
        self.assertEqual(self.fixture.process.command, [])

    def test_real_process_clears_inherited_environment_and_preserves_log(self) -> None:
        os.environ["OBK_SEALED_TEST_SECRET"] = "not inherited"
        self.addCleanup(os.environ.pop, "OBK_SEALED_TEST_SECRET", None)
        command = [
            sys.executable,
            "-c",
            "import os; print(os.getenv('OBK_SEALED_TEST_SECRET'))",
        ]
        self.assertEqual(run_process(command, self.fixture.log), 0)
        self.assertEqual(self.fixture.log.read_text().strip(), "None")
        with self.assertRaises(FileExistsError):
            run_process(command, self.fixture.log)
        self.assertEqual(self.fixture.log.read_text().strip(), "None")

    def test_output_under_runtime_or_source_is_refused(self) -> None:
        from dataclasses import replace

        request = replace(self.fixture.request(), log=self.fixture.runtime / "new-log")
        with self.assertRaisesRegex(ValueError, "outside all mounted roots"):
            run_sealed(request, self.fixture.process, self.fixture.verify)
        nested = self.fixture.checkout / "reports"
        nested.mkdir()
        request = replace(self.fixture.request(), reports=nested)
        with self.assertRaisesRegex(ValueError, "disjoint directories"):
            run_sealed(request, self.fixture.process, self.fixture.verify)

    def test_failed_run_is_not_a_proof(self) -> None:
        self.fixture.process.status = 5
        with self.assertRaisesRegex(ValueError, "exit"):
            run_sealed(
                self.fixture.request(), self.fixture.process, self.fixture.verify
            )


if __name__ == "__main__":
    unittest.main()
