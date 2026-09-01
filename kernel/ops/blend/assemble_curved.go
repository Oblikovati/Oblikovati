// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// filletLoop is one boundary loop of a face as a ring of points with the curve along each
// segment: curves[i] runs pts[i]→pts[(i+1)%n], or nil for a straight line. (Used to build
// curved-faced results like fillets, where some edges are arcs.)
// srcV[i]/srcE[i] carry the SOURCE topo identity of pts[i] and of the segment leaving it (0 =
// op-generated): they preserve the boolean's tangent-contact topology across the re-weld. srcV
// keeps pinch-split coincident VERTICES distinct; srcE keeps two coincident EDGES that share the
// same endpoints distinct (the flush/box tangency where 4 faces meet on one line), so neither
// collapses back into a non-manifold pinch (#1600, method C).
type filletLoop struct {
	pts    []math.Point3
	curves []geom.Curve3
	srcV   []uint64
	srcE   []uint64
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
	closed   bool        // a==b full-circle seam: the 2nd coedge is antiparallel by the manifold invariant
	curve    geom.Curve3 // the OFFER this edge was built from (nil = the offering consumer had none)
}

// collectLoopPoints gathers every loop point across all faces — the cloud the point-welder
// resolves into shared vertices.
func collectLoopPoints(faces []filletFace) []math.Point3 {
	var pts []math.Point3
	for _, f := range faces {
		for _, l := range f.loops {
			pts = append(pts, l.pts...)
		}
	}
	return pts
}

// filletAssemblyTag is the lineage token namespace every fillet result body is built under (its body,
// op-generated vertices, edges, and un-provenanced faces). A single constant — the tag is invariant
// across all assembleBody call sites, so it is not a per-call parameter.
const filletAssemblyTag = "fillet"

// assembleBody welds the faces' loop points into a watertight body: one shared edge per
// undirected vertex pair, carrying the best curve its two users offered (the first offer, oriented
// its way; a later curve replaces an earlier nil — see resolveCurveOffer), and a face per surface.
// A closed result (every edge used twice) is a solid. Curves let a face carry arc edges (a fillet's
// end arcs) alongside straight ones.
func assembleBody(faces []filletFace) *topo.Body {
	tag := filletAssemblyTag
	pts := collectLoopPoints(faces)
	weld := tol.ForPoints(pts).Weld()
	w := mesh.NewPointWelder(weld)
	rings, deadLoops := weldRings(faces, w, weld)
	classes := pairEdgeClasses(faces, rings)
	orientFilletShell(faces, rings, classes) // B2: unify loop windings before the catalog builds co-edges
	bld := topo.NewBuilder(curvedSolid(faces, rings, classes), topo.NewLineage(topo.Tok(tag, "body", 0)))
	recordDeadLoopRefusals(bld, deadLoops) // #3389: refuse a collapsing loop here, not at a later Validate
	tv := make([]*topo.Vertex, len(w.Points))
	for i, p := range w.Points {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(tag, "v", i)))
	}
	ec := &edgeCatalog{bld: bld, verts: w.Points, tv: tv, edges: map[seamEdgeKey]edgeRec{}, classes: classes, tag: tag, weld: weld}
	provByFace := addCurvedFaces(bld, faces, rings, ec)
	ec.diagnoseCatalogUse() // positive marker: this body IS the catalog's own output (I2/I4)
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
func addCurvedFaces(bld *topo.Builder, faces []filletFace, rings [][][]int, ec *edgeCatalog) map[*topo.Face]topo.Lineage {
	provByFace := map[*topo.Face]topo.Lineage{}
	for fi, f := range faces {
		specs := make([]topo.LoopSpec, len(f.loops))
		for li, ring := range rings[fi] {
			specs[li] = curvedLoopSpec(li == 0, ring, f.loops[li].curves, f.loops[li].srcE, ec)
		}
		lin := f.parent
		if len(lin.Key()) == 0 {
			lin = topo.NewLineage(topo.Tok(filletAssemblyTag, "f", fi))
		}
		face := bld.AddFace(f.surface, lin, specs...)
		if len(f.parent.Key()) > 0 {
			provByFace[face] = f.parent
		}
	}
	return provByFace
}

// curvedSolid reports whether every reconstructed edge is used exactly twice (the combinatorial
// test for a closed solid), computed before assembly to set the builder mode. Edges are keyed by
// their identity CLASS (a coincident tangent seam splits into two edges), matching what the edge
// catalog builds — so a manifold tangency counts as two twice-used edges, not one four-used one.
func curvedSolid(faces []filletFace, rings [][][]int, classes map[[2]int]int) bool {
	use := map[seamEdgeKey]int{}
	for fi := range faces {
		for li, ring := range rings[fi] {
			ids := faces[fi].loops[li].srcE
			for k := range ring {
				a, b := ring[k], ring[(k+1)%len(ring)]
				use[seamEdgeKey{probe.Canon2(a, b), edgeClassOf(a, b, probe.SrcIDAt(ids, k), classes)}]++
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

// pairEdgeClasses counts the distinct non-zero source-edge ids used along each welded vertex-pair.
// A pair carrying two or more is a coincident tangent seam (two input edges on one line sharing
// their endpoints) whose uses must stay on separate edges through the weld; one or zero is a plain
// shared or op-generated edge that welds to a single edge as before (#1600).
func pairEdgeClasses(faces []filletFace, rings [][][]int) map[[2]int]int {
	seen := map[[2]int]map[uint64]bool{}
	for fi := range faces {
		for li, ring := range rings[fi] {
			ids := faces[fi].loops[li].srcE
			for k := range ring {
				id := probe.SrcIDAt(ids, k)
				if id == 0 {
					continue
				}
				pair := probe.Canon2(ring[k], ring[(k+1)%len(ring)])
				if seen[pair] == nil {
					seen[pair] = map[uint64]bool{}
				}
				seen[pair][id] = true
			}
		}
	}
	out := make(map[[2]int]int, len(seen))
	for p, ids := range seen {
		out[p] = len(ids)
	}
	return out
}

// edgeClassOf returns the identity class of the segment a→b: its own source-edge id when the pair
// is a coincident tangent seam (>=2 distinct source edges meet on it), else 0 so ordinary uses of
// the pair all weld to one edge. It is the single keying rule shared by the catalog and the solid
// test (#1600).
func edgeClassOf(a, b int, srcE uint64, classes map[[2]int]int) uint64 {
	if srcE == 0 || classes[probe.Canon2(a, b)] < 2 {
		return 0
	}
	return srcE
}

// seamEdgeKey identifies a reconstructed edge by its welded vertex pair AND its identity class: a
// coincident tangent seam (two source edges on one line, same endpoints) resolves to two keys that
// share a pair but differ in class, so it stays two manifold edges instead of one 4-face edge.
type seamEdgeKey struct {
	pair  [2]int
	class uint64
}

// edgeCatalog creates one shared topo edge per (welded vertex pair, identity class) on demand,
// reusing it (reversed) for the second face. classes marks which pairs are coincident tangent
// seams that must split by source-edge id (#1600).
type edgeCatalog struct {
	bld     *topo.Builder
	verts   []math.Point3
	tv      []*topo.Vertex
	edges   map[seamEdgeKey]edgeRec
	classes map[[2]int]int
	tag     string
	weld    float64 // model-relative closure tolerance for isClosedSeam (ADR-0042)
}

// use returns the loop use for the directed segment a→b with the given curve (nil ⇒ a line) and
// source-edge id, creating the shared edge for its identity class the first time in its a→b
// direction. Two coincident seam edges (distinct ids at a >=2-id pair) get distinct edges.
// A second use offering geometry the edge does not yet carry upgrades it (resolveCurveOffer).
func (c *edgeCatalog) use(a, b int, curve geom.Curve3, srcE uint64) topo.Use {
	key := seamEdgeKey{probe.Canon2(a, b), edgeClassOf(a, b, srcE, c.classes)}
	if rec, ok := c.edges[key]; ok {
		c.resolveCurveOffer(key, a, b, rec, curve)
		// A closed seam edge welds both endpoints to one vertex, so rec.from!=a is false for
		// BOTH uses — the welded vertex order can't encode the traversal sense. The two coedges
		// of a manifold edge are antiparallel, so the 2nd use flips. Parity-only + tessellation-
		// safe: the periodic mesher rebuilds from the surface (u,v) domain and never reads this
		// flag (geometry-math consult 2026-07-12). Open edges keep the vertex-order derivation.
		if rec.closed {
			return topo.Use{Edge: rec.edge, Reversed: true}
		}
		return topo.Use{Edge: rec.edge, Reversed: rec.from != a}
	}
	closed := isClosedSeam(a, b, curve, c.weld)
	built := curve
	if built == nil {
		built = geom.NewLineSegment(c.verts[a], c.verts[b])
	}
	e := c.bld.AddEdge(built, c.tv[a], c.tv[b], topo.NewLineage(topo.Tok(c.tag, "e", len(c.edges))))
	c.edges[key] = edgeRec{edge: e, from: a, to: b, closed: closed, curve: curve}
	return topo.Use{Edge: e, Reversed: false}
}

// isClosedSeam reports whether a→b is a genuine closed seam edge: both endpoints weld to one
// vertex (a==b) AND the curve returns to its start over its full domain (within the model-
// relative weld tolerance). The geometric corroboration rejects a spuriously-welded micro-arc,
// so a real point-welder defect fails Validate loud rather than being laundered into a valid-
// looking topological ghost. A straight (nil-curve) a==b segment is a true zero-length
// degeneracy and is never a seam.
func isClosedSeam(a, b int, curve geom.Curve3, weld float64) bool {
	if a != b || curve == nil {
		return false
	}
	lo, hi := curve.Domain()
	return curve.PointAt(lo).DistanceTo(curve.PointAt(hi)) < weld
}

// curvedLoopSpec builds a face loop from a ring of welded indices, the per-segment curves and the
// per-segment source-edge ids (for tangent-seam identity, #1600).
func curvedLoopSpec(outer bool, ring []int, curves []geom.Curve3, srcE []uint64, ec *edgeCatalog) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for k := range ring {
		uses[k] = ec.use(ring[k], ring[(k+1)%len(ring)], curveAt(curves, k), probe.SrcIDAt(srcE, k))
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
