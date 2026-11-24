# SPDX-License-Identifier: GPL-2.0-only
from pathlib import Path
import ctypes
import hashlib
import json
import os
import subprocess
import sys
import tempfile
import time

repo, binary, project, fixture, output = map(Path, sys.argv[1:])
sys.path.insert(0, str(repo / 'tools/bdd'))
from docking_process_restart import NativePointer, capture, close_process, wait_for_window

output.mkdir()
pointer = NativePointer()
pointer.x11.XStringToKeysym.argtypes = [ctypes.c_char_p]
pointer.x11.XStringToKeysym.restype = ctypes.c_ulong
pointer.x11.XKeysymToKeycode.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
pointer.x11.XKeysymToKeycode.restype = ctypes.c_uint
pointer.xtest.XTestFakeKeyEvent.argtypes = [
    ctypes.c_void_p, ctypes.c_uint, ctypes.c_int, ctypes.c_ulong,
]

def keys(*names):
    codes = [pointer.x11.XKeysymToKeycode(pointer.display, pointer.x11.XStringToKeysym(name)) for name in names]
    assert all(codes)
    for code in codes:
        pointer.xtest.XTestFakeKeyEvent(pointer.display, code, 1, 0)
    for code in reversed(codes):
        pointer.xtest.XTestFakeKeyEvent(pointer.display, code, 0, 0)
    pointer.x11.XFlush(pointer.display)
    time.sleep(2)

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def crop(path, geometry):
    return subprocess.check_output(['convert', str(path), '-crop', geometry, '+repage', 'rgb:-'])

def capture_at_color(path, geometry, expected, ready=lambda: True):
    for _ in range(30):
        capture(path)
        if crop(path, geometry) == expected and ready():
            return
        time.sleep(0.4)
    raise AssertionError(f'{path.name}: expected {expected.hex()} at {geometry}')

accent = bytes([129, 96, 227])
details = '265x77+846+278'
selection = '350x16+465+320'

def start(profile, phase):
    environment = dict(os.environ, OBK_CONFIG_DIR=profile, OBK_ADDINS_DIR=profile)
    with (output / f'{phase}.log').open('wb') as log:
        return subprocess.Popen([str(binary), str(project), str(fixture)], env=environment,
                                stdout=log, stderr=log)

fixture_hash = sha(fixture)
with tempfile.TemporaryDirectory(prefix='obk-chooser-process-') as profile:
    store = Path(profile) / 'template-selection.json'
    expected = project.parent / 'document-creation-generic-candidate/Metric/Standard (mm).opd'
    assert not store.exists()
    template_hash = sha(expected)
    pids = []
    for phase in ['before', 'after']:
        process = start(profile, phase)
        pids.append(process.pid)
        try:
            wait_for_window(process)
            pointer.move((500, 140))
            keys(b'Control_L', b'n')
            if phase == 'before':
                capture_at_color(output / 'unselected.png', '1x1+500+254', accent)
                pointer.click((520, 325))
                pointer.move((500, 140))
            selected = output / f'{phase}.png'
            unselected = output / 'unselected.png'
            capture_at_color(selected, '1x1+750+324', accent,
                             lambda: crop(selected, details) != crop(unselected, details))
            keys(b'Escape')
            capture(output / f'{phase}-part.png')
            close_process(process, pointer)
            assert Path(json.loads(store.read_text())).resolve() == expected.resolve()
        finally:
            if process.poll() is None:
                process.kill()
                process.wait()
    assert pids[0] != pids[1]
    assert crop(output / 'before.png', details) != crop(output / 'unselected.png', details)
    assert crop(output / 'before.png', details) == crop(output / 'after.png', details)
    assert crop(output / 'before.png', selection) == crop(output / 'after.png', selection)
    assert sha(fixture) == fixture_hash and sha(expected) == template_hash
    (output / 'process-restart.json').write_text(json.dumps({
        'pids': pids, 'binary_sha256': sha(binary), 'fixture_sha256': fixture_hash,
        'template_sha256': template_hash, 'saved_selection': str(expected.relative_to(project.parent)),
        'details_crop': details, 'selection_crop': selection,
        'restored_details_and_selection_match': True,
    }, indent=2) + '\n')
