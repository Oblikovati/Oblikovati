# Committed sketch presentation — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go sketch overlays and default dark theme at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override.

Committed planar sketches now have a display path independent of solids and transient
placement outlines. A document/generation change publishes an owned sketch packet.
The owner does not clone unchanged geometry each frame. Finish leaves the packet
visible. Undo, Redo, active document changes and close replace or clear it. A solid
feature failure does not remove the authoring geometry. A missing sketch plane
produces a named local failure while other sketches remain available.

One immutable `SketchReadIndex` resolves entity and defining-point identities for
each batch. Saved entity order controls display. Hidden defining points do not
become standalone entities. The model also supplies circle/arc center markers,
matching Go issue #2159; shared centers and visible point entities are not drawn
twice. The borrowed index cannot outlive an edit. Line
intersection conversion, placement preview and committed display share the same
analytic resolver. Circular arc radius reads share the model's existing formula.
The index adds no persisted cache or alternate geometry owner.

Lines, circles, directed circular arcs and ellipses use the released native curve
adaptor and `QuasiUniformDeflection`. Preview and committed display share this
sampler. The requested deflection is half a view pixel. A resource admission floor
refuses excessively small conic deflections before sampling, and the native output
count must be within the configured limit. The ellipse admission floor is not a
proof of a strict bound on native allocations. Sampled coordinates serve display
only: no topology, naming or picking decision reads them.

The native structure builder is shared. Transient curves remain white and excluded
from Fit, including depth fitting. Committed curves use the Go default dark theme's
`wire_edit` grey, construction lines use dashes, and centerlines use dot-dash.
Standalone points use native circle-plus markers. Committed structures participate
in Fit, including when the scene has no solids. View changes resample curves;
unchanged packets and deflections retain their structures. Owned structure guards
remove retired native objects.

A failed entity conversion or render coordinate produces a sketch/entity diagnostic
without hiding supported entities. Spline construction is still absent from the
pinned release; its decline names kernel issue #9. The native title shows the
failure count and first plane/entity failures. Full diagnostic lists remain
available from the native window. Zoom-dependent failures update this feedback.
No spline approximation or local kernel override was added. The existing kernel
priority-one release dependency remains appropriate.

Model and adapter checks cover saved order, hidden point bits, roles, radius
failures, analytic samples, immutability, repeatability, zoom sampling and partial
failures. Native checks cover visible curves and standalone markers, preview
coexistence, unchanged-packet caching, Fit, zoom, resize, invalid render coordinates,
cleanup and named UI failures. Owner checks cover history, Finish, close and failed
solid evaluation. The live rectangle input tests now retain committed sketch
packets after Finish and continue to compare captures with foreground X11 pixels.

Per-sketch visibility controls, native hit testing, selection highlighting,
per-entity formatting, editable dimension widgets and the other placement tools
remain open. All current authored sketches use the model's visible default.
This step does not establish full R3/R4 completion or Windows/macOS runtime
acceptance. Work remains local only.

The cross-platform review checked saved entity order, identity-based center
suppression, finite binary64 model coordinates, checked binary32 render conversion,
and native signed vertex-count limits. Marker and line patterns use native enums.
The half-pixel sampling policy stays separate from modelling tolerance. The comment
review retained the Go center-marker rationale and transient Fit exclusion. One
overlay publication guard now owns live-state, extent and capture invalidation;
script and add-in wrappers share owner forwarding for scene and edit state.

The full local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used a 1920×1080×24
Xvfb display and software Vulkan. The acceptance image was inspected visually:
the circle, ellipse, solid outline, construction dashes, centerline pattern,
standalone point and circle-center marker are present. Captured geometry pixels
match the default theme's RGB value of 160 in each channel.

Every crate exceeded 80% line coverage. Model reached 98.83%, the native kernel
adapter 91.48%, viewport 94.17% and head 95.62%. Duplication was 0.80%
(587/73,047 scanned lines), below the wave base's 595 duplicated lines. Two
additional default-window contract tests passed after the gate: an unsupported
window accepts a clear request, refuses committed geometry, and cannot silently
drop a plane failure. No implementation changed after the full gate.
