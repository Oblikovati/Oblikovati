// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"sort"
)

// Sketch constraints — the DIMENSIONAL family (M48 #2230 split of sketchconstraint.go). The valued
// dimensions decoded from the constraint/dimension nodes: offset (distance-from-line), two-line angle,
// radius/diameter of a circle or arc, point-to-point distance, revolve radius-from-centreline, and
// axial step-length. Each value is taken from the current geometry, so applying the dimension pins a
// DOF without moving a point. The shared record decoding and geometry predicates live in
// sketchconstraint.go; the value-free geometric relations in sketchconstraint_geometric.go.

// OffsetDim is a decoded offset (distance-from-line) dimension: the perpendicular distance from a
// point to a reference line. Two Inventor forms decode to this: a point-to-line offset, and a
// line-to-line offset between two PARALLEL lines (reduced here to the reference line plus one
// endpoint of the other line, whose perpendicular distance is the line-to-line separation). The
// value is the current perpendicular distance, so applying it never moves geometry.
type OffsetDim struct {
	Line  [2]Point2D // the reference line (r2)
	Pt    Point2D    // the dimensioned point, or an endpoint of the parallel line
	Value float64    // perpendicular distance (cm)
}

// DecodeOffsetDimensions returns the sketch's offset (distance-from-line) dimensions. The node has
// the same head as an angle dimension — first ref = the 0x10 list sentinel, second ref (+40) = a
// line — but its t44 (+44) reference is either a POINT (point-to-line offset) or a LINE that is
// PARALLEL to the second (line-to-line offset). An angle dimension's t44 line is NON-parallel, so
// the angle decoder (which drops near-parallel pairs) and this one are disjoint. The value is the
// current perpendicular distance, self-validating.
func DecodeOffsetDimensions(seg []byte) []OffsetDim {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	var out []OffsetDim
	for _, c := range collectRawCons(seg) {
		if c.r1 != emptyListMark || c.r2&refBit == 0 || c.disc&refBit == 0 {
			continue
		}
		line, isLine := le[c.r2&^refBit]
		if !isLine {
			continue
		}
		if pt, ok := vc[c.disc&^refBit]; ok {
			if d := pointLineDistance(pt, line); d > 1e-4 {
				out = append(out, OffsetDim{Line: line, Pt: pt, Value: d})
			}
		} else if other, ok := le[c.disc&^refBit]; ok && linesParallel(line, other) {
			// line-to-line offset: an endpoint of the other (parallel) line to the reference line.
			if d := pointLineDistance(other[0], line); d > 1e-4 {
				out = append(out, OffsetDim{Line: line, Pt: other[0], Value: d})
			}
		}
	}
	return out
}

// pointLineDistance returns the perpendicular distance from a point to a segment's infinite line
// (falling back to point-to-endpoint distance for a degenerate segment).
func pointLineDistance(p Point2D, l [2]Point2D) float64 {
	dx, dy := l[1].X-l[0].X, l[1].Y-l[0].Y
	length := math.Hypot(dx, dy)
	if length < 1e-9 {
		return math.Hypot(p.X-l[0].X, p.Y-l[0].Y)
	}
	return math.Abs((p.X-l[0].X)*dy-(p.Y-l[0].Y)*dx) / length
}

// AngleDim is a decoded angle dimension between two lines, resolved to both lines' endpoints and
// the unsigned angle between them in degrees ([0,180]). The value is the current geometric angle,
// so applying AddAngle(l1, l2, "<deg> deg") pins the angle without moving geometry.
type AngleDim struct {
	L1, L2  [2]Point2D
	Degrees float64
}

// DecodeAngleDimensions returns the sketch's two-line angle dimensions. An angle-dimension node has
// its first reference = the 0x10 list sentinel (its label sits at +32, so the first entity ref is
// the empty-list marker, like a radius dim) and its second reference (+40) AND its t44 word (+44)
// BOTH resolve to lines — the two lines it dimensions. A distance dimension's t44 references a text
// point; a symmetry's first two refs are points; a radius/tangent's line-bearing ref is a circle —
// so none of those pass the two-lines gate. The value is the current unsigned angle between the
// lines, so the dimension reproduces without moving geometry. A near-parallel pair (angle ≈ 0/180)
// is dropped: you don't dimension the angle between parallel lines, and it is ill-posed to solve.
func DecodeAngleDimensions(seg []byte) []AngleDim {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	var out []AngleDim
	for _, c := range collectRawCons(seg) {
		if c.r1 != emptyListMark || c.r2&refBit == 0 || c.disc&refBit == 0 {
			continue
		}
		l1, ok1 := le[c.r2&^refBit]
		l2, ok2 := le[c.disc&^refBit]
		if !ok1 || !ok2 {
			continue
		}
		deg := lineAngleDegrees(l1, l2)
		if deg < 1e-2 || deg > 180-1e-2 {
			continue // parallel/anti-parallel — not an angle dimension
		}
		out = append(out, AngleDim{L1: l1, L2: l2, Degrees: deg})
	}
	return out
}

// lineAngleDegrees returns the unsigned angle in degrees ([0,180]) between two segments' directions,
// using the same measure AddAngle solves against: atan2(|cross|, dot).
func lineAngleDegrees(a, b [2]Point2D) float64 {
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[1].X-b[0].X, b[1].Y-b[0].Y
	return math.Atan2(math.Abs(ax*by-ay*bx), ax*bx+ay*by) * 180 / math.Pi
}

// RadiusDim is a decoded radius or diameter dimension of a circle or an arc, resolved to the
// curve's geometry so the translator can bind it by coordinate. Radius and diameter are
// byte-identical in the node (both pin the same degree of freedom — the curve's radius), so both
// decode to this and apply as a radius dimension of the curve's OWN radius; the value equals the
// current radius, so applying it removes a DOF without moving geometry. The label form (r vs ⌀) is
// not recoverable and not needed for DOF parity.
type RadiusDim struct {
	Center Point2D
	Radius float64
	Arc    bool    // true → bind an arc (matched by centre + radius), false → a circle
	Start  Point2D // arc endpoints (zero for a circle)
	End    Point2D
}

// DecodeRadiusDimensions returns the sketch's radius/diameter dimensions. A radius-dimension node
// has t44 == 0 (shared with midpoint) but its first reference is the 0x10 list sentinel — a
// midpoint's first reference is a LINE — and its SECOND reference resolves to a circle or arc
// entity id (circle = centre ref + 1; arc = its highest point ref + 1). The value is the curve's
// own decoded radius, so the dimension reproduces exactly. A node whose second reference resolves
// to neither a circle nor an arc is dropped rather than guessed.
//
// Radius, diameter, AND arc-length dimensions all share this exact node shape at the constraint
// level (t44 == 0, 0x10 sentinel, curve ref) — they are byte-identical apart from positional ids;
// their type lives only in a deep dimension sub-structure this decoder does not read. All three are
// therefore decoded here as a radius of the curve's own radius. This is faithful for DOF parity:
// each removes exactly one degree of freedom and, being self-validated against the current radius,
// moves no geometry — so an arc-length dimension reproduced as an arc-radius dimension yields the
// same DOF and the same solved geometry. The only unrecoverable detail is the label (r / ⌀ / arc
// length), which does not affect parity. (Same reasoning as radius-vs-diameter, which are likewise
// indistinguishable.)
func DecodeRadiusDimensions(seg []byte) []RadiusDim {
	vc := vertexCoords(seg)
	circ := circleByEntityID(seg, vc)
	arcs := arcByEntityID(seg, vc)
	var out []RadiusDim
	for _, c := range collectRawCons(seg) {
		if c.disc != 0 || c.r1 != emptyListMark || c.r2&refBit == 0 {
			continue
		}
		id := c.r2 &^ refBit
		if ce, ok := circ[id]; ok {
			out = append(out, RadiusDim{Center: ce.inline, Radius: ce.radius})
		} else if a, ok := arcs[id]; ok {
			out = append(out, RadiusDim{Center: a.Center, Radius: a.Radius, Arc: true, Start: a.Start, End: a.End})
		}
	}
	return out
}

// arcByEntityID maps each arc's ENTITY id to the arc with its centre and endpoints resolved and its
// inline radius. Inventor allocates the arc entity right after its centre/start/end point nodes, so
// the entity id is the highest of those three point references + 1 — the same "+1 after the
// constituent point(s)" rule circleByEntityID uses for a circle's single centre. This is the id a
// radius dimension uses to name the arc.
func arcByEntityID(seg []byte, vc map[uint32]Point2D) map[uint32]Arc {
	m := map[uint32]Arc{}
	for _, it := range collectItems(seg) {
		if it.arc == nil {
			continue
		}
		a := it.arc
		arc := Arc{Radius: a.radius}
		if p, ok := vc[a.center]; ok {
			arc.Center = p
		}
		if p, ok := vc[a.start]; ok {
			arc.Start = p
		}
		if p, ok := vc[a.end]; ok {
			arc.End = p
		}
		m[maxRef(a.center, a.start, a.end)+1] = arc
	}
	return m
}

func maxRef(a, b, c uint32) uint32 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

// DistanceDim is a decoded point-to-point distance dimension: the two sketch points it dimensions
// and their separation (cm). The value is taken from the geometry (the points' distance), which
// equals the dimension's stored value for a driving dimension at the solved geometry — so applying
// AddDistance(A,B,value) reproduces the dimension without moving a point.
type DistanceDim struct {
	A, B  Point2D
	Value float64
}

// DecodeDistanceDimensions returns the sketch's point-to-point distance dimensions. A distance
// dimension is the one constraint node whose BOTH references resolve to distinct sketch points: a
// coincidence's first reference is a line, parallel/perpendicular reference two lines, an
// axis-align references a line, and a radius uses the 0x10 sentinel — so none of those pass the
// two-points gate, leaving only the distance dimensions. (Aligned dimensions whose second endpoint
// isn't captured, and distance-from-line dimensions, are not covered here.)
func DecodeDistanceDimensions(seg []byte) []DistanceDim {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	var out []DistanceDim
	seen := map[[4]int64]bool{}
	for _, c := range collectRawCons(seg) {
		// A point-to-point distance dimension's t44 is itself a reference (its high bit is set — it
		// points at the dimension's text/label entity), unlike a geometric constraint whose t44 is a
		// small enum (coincidence 0x3e, axis-align 0x01030000, line-relate 0x00400000, radius 0).
		// Requiring t44 to be a reference keeps a two-point *geometric* relation (align/symmetry) from
		// being mistaken for a distance dimension.
		if c.disc&refBit == 0 || c.r1&refBit == 0 || c.r2&refBit == 0 {
			continue
		}
		// A SYMMETRY node has the same shape (high-bit t44, two point refs), but its t44 references
		// the symmetry AXIS line rather than a text point. So skip when t44 resolves to a line —
		// otherwise the two symmetric points would be misread as a spurious distance dimension.
		if _, isAxis := le[c.disc&^refBit]; isAxis {
			continue
		}
		p1, ok1 := vc[c.r1&^refBit]
		p2, ok2 := vc[c.r2&^refBit]
		if !ok1 || !ok2 {
			continue
		}
		d := math.Hypot(p1.X-p2.X, p1.Y-p2.Y)
		if d < 1e-3 {
			continue // coincident points are not a distance dimension
		}
		key := distKey(p1, p2)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, DistanceDim{A: p1, B: p2, Value: d})
	}
	return out
}

// DecodeRevolveRadii returns the x-distances at which a revolve's radius dimensions pin the profile
// edges from the x=0 centreline. A revolve radius node (t44=0x01150000) carries an inline value; we
// accept it only when it EQUALS the x-position of an actual vertical profile edge (so the value is a
// genuine radius, not a coordinate leak) and return that x. Applied as a horizontal distance from
// the centreline, the value is exactly the edge's current x — geometry-safe. Reuniting the
// centreline into the profile sketch (ReuniteRevolveAxis) is what lets these bind: the x=0 axis and
// the x=V edge then live in one sketch.
func DecodeRevolveRadii(seg []byte) []float64 {
	if !HasRevolve(seg) {
		return nil // radius-from-centreline dimensions are meaningful only on a revolve profile
	}
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	edgeX := map[int64]bool{}
	for _, l := range le {
		if math.Abs(l[0].X-l[1].X) < geoEps && math.Abs(l[0].X) > 1e-3 {
			edgeX[r4(l[0].X)] = true
		}
	}
	var out []float64
	seen := map[int64]bool{}
	for _, c := range collectRawCons(seg) {
		if c.disc != axisRadiusDisc {
			continue
		}
		for o := c.off + conDiscOff; o+8 <= c.off+140 && o+8 <= len(seg); o += 4 {
			v := math.Float64frombits(binary.LittleEndian.Uint64(seg[o:]))
			if edgeX[r4(v)] && !seen[r4(v)] {
				seen[r4(v)] = true
				out = append(out, v)
				break
			}
		}
	}
	return out
}

// AxialLength is a decoded axial (step-length) dimension of a revolve profile: the vertical gap
// between two horizontal profile edges. It is accepted only when the gap equals a model-parameter
// value — the modeller drives each step length by a parameter, so a gap that matches one is a real
// dimension, not an incidental alignment. Applied as a vertical distance, the value is the edges'
// current separation, so geometry never moves.
type AxialLength struct {
	Y1, Y2 float64
	Value  float64
}

// DecodeAxialLengths returns the revolve profile's axial step-length dimensions: the vertical gaps
// between ADJACENT horizontal edges whose size matches a model parameter (the driving dimension).
func DecodeAxialLengths(seg []byte) []AxialLength {
	if !HasRevolve(seg) {
		return nil // step-length dimensions are meaningful only on a revolve profile
	}
	le := lineEndpoints(seg, vertexCoords(seg))
	var ys []float64
	seen := map[int64]bool{}
	for _, l := range le {
		if math.Abs(l[0].Y-l[1].Y) < geoEps && !seen[r4(l[0].Y)] {
			seen[r4(l[0].Y)] = true
			ys = append(ys, l[0].Y)
		}
	}
	sort.Float64s(ys)
	params := paramValueSet(seg)
	var out []AxialLength
	for i := 1; i < len(ys); i++ {
		gap := ys[i] - ys[i-1]
		if gap > 1e-3 && params[r4(gap)] {
			out = append(out, AxialLength{Y1: ys[i-1], Y2: ys[i], Value: gap})
		}
	}
	return out
}

// paramValueSet is the set of model-parameter values (rounded), used to certify that a decoded gap
// is a driven dimension rather than an incidental one.
func paramValueSet(seg []byte) map[int64]bool {
	m := map[int64]bool{}
	for i := 0; i+20 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag || binary.LittleEndian.Uint32(seg[i+8:]) != nullRef {
			continue
		}
		n := int(binary.LittleEndian.Uint32(seg[i+16:]))
		if n < 1 || n > 64 || i+20+n > len(seg) {
			continue
		}
		if v, ok := firstNonZeroDouble(seg, i+20+n, 40); ok && v > 1e-3 && v < 100 {
			m[r4(v)] = true
		}
	}
	return m
}

// distKey is an order-independent rounded key for a point pair, so a doubly-recorded dimension
// de-duplicates.
func distKey(a, b Point2D) [4]int64 {
	ax, ay, bx, by := r4(a.X), r4(a.Y), r4(b.X), r4(b.Y)
	if ax > bx || (ax == bx && ay > by) {
		ax, ay, bx, by = bx, by, ax, ay
	}
	return [4]int64{ax, ay, bx, by}
}
