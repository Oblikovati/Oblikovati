# X11 window observations during desktop rendering

Oblikovati now uses one size read per live X11 extent check. A native
configure event still refreshes
the kernel window's fallback rectangle. Oblikovati uses the event's dimensions
for native resize notifications and resize-edge hit testing. Status telemetry
uses the same live extent as GUI layout. Projection and presentation retain
their live checks for an externally delivered resize.

## Attribute calls

The test used the optimized `1a74c3c9c` baseline and the optimized application
changes above on the same Martensite v0.14.1 build. Both ran in fresh
application profiles at 1440×900 in an isolated XWayland display, using
Radeon 8060S with RADV Mesa 26.1.6. X resources selected 144 DPI, and both builds
used the 125% automatic UI scale. Each swapchain requested and used immediate
presentation through `OBLIKOVATI_VSYNC=0`. Selection had the Mini-Toolbar
hidden to keep clicks on the CAD model.

| Workload | Baseline calls per present | Optimized calls per present | Baseline cumulative ms per present | Optimized cumulative ms per present |
| --- | ---: | ---: | ---: | ---: |
| Unchanged redraw | 94.7–95.0 | 63.8–64.2 | 4.02–7.55 | 4.15–5.12 |
| Orbit | 325.9–373.4 | 206.1–207.9 | 11.18–14.08 | 10.62–13.51 |
| Face hover and selection | 371.5–508.9 | 263.6–303.2 | 5.67–21.39 | 9.58–18.28 |

The glibc audit wraps the process's `XGetWindowAttributes` bindings,
including calls reached through X11 bindings and the renderer. It timestamps
each call on entry and return. Counts use the workload's monotonic time
interval, after 200 successful presents in continuous mode, divided by
successful presentation entries in the interval. The elapsed column sums
the call durations, which can include thread scheduling and overlapping calls.
The counts establish the reduction from the application changes. They do not
assign the remaining calls to application, renderer, or driver.

## Frame timing without the X11 audit

| Workload | Baseline FPS / p95 | Optimized FPS / p95 |
| --- | ---: | ---: |
| Unchanged redraw, 650-frame run | 285.6 / 5.61 ms | 299.6 / 6.14 ms |
| Orbit, first pair | 52.9 / 19.15 ms | 61.3 / 27.90 ms |
| Orbit, reverse order | 42.0 / 37.53 ms | 46.8 / 33.38 ms |
| Selection, first pair | 61.8 / 25.08 ms | 54.0 / 29.24 ms |
| Selection, reverse order | 33.8 / 53.35 ms | 38.2 / 46.42 ms |

The 120 FPS interval budget is 8.33 ms. The unchanged redraw run met this
budget at p95. Orbit throughput increased in both pairs, with mixed
changes to tail latency. Selection varied across pairs. The [native capture](selected-face.png)
shows the face highlight, viewport coordinates, and unchanged camera zoom.
[Unaudited timing traces](timing/)

## Reproduction and verification

From this receipt directory, compile the audit with
`cc -shared -fPIC -O2 -Wall -Wextra -Werror -o libx11_audit.so probe/x11_audit.c`.
Use the [shared present probe](../2026-11-25-martensite-v0141-performance/probe/)
and export `OBK_X11_AUDIT` as the absolute audit library path. Pass
`--observe-present-mode --dpi 144`, use `--frames 500` for continuous redraw,
and add `--hide-selection-toolbar` for selection. The work window runs for six
seconds in orbit and selection. Run `python3 probe/summarize_attributes.py`
with the archived run directories. [Raw traces](runs/)

The viewport library passed 186 tests, and 15 targeted native viewport
tests passed on llvmpipe with Xvfb. These include external resize,
GUI composition, pan and orbit, status pixels, and secondary-window events.
The live XSettings and DPI test passed for main and detached windows. The
configure test failed on the baseline when the event size differed from a
fresh server reply. It now verifies the event size and malformed input.

The remaining renderer instrumentation is recorded in the
[Martensite performance follow-up](../2026-11-25-oblikovati-performance/martensite.md).
