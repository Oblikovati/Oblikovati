# Martensite v0.14.1 desktop frame pacing

The optimized Linux desktop at `fe56b318b` was profiled on an AMD Radeon
8060S with RADV Mesa 26.1.6. It opened the published
`docking-visible-solid-current.opd` Part at 1440×900 pixels. The suite also
included the existing local filesystem edit. COSMIC reported a 150% output
scale and a 59.984 Hz monitor.

The probe verified `VK_PRESENT_MODE_IMMEDIATE_KHR` on the native X11 swapchain.
It timestamped calls to `vkQueuePresentKHR` without adding any GPU work. The
numbers measure application presentation throughput and pacing. The monitor's
physical refresh rate still bounds the visible scanout.

| Workload | Default FPS | Default p95 | Reduced checks FPS | Reduced checks p95 |
| --- | ---: | ---: | ---: | ---: |
| Continuous redraw | 88.5 | 14.52 ms | 134.9 | 9.43 ms |
| Orbit drag | 46.8 | 24.58 ms | 79.1 | 14.76 ms |
| Face hover and selection | 39.5 | 35.11 ms | 74.6 | 17.96 ms |
| Animated ViewCube turns | 56.2 | 20.58 ms | 54.8 | 19.53 ms |

The 120 FPS frame budget is 8.33 ms. Reduced validation crossed 120 FPS
only in the continuous redraw workload, with 1,199 of 1,398 intervals at
budget. Native orbit and selection used six seconds of pointer input, and
ViewCube was measured over eleven turns with the animate option enabled.
The [selected face capture](runs/flush-block-select/selected-face.png)
shows the active highlight, and the [animation setting](runs/final-animation/animation-enabled.png)
shows the enabled option. A direct desktop run averaged 84.3 FPS with
14.65 ms p95.

The ViewCube owner schedules one camera sample every 16.67 ms and waits for
the previous presentation. Orbit sampling attributed 16% of CPU cycles to
the validation layer and about 8% to GUI draw-data capture. Allocation and
copying appeared among the largest individual sample buckets. The reduced
run kept the validation layer loaded and disabled its core, thread safety,
API parameter, and object lifetime checks using `VK_LAYER_DISABLES`.
[CPU samples by library](orbit-cpu-by-library.txt)

## Window sizing and DPI

The original test desktop was 1600×1000 and centered the 1440×900 client at
`(80, 50)`. Matching the test desktop to the client placed the window at
`(0, 0)`. The direct client also measured 1440×900, with a zero-width X11
border and an equal-size Vulkan swapchain. ImageMagick reported zero
differing pixels across the matching 1440×800 client regions in the
[direct capture](dpi/direct-client.png) and the
[nested capture](dpi/nested-client.png).

The rootful XWayland `-hidpi` probe failed during the initial Wayland surface
configure. The final compositor screenshot attempt timed out, so the client
capture cannot resolve the reported residual blur at the 150% output scale.
The follow-up found 144 DPI in COSMIC's XSETTINGS manager while the X resource
database omitted `Xft.dpi`. See the [X11 DPI fallback](../../implementation-native-ui-scale.md#linux-x11-display-dpi).

## Reproduction

Build `cargo build --release --locked -p oblikovati-head --bin oblikovati`.
Compile the layer with `cc -shared -fPIC -O2 -o libpresent_probe.so present_probe.c`
in a writable copy of [probe](probe/). Launch `profile.py` with the binary,
fixture, output directory, and `continuous`, `orbit`, `select`, or `cube`
through `with-private-xwayland.sh`. The layer manifest resides beside its
shared library. The [raw traces](runs/) retain results and timing intervals.
Run `python3 probe/summarize.py runs/*` to recompute the table.
