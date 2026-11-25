# Native Wayland desktop

Linux selects Wayland when the session advertises a socket and the client can
connect. An unavailable socket falls back to X11. Once a Wayland connection
opens, errors creating its shell window or renderer surface are reported
through the desktop startup path.

Oblikovati owns the xdg-shell window, input queue, and text clipboard.
Martensite v0.14.2 wraps the live surface and presents through Vulkan.
The desktop loop waits on the compositor connection and the existing worker
wake source. Pointer positions use drawable pixels. The viewport uses
fractional-scale and viewporter when the compositor exposes both protocols,
and uses integer buffer scaling otherwise. A compositor close request leaves
the Wayland surface available for the application's save-changes prompt.
The X11 test launcher clears the Wayland socket variables for its XTest cases.
Linux builds require the libxkbcommon development package for keyboard maps.

The main and detached Wayland viewports presented and read back a rendered
scene in headless Weston with no X11 display and zero Vulkan validation errors.
The desktop executable submitted three Wayland frames in that session and
three X11 frames under Xvfb. An advertised but unavailable Wayland socket
also fell back to X11. The interactive development build connected
directly to the COSMIC Wayland socket and was inspected by the user.

Docking checks use the native window on both Linux backends. On Wayland,
the harness sends pointer motion and button transitions through the window
frame loop. On X11, it sends the same drag through XTest. Holding the
button across frames lets the checks observe edge and tab previews before
release. [Docking scenario results](receipts/2026-11-25-docking-pointer-backends-4133/README.md)

## Performance check

Two interleaved runs per backend presented the same
`docking-visible-solid-current.opd` Part at 1440 × 900. Both used Vulkan
Lavapipe with four threads, the same debug desktop binary, validation disabled,
and immediate presentation requested by Oblikovati with `OBLIKOVATI_VSYNC=0`.
The Vulkan probe recorded successful presents and confirmed immediate mode
on both Weston and Xvfb. Each run discarded the first 30 of 180 presents.

| Backend | Run | Presents/s | Median interval | 95th-percentile interval |
| --- | ---: | ---: | ---: | ---: |
| Wayland | 1 | 40.6 | 23.57 ms | 28.61 ms |
| X11 | 1 | 36.0 | 24.98 ms | 37.65 ms |
| Wayland | 2 | 46.2 | 21.40 ms | 23.96 ms |
| X11 | 2 | 30.8 | 31.02 ms | 41.26 ms |

The software-rendered continuous runs showed no Wayland slowdown.
The host was also building tests. Hardware 120 Hz, animated camera transitions,
and selection latency remain unmeasured in the paired setup.
[Frame timestamps](receipts/2026-11-25-wayland-cad-performance-3850/)

Run the native Wayland viewport tests on a compositor with a Vulkan validation
layer available.

```sh
cargo test --locked -p oblikovati-viewport --test native wayland:: -- --ignored --test-threads=1
```
