# SPDX-License-Identifier: GPL-2.0-only
"""Send a real Return key through the X11 test display."""
import ctypes
import ctypes.util
import time


def press_enter():
    x11 = ctypes.CDLL(ctypes.util.find_library('X11'))
    xtst = ctypes.CDLL(ctypes.util.find_library('Xtst'))
    x11.XOpenDisplay.restype = ctypes.c_void_p
    x11.XKeysymToKeycode.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
    x11.XFlush.argtypes = [ctypes.c_void_p]
    x11.XCloseDisplay.argtypes = [ctypes.c_void_p]
    xtst.XTestFakeKeyEvent.argtypes = [ctypes.c_void_p, ctypes.c_uint, ctypes.c_int, ctypes.c_ulong]
    display = x11.XOpenDisplay(None)
    assert display, 'expected an X11 display for keyboard input'
    key = x11.XKeysymToKeycode(display, 0xff0d)
    assert key, 'expected a Return keycode'
    xtst.XTestFakeKeyEvent(display, key, 1, 0)
    x11.XFlush(display)
    time.sleep(.1)
    xtst.XTestFakeKeyEvent(display, key, 0, 0)
    x11.XFlush(display)
    x11.XCloseDisplay(display)
