# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for validating the saved dock group and viewport."""
from pathlib import Path
from subprocess import CompletedProcess
import unittest
from unittest.mock import call, patch

import docking_process_restart as restart


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
                        restart.bright_pixel_fraction(Path("after.png"), restart.FIXTURE_FACE_CROP)
                else:
                    self.assertEqual(
                        restart.bright_pixel_fraction(Path("after.png"), restart.FIXTURE_FACE_CROP),
                        expected,
                    )

    def test_checks_face_contrast_against_the_viewport_background(self) -> None:
        image = Path("after.png")
        with patch.object(restart, "bright_pixel_fraction", side_effect=[1.0, 0.0]) as sample:
            result = restart.check_visible_part(image)
        self.assertEqual(result, {"face_bright_fraction": 1.0, "background_bright_fraction": 0.0})
        self.assertEqual(sample.call_args_list, [
            call(image, restart.FIXTURE_FACE_CROP),
            call(image, restart.VIEW_BACKGROUND_CROP),
        ])

    def test_rejects_blank_frames_and_bright_overlays(self) -> None:
        for samples in ([0.0, 0.0], [1.0, 1.0]):
            with self.subTest(samples=samples):
                with patch.object(restart, "bright_pixel_fraction", side_effect=samples):
                    with self.assertRaisesRegex(AssertionError, "bright face"):
                        restart.check_visible_part(Path("after.png"))


if __name__ == "__main__":
    unittest.main()
