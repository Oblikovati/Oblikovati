// SPDX-License-Identifier: GPL-2.0-only

package ipt

// The region an extrude actually consumes.
//
// A sketch usually holds several closed regions; the feature names the one (or ones) it uses. That
// naming runs patch -> FaceBound -> parts: each part is a Loop carrying an operation flag and an
// ordered list of edges, and each edge is a SketchEntityRef naming its curve by ASSOCIATIVE ID
// within the sketch — an id the curve itself carries in an FxAttrStyles attribute, reached through
// the attribute chain hanging off every node's `next` reference.
//
// Without this, the extrude falls back to "profile index 0", which was harmless only while the
// cluster decode saw almost no geometry. Once GraphSketches decoded whole sketches, index 0 became
// an arbitrary loop of many (BigChunkyPlate built 0.01x of its true volume).
//
// Grounded in the GPL InventorLoader reference: Read_1E3A132C "Loop" (operation 8=Fuse, 0=Cut),
// Read_90874D13 "SketchEntityRef" (entityAI), Read_90874D15 "FxAttrStyles" (associativeID),
// ReadHeaderLinkedElement, and createBoundary/addBoundaryPart's use of them.

import "encoding/binary"

const (
	loopNodeType       = 0x1E3A132C // Loop — an ordered edge cycle bounding part of a region
	loop3DNodeType     = 0xA3277869 // Loop3D — same layout, used on some parts
	entityRefNodeType  = 0x90874D13 // SketchEntityRef — names a curve by associative id
	attrStylesNodeType = 0x90874D15 // FxAttrStyles — carries a curve's associative id
)

// Byte layout for this Inventor generation (block size 0; the post-2023 4-byte gap included).
const (
	nodeAttrChainOffset = 6  // a content node's `next` child ref: the head of its attribute chain
	attrChainNextOffset = 26 // an attribute's own `next`, continuing the chain
	assocIDOffset       = 34 // FxAttrStyles.associativeID
	loopOperationOffset = 10 // Loop.operation
	loopEdgesOffset     = 18 // Loop.edges (List2 of refs)
	entityRefAIOffset   = 22 // SketchEntityRef.entityAI
	// The arc TRIM, from InventorLoader Read_90874D13. A loop names a circle WHOLE even where the
	// face walks only an arc of it, so which arc is not in the geometry — it is these fields.
	// point1AI/point2AI name the arc's start/end associative points, and posDir gives the
	// orientation ("required for e.g. circles", per the reference). The offsets corroborate
	// entityRefSketchOffset below: posDir is the single byte that pushes it off 4-alignment.
	entityRefPoint1Offset = 34
	entityRefPoint2Offset = 46
	entityRefPosDirOffset = 54
	// SketchEntityRef.sketch. NOT 4-aligned: the `posDir` boolean before it is a single byte, so
	// the reference lands at 55 (confirmed on every SketchEntityRef in the corpus).
	entityRefSketchOffset = 55
)

// loopAddsMaterial is the operation bit that marks a loop as bounding material (Fuse). A loop
// WITHOUT it is a Cut: a hole inside the region, not something to discard — dropping those lost
// the linkage's two bores and made its region unmatchable.
const loopAddsMaterial = 0x18

// RegionLoop is one closed boundary of a region: the curves bounding it, and whether it adds
// material (the outer boundary) or cuts it away (a hole).
type RegionLoop struct {
	Edges []RegionEdge
	Cut   bool
}

// RegionEdge is one curve of a region's boundary, identified by geometry so it can be matched
// against a rebuilt profile's entities. Exactly one of the shapes is meaningful, per Kind.
type RegionEdge struct {
	Kind    EdgeKind
	Line    Line
	Circle  Circle
	Arc     Arc
	Ellipse EllipseArc
}

// EllipseArc is a region-boundary ellipse edge: the full ellipse geometry plus the span the face
// walks (Start..End, ordered by the reference's posDir like an arc). Start == End (both zero) means
// the loop names the whole ellipse, as a bore's hole loop does with a circle.
type EllipseArc struct {
	Center     Point2D
	MajorAxis  Point2D // unit direction of the major axis
	MajorR     float64
	MinorR     float64
	Start, End Point2D
}

// EdgeKind names which curve a RegionEdge carries.
type EdgeKind int

const (
	EdgeLine EdgeKind = iota
	EdgeCircle
	EdgeArc
	EdgeEllipse
)

// ExtrudeRegions returns, for each extrude in feature order, the loops bounding the region it
// consumes — aligned with DecodeExtrudes and ExtrudeProfiles. An extrude whose region can't be
// resolved yields a nil slice, so callers decline rather than guess a profile.
func ExtrudeRegions(d *Document) [][]RegionLoop {
	nodes := dcNodes(d)
	_, sketchIndex := sketchOrdinals(nodes)
	assoc := associativeEntities(nodes, sketchIndex)
	patchLoops := boundPatchLoops(nodes)
	var out [][]RegionLoop
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if _, ok := extrudeDepth(nodes, n.payload); !ok {
			continue
		}
		out = append(out, regionOf(nodes, n.payload, patchLoops, assoc, sketchIndex))
	}
	return out
}

// assocKey identifies a curve by its associative id within one sketch: the ids restart per sketch,
// so the sketch is part of the key (14_box_two's two rectangles both use ids 5..8).
type assocKey struct {
	sketch int
	id     int
}

// regionOf collects every loop bounding one extrude's patch, outer and holes alike.
func regionOf(nodes []dcNode, pay []byte, patchLoops map[int][]int, assoc map[assocKey]dcNode, sketchIndex map[int]int) []RegionLoop {
	props, ok := featureProperties(pay)
	if !ok || len(props) <= propProfile {
		return nil
	}
	var out []RegionLoop
	for _, loopRef := range patchLoops[props[propProfile]] {
		loop, ok := nodeAt(nodes, loopRef)
		if !ok {
			return nil
		}
		l, ok := loopAt(nodes, loop, assoc, sketchIndex)
		if !ok {
			return nil // an unreadable loop makes the region unknown; say so rather than part-guess
		}
		out = append(out, l)
	}
	return out
}

// loopAt resolves one loop's ordered edges to their curves, and whether it cuts rather than adds.
func loopAt(nodes []dcNode, loop dcNode, assoc map[assocKey]dcNode, sketchIndex map[int]int) (RegionLoop, bool) {
	if len(loop.payload) < loopEdgesOffset+8 {
		return RegionLoop{}, false
	}
	cut := binary.LittleEndian.Uint32(loop.payload[loopOperationOffset:])&loopAddsMaterial == 0
	refs, _, ok := refList2(loop.payload, loopEdgesOffset)
	if !ok {
		return RegionLoop{}, false
	}
	out := make([]RegionEdge, 0, len(refs))
	for _, r := range refs {
		e, ok := resolveRegionEdge(nodes, r, assoc, sketchIndex)
		if !ok {
			return RegionLoop{}, false
		}
		out = append(out, e)
	}
	return RegionLoop{Edges: out, Cut: cut}, true
}

// resolveRegionEdge turns one SketchEntityRef into the curve it names, via the associative id.
func resolveRegionEdge(nodes []dcNode, ref int, assoc map[assocKey]dcNode, sketchIndex map[int]int) (RegionEdge, bool) {
	er, ok := nodeAt(nodes, ref)
	if !ok || er.typ != entityRefNodeType || len(er.payload) < entityRefAIOffset+4 {
		return RegionEdge{}, false
	}
	sk, ok := entityRefSketch(er, sketchIndex)
	if !ok {
		return RegionEdge{}, false
	}
	key := assocKey{sk, int(binary.LittleEndian.Uint32(er.payload[entityRefAIOffset:]) & refIndexMask)}
	curve, ok := assoc[key]
	if !ok {
		return RegionEdge{}, false
	}
	e, ok := regionEdgeOf(nodes, curve)
	if !ok {
		return RegionEdge{}, false
	}
	if e.Kind == EdgeCircle {
		e = trimCircleEdge(e, er, assoc, sk)
	}
	if e.Kind == EdgeEllipse {
		e = trimEllipseEdge(e, er, assoc, sk)
	}
	return e, true
}

// trimEllipseEdge records the span an ellipse-boundary edge walks, from the reference's own trim
// points — the ellipse analogue of trimCircleEdge. The start/end points lie ON the ellipse; the
// consumer (translate.ellipseArcPolyline) turns them into the swept parameter range. p1 == p2 (or an
// unresolvable point) means the face uses the whole ellipse, left as Start == End for the consumer.
func trimEllipseEdge(e RegionEdge, er dcNode, assoc map[assocKey]dcNode, sk int) RegionEdge {
	p1, ok1 := entityRefPoint(er, entityRefPoint1Offset, assoc, sk)
	p2, ok2 := entityRefPoint(er, entityRefPoint2Offset, assoc, sk)
	if !ok1 || !ok2 || samePoint2D(p1, p2) {
		return e // whole ellipse
	}
	start, end := p1, p2
	if !entityRefPosDir(er) {
		start, end = p2, p1
	}
	e.Ellipse.Start, e.Ellipse.End = start, end
	return e
}

// trimCircleEdge cuts a named WHOLE circle back to the arc the face actually walks, from the
// reference's own trim fields. Without it the arc has to be guessed from geometry, and no geometric
// rule can do it: an obround's two arcs are EXACTLY 180 degrees, so "the shorter one" is a coin
// flip and picks the inner halves — a lens instead of the obround, at half the area (#26).
//
// Mirrors InventorLoader's Part.Circle branch (importerFreeCAD Create..._createBoundary): the arc
// runs counter-clockwise p1 -> p2 when posDir is set and p2 -> p1 when it is not; p1 == p2 (or an
// unresolvable point) means the face really does use the whole circle, as a bore's hole loop does.
func trimCircleEdge(e RegionEdge, er dcNode, assoc map[assocKey]dcNode, sk int) RegionEdge {
	p1, ok1 := entityRefPoint(er, entityRefPoint1Offset, assoc, sk)
	p2, ok2 := entityRefPoint(er, entityRefPoint2Offset, assoc, sk)
	if !ok1 || !ok2 || samePoint2D(p1, p2) {
		return e // whole circle
	}
	start, end := p1, p2
	if !entityRefPosDir(er) {
		start, end = p2, p1
	}
	return RegionEdge{Kind: EdgeArc, Arc: Arc{Center: e.Circle.Center, Radius: e.Circle.Radius, Start: start, End: end}}
}

// entityRefPoint resolves one of a SketchEntityRef's trim points.
func entityRefPoint(er dcNode, off int, assoc map[assocKey]dcNode, sk int) (Point2D, bool) {
	if len(er.payload) < off+4 {
		return Point2D{}, false
	}
	n, ok := assoc[assocKey{sk, int(binary.LittleEndian.Uint32(er.payload[off:]) & refIndexMask)}]
	if !ok || n.typ != point2DNodeType {
		return Point2D{}, false
	}
	return point2DAt(n.payload)
}

// entityRefPosDir reports the edge's orientation flag.
func entityRefPosDir(er dcNode) bool {
	if len(er.payload) <= entityRefPosDirOffset {
		return false
	}
	return er.payload[entityRefPosDirOffset] != 0
}

// entityRefSketch reports which sketch a SketchEntityRef's associative id is scoped to — the ids
// restart per sketch, so the ref alone is meaningless without it.
func entityRefSketch(er dcNode, sketchIndex map[int]int) (int, bool) {
	if len(er.payload) < entityRefSketchOffset+4 {
		return 0, false
	}
	i, ok := sketchIndex[int(binary.LittleEndian.Uint32(er.payload[entityRefSketchOffset:])&refIndexMask)]
	return i, ok
}

// regionEdgeOf reads a curve node into a geometric edge descriptor.
func regionEdgeOf(nodes []dcNode, n dcNode) (RegionEdge, bool) {
	switch n.typ {
	case line2DNodeType:
		a, b, ok := edgeEndpoints(nodes, n.payload)
		return RegionEdge{Kind: EdgeLine, Line: Line{A: a, B: b}}, ok
	case circle2DNodeType:
		c, ok := circle2DAt(nodes, n.payload)
		return RegionEdge{Kind: EdgeCircle, Circle: c}, ok
	case arc2DNodeType:
		a, ok := arc2DAt(nodes, n.payload)
		return RegionEdge{Kind: EdgeArc, Arc: a}, ok
	case ellipse2DNodeType:
		el, ok := ellipse2DAt(nodes, n.payload)
		return RegionEdge{Kind: EdgeEllipse, Ellipse: EllipseArc{
			Center: el.Center, MajorAxis: el.MajorAxis, MajorR: el.MajorR, MinorR: el.MinorR,
		}}, ok
	}
	return RegionEdge{}, false
}

// associativeEntities maps (sketch, associative id) to the curve node carrying that id.
func associativeEntities(nodes []dcNode, sketchIndex map[int]int) map[assocKey]dcNode {
	out := map[assocKey]dcNode{}
	for _, n := range nodes {
		owner, ok := entityOwner(n, sketchIndex)
		if !ok {
			continue
		}
		if id, ok := associativeID(nodes, n); ok {
			out[assocKey{owner, id}] = n
		}
	}
	return out
}

// associativeID walks a node's attribute chain to the FxAttrStyles that carries its id.
func associativeID(nodes []dcNode, n dcNode) (int, bool) {
	if len(n.payload) < nodeAttrChainOffset+4 {
		return 0, false
	}
	ref := int(binary.LittleEndian.Uint32(n.payload[nodeAttrChainOffset:]) & refIndexMask)
	for hops := 0; hops < maxAttrChainHops; hops++ {
		a, ok := nodeAt(nodes, ref)
		if !ok {
			return 0, false
		}
		if a.typ == attrStylesNodeType {
			if len(a.payload) < assocIDOffset+4 {
				return 0, false
			}
			return int(binary.LittleEndian.Uint32(a.payload[assocIDOffset:]) & refIndexMask), true
		}
		if len(a.payload) < attrChainNextOffset+4 {
			return 0, false
		}
		ref = int(binary.LittleEndian.Uint32(a.payload[attrChainNextOffset:]) & refIndexMask)
	}
	return 0, false
}

// maxAttrChainHops bounds the attribute walk so a corrupt or cyclic chain can't spin.
const maxAttrChainHops = 32

// boundPatchLoops maps each BoundaryPatch ordinal to the loops bounding it, from the FaceBound that
// names the patch as its proxy. The loops are the FaceBound's first reference list.
func boundPatchLoops(nodes []dcNode) map[int][]int {
	out := map[int][]int{}
	for _, n := range nodes {
		if n.typ != faceBoundNodeType && n.typ != faceBoundOuterNodeType {
			continue
		}
		parts, _, ok := refList2(n.payload, faceBoundHeaderEnd)
		if !ok {
			continue
		}
		_, patchRef, ok := faceBoundRefs(n.payload)
		if !ok {
			continue
		}
		out[patchRef] = append(out[patchRef], loopRefsAmong(nodes, parts)...)
	}
	return out
}

// loopRefsAmong keeps the references that name a Loop.
func loopRefsAmong(nodes []dcNode, refs []int) []int {
	var out []int
	for _, r := range refs {
		if n, ok := nodeAt(nodes, r); ok && (n.typ == loopNodeType || n.typ == loop3DNodeType) {
			out = append(out, r)
		}
	}
	return out
}
