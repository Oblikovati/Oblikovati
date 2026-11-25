# X11 frame pacing after the DPI fix

The optimized build at `ab5905c1b` used Martensite v0.14.1 on Radeon 8060S
with RADV Mesa 26.1.6. It included a local change to the visibility of a
test-only module in the GUI filesystem code. Oblikovati ran through the X11
backend in an isolated rootful XWayland session with DRI3. Both DPI settings
used the same 1440×900 physical client and Vulkan swapchain. X resources
explicitly set 96 or 144 DPI, and each swapchain trace verified immediate
presentation. The physical output refresh was 59.984 Hz.

The initial series below ran alongside other local builds. The repeat series
recorded Linux CPU and I/O pressure before each trial. The large variation
within a single scale prevents attributing a throughput change to DPI or
validation from this series.

| Workload | 100% first FPS / p95 | 100% repeat FPS / p95 | 150% first FPS / p95 | 150% repeats FPS / p95 |
| --- | ---: | ---: | ---: | ---: |
| Continuous redraw | 47.0 / 28.59 ms | 11.5 / 272.84 ms | 65.6 / 20.33 ms | 19.4–20.1 / 104.75–138.22 ms |
| Orbit drag | 26.3 / 62.20 ms | 5.3 / 426.49 ms | 25.0 / 88.31 ms | 9.7–11.8 / 201.48–207.91 ms |
| Face hover and selection | 28.9 / 52.87 ms | 3.8 / 547.62 ms | 11.2 / 140.19 ms | 4.2–8.2 / 280.69–436.09 ms |
| Animated ViewCube turns | — | 20.4 / 83.65 ms | 38.2 / 35.86 ms | 13.0–37.9 / 35.04–163.71 ms |

Every interval in the first-pass default 150% workloads exceeded the 8.33 ms
budget for 120 FPS. The [earlier v0.14.1 baseline](../2026-11-25-martensite-v0141-performance/README.md)
remains the reference for the lower-load hardware run. It reached 134.9 FPS in
continuous redraw with reduced validation checks and 88.5 FPS at default
settings. The separate reduced-checks runs here also missed the target under
load.

An additional 150% orbit on the [two-body rounded cylinder](rounded-cylinder.png)
measured 6.2 FPS with a 325.60 ms p95 under contention. This is a fixture
coverage result. The [selection capture](selected-face.png) shows a cyan face
highlight and `1 Face` in the status line after the pointer moved away. The
[150% animation capture](animation-enabled.png) and the
[100% repeat capture](animation-enabled-100.png) show the enabled ViewCube setting.
The 100% animation startup in the first series did not apply the setting, so
that trial is omitted.

## CPU and presentation

Independent `perf record` runs at 150% with default validation collected
788 user-space cycle samples in orbit and 810 in selection. The flat
`oblikovati_gui::snapshot::capture_frame` symbol accounted for 10.35% of
sampled orbit cycles and 7.43% during selection. The four largest allocator
symbols together accounted for 25.75% and 20.98% respectively. Those allocator
percentages include all application threads and their callers. The Vulkan
validation layer accounted for 7.05% and 4.12% respectively.
[Orbit symbols](orbit-cpu.txt) and [selection symbols](select-cpu.txt).

The first-pass 150% p95 for time spent inside `vkQueuePresentKHR` was
0.179 ms for continuous redraw, 0.183 ms for orbit, 0.220 ms for selection,
and 0.287 ms for animated turns. The call latency excludes execution on the
GPU after the call returns and physical display latency.

In a separate 150% trace, the longest orbit frame interval was 884.62 ms
and the longest selection interval was 337.59 ms. The summed time in image
acquisition, queue submission, fence waits, and presentation was 1.44 ms
and 1.39 ms inside those intervals. Continuous redraw had a 902.19 ms gap,
with 6.71 ms summed inside those calls. Application work, host scheduling,
and other blocking calls account for the unmeasured portion.

The ViewCube scheduler uses a 16.67 ms sample tick and waits for the
previous frame to present. Its cadence needs to change for transitions
to reach 120 distinct camera samples per second.

The CPU profile identifies GUI draw-data capture as a first candidate for
an isolated optimization. The current path builds a per-command index vector
and a tree mapping during vertex compaction. An A/B measurement should use
the same physical pixel extent, validation settings, and pointer input on a
quiet machine, and verify the submitted draw data alongside frame pacing.

## Reproduction

Use the [Vulkan probe](../2026-11-25-martensite-v0141-performance/probe/) from
the original performance receipt. Build the desktop with
`cargo build --release --locked -p oblikovati-head --bin oblikovati`. Compile
the probe with `cc -shared -fPIC -O2 -o libpresent_probe.so present_probe.c`
beside its JSON manifest in a writable copy of the probe directory.

Start the isolated X11 session with `with-private-xwayland.sh`. Inside that
session, run `printf 'Xft.dpi: 144\n' | xrdb -merge` and pass `--dpi 144` to
`profile.py`. Use 96 in both places for the 100% cases. Supply the binary, the
model file, an unused output directory, and one of `continuous`, `orbit`,
`select`, or `cube`. Add `--frames 800` for continuous redraw, `--cpu`
for the separate sampling runs, or `--phases` for the optional
acquisition, submission, and fence timing. Reduced-checks trials record their
`VK_LAYER_DISABLES` values in each `run.json`.

The [raw traces](runs/) retain timestamps and Vulkan results. Run
`python3 probe/summarize.py` with the paths to those trial directories to
recompute the measurements. The frame interval distribution includes adjacent
`VK_SUCCESS` calls within each workload, trimming the first 200 successful
presents for continuous redraw. The orbit and selection drivers send six
seconds of XTest input at a nominal 240 Hz. The ViewCube driver measures
eleven separate animation bursts. Frame intervals report application
presentation pacing, and the selection workload uses redraw-on-demand.

Fixture SHA-256 digests

- `docking-visible-solid-current.opd` · `54a6f807b8084cc19061bce4079aaee3d4cefc03eaad2ac33123544997999f8a`
- `fillet-two-bodies-rounded-cylinder.opd` · `aab0cf09f7094188e31ab679c704257b19aa146715b73e6bb43aa0c63d844095`
