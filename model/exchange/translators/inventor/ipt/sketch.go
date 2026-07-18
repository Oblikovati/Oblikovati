// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"sort"
)

// Sketch geometry decoded from PmDCSegment. Coordinates are in cm (database units).
// A sketch entity is a nameless graph node whose payload begins with the 2D-geometry
// tag; a point carries two coordinates and an entity id, a curve a reference/parameter
// block. A line node references its two endpoint points by their (SketchPoint) entity ids.
const (
	geomTag     = 0x8000000C // 2D geometry payload tag on a standalone point
	curveMarker = 0x30000002 // first word of the curve signature
	refBit      = 0x80000000 // set on an entity-reference u32 (a line's endpoint refs)
	// nodeTagBase is the geometry-node type tag with its low byte masked off: a real tag is
	// 0x800000XX, so `tag & 0xFFFFFF00 == nodeTagBase` accepts only genuine node tags.
	nodeTagBase = 0x80000000
)

// curveHead is the first word (0x30000002, little-endian bytes) of a sketch curve's 16-byte
// header, which is: 0x30000002 | w1 | w2 | 0x10. Keying on it (rather than the preceding tag)
// finds curves in both standalone sketches and feature-consumed profiles, whose leading tag
// differs (0x8000000C vs 0x80000016). After the header: for a line, two endpoint entity-refs (at
// +16 and +20); for a circle, the centre (2 float64 from +16), an entity-ref, and the radius
// (float64 at +36).
//
// The two middle words w1,w2 are small per-endpoint counts — 2 at a plain vertex, but 3 (rarely
// 4) where extra geometry meets it. They must NOT be hardcoded: keying on the old fixed 0x02,0x02
// header silently dropped a shaft's axis edge and an interior segment (LeverShaft decoded 11 of 13
// lines), leaving an OPEN profile that resolved by count yet revolved into a wrong solid. Anchoring
// on the invariant 0x30000002 … 0x10 with w1,w2 in [curveWordMin,curveWordMax] recovers them while
// still rejecting a stray 0x30000002 whose following words are float mantissa (large) or 1.
var curveHead = []byte{0x02, 0, 0, 0x30}

const (
	curveTailMark = 0x10 // the invariant 4th word of a curve header
	curveWordMin  = 2    // w1,w2 bounds: a curve endpoint's count is ≥2; >4 is not a curve header
	curveWordMax  = 4
	// ellipseSentinel is the u32 at curve+72 that marks a sketch ellipse. An ellipse's node has the
	// same shape as a circle's (centre by reference, +16 not a ref) but adds an inline unit
	// major-axis (+36/+44) and two semi-axis lengths (majorR@+52, minorR@+60); a real circle instead
	// carries denormal garbage (~1e-308) in those slots and a node reference (0x800000xx) at +72, so
	// the sentinel plus normal-range radii cleanly tell the two apart.
	ellipseSentinel = 0x01170000

	// constructionFlag marks a curve entity as Inventor construction / reference geometry — a dashed
	// diagonal, centreline, or reference line that constrains the sketch but is NOT part of a
	// profile. It is bit 0x00080000 of the curve's dcNode FLAG word, which sits in the 24-byte node
	// header preceding the curve signature: tag@C-24, id@C-20, nullRef@C-16, marker@C-12, flag@C-8,
	// typetag@C-4, then the 0x30000002 curve header at C. Validated against live Inventor's
	// Construction property: bit-exact counts on FlangeReelMotor (2/2) and EncoderFrame (2/2), zero
	// on the shaft fixtures, and it never fires on real profile geometry.
	//
	// NOT YET ACTED ON — deliberately. A construction curve references the sketch's List6 constraint
	// nodes rather than geometry points, so today it decodes as a phantom line; but simply skipping
	// construction curves in collectItems regressed the library batch (SOLID 35→29). Removing the
	// curve leaves its now-unreferenced points still collected, which breaks the #refs==#points
	// parity resolveByRefs relies on — the same entanglement that blocks the denormal-garbage-point
	// fix. Construction geometry must be dropped together with its construction-only points, and that
	// requires the resolver rework (see the memory note ipt-sketch-entities). This constant is kept
	// so that work has the marker ready. To read it: flag@C-8, guarded by nullRef present at C-16.
	constructionFlag = 0x00080000
)

type Point2D struct{ X, Y float64 }

// Sketch is a decoded 2D sketch: standalone points, lines, circles, arcs, and ellipses.
// Constraints and dimensions are not decoded here (see sketchconstraint.go).
type Sketch struct {
	Points   []Point2D
	Lines    []Line
	Circles  []Circle
	Arcs     []Arc
	Ellipses []Ellipse
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
}

// LineIsConstruction reports whether line i is construction geometry, tolerating an absent flag
// slice (the decoders that don't carry the flag).
func (s Sketch) LineIsConstruction(i int) bool { return flagAt(s.LineConstruction, i) }

// CircleIsConstruction reports whether circle i is construction geometry.
func (s Sketch) CircleIsConstruction(i int) bool { return flagAt(s.CircleConstruction, i) }

// ArcIsConstruction reports whether arc i is construction geometry.
func (s Sketch) ArcIsConstruction(i int) bool { return flagAt(s.ArcConstruction, i) }

func flagAt(flags []bool, i int) bool { return i < len(flags) && flags[i] }

type Line struct{ A, B Point2D }
type Circle struct {
	Center Point2D
	Radius float64
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

// idPoint is a decoded geometry point tagged with its entity id, needed to resolve which
// point a line's endpoint reference names.
type idPoint struct {
	id uint32
	p  Point2D
}

// lineRefs is a line's two endpoint entity references (paired == both were real refs), plus the
// line's inline geometry: a curve node caches the exact infinite line as (midpoint, unit-direction)
// f64 immediately after its endpoint refs (at C+24+4*w1). This is decoded WITHOUT reference
// resolution, so it recovers a line's position/orientation even when the point-ref rank-alignment
// fails — the basis for reconstructInlineLoop. unit is set when the stored direction is unit length
// (a genuine line); an uninitialised/garbage slot holds a huge non-unit value and clears it.
type lineRefs struct {
	a, b   uint32
	paired bool
	mid    Point2D
	dir    Point2D
	unit   bool
	constr bool // Inventor construction/reference geometry (not part of a profile)
}

// circleEnt is a decoded circle plus the entity reference to its centre point. Inventor
// stores a circle's centre BY REFERENCE (the inline centre bytes are zero); the inline
// value is correct only for an origin circle that has no separate centre-point node.
type circleEnt struct {
	radius float64
	inline Point2D
	ref    uint32
	hasRef bool
}

// arcEnt is a decoded arc: entity references to its two endpoints and its centre, plus the
// radius (stored inline). Endpoints and centre resolve via the shared point-reference map.
type arcEnt struct {
	start, end, center uint32
	radius             float64
}

// ellipseEnt is a decoded ellipse: the entity reference to its centre point (resolved via the
// shared point-reference map, like a circle's centre), plus the inline unit major-axis direction
// and the two semi-axis lengths.
type ellipseEnt struct {
	ref            uint32
	axisX, axisY   float64
	majorR, minorR float64
}

// sketchItem is one decoded sketch entity in file order (by byte offset): a point, a
// circle, a line, an arc, or an ellipse — exactly one is non-nil.
type sketchItem struct {
	off     int
	pt      *idPoint
	circle  *circleEnt
	line    *lineRefs
	arc     *arcEnt
	ellipse *ellipseEnt
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

// collectItems reads every sketch entity — points (nameless node whose geometry tag is
// not a curve, plus its id and coords) and curves (by their invariant signature, split
// into lines with endpoint refs vs circles) — and returns them sorted by byte offset.
func collectItems(seg []byte) []sketchItem {
	var items []sketchItem
	forEachNamelessNode(seg, func(j int) {
		// The geometry tag at +20 must be a real node-type tag of the form 0x800000XX (line
		// 0x8000000C, consumed 0x8000000E, point 0x80000017, …) — not merely a word with the high
		// bit set. Requiring the exact shape (matching the curve scan below) rejects phantom points:
		// a stray node whose +16 flag holds a float and whose +20 word is garbage-with-the-top-bit
		// (e.g. 0xFFFF9673) otherwise reads as a spurious (0,0) vertex, and — now that the point
		// gate accepts nonzero +16 flags — would land in a neighbouring cluster and break its loop.
		if binary.LittleEndian.Uint32(seg[j+20:])&0xFFFFFF00 != nodeTagBase ||
			binary.LittleEndian.Uint32(seg[j+24:]) == curveMarker {
			return
		}
		x, y := f64(seg, j+24), f64(seg, j+32)
		if !math.IsNaN(x) && !math.IsNaN(y) && absf(x) < 1e4 && absf(y) < 1e4 {
			ip := idPoint{id: binary.LittleEndian.Uint32(seg[j+4:]), p: Point2D{x, y}}
			items = append(items, sketchItem{off: j, pt: &ip})
		}
	})
	for c := 0; ; {
		k := indexOf(seg[c:], curveHead)
		if k < 0 {
			break
		}
		C := c + k
		c = C + 1
		if C < 4 || C+44 > len(seg) {
			continue
		}
		// A curve header is 0x30000002 | w1 | w2 | 0x10 with w1,w2 the small per-endpoint counts,
		// preceded by a geometry-node TYPE tag of the form 0x800000XX (line 0x80000019, consumed
		// 0x80000016, standalone 0x8000000C). Requiring the invariant tail word (0x10), the bounded
		// middle words, and the exact node-tag shape — not merely the top bit — rejects both a stray
		// 0x30000002 inside float data and the parallel dimension/constraint nodes (whose preceding
		// word is a float, not a node tag), either of which would otherwise inject a phantom curve.
		if w1 := binary.LittleEndian.Uint32(seg[C+4:]); w1 < curveWordMin || w1 > curveWordMax {
			continue
		}
		if w2 := binary.LittleEndian.Uint32(seg[C+8:]); w2 < curveWordMin || w2 > curveWordMax {
			continue
		}
		if binary.LittleEndian.Uint32(seg[C+12:]) != curveTailMark ||
			binary.LittleEndian.Uint32(seg[C-4:])&0xFFFFFF00 != nodeTagBase {
			continue
		}
		if w16 := binary.LittleEndian.Uint32(seg[C+16:]); w16&refBit != 0 {
			w20 := binary.LittleEndian.Uint32(seg[C+20:])
			// A curve with two endpoint refs AND a centre ref + positive radius is an arc; the
			// same shape without those is a straight line.
			if center := binary.LittleEndian.Uint32(seg[C+32:]); center&refBit != 0 && w20&refBit != 0 {
				if r := f64(seg, C+36); r > 1e-6 && r < 1e5 {
					items = append(items, sketchItem{off: C, arc: &arcEnt{
						start: w16 &^ refBit, end: w20 &^ refBit, center: center &^ refBit, radius: r,
					}})
					continue
				}
			}
			lr := lineRefs{a: w16 &^ refBit, b: w20 &^ refBit, paired: w20&refBit != 0}
			readInlineLine(seg, C, &lr)
			items = append(items, sketchItem{off: C, line: &lr})
		} else if cref := binary.LittleEndian.Uint32(seg[C+32:]); cref&refBit != 0 && isEllipseNode(seg, C) {
			// Same node shape as a referenced-centre circle, but the sentinel + normal-range radii
			// (a circle carries denormals and a node ref there) identify a genuine ellipse.
			items = append(items, sketchItem{off: C, ellipse: &ellipseEnt{
				ref: cref &^ refBit, axisX: f64(seg, C+36), axisY: f64(seg, C+44),
				majorR: f64(seg, C+52), minorR: f64(seg, C+60),
			}})
		} else {
			ce := circleEnt{radius: f64(seg, C+36), inline: Point2D{f64(seg, C+16), f64(seg, C+24)}}
			if cref&refBit != 0 {
				ce.ref, ce.hasRef = cref&^refBit, true
			}
			items = append(items, sketchItem{off: C, circle: &ce})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].off < items[j].off })
	return items
}

// isEllipseNode reports whether the referenced-centre curve at C is a sketch ellipse rather than
// a circle. It requires the ellipse sentinel at +72 AND both semi-axis lengths (majorR@+52,
// minorR@+60) in a normal positive range: a circle stores denormal garbage (~1e-308) in those
// two slots and a node reference (0x800000xx) at +72, so all three checks fail on a circle. The
// three independent signals make a false positive (a circle read as an ellipse) impossible.
func isEllipseNode(seg []byte, C int) bool {
	if C+80 > len(seg) || binary.LittleEndian.Uint32(seg[C+72:]) != ellipseSentinel {
		return false
	}
	majorR, minorR := f64(seg, C+52), f64(seg, C+60)
	return majorR > 1e-6 && majorR < 1e5 && minorR > 1e-6 && minorR < 1e5
}

// assembleCluster reconstructs one sketch from its entities. When the cluster is clean
// (every curve's point references resolve), lines and circle centres are emitted from
// their exact referenced points — reconstructing non-convex loops and off-origin circles
// faithfully. Otherwise it falls back to inline circle centres and the convex-ring
// heuristic (good for simple convex profiles and origin circles only).
func assembleCluster(items []sketchItem) Sketch {
	var pts []idPoint
	var circs []circleEnt
	var lines []lineRefs
	var arcs []arcEnt
	var ells []ellipseEnt
	for _, it := range items {
		switch {
		case it.pt != nil:
			pts = append(pts, *it.pt)
		case it.circle != nil:
			circs = append(circs, *it.circle)
		case it.line != nil:
			lines = append(lines, *it.line)
		case it.arc != nil:
			arcs = append(arcs, *it.arc)
		case it.ellipse != nil:
			ells = append(ells, *it.ellipse)
		}
	}
	if s, ok := resolveByRefs(pts, circs, lines, arcs, ells); ok {
		s.Resolved = true
		return s
	}
	// resolveByRefs failed (its reference and point counts disagree). For a pure-line cluster,
	// try reconstructing the profile from each line's inline infinite-line geometry, which needs no
	// reference resolution — accepted only when every corner lands on a collected point (self-
	// validated), so it never emits guessed geometry. Falls through to the convex-ring heuristic.
	if s, ok := reconstructInlineLoop(items); ok {
		return s
	}
	inline := make([]Circle, len(circs))
	for i, ce := range circs {
		inline[i] = Circle{Center: ce.inline, Radius: ce.radius}
	}
	coords := make([]Point2D, len(pts))
	for i, ip := range pts {
		coords[i] = ip.p
	}
	return assembleSketch(coords, inline, len(lines))
}

// hasOriginPoint reports whether any decoded point sits at the sketch origin (0,0).
func hasOriginPoint(pts []idPoint) bool {
	for _, p := range pts {
		if absf(p.p.X) < 1e-9 && absf(p.p.Y) < 1e-9 {
			return true
		}
	}
	return false
}

// resolveByRefs reconstructs lines and circle centres from the point references every
// sketch curve carries. Rank-aligning the sorted distinct references to the sorted
// point-entity ids (both assigned in creation order, so equal rank == same point) recovers
// each point's coordinate exactly — so any polygon (convex or not) and any off-origin
// circle rebuild faithfully. Returns ok=false (caller falls back) unless the cluster is
// clean: at least one curve, every line paired, every circle referenced, and #distinct
// references == #points.
func resolveByRefs(pts []idPoint, circs []circleEnt, lines []lineRefs, arcs []arcEnt, ells []ellipseEnt) (Sketch, bool) {
	if len(lines) == 0 && len(circs) == 0 && len(arcs) == 0 && len(ells) == 0 {
		return Sketch{}, false
	}
	refSet := map[uint32]struct{}{}
	for _, l := range lines {
		if !l.paired {
			return Sketch{}, false
		}
		refSet[l.a] = struct{}{}
		refSet[l.b] = struct{}{}
	}
	for _, c := range circs {
		if !c.hasRef {
			return Sketch{}, false
		}
		refSet[c.ref] = struct{}{}
	}
	for _, a := range arcs {
		refSet[a.start] = struct{}{}
		refSet[a.end] = struct{}{}
		refSet[a.center] = struct{}{}
	}
	for _, e := range ells {
		refSet[e.ref] = struct{}{}
	}
	// The sketch origin (0,0) is an implicit centre point that geometry can reference but that
	// carries no coordinate node (so it never appears in pts). When exactly one referenced point
	// is otherwise unaccounted for and none of the decoded points is at the origin, synthesise it
	// with the lowest id — Inventor creates the origin first, so it rank-aligns to the smallest
	// reference. This makes a revolve profile that touches its centreline (a shaft) resolve
	// exactly instead of collapsing to the convex fallback.
	if len(refSet) == len(pts)+1 && !hasOriginPoint(pts) {
		pts = append([]idPoint{{id: 0, p: Point2D{0, 0}}}, pts...)
	}
	if len(refSet) != len(pts) {
		return Sketch{}, false
	}
	refs := make([]uint32, 0, len(refSet))
	for r := range refSet {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
	verts := append([]idPoint(nil), pts...)
	sort.Slice(verts, func(i, j int) bool { return verts[i].id < verts[j].id })
	coord := make(map[uint32]Point2D, len(refs))
	for i, r := range refs {
		coord[r] = verts[i].p
	}
	var s Sketch
	for _, c := range circs {
		s.Circles = append(s.Circles, Circle{Center: coord[c.ref], Radius: c.radius})
	}
	for _, l := range lines {
		s.Lines = append(s.Lines, Line{A: coord[l.a], B: coord[l.b]})
	}
	for _, a := range arcs {
		s.Arcs = append(s.Arcs, Arc{Center: coord[a.center], Radius: a.radius, Start: coord[a.start], End: coord[a.end]})
	}
	for _, e := range ells {
		s.Ellipses = append(s.Ellipses, Ellipse{
			Center: coord[e.ref], MajorAxis: Point2D{e.axisX, e.axisY}, MajorR: e.majorR, MinorR: e.minorR,
		})
	}
	return s, true
}

// assembleSketch is the fallback assembly when endpoint refs don't resolve cleanly: a
// circle owns its centre point, a single line owns its two endpoints, a convex ring is
// ordered by angle, and whatever remains is emitted as standalone sketch points.
func assembleSketch(pts []Point2D, circles []Circle, lineCurves int) Sketch {
	remaining := make([]Point2D, 0, len(pts))
	for _, p := range pts {
		if !nearAnyCenter(p, circles) {
			remaining = append(remaining, p)
		}
	}
	s := Sketch{Circles: circles}
	switch {
	case lineCurves == 0:
		s.Points = remaining // genuine standalone points
	case lineCurves == 1 && len(remaining) == 2:
		s.Lines = []Line{{A: remaining[0], B: remaining[1]}}
	case len(remaining) >= 3 && lineCurves >= len(remaining):
		// A closed polygon we could not resolve exactly: order the N corners into a simple
		// convex ring and connect consecutive corners. Correct only for convex profiles.
		s.Lines = closeLoop(orderConvex(remaining))
	default:
		// Ambiguous endpoint set — omit rather than emit misleading geometry.
	}
	return s
}

// orderConvex sorts points into convex (CCW) order around their centroid.
func orderConvex(pts []Point2D) []Point2D {
	var cx, cy float64
	for _, p := range pts {
		cx, cy = cx+p.X, cy+p.Y
	}
	n := float64(len(pts))
	cx, cy = cx/n, cy/n
	out := append([]Point2D(nil), pts...)
	sort.SliceStable(out, func(i, j int) bool {
		return math.Atan2(out[i].Y-cy, out[i].X-cx) < math.Atan2(out[j].Y-cy, out[j].X-cx)
	})
	return out
}

func closeLoop(ring []Point2D) []Line {
	lines := make([]Line, len(ring))
	for i := range ring {
		lines[i] = Line{A: ring[i], B: ring[(i+1)%len(ring)]}
	}
	return lines
}

func nearAnyCenter(p Point2D, circles []Circle) bool {
	for _, c := range circles {
		if absf(p.X-c.Center.X) < 1e-9 && absf(p.Y-c.Center.Y) < 1e-9 {
			return true
		}
	}
	return false
}

// forEachNamelessNode calls fn(nodeStart) for each nameless graph node (a sketch entity
// carrier): dcNodeTag(+0), id(+4), nullRef(+8), a schema marker(+12), a flag word(+16), then
// a geometry tag(+20). The +12 marker's VALUE is not keyed on (it is 0x0C20 in API-authored
// corpus parts but varies in real parts — 0x0A96, 0x0B5C, …), but its HIGH BIT is: a
// geometry carrier's marker is a small enum (< 0x80000000), whereas a structural/typed node
// sets the high bit there (e.g. 0x80000018), so the high bit cleanly rejects non-geometry
// nodes. The +16 flag word is NOT constrained: it is 0 for interior vertices but 0x40 for the
// sketch origin and 0x8000 for some points, so the earlier w@16==0 gate silently dropped the
// origin — forcing a fragile off-by-one synthesis in resolveByRefs and failing outright on any
// profile that drops two or more flagged points. collectItems still validates the geometry tag
// and coordinate range, so this coarse structural filter (nullRef + small marker) suffices.
func forEachNamelessNode(seg []byte, fn func(j int)) {
	for i := 0; i+24 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag {
			continue
		}
		if binary.LittleEndian.Uint32(seg[i+8:]) != nullRef ||
			binary.LittleEndian.Uint32(seg[i+12:]) >= nodeTagBase {
			continue
		}
		fn(i)
	}
}
