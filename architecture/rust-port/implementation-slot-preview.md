# Native slot outline — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go `head/ui/sketch_placement_overlay.go` and `theme/dark.xml` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The owner publishes its current placement recipe and sketch plane after input
and queued document work. The native window renders this transient outline before
presentation. Commit, cancellation, or invalid document context removes it. Script
and native add-in owners forward the same packet. Preview failure clears the old
outline and shows a command notice; it does not commit geometry or stop the tool.
A successful retry removes the old preview error without clearing other notices.

The adapter uses the existing placement constructor in a private temporary history.
It passes each resulting bounded line or arc through the existing curve conversion
and the released kernel `QuasiUniformDeflection` sampler. No application document,
allocator, save state or undo history is borrowed. Returned members retain recipe
order and construction roles. Sampled coordinates are presentation inputs only.

The native view supplies world distance per framebuffer pixel. The requested chord
error is half a pixel, with a 4,096-point limit per curve. A conservative full-circle
admission bound rejects excessive allocation before calling the native sampler.
The ratio must also keep the kernel's `1-d` expression distinct from one. This is
an explicit resource and arithmetic limit, not a second curve discretizer or a
model tolerance. No tolerance widening or geometry perturbation is used.

Graphic3d draws solid boundaries and dashed construction lines in the pinned Go
dark theme's white preview color. These native structures are excluded from Fit
bounds and model depth tests. Fit temporarily hides the preview because v0.2.5
includes infinite structures in graphic depth bounds. A guard restores visibility
after success or failure. Replacement structures are owned and removed on
failure, replacement or view retirement. Invalid f32 render positions are refused.
An unchanged packet and pixel error preserve the capture cache. Zoom and resize
refresh the sampling before drawing. Only overlay generation changes for a new
preview; document and scene generations stay unchanged.

Adapter tests cover straight and curved slots, both arc directions, ordered
endpoints and roles, chord error, deterministic repeats, finer sampling, malformed
recipes and resource refusal. Native pixel tests check visible outlines, continuous
boundaries, construction dashes, exact clearing, unchanged scene generation,
replacement failure, zoom, resize and Fit bounds. Owner-loop tests use the real
native camera and renderer, compare outline-on/off pixels with the same prompt,
and verify clearing after commit and Escape without a preview edit to the document.
The captured native outline was also inspected visually.

The existing owner activation entry point remains the way to select this tool.
Menu activation, editable placement fields, dimension witnesses, theme switching,
snapping and other placement tools remain open. This step does not establish full
interactive sketch editing or general named solids. Kernel priorities in
`RustyCADKernel/docs/application-blockers.md` remain unchanged.

Kernel/API pins remain v0.2.5/v0.158.0. There is no local kernel source override.
Work is local only. Windows/macOS native runtime acceptance remains open.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used a 1920×1080×24
Xvfb display, software Vulkan and validation layer 1.4.321. Each crate exceeded
80% line coverage. New preview modules ranged from 85.29% to 100%; the head crate
reached 95.50%. Duplication was 0.81% (578/71,561 scanned lines). The added match
is a blank-document field initializer shared with a test fixture; existing trait
forwarding also grew. No geometry algorithm was copied.

The cross-platform and comment review checked physical pixel units, camera-derived
chord error, bounded native point counts, checked i32 conversion and explicit f32
render-range refusal. Capture checks use the observed extent. Recipe indices stay
local to the temporary graph; no platform-sized index becomes a saved identity.
The comments record why preview structures are excluded during the complete Fit
operation and why the allocation guard precedes the native sampler.
