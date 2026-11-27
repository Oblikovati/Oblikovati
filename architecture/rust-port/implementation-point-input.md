# Point mouse placement

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go PointTool and PointRecipe at
aa8297d1b0ac33c024f19e54821536d76e8a447a.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

Point placement now uses the shared mouse path and the
[point recipe](implementation-point-recipe.md). Motion shows the point before
any click. The first press commits one point and ends the tool, as in Go.
Drag release and further clicks cannot create another point without a new
activation. Preview does not allocate document identities or change history.
A projection failure or an edge-on plane leaves the tool available for retry.
Escape, lost focus, unmap and close clear the preview without editing geometry.

Preview members now carry the same point-or-curve display type as committed
entities. Both use one display conversion function. The previous curve-only
preview record and duplicate conversion call are removed. Points use the
existing native marker primitive, with transient bounds and preview color.
Point and curve drawing share structure and group allocation.
Curves keep their line style and analytic sampling. Preview failure retains
the previous valid packet, and clearing preview leaves committed display intact.
No degenerate polyline is used to display a point.

Input tests check hover, direct press without prior motion, one commit per
activation, stable identity through Undo/Redo, cancellation and projection
failure with retry. Adapter tests check independent Go point positions and a
mixed point/line recipe with a shared endpoint. Existing curve preview tests
retain their geometric assertions after the display type change.

Native viewport tests compare the full marker pixel mask before and after
commit on a translated, rotated sketch plane. They check caching, invalid
packet refusal, clear, Fit and zoom. The shared foreground input test now
covers points as well as the existing curve tools through preview, commit,
Finish, Undo/Redo, save, close and reopen. Captures are compared with mapped
X11 pixels. The preview and final captures were inspected visually:
`/tmp/point-input-preview.png` and `/tmp/point-input-committed.png` (1440x900).
They show one point below the existing rectangular fixture. The blue background
belongs to the fixture. Temporary capture code was removed before the gate.

Native menu/ribbon activation, snapping and point selection remain open.
Activation is through the owner tool interface, as for the other placement
tools. This step does not complete R3/R4 or the full port. Windows/macOS runtime
acceptance remains open. Work is local only.

Cross-platform and comment review checked coordinate conversion, checked click
arity, deterministic member order, one identity per commit and removal of the
completed tool before later events. No new numeric tolerance, public wire
contract or kernel algorithm was added. The released marker path supports this
work, so no kernel priority change was needed.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb with software
Vulkan. Every crate exceeded 80% line coverage. Head reached 96.14%, the kernel
adapter 91.65% and viewport 94.25%. Tool dispatch, preview conversion and preview
rendering reached 100%. Duplication was 0.79% (587/74,253 scanned lines, 68 clone
pairs), with no added duplicated lines or pairs. Full evidence is in
`/tmp/point-input-gate-final.log`. This result paragraph was added after the gate
and checked with `make docs-lint`; no code changed afterward.
