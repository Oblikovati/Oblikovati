// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"sort"
)

// Sketch geometric-constraint decode (Inventor 2027). The type tags InventorLoader keys on
// (Horizontal2D=0x90874D98, Parallel2D=0x90874D95, …) are ABSENT in the 2027 node graph; a
// constraint is instead a dc node whose +16 word is the constraint List6 header 0x30000006,
// followed by two entity references (+36/+40) and a discriminator word at +44 (t44). The t44
// families were pinned by a single-variable corpus (one constraint per part, byte-diffed):
//
//	0x0000003e  coincidence  (line ref, endpoint ref)     — reproduced structurally by shared points
//	0x01030000  axis-align   (line ref, 0x00003b00)       — Horizontal if the line is horizontal, else Vertical
//	0x00400000  line-relate  (line ref, line ref)         — Parallel if the two lines are parallel, else Perpendicular
//	0x00000000  radius/dia   (0x10, circle ref, param@+60) — a dimension (value in the linked dN parameter)
//
// Horizontal vs Vertical, and Parallel vs Perpendicular, are byte-IDENTICAL in the node — the
// distinction is carried by the referenced geometry (which the diff proved). So these constraints
// are classified from the resolved lines' orientation, and because the geometry already satisfies
// them, applying them never moves a point (correctness-first): they only remove degrees of freedom.

const (
	constraintHdr = 0x30000006 // a constraint/dimension node's +16 word
	conRef1Off    = 36         // first entity reference
	conRef2Off    = 40         // second entity reference
	conDiscOff    = 44         // the t44 discriminator

	coincidenceDisc = 0x0000003e // line↔endpoint coincidence
	axisAlignDisc   = 0x01030000 // horizontal/vertical (one line)
	lineRelateDisc  = 0x00400000 // parallel/perpendicular (two lines)
	axisRadiusDisc  = 0x01150000 // a revolve radius dimension (distance from the x=0 centreline)
	midpointDisc    = 0x00000000 // midpoint (line + a point at its midpoint); shared with radius dims
)

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
)

// GeoConstraint is a decoded geometric constraint with its lines resolved to endpoint coordinates,
// so the translator can bind it to the emitted sketch by coordinate. L2 is unused for H/V; Pt is
// set only for point-on-geometry constraints (midpoint), where it is the pinned point's coordinate.
type GeoConstraint struct {
	Kind   GeoKind
	L1, L2 [2]Point2D
	Pt     Point2D
}

// rawCon is a decoded constraint/dimension node before its references are resolved.
type rawCon struct {
	off    int
	r1, r2 uint32
	disc   uint32
}

// collectRawCons reads every constraint/dimension node (a dc node whose +16 word is the constraint
// List6 header), recording its two entity references and the t44 discriminator.
func collectRawCons(seg []byte) []rawCon {
	var out []rawCon
	for i := 0; i+conDiscOff+4 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag ||
			binary.LittleEndian.Uint32(seg[i+8:]) != nullRef ||
			binary.LittleEndian.Uint32(seg[i+16:]) != constraintHdr {
			continue
		}
		out = append(out, rawCon{
			off:  i,
			r1:   binary.LittleEndian.Uint32(seg[i+conRef1Off:]),
			r2:   binary.LittleEndian.Uint32(seg[i+conRef2Off:]),
			disc: binary.LittleEndian.Uint32(seg[i+conDiscOff:]),
		})
	}
	return out
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

// circleByEntityID maps each circle's ENTITY id (its centre-point reference + 1, the id a
// constraint uses to name the circle) to the circle with its centre resolved.
func circleByEntityID(seg []byte, vc map[uint32]Point2D) map[uint32]circleEnt {
	m := map[uint32]circleEnt{}
	for _, it := range collectItems(seg) {
		if it.circle == nil || !it.circle.hasRef {
			continue
		}
		ce := *it.circle
		if p, ok := vc[ce.ref]; ok {
			ce.inline = p
		}
		m[ce.ref+1] = ce
	}
	return m
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

func r4(v float64) int64 { return int64(math.Round(v * 1e4)) }

// geoEps bounds the direction test for classifying axis-alignment and parallel/perpendicular.
const geoEps = 1e-6

func isHorizontal(l [2]Point2D) bool { return math.Abs(l[0].Y-l[1].Y) < geoEps }
func isVertical(l [2]Point2D) bool   { return math.Abs(l[0].X-l[1].X) < geoEps }

// linesParallel reports whether the two segments are parallel (their direction cross product ≈ 0).
func linesParallel(a, b [2]Point2D) bool {
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[1].X-b[0].X, b[1].Y-b[0].Y
	return math.Abs(ax*by-ay*bx) < geoEps*maxLen(ax, ay, bx, by)
}

// linesPerpendicular reports whether the two segments are perpendicular (direction dot ≈ 0).
func linesPerpendicular(a, b [2]Point2D) bool {
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[1].X-b[0].X, b[1].Y-b[0].Y
	return math.Abs(ax*bx+ay*by) < geoEps*maxLen(ax, ay, bx, by)
}

// linesCollinear reports whether the two segments lie on the same infinite line: parallel AND
// one segment's start lies on the other's line (its offset from that line is ≈ 0).
func linesCollinear(a, b [2]Point2D) bool {
	if !linesParallel(a, b) {
		return false
	}
	ax, ay := a[1].X-a[0].X, a[1].Y-a[0].Y
	bx, by := b[0].X-a[0].X, b[0].Y-a[0].Y
	return math.Abs(ax*by-ay*bx) < geoEps*maxLen(ax, ay, bx, by)
}

// lengthsEqual reports whether two segments have the same length (to sketch precision).
func lengthsEqual(a, b [2]Point2D) bool {
	la := math.Hypot(a[1].X-a[0].X, a[1].Y-a[0].Y)
	lb := math.Hypot(b[1].X-b[0].X, b[1].Y-b[0].Y)
	return math.Abs(la-lb) < 1e-4
}

// midpointOf returns the midpoint of a segment.
func midpointOf(l [2]Point2D) Point2D {
	return Point2D{X: (l[0].X + l[1].X) / 2, Y: (l[0].Y + l[1].Y) / 2}
}

func maxLen(ax, ay, bx, by float64) float64 {
	return math.Max(math.Hypot(ax, ay), math.Hypot(bx, by))
}

// vertexCoords maps each geometry vertex reference (a curve endpoint id) to its coordinate. It
// resolves PER CLUSTER — each sketch's entities sit in one byte-offset run (clusterItems), and
// within a run the sorted distinct vertex references rank-align to the cluster's points in creation
// order (the correspondence resolveByRefs uses). Per-cluster alignment is what lets a multi-sketch
// part (a shaft's axis sketch + profile + work geometry) resolve — a global alignment fails on the
// count mismatch. A cluster whose counts disagree is skipped, so its constraints aren't bound to
// mis-aligned geometry.
func vertexCoords(seg []byte) map[uint32]Point2D {
	m := map[uint32]Point2D{}
	for _, cl := range clusterItems(seg) {
		refs := clusterVertexRefs(cl)
		pts := clusterPoints(cl)
		// A dimension inserts an extra TEXT point into the cluster (its label position), created
		// after the geometry, so a cluster can hold more points than curve-referenced vertices. The
		// geometry points come first (lower node id / byte offset), so the sorted references still
		// rank-align to the leading points; the trailing text points are simply left unmapped. Only
		// a cluster with FEWER points than references is genuinely inconsistent and skipped.
		if len(refs) == 0 || len(refs) > len(pts) {
			continue
		}
		for i, r := range refs {
			m[r] = pts[i]
		}
	}
	return m
}

// clusterVertexRefs returns the distinct point references in one cluster, ascending — every
// reference a curve makes to a point node: a line's two endpoints, an arc's endpoints, and a
// circle's centre. The centre matters even though it is not a "vertex": omitting it leaves the
// reference set one short of the point set whenever the sketch has a circle, so the count check in
// vertexCoords would reject the whole cluster (a part with any circle would resolve no constraints).
func clusterVertexRefs(cl []sketchItem) []uint32 {
	seen := map[uint32]bool{}
	for _, it := range cl {
		switch {
		case it.line != nil && it.line.paired:
			seen[it.line.a], seen[it.line.b] = true, true
		case it.arc != nil:
			seen[it.arc.start], seen[it.arc.end] = true, true
		case it.circle != nil && it.circle.hasRef:
			seen[it.circle.ref] = true
		}
	}
	refs := make([]uint32, 0, len(seen))
	for r := range seen {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
	return refs
}

// clusterPoints returns the cluster's point coordinates in creation (byte-offset) order.
func clusterPoints(cl []sketchItem) []Point2D {
	var pts []Point2D
	for _, it := range cl {
		if it.pt != nil {
			pts = append(pts, it.pt.p)
		}
	}
	return pts
}

// coincidenceGroups maps each line reference to every point pinned to it by a 0x3e coincidence
// node — its two endpoints, plus any interior point-on-line vertex.
func coincidenceGroups(seg []byte, vc map[uint32]Point2D) map[uint32][]Point2D {
	grp := map[uint32][]Point2D{}
	for _, c := range collectRawCons(seg) {
		if c.disc != coincidenceDisc || c.r1&refBit == 0 || c.r2&refBit == 0 {
			continue
		}
		line, vert := c.r1&^refBit, c.r2&^refBit
		p, ok := vc[vert]
		if !ok {
			line, vert = c.r2&^refBit, c.r1&^refBit
			if p, ok = vc[vert]; !ok {
				continue
			}
		}
		grp[line] = append(grp[line], p)
	}
	return grp
}

// lineEndpoints maps each line reference to its two endpoint coordinates (a line with exactly two
// coincidence points). A line carrying an extra interior point-on-line vertex has >2 points and is
// intentionally left out here — point-on-line resolves its own endpoints via extremePair in
// decodePointOnLine, so this core resolver stays unchanged (recovering such lines here showed no
// benefit on the corpus and would broaden the resolver's behaviour without cause).
func lineEndpoints(seg []byte, vc map[uint32]Point2D) map[uint32][2]Point2D {
	out := map[uint32][2]Point2D{}
	for line, ps := range coincidenceGroups(seg, vc) {
		if len(ps) == 2 {
			out[line] = [2]Point2D{ps[0], ps[1]}
		}
	}
	return out
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
