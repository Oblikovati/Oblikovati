# SPDX-License-Identifier: GPL-2.0-only
"""Regression tests for validating the saved dock group."""
import unittest

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


if __name__ == "__main__":
    unittest.main()
