# Interactive viewport preview readback

Oblikovati now reuses status projections while pointer position, extent,
camera and surface generations, and sketch plane match. The selected face's
label and toolbar share their projection when both use the same world point.

For a changed tab thumbnail on X11, the viewport uses Martensite's packed
RGB buffer dump and reuses the destination pixel map. The GUI samples its
64×36 preview directly from borrowed pixels. If the packed route declines,
the existing checked capture supplies the result or reports the renderer's
typed failure. Explicit screenshots retain their full RGBA capture.
Detached X11 drawings present without reading pixels for which they have
no consumer.

At 1440×900, a full RGBA-plus-depth capture carries 10,368,000 bytes across
its two planes. The packed destination holds 3,888,000 bytes. These are
logical buffer sizes. Individual GPU transfers and renderer allocations
can differ.

## Native results

The test compared optimized builds on Radeon 8060S with RADV Mesa 26.1.6
and Martensite v0.14.1. Each app opened the same saved Part in an isolated
XWayland session with a fresh profile, 1440×900 drawable, 144 DPI, and 125%
automatic UI sizing. `OBLIKOVATI_VSYNC=0` requested immediate presentation,
and the probe verified the selected mode. The selection runs disabled
the Mini-Toolbar to avoid activating a command from the pointer trace.

| Workload | `d984ea78b` FPS / p95 | Projection reuse FPS / p95 | Projection reuse and RGB FPS / p95 |
| --- | ---: | ---: | ---: |
| Orbit | 55.5–74.6 / 17.36–18.05 ms | 66.1 / 19.68 ms | 162.9–196.4 / 6.94–9.01 ms |
| Face hover and selection | 52.8–68.8 / 21.14–26.51 ms | 61.2 / 22.36 ms | 85.9–123.3 / 16.52–23.51 ms |
| Animated ViewCube turns | 103.3–112.1 / 11.39–12.20 ms | — | 164.1–206.2 / 6.31–13.29 ms |

Orbit and ViewCube turns averaged over 120 FPS in both RGB trials. Each
workload met the 8.33 ms p95 budget in one trial. The faster runs placed
1,151 of 1,176 orbit intervals and 778 of 793 ViewCube intervals within
budget. Selection exceeded 120 FPS on average in the faster run, with p95
remaining above budget in both trials. The projection-only results fall
within the earlier run-to-run variation. Throughput measures app presents
on the 59.984 Hz test display.

Use the [shared present probe](../2026-11-25-martensite-v0141-performance/probe/)
with Xft.dpi set to 144 and `--dpi 144 --observe-present-mode`. Add
`--hide-selection-toolbar` for selection and `--cube-ui-scale 125` for
eleven animated ViewCube turns. Timing uses successful present-entry
intervals within each recorded workload. [Raw traces](runs/) ·
[Selected face](selected-face.png) · [Animation setting](animation-enabled.png)

## Verification

The viewport test compared every packed RGB pixel with the full capture at
320×240 and at 319×233 after native resize. It also verified that an explicit
capture still returned the same frame. The thumbnail tests compared packed
RGB and RGBA crops. The projection unit tests exercised relevant cache
invalidation and the shared face anchor, and the native “Select a solid face”
scenario passed all six steps. Four native screenshot tests and the live X11
DPI test for the main and detached windows passed. The native ViewCube
scenario passed all seven steps and verified intermediate camera positions,
consistent control and model updates, and the final Top view.

The remaining cropped readback and renderer phase work is in the
[Martensite report](../2026-11-25-oblikovati-performance/martensite.md).
