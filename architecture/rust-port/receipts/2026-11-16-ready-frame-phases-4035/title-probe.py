import ctypes
import ctypes.util
import statistics
import time
x = ctypes.CDLL(ctypes.util.find_library('X11'))
x.XOpenDisplay.argtypes = [ctypes.c_char_p]
x.XOpenDisplay.restype = ctypes.c_void_p
x.XDefaultRootWindow.argtypes = [ctypes.c_void_p]
x.XDefaultRootWindow.restype = ctypes.c_ulong
x.XCreateSimpleWindow.argtypes = [ctypes.c_void_p, ctypes.c_ulong, ctypes.c_int, ctypes.c_int, ctypes.c_uint, ctypes.c_uint, ctypes.c_uint, ctypes.c_ulong, ctypes.c_ulong]
x.XCreateSimpleWindow.restype = ctypes.c_ulong
x.XStoreName.argtypes = [ctypes.c_void_p, ctypes.c_ulong, ctypes.c_char_p]
x.XFlush.argtypes = [ctypes.c_void_p]
x.XDestroyWindow.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
x.XCloseDisplay.argtypes = [ctypes.c_void_p]
d = x.XOpenDisplay(None)
if not d:
    raise RuntimeError('display unavailable')
w = x.XCreateSimpleWindow(d, x.XDefaultRootWindow(d), 0, 0, 20, 20, 0, 0, 0)
try:
    samples = []
    for i in range(11000):
        start = time.perf_counter_ns()
        x.XStoreName(d, w, b'Oblikovati - Part1')
        x.XFlush(d)
        elapsed = time.perf_counter_ns() - start
        if i >= 1000:
            samples.append(elapsed)
    samples.sort()
    print('n=', len(samples), 'p50_us=', samples[5000] / 1000, 'p95_us=', samples[9500] / 1000, 'p99_us=', samples[9900] / 1000, 'max_us=', samples[-1] / 1000)
finally:
    x.XDestroyWindow(d, w)
    x.XCloseDisplay(d)
