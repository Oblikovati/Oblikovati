# SPDX-License-Identifier: GPL-2.0-only
import argparse
import ctypes as C
import json
import math
import os
from pathlib import Path
import re
import subprocess
import time

parser = argparse.ArgumentParser()
parser.add_argument("binary", type=Path)
parser.add_argument("model", type=Path)
parser.add_argument("folder", type=Path)
parser.add_argument("mode", choices=["probe", "continuous", "orbit", "select", "cube", "menu"])
parser.add_argument("--hover-center", nargs=2, default=(808, 386), type=int)
parser.add_argument("--cpu", action="store_true")
args = parser.parse_args()
args.folder.mkdir(parents=True, exist_ok=False)
folder = args.folder.resolve()
csv_path = folder / "presents.csv"

def rows():
    if not csv_path.exists():
        return 0
    with csv_path.open() as stream:
        return max(0, sum(1 for _ in stream) - 1)

def wait_window(app):
    for _ in range(160):
        if app.poll() is not None:
            raise RuntimeError(f"Oblikovati exited during startup with status {app.returncode}")
        tree = subprocess.check_output(["xwininfo", "-root", "-tree"], text=True)
        for line in tree.splitlines():
            candidate = re.search(r'(0x[0-9a-f]+) +"[^"]*Oblikovati[^"]*"', line)
            if candidate:
                return candidate.group(1)
        time.sleep(0.05)
    raise RuntimeError("Oblikovati window never mapped")

def screenshot(xid, name):
    subprocess.run(["import", "-window", xid, str(folder / f"{name}.png")], check=True)

def extent(xid):
    info = subprocess.check_output(["xwininfo", "-id", xid], text=True)
    return [int(re.search(rf"{key}:\s*(\d+)", info).group(1))
            for key in ["Absolute upper-left X", "Absolute upper-left Y", "Width", "Height"]]

def prototype(library, name, arguments, result=C.c_int):
    function = getattr(library, name)
    function.argtypes = arguments
    function.restype = result
    return function

x11 = C.CDLL("libX11.so.6")
xtst = C.CDLL("libXtst.so.6")
xopen = prototype(x11, "XOpenDisplay", [C.c_char_p], C.c_void_p)
flush = prototype(x11, "XSync", [C.c_void_p, C.c_int])
close_display = prototype(x11, "XCloseDisplay", [C.c_void_p])
focus = prototype(x11, "XSetInputFocus", [C.c_void_p, C.c_ulong, C.c_int, C.c_ulong])
keysym = prototype(x11, "XStringToKeysym", [C.c_char_p], C.c_ulong)
keycode = prototype(x11, "XKeysymToKeycode", [C.c_void_p, C.c_ulong], C.c_ubyte)
fake_key = prototype(xtst, "XTestFakeKeyEvent", [C.c_void_p, C.c_uint, C.c_int, C.c_ulong])
fake_button = prototype(xtst, "XTestFakeButtonEvent", [C.c_void_p, C.c_uint, C.c_int, C.c_ulong])
fake_motion = prototype(xtst, "XTestFakeMotionEvent", [C.c_void_p, C.c_int, C.c_int, C.c_int, C.c_ulong])

def motion(position):
    fake_motion(display, -1, origin[0] + position[0], origin[1] + position[1], 0)

def key(label, active):
    fake_key(display, keycode(display, keysym(label.encode())), int(active), 0)

def click():
    fake_button(display, 1, 1, 0)
    fake_button(display, 1, 0, 0)

def timed(action, seconds=6, hz=240):
    start_ns = time.monotonic_ns()
    start = start_ns / 1e9
    for index in range(seconds * hz):
        due = start + index / hz
        delay = due - time.monotonic()
        if delay > 0:
            time.sleep(delay)
        action(index / hz, index)
        flush(display, 0)
    return [start_ns, time.monotonic_ns()]

def orbit():
    motion((800, 430))
    key("Shift_L", True)
    fake_button(display, 2, 1, 0)
    flush(display, 0)
    timeline = timed(lambda t, _: motion((800 + int(126 * math.sin(t * 6.7)),
                                          430 + int(76 * math.sin(t * 8.1)))))
    fake_button(display, 2, 0, 0)
    key("Shift_L", False)
    flush(display, 0)
    return timeline

def selection():
    # Tilt the model so the pointer crosses multiple visible face and edge boundaries.
    for label in ["Right", "Up", "Right"]:
        key(label, True)
        key(label, False)
        flush(display, 0)
        time.sleep(0.12)
    time.sleep(0.35)
    screenshot(xid, "selection-start")
    def sample(t, index):
        x, y = args.hover_center
        motion((x + int(117 * math.sin(t * 8.0)), y + int(66 * math.sin(t * 11.0))))
        if index % 20 == 0:
            click()
    return timed(sample)

def press_at(position, button=1):
    motion(position)
    flush(display, 0)
    time.sleep(0.10)
    fake_button(display, button, 1, 0)
    flush(display, 0)
    time.sleep(0.05)
    fake_button(display, button, 0, 0)
    flush(display, 0)
    time.sleep(0.20)

def enable_animation():
    press_at((width - 102, 339), 3)
    press_at((1203, 540))
    press_at((414, 361))
    screenshot(xid, "animation-enabled")
    press_at((461, 574))

def cube():
    enable_animation()
    bursts = []
    for index in range(11):
        target = [width - 102, 267 if index % 2 == 0 else 411]
        motion(target)
        flush(display, 0)
        time.sleep(0.08)
        start_ns = time.monotonic_ns()
        fake_button(display, 1, 1, 0)
        flush(display, 0)
        time.sleep(0.05)
        fake_button(display, 1, 0, 0)
        flush(display, 0)
        time.sleep(0.43)
        bursts.append([start_ns, time.monotonic_ns()])
    return bursts

env = dict(os.environ)
env["XDG_CONFIG_HOME"] = str(folder / "config")
env["XDG_DATA_HOME"] = str(folder / "data")
env["VK_INSTANCE_LAYERS"] = "VK_LAYER_OBK_present_probe"
env["VK_ADD_LAYER_PATH"] = str(Path(__file__).parent)
env["OBK_PRESENT_TRACE"] = str(csv_path)
env["OBK_PRESENT_IMMEDIATE"] = "1"
env.pop("MESA_VK_WSI_PRESENT_MODE", None)
command = [str(args.binary.resolve()), str(args.model.resolve())]
if args.mode == "continuous":
    command += ["--frames", "1600"]
with (folder / "app.log").open("w") as log:
    app = subprocess.Popen(command, env=env, stdout=log, stderr=subprocess.STDOUT)
    metadata = dict(mode=args.mode, app_pid=app.pid, fixture=args.model.name, validation_layers_disabled=os.getenv("VK_LOADER_LAYERS_DISABLE", ""))
    try:
        if args.mode == "continuous":
            if app.wait(timeout=90) != 0:
                raise RuntimeError("Continuous frame run failed")
        else:
            xid = wait_window(app)
            *origin, width, height = extent(xid)
            metadata.update(window_size=[width, height], window_origin=origin)
            display = xopen(None)
            if not display:
                raise RuntimeError("Private X display is unavailable")
            focus(display, int(xid, 16), 1, 0)
            flush(display, 0)
            time.sleep(1)
            screenshot(xid, "before")
            metadata["first_row"] = rows()
            if args.cpu:
                perf = subprocess.Popen(["perf", "record", "-F", "149", "-e", "cycles:u",
                    "--call-graph", "dwarf,8192", "-p", str(app.pid),
                    "-o", str(folder / "perf.data"), "--", "sleep", "6"],
                    stdout=log, stderr=log)
            if args.mode == "orbit":
                metadata["intervals"] = [orbit()]
            elif args.mode == "select":
                metadata["intervals"] = [selection()]
            elif args.mode == "menu":
                motion((width - 102, 339))
                flush(display, 0)
                time.sleep(0.15)
                fake_button(display, 3, 1, 0)
                flush(display, 0)
                time.sleep(0.05)
                fake_button(display, 3, 0, 0)
                flush(display, 0)
                time.sleep(0.25)
                motion((1203, 540))
                flush(display, 0)
                time.sleep(0.1)
                click()
                flush(display, 0)
                time.sleep(0.45)
            elif args.mode == "cube":
                metadata["intervals"] = cube()
            time.sleep(0.2)
            metadata["last_row"] = rows()
            screenshot(xid, "after")
            if args.cpu:
                metadata["perf_exit"] = perf.wait(timeout=10)
            key("Alt_L", True)
            key("F4", True)
            key("F4", False)
            key("Alt_L", False)
            flush(display, 0)
            close_display(display)
            if app.wait(timeout=8) != 0:
                raise RuntimeError("Oblikovati failed to close normally")
    finally:
        if app.poll() is None:
            app.terminate()
            try:
                app.wait(timeout=5)
            except subprocess.TimeoutExpired:
                app.kill()
                app.wait()
        metadata["total_rows"] = rows()
        (folder / "run.json").write_text(json.dumps(metadata, indent=2) + "\n")
