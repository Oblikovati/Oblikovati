// SPDX-License-Identifier: GPL-2.0-only

package ipt

// Exact sketch decoding from the DC node graph.
//
// Every 2D entity node carries a back-reference to the Sketch2D node that OWNS it, and every edge
// names its endpoints by reference to Point2D nodes. So a sketch's membership and its connectivity
// are both stated in the file — there is nothing to infer. This replaces the 800-byte "cluster"
// heuristic, which split one sketch across clusters or merged several into one, and the
// rank-alignment that guessed which point a curve's endpoint reference meant.
//
// Measured against the files: CompressionRollerActuatorLinkage1 has ONE Sketch2D node but the
// cluster decode produced 3 sketches; BigChunkyPlate has 63 but the cluster decode produced 8.
//
// Node types and layouts are grounded in the GPL InventorLoader reference
// (_iptformat/reference-InventorLoader): Read_90874D11 "PlanarSketch", Read_CE52DF35 "SketchPoint",
// Read_CE52DF3A "SketchLine", Read_CE52DF3B "SketchCircle", Read_160915E2 "SketchArc", and the
// shared ReadHeaderSketch2DEntity / ReadHeader2DEdge headers.

import (
	"encoding/binary"
	"math"
)

const (
	sketch2DNodeType  = 0x90874D11 // PlanarSketch — one per sketch
	point2DNodeType   = 0xCE52DF35 // SketchPoint — carries its coordinates inline
	line2DNodeType    = 0xCE52DF3A // SketchLine — endpoints by reference
	circle2DNodeType  = 0xCE52DF3B // SketchCircle — centre by reference, radius inline
	arc2DNodeType     = 0x160915E2 // SketchArc — centre by reference, radius inline
	ellipse2DNodeType = 0x4507D460 // SketchEllipse — centre by reference, axis + both radii inline
	spline2DNodeType  = 0xF9372FD4 // SketchSpline — interpolating (fit) spline through point refs
)

// Byte layout of a 2D entity's payload for this Inventor generation. The content header runs to 34
// (block size 0 since 2010, plus the >2018 and >2020 4-byte gaps), then a flags word, so the
// owning-sketch reference sits at 38 and a point's coordinates at 42. An edge instead has its
// endpoint List2 at 42.
const (
	entityFlags2Offset    = 34 // the entity's flag word; constructionFlag lives here
	entitySketchRefOffset = 38
	point2DPosOffset      = 42
	edgePointsListOffset  = 42
)

// arcFlag marks a SketchCircle node that is really an ARC (open), sitting in the entity flag word
// at offset 34 one bit below constructionFlag (0x00080000). This Inventor generation serialises a
// sketch arc as a SketchCircle (0xCE52DF3B) node with this bit set and two on-circle endpoints,
// NOT as a distinct SketchArc (0x160915E2) — that type does not occur in the ReelToReel corpus at
// all. A full circle whose rim merely carries coincidence points (a slot's pin touched by two
// tangent lines) leaves the bit clear, so it is still emitted as a circle. Verified across the
// corpus: the linkage's tangent-point circles read clear (4 circles / 0 arcs, matching Inventor)
// while BigChunkyPlate's and TorquimeterDisk's corner rounds read set.
const arcFlag = 0x00040000

// GraphSketches returns one sketch per Sketch2D node, in node order, decoded exactly: entities are
// grouped by the sketch they declare as their owner, and every edge's endpoints are resolved
// through its point references. Returns nil when the part has no Sketch2D node, so callers can fall
// back to the cluster decode.
func GraphSketches(d *Document) []Sketch {
	nodes := dcNodes(d)
	order, index := sketchOrdinals(nodes)
	if len(order) == 0 {
		return nil
	}
	out := make([]Sketch, len(order))
	for i := range out {
		out[i].Resolved = true // exact by construction: no inference to be unsure about
		if n, ok := nodeAt(nodes, order[i]); ok {
			out[i].Plane, out[i].PlaneOK = sketchPlane(nodes, n)
		}
	}
	for _, n := range nodes {
		owner, ok := entityOwner(n, index)
		if !ok {
			continue
		}
		addGraphEntity(&out[owner], nodes, n)
	}
	return out
}

// sketchOrdinals returns the Sketch2D nodes' 1-based ordinals in order, and the reverse map from
// ordinal to sketch index.
func sketchOrdinals(nodes []dcNode) ([]int, map[int]int) {
	var order []int
	index := map[int]int{}
	for i, n := range nodes {
		if n.typ == sketch2DNodeType {
			index[i+1] = len(order)
			order = append(order, i+1)
		}
	}
	return order, index
}

// entityOwner reports which sketch a 2D entity node belongs to, from the owning-sketch reference
// every sketch entity carries. Non-entity nodes and entities owned by something that isn't a
// Sketch2D (a block definition, say) are declined.
func entityOwner(n dcNode, index map[int]int) (int, bool) {
	switch n.typ {
	case point2DNodeType, line2DNodeType, circle2DNodeType, arc2DNodeType, ellipse2DNodeType, spline2DNodeType:
	default:
		return 0, false
	}
	if len(n.payload) < entitySketchRefOffset+4 {
		return 0, false
	}
	ref := int(binary.LittleEndian.Uint32(n.payload[entitySketchRefOffset:]) & refIndexMask)
	i, ok := index[ref]
	return i, ok
}

// addGraphEntity decodes one entity onto its sketch. An entity that can't be read is skipped: a
// missing curve is a visibly incomplete profile, which the body gate then catches, whereas a
// guessed one would be silently wrong.
func addGraphEntity(s *Sketch, nodes []dcNode, n dcNode) {
	c := isConstructionEntity(n.payload)
	switch n.typ {
	case point2DNodeType:
		if p, ok := point2DAt(n.payload); ok {
			s.Points = append(s.Points, p)
		}
	case line2DNodeType:
		if a, b, ok := edgeEndpoints(nodes, n.payload); ok {
			s.Lines = append(s.Lines, Line{A: a, B: b})
			s.LineConstruction = append(s.LineConstruction, c)
		}
	case circle2DNodeType:
		// A SketchCircle marked open (arcFlag) with two endpoints is an arc — see arcFlag. Fall
		// through to a circle when the bit is clear, or when its endpoints don't resolve.
		if a, ok := arcFromCircleNode(nodes, n.payload); ok {
			s.Arcs = append(s.Arcs, a)
			s.ArcConstruction = append(s.ArcConstruction, c)
		} else if circ, ok := circle2DAt(nodes, n.payload); ok {
			s.Circles = append(s.Circles, circ)
			s.CircleConstruction = append(s.CircleConstruction, c)
		}
	case arc2DNodeType:
		if a, ok := arc2DAt(nodes, n.payload); ok {
			s.Arcs = append(s.Arcs, a)
			s.ArcConstruction = append(s.ArcConstruction, c)
		}
	case ellipse2DNodeType:
		if e, ok := ellipse2DAt(nodes, n.payload); ok {
			s.Ellipses = append(s.Ellipses, e)
		}
	case spline2DNodeType:
		if sp, ok := spline2DAt(nodes, n.payload); ok {
			s.Splines = append(s.Splines, sp)
			s.SplineConstruction = append(s.SplineConstruction, c)
		}
	}
}

// isConstructionEntity reports whether a 2D entity is construction (reference) geometry, from the
// flag word every sketch entity carries. Verified on the linkage: its centreline reads 0x00080000
// there while the two profile lines read 0.
func isConstructionEntity(pay []byte) bool {
	if len(pay) < entityFlags2Offset+4 {
		return false
	}
	return binary.LittleEndian.Uint32(pay[entityFlags2Offset:])&constructionFlag != 0
}

// ellipse2DAt reads a SketchEllipse: centre by reference, then the major-axis unit direction and
// the two radii inline.
func ellipse2DAt(nodes []dcNode, pay []byte) (Ellipse, bool) {
	i, ok := skipEdgeHeader(pay)
	if !ok || i+36 > len(pay) {
		return Ellipse{}, false
	}
	c, ok := referencedPoint(nodes, int(binary.LittleEndian.Uint32(pay[i:])&refIndexMask))
	if !ok {
		return Ellipse{}, false
	}
	e := Ellipse{
		Center:    c,
		MajorAxis: Point2D{f64(pay, i+4), f64(pay, i+12)},
		MajorR:    f64(pay, i+20),
		MinorR:    f64(pay, i+28),
	}
	if math.IsNaN(e.MajorR) || math.IsNaN(e.MinorR) || e.MajorR <= 0 || e.MinorR <= 0 {
		return Ellipse{}, false
	}
	return e, true
}

// spline2DAt reads a SketchSpline: its fit points, in order, from the entity's point-reference
// list. Inventor stores an interpolating spline whose referenced points lie ON the curve, so the
// same points fed to a fit spline reproduce it. Needs at least two points; returns ok=false (the
// caller then drops the curve) if any reference doesn't resolve to a Point2D. Closed is left false
// until a closed-spline node is available to locate its flag — the corpus fixtures are all open.
func spline2DAt(nodes []dcNode, pay []byte) (Spline, bool) {
	refs, _, ok := refList2(pay, edgePointsListOffset)
	if !ok || len(refs) < 2 {
		return Spline{}, false
	}
	pts := make([]Point2D, 0, len(refs))
	for _, r := range refs {
		p, ok := referencedPoint(nodes, r)
		if !ok {
			return Spline{}, false
		}
		pts = append(pts, p)
	}
	return Spline{Points: pts}, true
}

// point2DAt reads a SketchPoint's inline coordinates (cm).
func point2DAt(pay []byte) (Point2D, bool) {
	if len(pay) < point2DPosOffset+16 {
		return Point2D{}, false
	}
	x, y := f64(pay, point2DPosOffset), f64(pay, point2DPosOffset+8)
	if math.IsNaN(x) || math.IsNaN(y) {
		return Point2D{}, false
	}
	return Point2D{x, y}, true
}

// edgeEndpoints resolves an edge's endpoints to coordinates — the exact connectivity, stated by the
// file rather than inferred from creation order.
//
// The FIRST TWO references are the endpoints (InventorLoader's addSketch_Line2D reads points[0] and
// points[1]); an edge lists every point lying ON it, so further references are midpoints and
// coincidences, not ends. Requiring exactly two dropped any line a point was constrained to —
// BigChunkyPlate's base profile lost an edge that way, which nulled its whole region and left the
// part with only its cuts.
func edgeEndpoints(nodes []dcNode, pay []byte) (a, b Point2D, ok bool) {
	refs, _, ok := refList2(pay, edgePointsListOffset)
	if !ok || len(refs) < 2 {
		return a, b, false
	}
	if a, ok = referencedPoint(nodes, refs[0]); !ok {
		return a, b, false
	}
	b, ok = referencedPoint(nodes, refs[1])
	return a, b, ok
}

// circle2DAt reads a SketchCircle: its centre by reference, its radius inline after the edge's
// point list.
func circle2DAt(nodes []dcNode, pay []byte) (Circle, bool) {
	c, r, ok := centreAndRadius(nodes, pay)
	if !ok {
		return Circle{}, false
	}
	return Circle{Center: c, Radius: r}, true
}

// arc2DAt reads a SketchArc: centre + radius like a circle, with the endpoints from the edge's
// point list (start → end in creation order).
func arc2DAt(nodes []dcNode, pay []byte) (Arc, bool) {
	c, r, ok := centreAndRadius(nodes, pay)
	if !ok {
		return Arc{}, false
	}
	start, end, ok := edgeEndpoints(nodes, pay)
	if !ok {
		return Arc{}, false
	}
	return Arc{Center: c, Radius: r, Start: start, End: end}, true
}

// arcFromCircleNode reads a SketchCircle node as an arc when the file marks it open (arcFlag at
// offset 34) and it carries two resolvable endpoints. Returns ok=false — so the caller emits a
// circle — when the bit is clear or the endpoints don't resolve. See arcFlag for why arcs live
// under the SketchCircle type in this Inventor generation.
func arcFromCircleNode(nodes []dcNode, pay []byte) (Arc, bool) {
	if len(pay) < entityFlags2Offset+4 {
		return Arc{}, false
	}
	if binary.LittleEndian.Uint32(pay[entityFlags2Offset:])&arcFlag == 0 {
		return Arc{}, false
	}
	a, ok := arc2DAt(nodes, pay)
	if !ok || a.Start == a.End { // a zero-span "arc" is a degenerate full circle
		return Arc{}, false
	}
	return a, true
}

// centreAndRadius reads the centre reference and radius that follow an edge's header on a circle
// or arc. The lists in that header are variable length, so the fields are found by walking them.
func centreAndRadius(nodes []dcNode, pay []byte) (Point2D, float64, bool) {
	i, ok := skipEdgeHeader(pay)
	if !ok || i+12 > len(pay) {
		return Point2D{}, 0, false
	}
	c, ok := referencedPoint(nodes, int(binary.LittleEndian.Uint32(pay[i:])&refIndexMask))
	if !ok {
		return Point2D{}, 0, false
	}
	r := f64(pay, i+4)
	if math.IsNaN(r) || r <= 0 {
		return Point2D{}, 0, false
	}
	return c, r, true
}

// skipEdgeHeader advances past a 2D edge's variable-length header — the endpoint list, then a u32
// pair and up to two further reference lists — returning the offset of the type-specific fields.
func skipEdgeHeader(pay []byte) (int, bool) {
	_, i, ok := refList2(pay, edgePointsListOffset)
	if !ok || i+8 > len(pay) {
		return 0, false
	}
	kind := binary.LittleEndian.Uint32(pay[i:])
	i += 8
	if kind != 1 {
		return i, true
	}
	for k := 0; k < 2; k++ { // lst1 then lst2 (this generation writes both)
		_, next, ok := refList2(pay, i)
		if !ok {
			return i, true
		}
		i = next
	}
	return i, true
}

// referencedPoint resolves a reference that must name a Point2D, and returns its coordinates.
func referencedPoint(nodes []dcNode, ref int) (Point2D, bool) {
	n, ok := nodeAt(nodes, ref)
	if !ok || n.typ != point2DNodeType {
		return Point2D{}, false
	}
	return point2DAt(n.payload)
}

// refList2 reads a List2 of 4-byte references at off, returning them and the offset past the list.
// The list is a marker, a count, an 8-byte header when non-empty, then the elements.
func refList2(pay []byte, off int) (refs []int, next int, ok bool) {
	if off+8 > len(pay) ||
		binary.LittleEndian.Uint16(pay[off:]) != list2Marker ||
		binary.LittleEndian.Uint16(pay[off+2:]) != list2Owner {
		return nil, 0, false
	}
	cnt := int(binary.LittleEndian.Uint32(pay[off+4:]))
	if cnt < 0 || cnt > maxListElements {
		return nil, 0, false
	}
	ds := off + 8
	if cnt > 0 {
		ds += 8
	}
	if ds+4*cnt > len(pay) {
		return nil, 0, false
	}
	for i := 0; i < cnt; i++ {
		refs = append(refs, int(binary.LittleEndian.Uint32(pay[ds+i*4:])&refIndexMask))
	}
	return refs, ds + 4*cnt, true
}
