# Native GUI draw and preview reuse

The viewport compiler keeps prepared draws for the immediately preceding
GUI layer. Draws reuse the existing immutable geometry when their positions
in the layer, vertices, indices, texture, and clipping match. The compiler
owns a single texture-resource generation, and an extent change rebuilds
the projection and its draws. The regression test compares the retained
vertex storage, a changed neighbor draw, and a resized layer against a
fresh compilation. The test failed before retention and passed with it.

A synthetic CPU measurement used 32 batches of 128 quads at 1440×900. One
batch changed per frame. Each of five repetitions discarded ten warmup frames
and timed 600 calls to `GuiFrameCompiler::layer`. Alternating 1440 and 1441
pixel widths forced a full rebuild without reloading the texture resources.
The median time was **238.2 µs** for rebuilds and **39.7 µs** with retention.
This isolates preparation CPU time from GPU execution and presentation.

The X11 owner also skips repeated tab-preview readback when the same front
region already has pixels from the same scene, camera, surface, and overlay
generations. Output resize, selection or camera updates, and a changed crop
request fresh pixels. Explicit screenshots still use the completed capture.
A failed crop never marks a preview current.

## Native measurements

The optimized builds used Martensite v0.14.1 on Radeon 8060S with RADV Mesa
26.1.6. The baseline was `a27834d3d`. The three build variants stayed on the
same source and toolchain apart from the changes above. Each run opened the
same `docking-visible-solid-current.opd` Part in a fresh profile at 1440×900,
150% desktop scale, and 125% automatic UI scaling. The desktop requested
immediate mode with `OBLIKOVATI_VSYNC=0`; the probe measured the mode selected
by the driver. Default optimized validation settings were used.

| Workload | Baseline FPS / p95 | Prepared draw reuse FPS / p95 | Draw and preview reuse FPS / p95 |
| --- | ---: | ---: | ---: |
| Repeated unchanged redraw | 58.4–86.8 / 16.21–24.34 ms | 63.0–95.1 / 14.65–22.84 ms | 113.8–188.6 / 11.94–17.40 ms |
| Orbit drag | 54.6 / 27.03 ms | 69.8 / 21.63 ms | 52.7 / 29.76 ms |
| Face hover and selection | 65.3–67.9 / 20.58–22.28 ms | 62.6 / 25.63 ms | 69.5–77.8 / 17.48–20.25 ms |

The final build's two unchanged-frame runs each made 398 submissions and
796 fence waits across the measured intervals. The corresponding
prepared-only control made 796 submissions and 1,194 fence waits in its
second run. Both had 398 acquisitions. Preview reuse removes one
submission and one wait per unchanged frame. The 188.6 FPS run still had
an 11.94 ms p95, above the 8.33 ms budget for 120 FPS.

Run-to-run variation leaves the end-to-end benefit of draw reuse alone
unresolved. The orbit result did not improve reliably. The selection
runs used the Mini-Toolbar hidden because preliminary clicks activated
its Zoom Selected action. The [selected face capture](selected-face.png)
shows the face highlighted at the original camera magnification.
[Raw present and phase traces](runs/)

A separate final-build orbit sample collected 890 user-space cycle samples.
The top leaf was `memmove` at 14.85%. Stacks included device-buffer readback
and geometry transfer. `XGetWindowAttributes` accounted for 6.70% inclusive
and resolved through Martensite's X11 window implementation.
These percentages measure CPU samples and do not establish frame wall time.
[Orbit symbols](orbit-cpu.txt)

Use the [shared X11 probe](../2026-11-25-martensite-v0141-performance/probe/)
with `--dpi 144 --observe-present-mode --phases` in a private XWayland
session. Use `--frames 600` for continuous redraw and
`--hide-selection-toolbar` for selection. Orbit and selection use six seconds
of XTest input and preserve their input time intervals. The timing CSVs expose
present-entry spacing and Vulkan API call duration.

## Verification

The viewport library passed 186 tests, the GUI render integration suite passed
12, and the tab-preview unit suite passed 20. Native viewport tests passed 48
with three existing sketch-preview pixel failures. The native owner capture
suite passed all four tests. The same three assertions
failed in the pre-change native test executable on the same isolated Vulkan
software display.

The [Martensite follow-up](../2026-11-25-oblikovati-performance/martensite.md)
tracks renderer-side capture and instrumentation.
