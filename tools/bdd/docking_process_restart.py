#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Exercise a saved dock group across two desktop processes on a 1440x900 X11 display."""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time
from typing import TypedDict

# These crops sample the front face and adjacent background of the native Part
# fixture after the two browsers occupy the right dock on the 1440 × 900 display.
FIXTURE_FACE_CROP = "120x80+747+343"
VIEW_BACKGROUND_CROP = "120x80+550+343"


class WorkspaceSettings(TypedDict):
    imgui_ini: str
    material_browser: bool
    appearance_browser: bool


class PanelRecord(TypedDict):
    dock: str
    tab: int
    geometry: str


class ViewportSamples(TypedDict):
    face_bright_fraction: float
    background_bright_fraction: float


class RestartResult(TypedDict):
    fixture_sha256: str
    binary_sha256: str
    panels: list[PanelRecord]
    viewport_before: ViewportSamples
    viewport_after: ViewportSamples


class NativePointer:
    """Send real pointer events through XTest; for example, `pointer.click(10, 10)`."""

    def __init__(self) -> None:
        self.x11 = ctypes.CDLL("libX11.so.6")
        self.xtest = ctypes.CDLL("libXtst.so.6")
        self.configure_x11()
        self.display = self.x11.XOpenDisplay(None)
        if not self.display:
            raise RuntimeError("DISPLAY: expected an available X11 display")
        self.check_extent()

    def configure_x11(self) -> None:
        """Configure X11 signatures; for example, `pointer.configure_x11()`."""
        self.x11.XOpenDisplay.restype = ctypes.c_void_p
        self.x11.XOpenDisplay.argtypes = [ctypes.c_char_p]
        self.x11.XFlush.argtypes = [ctypes.c_void_p]
        self.x11.XDefaultScreen.argtypes = [ctypes.c_void_p]
        self.x11.XDisplayWidth.argtypes = [ctypes.c_void_p, ctypes.c_int]
        self.x11.XDisplayHeight.argtypes = [ctypes.c_void_p, ctypes.c_int]
        self.xtest.XTestFakeMotionEvent.argtypes = [
            ctypes.c_void_p, ctypes.c_int, ctypes.c_int, ctypes.c_int, ctypes.c_ulong
        ]
        self.xtest.XTestFakeButtonEvent.argtypes = [
            ctypes.c_void_p, ctypes.c_uint, ctypes.c_int, ctypes.c_ulong
        ]

    def check_extent(self) -> None:
        """Reject displays with unsuitable coordinates; for example, `pointer.check_extent()`."""
        screen = self.x11.XDefaultScreen(self.display)
        extent = (
            self.x11.XDisplayWidth(self.display, screen),
            self.x11.XDisplayHeight(self.display, screen),
        )
        if extent != (1440, 900):
            raise RuntimeError(f"X11 display size {extent}: expected (1440, 900)")

    def move(self, point: tuple[int, int]) -> None:
        """Move the pointer; for example, `pointer.move((500, 140))`."""
        self.xtest.XTestFakeMotionEvent(self.display, -1, *point, 0)
        self.x11.XFlush(self.display)
        time.sleep(0.15)

    def button(self, pressed: bool) -> None:
        """Change the left button state; for example, `pointer.button(True)`."""
        self.xtest.XTestFakeButtonEvent(self.display, 1, pressed, 0)
        self.x11.XFlush(self.display)

    def click(self, point: tuple[int, int]) -> None:
        """Click a screen point; for example, `pointer.click((307, 15))`."""
        self.move(point)
        self.button(True)
        time.sleep(0.12)
        self.button(False)
        time.sleep(1)

    def drag(self, start: tuple[int, int], end: tuple[int, int]) -> None:
        """Drag between screen points; for example, `pointer.drag((1, 1), (10, 10))`."""
        self.move(start)
        self.button(True)
        for step in range(1, 21):
            point = tuple(start[i] + (end[i] - start[i]) * step // 20 for i in range(2))
            self.move(point)
        time.sleep(0.5)
        self.button(False)
        time.sleep(2)


def panel_record(settings: WorkspaceSettings, name: str) -> PanelRecord:
    """Extract a panel record; for example, `panel_record(settings, 'Material Browser')`."""
    section = f"[Window][{name}]\n"
    ini = settings["imgui_ini"]
    if section not in ini:
        raise ValueError(f"window {name!r}: expected a saved ImGui window")
    block = ini.split(section, 1)[1].split("\n\n", 1)[0]
    match = re.search(r"^DockId=(0x[0-9A-Fa-f]+),(\d+)$", block, re.MULTILINE)
    if not match or int(match.group(1), 16) == 0:
        raise ValueError(f"window {name!r}: expected a nonzero saved dock ID")
    return {"dock": match.group(1), "tab": int(match.group(2)), "geometry": block}


def check_group(settings: WorkspaceSettings) -> list[PanelRecord]:
    """Require two visible tabs in one dock; for example, `check_group(saved_settings)`."""
    panels = [
        panel_record(settings, name) for name in ("Material Browser", "Appearance Browser")
    ]
    if not settings["material_browser"] or not settings["appearance_browser"]:
        raise ValueError("browser visibility: expected both material and appearance panels open")
    if panels[0]["dock"] != panels[1]["dock"] or panels[0]["tab"] == panels[1]["tab"]:
        raise ValueError(f"saved panels {panels!r}: expected distinct tabs in one dock")
    return panels


def capture(path: Path) -> None:
    """Capture the X11 screen; for example, `capture(Path('before.png'))`."""
    subprocess.run(["import", "-window", "root", str(path)], check=True)


def compare_panel(before: Path, after: Path) -> None:
    """Check the dock pixels after restart; for example, `compare_panel(a, b)`."""
    result = subprocess.run(
        ["compare", "-metric", "AE", "-crop", "440x694+1000+158", str(before), str(after), "null:"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise AssertionError(f"restored dock pixel difference {result.stderr.strip()!r}: expected 0")


def bright_pixel_fraction(image: Path, crop: str) -> float:
    """Measure the bright area in a crop, for example bright_pixel_fraction(image, FIXTURE_FACE_CROP)."""
    result = subprocess.run(
        ["convert", str(image), "-crop", crop, "+repage", "-colorspace", "gray",
         "-threshold", "48%", "-format", "%[fx:mean]", "info:"],
        check=True,
        capture_output=True,
        text=True,
    )
    output = result.stdout.strip()
    try:
        fraction = float(output)
    except ValueError as cause:
        raise ValueError(f"viewport crop {crop!r}: expected a pixel fraction, got {output!r}") from cause
    if not 0.0 <= fraction <= 1.0:
        raise ValueError(f"viewport crop {crop!r}: expected a pixel fraction from 0 to 1, got {fraction}")
    return fraction


def check_visible_part(image: Path) -> ViewportSamples:
    """Require face and background contrast, for example check_visible_part(Path('after.png'))."""
    face = bright_pixel_fraction(image, FIXTURE_FACE_CROP)
    background = bright_pixel_fraction(image, VIEW_BACKGROUND_CROP)
    if face < 0.95 or background > 0.05:
        raise AssertionError(
            f"viewport capture {image}: face fraction {face}, background fraction {background}; "
            "expected at least 0.95 bright face and at most 0.05 bright background"
        )
    return {"face_bright_fraction": face, "background_bright_fraction": background}


def launch(binary: Path, fixture: Path, environment: dict[str, str]) -> subprocess.Popen[bytes]:
    """Start the desktop; for example, `launch(binary, fixture, environment)`."""
    return subprocess.Popen(
        [str(binary), str(fixture)], env=environment,
        stdout=subprocess.DEVNULL, stderr=subprocess.PIPE,
    )


def close_process(process: subprocess.Popen[bytes], pointer: NativePointer) -> None:
    """Close the desktop through its X button; for example, `close_process(child, pointer)`."""
    pointer.click((1416, 16))
    try:
        code = process.wait(timeout=20)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()
        raise RuntimeError("desktop close: expected the process to exit within 20 seconds")
    if code:
        error = process.stderr.read().decode(errors="replace")
        raise RuntimeError(f"desktop exit {code}: expected zero; {error}")


def arrange(pointer: NativePointer) -> None:
    """Dock two browser panels; for example, `arrange(pointer)`."""
    pointer.click((307, 15))
    pointer.drag((1150, 210), (1420, 480))
    pointer.click((371, 15))
    pointer.drag((1150, 210), (1200, 168))
    pointer.move((500, 140))
    time.sleep(2)


def file_hash(path: Path) -> str:
    """Hash evidence input bytes; for example, `file_hash(Path('fixture.opd'))`."""
    return hashlib.sha256(path.read_bytes()).hexdigest()


def wait_for_window(process: subprocess.Popen[bytes]) -> None:
    """Wait for desktop startup; for example, `wait_for_window(child)`."""
    time.sleep(15)
    if process.poll() is not None:
        error = process.stderr.read().decode(errors="replace")
        raise RuntimeError(f"desktop exit {process.returncode}: expected a running window; {error}")


def run_phase(
    binary: Path, fixture: Path, environment: dict[str, str],
    pointer: NativePointer, output: Path, first: bool,
) -> WorkspaceSettings:
    """Capture one application process; for example, `run_phase(bin, part, env, pointer, out, True)`."""
    process = launch(binary, fixture, environment)
    try:
        wait_for_window(process)
        if first:
            arrange(pointer)
        pointer.move((500, 140))
        capture(output / ("before.png" if first else "after.png"))
        close_process(process, pointer)
        settings = Path(environment["OBK_CONFIG_DIR"]) / "workspace-layout.json"
        return json.loads(settings.read_text())
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()


def restart_in_same_profile(binary: Path, fixture: Path, output: Path) -> list[PanelRecord]:
    """Check the two runs sharing one profile, for example restart_in_same_profile(binary, fixture, output)."""
    pointer = NativePointer()
    with tempfile.TemporaryDirectory(prefix="obk-dock-process-") as config:
        environment = dict(os.environ, OBK_CONFIG_DIR=config, OBK_ADDINS_DIR=config)
        saved = run_phase(binary, fixture, environment, pointer, output, True)
        panels = check_group(saved)
        (output / "saved-layout.json").write_text(json.dumps(saved, indent=2) + "\n")
        restored = run_phase(binary, fixture, environment, pointer, output, False)
        compare_panel(output / "before.png", output / "after.png")
        if check_group(restored) != panels:
            raise AssertionError("restored panel records: expected saved positions, sizes, and tabs")
    return panels


def exercise(binary: Path, fixture: Path, output: Path) -> RestartResult:
    """Test a full restart; for example, `exercise(binary, fixture, receipt_dir)`."""
    output.mkdir(parents=True, exist_ok=True)
    fixture_hash = file_hash(fixture)
    panels = restart_in_same_profile(binary, fixture, output)
    if file_hash(fixture) != fixture_hash:
        raise AssertionError("fixture hash changed during the restart test")
    return {
        "fixture_sha256": fixture_hash,
        "binary_sha256": file_hash(binary),
        "panels": panels,
        "viewport_before": check_visible_part(output / "before.png"),
        "viewport_after": check_visible_part(output / "after.png"),
    }


def main() -> None:
    """Run the isolated process test; for example, `python3 docking_process_restart.py ...`."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("fixture", type=Path)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    result = exercise(args.binary.resolve(), args.fixture.resolve(), args.output.resolve())
    (args.output / "restart-summary.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
