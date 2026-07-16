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
)

// Byte layout of a 2D entity's payload for this Inventor generation. The content header runs to 34
// (block size 0 since 2010, plus the >2018 and >2020 4-byte gaps), then a flags word, so the
// owning-sketch reference sits at 38 and a point's coordinates at 42. An edge instead has its
// endpoint List2 at 42.
const (
	entitySketchRefOffset = 38
	point2DPosOffset      = 42
	edgePointsListOffset  = 42
)

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
	case point2DNodeType, line2DNodeType, circle2DNodeType, arc2DNodeType, ellipse2DNodeType:
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
	switch n.typ {
	case point2DNodeType:
		if p, ok := point2DAt(n.payload); ok {
			s.Points = append(s.Points, p)
		}
	case line2DNodeType:
		if a, b, ok := edgeEndpoints(nodes, n.payload); ok {
			s.Lines = append(s.Lines, Line{A: a, B: b})
		}
	case circle2DNodeType:
		if c, ok := circle2DAt(nodes, n.payload); ok {
			s.Circles = append(s.Circles, c)
		}
	case arc2DNodeType:
		if a, ok := arc2DAt(nodes, n.payload); ok {
			s.Arcs = append(s.Arcs, a)
		}
	case ellipse2DNodeType:
		if e, ok := ellipse2DAt(nodes, n.payload); ok {
			s.Ellipses = append(s.Ellipses, e)
		}
	}
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

// edgeEndpoints resolves an edge's two endpoint references to coordinates — the exact
// connectivity, stated by the file rather than inferred from creation order.
func edgeEndpoints(nodes []dcNode, pay []byte) (a, b Point2D, ok bool) {
	refs, _, ok := refList2(pay, edgePointsListOffset)
	if !ok || len(refs) != 2 {
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
