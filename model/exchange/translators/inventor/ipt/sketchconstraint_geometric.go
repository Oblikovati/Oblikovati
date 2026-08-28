// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
)

// Sketch constraints — the GEOMETRIC family (M48 #2230 split of sketchconstraint.go). The value-free
// relations decoded from the constraint/dimension nodes: horizontal/vertical/parallel/perpendicular/
// collinear/equal-length/midpoint (DecodeGeometricConstraints), circle concentric/equal-radius,
// ground (fully-fix), symmetry, line-circle tangent, and point-on-line. Each is classified from the
// resolved geometry (which already satisfies it), so applying it only removes DOF and moves nothing.
// The shared record decoding and geometry predicates live in sketchconstraint.go; the dimensional
// (valued) constraints in sketchconstraint_dimensional.go.

// GeoKind is a decoded geometric-constraint type.
type GeoKind int

const (
	GeoHorizontal GeoKind = iota
	GeoVertical
	GeoParallel
	GeoPerpendicular
	GeoCollinear
	GeoEqualLength
	GeoMidpoint
	GeoPointOnLine
	GeoConcentric
	GeoEqualRadius
)

// GeoConstraint is a decoded geometric constraint with its lines resolved to endpoint coordinates,
// so the translator can bind it to the emitted sketch by coordinate. L2 is unused for H/V; Pt is
// set only for point-on-geometry constraints (midpoint), where it is the pinned point's coordinate.
type GeoConstraint struct {
	Kind   GeoKind
	L1, L2 [2]Point2D
	Pt     Point2D
}

// DecodeGeometricConstraints returns the sketch's value-free geometric constraints (horizontal,
// vertical, parallel, perpendicular), each resolved to the coordinates of the line(s) it relates.
// Coincidences are omitted (reproduced by shared points), as are dimensions (their value lives in a
// parameter node — decoded separately). A constraint whose lines don't resolve, or whose geometry
// matches neither expected relation, is dropped rather than guessed.
func DecodeGeometricConstraints(seg []byte) []GeoConstraint {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	var out []GeoConstraint
	for _, c := range collectRawCons(seg) {
		switch c.disc {
		case axisAlignDisc:
			l, ok := le[c.r1&^refBit]
			if !ok || c.r1&refBit == 0 {
				continue
			}
			if isHorizontal(l) {
				out = append(out, GeoConstraint{Kind: GeoHorizontal, L1: l})
			} else if isVertical(l) {
				out = append(out, GeoConstraint{Kind: GeoVertical, L1: l})
			}
		case lineRelateDisc:
			l1, ok1 := le[c.r1&^refBit]
			l2, ok2 := le[c.r2&^refBit]
			if !ok1 || !ok2 || c.r1&refBit == 0 || c.r2&refBit == 0 {
				continue
			}
			// Collinear shares this discriminator with parallel/perpendicular (all "line relate"
			// two lines) and IS parallel, so it must be tested first: two collinear lines lie on one
			// infinite line, which parallel-but-offset lines do not.
			if linesCollinear(l1, l2) {
				out = append(out, GeoConstraint{Kind: GeoCollinear, L1: l1, L2: l2})
			} else if linesParallel(l1, l2) {
				out = append(out, GeoConstraint{Kind: GeoParallel, L1: l1, L2: l2})
			} else if linesPerpendicular(l1, l2) {
				out = append(out, GeoConstraint{Kind: GeoPerpendicular, L1: l1, L2: l2})
			}
		case coincidenceDisc:
			// A coincidence node whose BOTH references are lines (not the usual line↔endpoint) is an
			// equal-length constraint. lineEndpoints already ignores it (it needs one ref to be a
			// point), so this read doesn't disturb endpoint resolution. Emitted only when the lengths
			// actually match, so it never implies geometry the sketch doesn't already have.
			l1, ok1 := le[c.r1&^refBit]
			l2, ok2 := le[c.r2&^refBit]
			if ok1 && ok2 && c.r1&refBit != 0 && c.r2&refBit != 0 && lengthsEqual(l1, l2) {
				out = append(out, GeoConstraint{Kind: GeoEqualLength, L1: l1, L2: l2})
			}
		case midpointDisc:
			// A disc-0 node whose first ref resolves to a line is a MIDPOINT constraint: a sketch
			// point pinned to the line's midpoint. Radius/diameter dimensions share disc 0 but their
			// first ref is the 0x10 sentinel (not a ref), so lineEndpoints won't resolve it — the
			// line gate cleanly excludes them. The pinned point's coordinate is the line's geometric
			// midpoint, COMPUTED from the resolved line rather than read from the (offset, hard to
			// resolve) point reference; the apply step binds it only when a sketch point actually sits
			// there, so a stray disc-0 node can't invent a constraint.
			if c.r1&refBit == 0 {
				continue
			}
			if l, ok := le[c.r1&^refBit]; ok {
				out = append(out, GeoConstraint{Kind: GeoMidpoint, L1: l, Pt: midpointOf(l)})
			}
		}
	}
	out = append(out, decodePointOnLine(seg, vc)...)
	return out
}

// CircleRelation is a decoded circle↔circle constraint (concentric or equal-radius), resolved to
// both circles' centres and radii so the translator can bind them by coordinate.
type CircleRelation struct {
	Kind   GeoKind // GeoConcentric or GeoEqualRadius
	C1, C2 Point2D // centres
	R1, R2 float64 // radii
}

// DecodeCircleRelations returns the sketch's concentric and equal-radius constraints. Both are
// stored as a coincidence node (t44 0x3e) whose two references resolve to CIRCLES (each named by
// its entity id = centre ref + 1, via circleByEntityID) — the same shape equal-length uses for two
// lines. Concentric and equal-radius share the discriminator and are told apart by geometry: a
// concentric pair shares a centre (different radii); an equal-radius pair shares a radius
// (different centres). Emitted only when the geometry actually exhibits the relation, so a stray
// node never yields a spurious constraint and nothing moves.
func DecodeCircleRelations(seg []byte) []CircleRelation {
	circ := circleByEntityID(seg, vertexCoords(seg))
	var out []CircleRelation
	for _, c := range collectRawCons(seg) {
		if c.disc != coincidenceDisc || c.r1&refBit == 0 || c.r2&refBit == 0 {
			continue
		}
		a, ok1 := circ[c.r1&^refBit]
		b, ok2 := circ[c.r2&^refBit]
		if !ok1 || !ok2 {
			continue
		}
		sameCentre := samePoint2D(a.inline, b.inline)
		sameRadius := math.Abs(a.radius-b.radius) < 1e-4
		switch {
		case sameCentre && !sameRadius:
			out = append(out, CircleRelation{Kind: GeoConcentric, C1: a.inline, C2: b.inline, R1: a.radius, R2: b.radius})
		case sameRadius && !sameCentre:
			out = append(out, CircleRelation{Kind: GeoEqualRadius, C1: a.inline, C2: b.inline, R1: a.radius, R2: b.radius})
		}
	}
	return out
}

// GroundKind selects which entity a decoded ground constraint fixes.
type GroundKind int

const (
	GroundLine GroundKind = iota
	GroundCircle
	GroundPoint
)

// GroundConstraint is a decoded "fully fix this geometry" (Inventor Ground): one entity frozen at
// its current position. It carries the resolved entity by coordinate; only the field selected by
// Kind is meaningful. Grounding pins the entity where it already sits, so applying it removes
// degrees of freedom WITHOUT moving anything.
type GroundConstraint struct {
	Kind   GroundKind
	Line   [2]Point2D // GroundLine
	Center Point2D    // GroundCircle
	Radius float64    // GroundCircle
	Pt     Point2D    // GroundPoint
}

// DecodeGroundConstraints returns the sketch's ground constraints. A ground node's t44 is
// groundDisc and its FIRST reference names the grounded entity (its second word is a non-ref
// count, not an entity). The entity is resolved as a line, a circle (by entity id = centre + 1),
// or a bare point — whichever the reference matches — so the translator can bind it by coordinate.
// A node whose reference resolves to none of these is dropped rather than guessed.
func DecodeGroundConstraints(seg []byte) []GroundConstraint {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	circ := circleByEntityID(seg, vc)
	var out []GroundConstraint
	for _, c := range collectRawCons(seg) {
		if c.disc != groundDisc || c.r1&refBit == 0 {
			continue
		}
		id := c.r1 &^ refBit
		switch {
		case hasLine(le, id):
			out = append(out, GroundConstraint{Kind: GroundLine, Line: le[id]})
		case hasCircle(circ, id):
			ce := circ[id]
			out = append(out, GroundConstraint{Kind: GroundCircle, Center: ce.inline, Radius: ce.radius})
		case hasPoint(vc, id):
			out = append(out, GroundConstraint{Kind: GroundPoint, Pt: vc[id]})
		}
	}
	return out
}

func hasLine(m map[uint32][2]Point2D, id uint32) bool  { _, ok := m[id]; return ok }
func hasCircle(m map[uint32]circleEnt, id uint32) bool { _, ok := m[id]; return ok }
func hasPoint(m map[uint32]Point2D, id uint32) bool    { _, ok := m[id]; return ok }

// SymmetryConstraint is a decoded symmetry: two points mirror-symmetric about an axis line. The
// two symmetric points sit at the usual +36/+40 references; the axis line is a THIRD reference at
// +44 (where a plain two-ref constraint carries a small discriminator — here its high bit is set,
// naming the axis). Resolved to coordinates and self-validated (each point reflects onto the other
// across the axis) so a coincidentally-shaped node never yields a spurious constraint.
type SymmetryConstraint struct {
	P1, P2 Point2D
	Axis   [2]Point2D
}

// DecodeSymmetryConstraints returns the sketch's symmetry constraints. Gate: the +32 empty-list
// sentinel (a plain two-ref layout — a tangent puts a line ref there instead), both entity refs
// resolve to POINTS, and the t44 word is itself a reference resolving to the axis LINE. The last
// point distinguishes symmetry from a point-to-point distance dimension, whose t44 also has its
// high bit set but references a text/label point (not a line). Finally the geometry must actually
// be symmetric (pointsSymmetric), so a stray node can't invent a constraint and nothing moves.
func DecodeSymmetryConstraints(seg []byte) []SymmetryConstraint {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	var out []SymmetryConstraint
	for _, c := range collectRawCons(seg) {
		if c.r1&refBit == 0 || c.r2&refBit == 0 || c.disc&refBit == 0 ||
			binary.LittleEndian.Uint32(seg[c.off+32:]) != emptyListMark {
			continue
		}
		axis, isLine := le[c.disc&^refBit]
		if !isLine {
			continue
		}
		p1, ok1 := vc[c.r1&^refBit]
		p2, ok2 := vc[c.r2&^refBit]
		if ok1 && ok2 && pointsSymmetric(p1, p2, axis) {
			out = append(out, SymmetryConstraint{P1: p1, P2: p2, Axis: axis})
		}
	}
	return out
}

// pointsSymmetric reports whether p1 and p2 are mirror images across the infinite axis line
// (reflecting p1 through its perpendicular foot on the axis lands on p2).
func pointsSymmetric(p1, p2 Point2D, axis [2]Point2D) bool {
	ax, ay := axis[1].X-axis[0].X, axis[1].Y-axis[0].Y
	l2 := ax*ax + ay*ay
	if l2 < 1e-12 {
		return false
	}
	tt := ((p1.X-axis[0].X)*ax + (p1.Y-axis[0].Y)*ay) / l2
	fx, fy := axis[0].X+tt*ax, axis[0].Y+tt*ay
	rx, ry := 2*fx-p1.X, 2*fy-p1.Y
	return math.Abs(rx-p2.X) < 1e-4 && math.Abs(ry-p2.Y) < 1e-4
}

// TangentConstraint is a decoded line↔circle tangent constraint, resolved to the line's endpoints
// and the circle's centre and radius so the translator can bind it by coordinate.
type TangentConstraint struct {
	Line   [2]Point2D
	Center Point2D
	Radius float64
}

// DecodeTangentConstraints returns the sketch's line↔circle tangents. A tangent node has a
// different layout from the two-ref geometric constraints: its List6 map is non-empty, so the
// LINE reference sits at +32 (a plain two-ref constraint holds the 0x10 sentinel there) and the
// CIRCLE reference is the +44 word (which doubles as the discriminator, high bit set). The circle
// is named by its ENTITY id = centre-point ref + 1 (circleByEntityID). Emitted only when the line
// is actually tangent to the circle (perpendicular distance from centre == radius), so a
// coincidentally-shaped node never yields a spurious constraint and no geometry moves.
func DecodeTangentConstraints(seg []byte) []TangentConstraint {
	vc := vertexCoords(seg)
	le := lineEndpoints(seg, vc)
	circ := circleByEntityID(seg, vc)
	var out []TangentConstraint
	seen := map[[6]int64]bool{}
	for i := 0; i+48 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag ||
			binary.LittleEndian.Uint32(seg[i+8:]) != nullRef ||
			binary.LittleEndian.Uint32(seg[i+16:]) != constraintHdr {
			continue
		}
		lref := binary.LittleEndian.Uint32(seg[i+32:])
		cref := binary.LittleEndian.Uint32(seg[i+conDiscOff:])
		if lref&refBit == 0 || cref&refBit == 0 {
			continue
		}
		l, ok1 := le[lref&^refBit]
		c, ok2 := circ[cref&^refBit]
		if !ok1 || !ok2 || !lineTangentToCircle(l, c.inline, c.radius) {
			continue
		}
		key := [6]int64{r4(l[0].X), r4(l[0].Y), r4(l[1].X), r4(l[1].Y), r4(c.inline.X), r4(c.inline.Y)}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, TangentConstraint{Line: l, Center: c.inline, Radius: c.radius})
	}
	return out
}

// lineTangentToCircle reports whether the segment's infinite line is tangent to the circle: the
// perpendicular distance from the centre to the line equals the radius.
func lineTangentToCircle(l [2]Point2D, center Point2D, radius float64) bool {
	dx, dy := l[1].X-l[0].X, l[1].Y-l[0].Y
	length := math.Hypot(dx, dy)
	if length < 1e-9 {
		return false
	}
	dist := math.Abs((center.X-l[0].X)*dy-(center.Y-l[0].Y)*dx) / length
	return math.Abs(dist-radius) < 1e-4
}

// extremePair returns the two points farthest apart in a collinear set — a line's endpoints.
func extremePair(ps []Point2D) [2]Point2D {
	ai, bi, best := 0, 1, -1.0
	for i := range ps {
		for j := i + 1; j < len(ps); j++ {
			if d := math.Hypot(ps[i].X-ps[j].X, ps[i].Y-ps[j].Y); d > best {
				best, ai, bi = d, i, j
			}
		}
	}
	return [2]Point2D{ps[ai], ps[bi]}
}

// decodePointOnLine returns the sketch's point-on-line constraints: a curve vertex pinned onto a
// line's INTERIOR by a 0x3e coincidence node (a coincidence at an endpoint is a plain corner, not a
// point-on-line). Each is real — a decoded coincidence node whose point resolves to a vertex lying
// strictly between the line's endpoints — so applying it moves no geometry. (A point-on-line on a
// STANDALONE point isn't covered: a standalone point isn't a curve vertex, so its arbitrary
// position along the line can't be resolved.)
func decodePointOnLine(seg []byte, vc map[uint32]Point2D) []GeoConstraint {
	var out []GeoConstraint
	for _, ps := range coincidenceGroups(seg, vc) {
		if len(ps) < 3 {
			continue
		}
		e := extremePair(ps)
		for _, p := range ps {
			if !samePoint2D(p, e[0]) && !samePoint2D(p, e[1]) && onSegmentInterior(p, e[0], e[1]) {
				out = append(out, GeoConstraint{Kind: GeoPointOnLine, L1: e, Pt: p})
			}
		}
	}
	return out
}

// onSegmentInterior reports whether p lies on segment a–b strictly between the endpoints.
func onSegmentInterior(p, a, b Point2D) bool {
	dx, dy := b.X-a.X, b.Y-a.Y
	length := math.Hypot(dx, dy)
	if length < 1e-6 {
		return false
	}
	if math.Abs((p.X-a.X)*dy-(p.Y-a.Y)*dx)/length > 1e-4 {
		return false // not on the line
	}
	tt := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / (length * length)
	return tt > 1e-3 && tt < 1-1e-3
}
