// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// filletLoop is one boundary loop of a face as a ring of points with the curve along each
// segment: curves[i] runs pts[i]→pts[(i+1)%n], or nil for a straight line. (Used to build
// curved-faced results like fillets, where some edges are arcs.)
type filletLoop struct {
	pts    []math.Point3
	curves []geom.Curve3
}

// filletFace is a result face: its surface (plane, cylinder, …) and boundary loops (outer
// first). parent is the lineage of the input entity that GENERATED this face (ADR-0043): the
// original face for a transformed face, the filleted edge (with a "cyl" marker) for a blend
// cylinder. The zero lineage means "no provenance" — that face, and any edge/vertex touching it,
// keeps its build-order name (see assembleBody / topo.RelineageByFaceProvenance).
type filletFace struct {
	surface geom.Surface
	loops   []filletLoop
	parent  topo.Lineage
}

// edgeRec remembers a built shared edge and the direction it was stored in, so a second
// face referencing the same vertex pair gets it reversed.
type edgeRec struct {
	edge     *topo.Edge
	from, to int
}

// assembleBody welds the faces' loop points into a watertight body: one shared edge per
// undirected vertex pair (carrying the curve the first user supplied, oriented its way), and
// a face per surface. A closed result (every edge used twice) is a solid. Curves let a face
// carry arc edges (a fillet's end arcs) alongside straight ones.
func assembleBody(faces []filletFace, tag string) *topo.Body {
	var pts []math.Point3
	for _, f := range faces {
		for _, l := range f.loops {
			pts = append(pts, l.pts...)
		}
	}
	w := newPointWelder(ResolutionForPoints(pts).Weld())
	rings := make([][][]int, len(faces))
	for i, f := range faces {
		for _, l := range f.loops {
			rings[i] = append(rings[i], w.weldRing(l.pts))
		}
	}
	bld := topo.NewBuilder(curvedSolid(faces, rings, w.points), topo.NewLineage(topo.Tok(tag, "body", 0)))
	tv := make([]*topo.Vertex, len(w.points))
	for i, p := range w.points {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(tag, "v", i)))
	}
	ec := &edgeCatalog{bld: bld, verts: w.points, tv: tv, edges: map[[2]int]edgeRec{}, tag: tag}
	provByFace := addCurvedFaces(bld, faces, rings, ec, tag)
	body := bld.Build()
	// Provenance naming (ADR-0043): once the faces are named by their parents, the edges and
	// vertices are renamed by their bordering faces' provenance, replacing the build-order counter
	// that renumbered on an upstream edit. A body with no provenanced faces is left as built.
	if len(provByFace) > 0 {
		body.RelineageByFaceProvenance(provByFace, topo.Tok("fillet", "x", 0), topo.Tok("fillet", "seg", 0))
	}
	return body
}

// addCurvedFaces builds each result face against the edge catalog and names it by its provenance
// parent when it has one (a transformed original face, a blend cylinder), else a build-order
// ordinal under tag (e.g. a variable-fillet ruling strip, not yet provenanced). Returns the
// provenanced faces so the caller can rename their edges/vertices by provenance (ADR-0043).
func addCurvedFaces(bld *topo.Builder, faces []filletFace, rings [][][]int, ec *edgeCatalog, tag string) map[*topo.Face]topo.Lineage {
	provByFace := map[*topo.Face]topo.Lineage{}
	for fi, f := range faces {
		specs := make([]topo.LoopSpec, len(f.loops))
		for li, ring := range rings[fi] {
			specs[li] = curvedLoopSpec(li == 0, ring, f.loops[li].curves, ec)
		}
		lin := f.parent
		if len(lin.Key()) == 0 {
			lin = topo.NewLineage(topo.Tok(tag, "f", fi))
		}
		face := bld.AddFace(f.surface, lin, specs...)
		if len(f.parent.Key()) > 0 {
			provByFace[face] = f.parent
		}
	}
	return provByFace
}

// curvedSolid reports whether every undirected edge of the faces is used exactly twice (the
// combinatorial test for a closed solid), computed before assembly to set the builder mode.
func curvedSolid(faces []filletFace, rings [][][]int, _ []math.Point3) bool {
	use := map[[2]int]int{}
	for fi := range faces {
		for _, ring := range rings[fi] {
			for k := 0; k < len(ring); k++ {
				use[canon2(ring[k], ring[(k+1)%len(ring)])]++
			}
		}
	}
	for _, c := range use {
		if c != 2 {
			return false
		}
	}
	return true
}

// edgeCatalog creates one shared topo edge per undirected vertex pair on demand, reusing it
// (reversed) for the second face.
type edgeCatalog struct {
	bld   *topo.Builder
	verts []math.Point3
	tv    []*topo.Vertex
	edges map[[2]int]edgeRec
	tag   string
}

// use returns the loop use for the directed segment a→b with the given curve (nil ⇒ a line),
// creating the shared edge the first time in its a→b direction.
func (c *edgeCatalog) use(a, b int, curve geom.Curve3) topo.Use {
	key := canon2(a, b)
	if rec, ok := c.edges[key]; ok {
		return topo.Use{Edge: rec.edge, Reversed: rec.from != a}
	}
	if curve == nil {
		curve = geom.NewLineSegment(c.verts[a], c.verts[b])
	}
	e := c.bld.AddEdge(curve, c.tv[a], c.tv[b], topo.NewLineage(topo.Tok(c.tag, "e", len(c.edges))))
	c.edges[key] = edgeRec{edge: e, from: a, to: b}
	return topo.Use{Edge: e, Reversed: false}
}

// curvedLoopSpec builds a face loop from a ring of welded indices and the per-segment curves.
func curvedLoopSpec(outer bool, ring []int, curves []geom.Curve3, ec *edgeCatalog) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for k := range ring {
		uses[k] = ec.use(ring[k], ring[(k+1)%len(ring)], curveAt(curves, k))
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// curveAt returns curves[k] if present, else nil (a straight segment).
func curveAt(curves []geom.Curve3, k int) geom.Curve3 {
	if k < len(curves) {
		return curves[k]
	}
	return nil
}
