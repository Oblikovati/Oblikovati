# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for validating the saved dock group and viewport."""
from pathlib import Path
from subprocess import CompletedProcess
from tempfile import TemporaryDirectory
import unittest
from unittest.mock import call, patch

import docking_process_restart as restart

DOCUMENT_AREA = (248, 149, 1041, 850)


def settings(material: str = "0x00000004,0", appearance: str = "0x00000004,1") -> restart.WorkspaceSettings:
    return {
        "imgui_ini": (
            f"[Window][Material Browser]\nPos=10,20\nSize=100,200\nDockId={material}\n\n"
            f"[Window][Appearance Browser]\nPos=10,20\nSize=100,200\nDockId={appearance}\n\n"
        ),
        "material_browser": True,
        "appearance_browser": True,
    }


class DockGroupTests(unittest.TestCase):
    def test_distinct_tabs_in_one_dock_are_accepted(self) -> None:
        panels = restart.check_group(settings())
        self.assertEqual([panel["tab"] for panel in panels], [0, 1])
        self.assertEqual(panels[0]["dock"], panels[1]["dock"])

    def test_split_or_duplicate_tabs_are_rejected(self) -> None:
        for appearance in ("0x00000005,1", "0x00000004,0", "0x00000000,1"):
            with self.subTest(appearance=appearance), self.assertRaises(ValueError):
                restart.check_group(settings(appearance=appearance))

    def test_hidden_or_missing_panels_are_rejected(self) -> None:
        hidden = settings()
        hidden["material_browser"] = False
        with self.assertRaises(ValueError):
            restart.check_group(hidden)
        missing = settings()
        missing["imgui_ini"] = ""
        with self.assertRaises(ValueError):
            restart.check_group(missing)


class ViewportVisibilityTests(unittest.TestCase):
    def test_reads_an_image_fraction_and_rejects_invalid_output(self) -> None:
        for output, expected in (("0.5", 0.5), ("nan", None), ("2", None), ("unknown", None)):
            with self.subTest(output=output), patch.object(
                restart.subprocess, "run", return_value=CompletedProcess(["convert"], 0, output)
            ):
                if expected is None:
                    with self.assertRaisesRegex(ValueError, "expected a pixel fraction"):
                        restart.bright_pixel_fraction(Path("after.png"), "120x80+671+392")
                else:
                    self.assertEqual(
                        restart.bright_pixel_fraction(Path("after.png"), "120x80+671+392"),
                        expected,
                    )

    def test_checks_face_contrast_against_the_viewport_background(self) -> None:
        image = Path("after.png")
        with patch.object(restart, "bright_pixel_fraction", side_effect=[1.0, 0.0]) as sample:
            result = restart.check_visible_part(image, DOCUMENT_AREA)
        self.assertEqual(result, {"face_bright_fraction": 1.0, "background_bright_fraction": 0.0})
        self.assertEqual(sample.call_args_list, [
            call(image, "120x80+671+392"),
            call(image, "120x80+474+392"),
        ])

    def test_rejects_blank_frames_and_bright_overlays(self) -> None:
        for samples in ([0.0, 0.0], [1.0, 1.0]):
            with self.subTest(samples=samples):
                with patch.object(restart, "bright_pixel_fraction", side_effect=samples):
                    with self.assertRaisesRegex(AssertionError, "bright face"):
                        restart.check_visible_part(Path("after.png"), DOCUMENT_AREA)

    def test_rejects_a_probe_outside_the_document_area(self) -> None:
        with self.assertRaisesRegex(ValueError, "inside document area"):
            restart.viewport_crop((0, 0, 120, 80), restart.FIXTURE_FACE_OFFSET)


class WaylandProfileTests(unittest.TestCase):
    def test_successful_exit_without_a_profile_write_is_rejected(self) -> None:
        with TemporaryDirectory() as directory, patch.object(
            restart.subprocess, "run", return_value=CompletedProcess([], 0)
        ) as child:
            profile = Path(directory)
            (profile / "workspace-layout.json").write_text("{}")
            with self.assertRaisesRegex(RuntimeError, "without saving"):
                restart.run_wayland_phase(Path("oblikovati"), Path("part.opd"), profile)
            self.assertEqual(child.call_args.args[0], ["oblikovati", "--frames", "20", "part.opd"])
            self.assertNotIn("DISPLAY", child.call_args.kwargs["env"])


if __name__ == "__main__":
    unittest.main()
