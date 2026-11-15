# SPDX-License-Identifier: GPL-2.0-only
"""Exercise the native runner's report ownership boundary."""

import contextlib
import io
import tempfile
import unittest
from pathlib import Path

from capture_cli import NativeCargoRun


class FakeCargoProcess:
    """A native runner that refuses a report directory containing prior artifacts."""

    def __init__(self, command: list[str], **options: object) -> None:
        environment = options["env"]
        assert isinstance(environment, dict)
        self.reports = Path(environment["OBK_BDD_REPORTS"])
        if list(self.reports.iterdir()):
            raise ValueError(f"reports {self.reports}: expected an empty directory")
        self.stdout = iter(["native output\n"])

    def __enter__(self) -> "FakeCargoProcess":
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def wait(self) -> int:
        return 0


class FakeFailingCargoProcess(FakeCargoProcess):
    def wait(self) -> int:
        return 7


class FakeConcurrentLogProcess(FakeCargoProcess):
    def wait(self) -> int:
        (self.reports / "native-test.log").write_text("concurrent evidence\n")
        return 0


class NativeCargoRunTests(unittest.TestCase):
    def test_child_sees_empty_reports_and_log_is_retained(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            with contextlib.redirect_stdout(io.StringIO()):
                status = NativeCargoRun(reports, "application", FakeCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(status, 0)
            self.assertEqual((reports / "native-test.log").read_text(), "native output\n")

    def test_failed_child_output_is_still_recorded(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            with contextlib.redirect_stdout(io.StringIO()):
                status = NativeCargoRun(reports, "application", FakeFailingCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(status, 7)
            self.assertEqual((reports / "native-test.log").read_text(), "native output\n")

    def test_concurrent_log_is_not_overwritten(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            with contextlib.redirect_stdout(io.StringIO()):
                with self.assertRaises(FileExistsError):
                    NativeCargoRun(reports, "application", FakeConcurrentLogProcess)(
                        reports, {"OBK_BDD_REPORTS": str(reports)}
                    )
            self.assertEqual((reports / "native-test.log").read_text(), "concurrent evidence\n")

    def test_other_existing_artifacts_are_left_untouched(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            existing = reports / "existing.json"
            existing.write_text("prior evidence\n")
            with self.assertRaisesRegex(ValueError, "expected an empty directory"):
                NativeCargoRun(reports, "application", FakeCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(existing.read_text(), "prior evidence\n")
            self.assertFalse((reports / "native-test.log").exists())

    def test_existing_log_is_not_overwritten(self) -> None:
        with tempfile.TemporaryDirectory() as folder:
            reports = Path(folder)
            log = reports / "native-test.log"
            log.write_text("prior evidence\n")
            with self.assertRaises((OSError, ValueError)):
                NativeCargoRun(reports, "application", FakeCargoProcess)(
                    reports, {"OBK_BDD_REPORTS": str(reports)}
                )
            self.assertEqual(log.read_text(), "prior evidence\n")


if __name__ == "__main__":
    unittest.main()
