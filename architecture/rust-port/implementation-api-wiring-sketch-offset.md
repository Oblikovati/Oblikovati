# Sketch offset API wiring

API 0.166.0 `sketch.offset` supports signed offsets of single lines,
circles, and arcs, mitered offsets of connected line chains, and rounded
offsets of the outer boundary of a certified profile. It resolves the
distance as a document length expression. The new entities share one
undoable edit. The closed-profile path uses the pinned Martensite planar
wire builder and authors the requested corner spans as sketch lines.

The typed tests check signed line and radial offsets, the shared point
at a chain miter, failure on a collapsed radius, and undo and redo. The
adapter regression checks a profile on a YZ sketch, including inward,
outward and zero distances, and all 68 segments of a rounded rectangle
with 16 spans per corner.

## Remaining Martensite certification limit

The original `offset`, `offsetconcave`, and `hdmi` live drivers now
create their offsets. Their next read, `list_sketch_profiles`, exceeds
the maximum curve count in Martensite's signed boundary area
integration. The diagnostics show 68 curves for the rectangle, 86
for the concave L, and 102 for the HDMI shell against a limit of 64.
The host preserves the supplied `arcSegments` value.

A separate native Wayland run changed only `arcSegments` from 16 to 8
and added profile-query logging. All three cases passed through the
MCP Bridge with certified annular regions and the expected extrusion
volumes. The original 16-span workflows need Martensite to certify
the larger closed boundaries at the existing geometric tolerances.
