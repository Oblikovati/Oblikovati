# SPDX-License-Identifier: GPL-2.0-only
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time

binary, fixture, folder = map(Path, sys.argv[1:])
folder.mkdir()
folder = folder.resolve()

def windows():
    tree = subprocess.check_output(["xwininfo", "-root", "-tree"], text=True)
    return {match[1] for line in tree.splitlines()
            if (match := re.search(r'(0x[0-9a-f]+) +"[^"]*Oblikovati[^"]*"', line))}

existing = windows()
env = dict(os.environ, XDG_CONFIG_HOME=str(folder / "config"),
           XDG_DATA_HOME=str(folder / "data"), VK_INSTANCE_LAYERS="VK_LAYER_OBK_present_probe",
           VK_ADD_LAYER_PATH=str(Path(__file__).parent),
           VK_ICD_FILENAMES="/usr/share/vulkan/icd.d/radeon_icd.json",
           OBK_PRESENT_IMMEDIATE="1", OBK_PRESENT_TRACE=str(folder / "presents.csv"))
with (folder / "app.log").open("w") as log:
    app = subprocess.Popen([str(binary.resolve()), "--frames", "850", str(fixture.resolve())],
                           env=env, stdout=log, stderr=subprocess.STDOUT)
    try:
        for _ in range(70):
            if app.poll() is not None:
                raise RuntimeError("Finished before the client window could be inspected")
            created = windows() - existing
            if len(created) == 1:
                xid, = created
                break
            time.sleep(0.08)
        else:
            raise RuntimeError("Could not identify the newly opened Oblikovati window")
        time.sleep(1)
        with (folder / "window.txt").open("w") as stream:
            subprocess.run(["xwininfo", "-id", xid], stdout=stream, check=True)
        subprocess.run(["import", "-window", xid, str(folder / "window.png")], check=True)
        if app.wait(timeout=40) != 0:
            raise RuntimeError("Direct desktop run failed")
    finally:
        if app.poll() is None:
            app.terminate()
            app.wait(timeout=7)
(folder / "run.json").write_text(json.dumps({"mode": "continuous", "fixture": fixture.name}) + "\n")
