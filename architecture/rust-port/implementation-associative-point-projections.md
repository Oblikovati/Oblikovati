# Owned point projection bindings — partial R3

Tracking: [#3761](https://github.com/Oblikovati/Oblikovati/issues/3761).

Placed sketches store an ordered association from an owned point ID to the
document origin or a stable datum-point feature ID. The source identity is the
driver. Saved planar coordinates remain a warm seed. No evaluated source, frame,
kernel handle, or verification stamp is serialized as a source definition.

Binding edits check existing anchors, unique anchors and sources, and nonzero
feature references before publication. `projected_point_for_source` permits a
creation caller to reuse the existing anchor before reserving another ID. Geometry
replacement cannot delete a bound anchor. Exact binding no-ops retain verification.
Changing or removing a binding clears verification and updates feature reads.

Collection recipe version 4 supports bindings. Versions 1–3 reject nonempty
binding content. Collections without bindings continue to use their earlier
minimum placement/orientation version. Native document schema remains 3. The
package codec preserves the source associations and permanent reservation marks.
Loading rebuilds verification against the current owner before feature use.

The immutable document evaluates the mixed-datum catalogue once for each solve
batch. It resolves every requested anchor with the
[current analytic projection seam](implementation-sketch-point-projection.md).
Host planes, orientation axes, source points, and model-value parameters are
current inputs. The injected input interface requires finite coordinates and
one point per binding in binding order. A missing source or host is a named
projection error that retains the underlying typed decline.

All document solve modes and analysis use the
[exact grounding path](implementation-grounded-sketch-points.md). Grounded
coordinates stay outside the numerical variable universe. Source failures remain
local to their sketch; independent sketches continue. Cancellation discards the
entire candidate batch. Numerical nonconvergence retains the existing adoption
policy and cannot become fully constrained verification.

Verification captures both dimension inputs and current projected coordinates.
Every consuming owner checks current source resolution and exact coordinate bits
against the stamp. A point, host, axis, or source-parameter edit cannot reuse a
stale stamp. Parameter-only sketch health and solving refuse unresolved bindings.
The session uses owner resolution for automatic edits, explicit and selected
solves, analysis, and open-time refresh. Source recovery triggers recompute.

Source feature IDs join the existing sketch read footprint. Datum deletion sees
these consumers, and deleting the last sketch consumer can prune construction
sources through the existing incidence graph. The projection creates no separate
deletion or naming mechanism.

Tests cover current sources, hosts, axes and reversed normals; local source loss
and recovery; stale verification; identity lookup and no-ops; malformed bindings,
injected inputs and recipes; exact signed-zero freshness; current analysis;
cancelled/aborted/failed edits; construction-source deletion; and native YAML
save/reopen with reservation marks. Changing a stored driver refreshes anchors
on open without adding undo steps, and the changed warm seed remains dirty until
saved. An extreme stored seed cannot override the current source.

[Creation and origin preferences](implementation-sketch-creation-api.md) now use
these bindings. Kernel vertex/edge projections need the released reference catalogue.
Explicit break-link policy and projected curves remain separate work. This
step does not establish native application, rendering, or kernel acceptance.
