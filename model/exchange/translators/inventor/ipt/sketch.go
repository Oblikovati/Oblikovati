// SPDX-License-Identifier: GPL-2.0-only

package ipt

// Sketch geometry decoded from PmDCSegment. Coordinates are in cm (database units).
// A sketch entity is a nameless graph node whose payload begins with the 2D-geometry
// tag; a point carries two coordinates and an entity id, a curve a reference/parameter
// block. A line node references its two endpoint points by their (SketchPoint) entity ids.
//
// This file holds the PUBLIC sketch model (the decoded geometry types) and the decode entry
// points (DecodeSketch / DecodeSketches / clusterItems). The byte-level record scan lives in
// sketch_records.go; the record-to-Sketch assembly and reference resolution in sketch_assemble.go.

type Point2D struct{ X, Y float64 }

// Sketch is a decoded 2D sketch: standalone points, lines, circles, arcs, and ellipses.
// Constraints and dimensions are not decoded here (see sketchconstraint.go).
type Sketch struct {
	Points   []Point2D
	Lines    []Line
	Circles  []Circle
	Arcs     []Arc
	Ellipses []Ellipse
	Splines  []Spline
	// Resolved is true when the curve endpoints were reconstructed exactly from their entity
	// references (a faithful loop), false when the convex-ordering fallback guessed the
	// connectivity — which mangles a non-convex profile. Revolve gates on this: a scrambled
	// profile turned about an axis is a blob, so an unresolved revolve yields to the mesh body.
	Resolved bool
	// Plane is where the sketch lives; PlaneOK is false when the file states no placement this
	// layout can read, and the caller must then fall back to XY rather than invent one.
	Plane   SketchPlacement
	PlaneOK bool
	// Construction marks, per curve, whether it is construction (reference) geometry — a centreline
	// or similar, which Inventor draws but does not let bound a region. Each slice runs parallel to
	// the curve slice it names and may be shorter/absent, meaning "none known". This must be
	// carried, not dropped: a construction line left as real geometry CUTS the regions around it
	// (the linkage's centreline split its one region into two, so no profile matched what the file
	// named). Filled by GraphSketches; the cluster decode drops construction curves instead.
	LineConstruction   []bool
	CircleConstruction []bool
	ArcConstruction    []bool
	SplineConstruction []bool
}

// Spline is a decoded 2D sketch spline: its fit points in order, and whether the curve closes
// back to the first. Inventor stores an interpolating (fit) spline through these points, so the
// points lie ON the curve (they are not off-curve control vertices).
type Spline struct {
	Points []Point2D
	Closed bool
}

// LineIsConstruction reports whether line i is construction geometry, tolerating an absent flag
// slice (the decoders that don't carry the flag).
func (s Sketch) LineIsConstruction(i int) bool { return flagAt(s.LineConstruction, i) }

// CircleIsConstruction reports whether circle i is construction geometry.
func (s Sketch) CircleIsConstruction(i int) bool { return flagAt(s.CircleConstruction, i) }

// ArcIsConstruction reports whether arc i is construction geometry.
func (s Sketch) ArcIsConstruction(i int) bool { return flagAt(s.ArcConstruction, i) }

// SplineIsConstruction reports whether spline i is construction geometry.
func (s Sketch) SplineIsConstruction(i int) bool { return flagAt(s.SplineConstruction, i) }

func flagAt(flags []bool, i int) bool { return i < len(flags) && flags[i] }

type Line struct{ A, B Point2D }
type Circle struct {
	Center Point2D
	Radius float64
	// ArcStart/ArcEnd are the circle node's two on-rim endpoints, when it carries a distinct
	// resolvable pair. Inventor serialises a sketch arc as a SketchCircle with an open-flag bit (see
	// arcFlag); a few real arcs carry that pair yet leave the bit CLEAR, so they decode as full
	// circles. These endpoints let the revolve path recover the arc where a full circle would cross
	// the axis — an impossible revolve profile — without touching the global arc/circle discriminator.
	// ArcEndsOK reports the pair is present and distinct.
	ArcStart, ArcEnd Point2D
	ArcEndsOK        bool
}

// Arc is a decoded circular arc: its centre, radius, and the two endpoints (start → end in
// creation order). Minor vs major is not carried; consumers take the minor arc.
type Arc struct {
	Center Point2D
	Radius float64
	Start  Point2D
	End    Point2D
}

// Ellipse is a decoded sketch ellipse: its centre, the unit direction of the major axis, and
// the two semi-axis lengths (cm). Inventor stores the centre BY REFERENCE (like a circle); the
// major-axis direction and both radii are inline.
type Ellipse struct {
	Center    Point2D
	MajorAxis Point2D // unit direction of the major axis
	MajorR    float64
	MinorR    float64
}

// DecodeSketch extracts one part's sketch geometry from PmDCSegment (all entities grouped
// into a single sketch). Used by the decode unit tests on single-sketch parts.
func DecodeSketch(seg []byte) Sketch {
	return assembleCluster(collectItems(seg))
}

// sketchGap is the byte gap between two sketches' entity clusters in PmDCSegment.
const sketchGap = 800

// clusterItems groups the sketch entities into per-sketch runs by byte-offset gap — each sketch's
// geometry sits in its own contiguous run, with a large gap (sketchGap) between sketches.
func clusterItems(seg []byte) [][]sketchItem {
	var clusters [][]sketchItem
	var cur []sketchItem
	prev := -1
	for _, it := range collectItems(seg) {
		if prev >= 0 && it.off-prev > sketchGap {
			clusters = append(clusters, cur)
			cur = nil
		}
		cur = append(cur, it)
		prev = it.off
	}
	if len(cur) > 0 {
		clusters = append(clusters, cur)
	}
	return clusters
}

// DecodeSketches groups a part's sketch entities into separate sketches by clustering on
// their byte offset. Used for multi-feature parts where each feature consumes its own sketch.
func DecodeSketches(seg []byte) []Sketch {
	var out []Sketch
	for _, cl := range clusterItems(seg) {
		if s := assembleCluster(cl); len(s.Points) > 0 || len(s.Lines) > 0 || len(s.Circles) > 0 || len(s.Arcs) > 0 || len(s.Ellipses) > 0 {
			out = append(out, s)
		}
	}
	return out
}
