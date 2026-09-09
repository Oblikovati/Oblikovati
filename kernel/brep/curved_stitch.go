// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved stitch (M2 Phase 1, Oblikovati/Oblikovati#1334). The general curved boolean's split stage
// produces a set of curvedFaces (each an analytic surface + curved boundary loops); this welds them
// into one topology. It is the curved analogue of the planar boolean's stitch (boolean_stitch.go):
// vertices weld by position, edges weld by their endpoints PLUS a midpoint (so the two different curves
// that can join the same pair of points — a boundary arc and the imprint arc between the same two cut
// vertices — stay distinct, not merged). A sub-range of a circle is stored as an Arc3d so the edge
// tessellates over the arc, not the whole circle (TessellateEdge walks the curve's whole Domain).

// curvedStitch welds curvedFaces into a body, sharing welded vertices and edges. Each face keeps its
// surface, sense (reversed) and lineage; loops[0] is the outer loop, the rest holes. The weld grid is
// the faces' own stitch resolution (ADR-0042, #1602): the seam points fed in carry SSI-tracer noise
// proportional to the operands' extent, so an absolute grid tears seams on parts it was never
// calibrated for. It is the ONE boolean stitch (ADR-0058): a two-pass build whose plan resolves
// tangent/grazing contacts through the surface-agnostic Weiler radial sew and splits pinched vertices
// into per-disk coincident duplicates — planar and curved faces alike, the OCCT BuilderSolid shape.
func curvedStitch(faces []curvedFace) *topo.Body {
	return curvedStitchNamed(faces, stitchNaming{relineage: true})
}

// stitchNaming is a stitch build's entity-naming policy (ADR-0043). The curved boolean's default
// (zero hooks, relineage on) mints ordinal names and then renames SSI edges by bordering face pair;
// the planar boolean injects its imprint-provenance naming instead — parent-pair intersection edges
// and meeting-faces vertices — so the ONE construction serves both pipelines' naming contracts.
type stitchNaming struct {
	// edges returns one lineage per plan group (nil → curvedbool:e#N ordinals).
	edges func(groups []edgeGroup, verts []math.Point3) []topo.Lineage
	// vertex names a non-pinch shared vertex (nil → curvedbool:v#N ordinals).
	vertex func(p math.Point3) topo.Lineage
	// relineage runs RelineageByFaceProvenance after the build (the curved SSI-edge naming).
	relineage bool
}

// curvedStitchNamed is curvedStitch with an explicit naming policy.
func curvedStitchNamed(faces []curvedFace, naming stitchNaming) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("curvedbool", "body", 0)))
	pw := newWelder3(geom.ResolutionForBox(curvedFaceBox(faces)).Stitch())
	m := newRadialMinter(bld, pw, buildCurvedStitchPlan(faces, pw))
	if naming.edges != nil {
		m.edgeLin = naming.edges(m.plan.sew.groups, pw.points)
	}
	m.vertexLin = naming.vertex
	provByFace := map[*topo.Face]topo.Lineage{}
	for fi := range faces {
		m.buildFace(fi, faces[fi], provByFace)
	}
	body := bld.Build()
	// ADR-0043 SSI-edge provenance: the welded edges are minted with build-order ordinals
	// (curvedbool:e#N) that renumber under any upstream edit. Each result face carries a stable
	// provenance lineage (an original face's key, or a wall/cap's parent-derived name), so rename the
	// surface-intersection edges by their bordering face pair — a build-order-independent name. The
	// caller's InheritOriginalEdges then restores the identity of original boundaries passed through
	// whole (a survivor must keep its OWN key, not a face-pair name), see booleanGeneral.
	if naming.relineage && len(provByFace) > 0 {
		body.RelineageByFaceProvenance(provByFace, topo.Tok("curvedbool", "x", 0), topo.Tok("curvedbool", "seg", 0))
	}
	return body
}

// radialMinter mints topo entities from a curvedStitchPlan lazily, in first-use order, so the edge and
// vertex ordinals of every previously-manifold case match the retired single-pass welder; only a
// tangent contact (extra edge groups) or a pinch (per-disk duplicate vertices) mints entities the old
// welder could not. Positions canonicalise through the shared point welder (grid + 26-neighbour
// search), so two independently computed copies of one seam point weld even across a grid-cell
// boundary (#1602).
type radialMinter struct {
	bld       *topo.Builder
	pw        *welder3
	plan      curvedStitchPlan
	groupDisk map[[2]int]int          // (group, welded vertex) → disk ordinal at that vertex
	edges     []*topo.Edge            // group → minted edge (nil until first demand)
	repFlip   []bool                  // group → stored curve runs opposite the rep (spiric native)
	diskVert  map[[2]int]*topo.Vertex // (welded vertex, disk ordinal) → minted vertex
	edgeLin   []topo.Lineage          // per-group edge lineage override (stitchNaming.edges)
	vertexLin func(math.Point3) topo.Lineage
	nv, ne    int
	npinch    int
}

// newRadialMinter indexes the plan's disk partition for endpoint resolution.
func newRadialMinter(bld *topo.Builder, pw *welder3, plan curvedStitchPlan) *radialMinter {
	m := &radialMinter{bld: bld, pw: pw, plan: plan, groupDisk: map[[2]int]int{},
		edges: make([]*topo.Edge, len(plan.sew.groups)), repFlip: make([]bool, len(plan.sew.groups)),
		diskVert: map[[2]int]*topo.Vertex{}}
	for v, disks := range plan.sew.disks {
		for di, d := range disks {
			for _, gi := range d.groups {
				m.groupDisk[[2]int{gi, v}] = di
			}
		}
	}
	return m
}

// buildFace mints one face from its loops, resolving every loop edge through the radial plan.
func (m *radialMinter) buildFace(fi int, f curvedFace, provByFace map[*topo.Face]topo.Lineage) {
	specs := m.loopSpecs(fi, f.loops, f.outerless)
	var built *topo.Face
	if f.reversed {
		built = m.bld.AddReversedFace(f.surface, f.lineage, specs...)
	} else {
		built = m.bld.AddFace(f.surface, f.lineage, specs...)
	}
	built.SetChart(f.chart) // the parametric trim travels with the face (ADR-0063)
	if len(f.lineage.Key()) > 0 {
		provByFace[built] = f.lineage
	}
	for _, k := range f.aliasKeys { // ADR-0057: resolve the merged coplanar parents' keys to this face
		built.AddAliasKey(k)
	}
}

// vertexFor returns the minted vertex for group gi's endpoint at welded index v — the shared vertex on
// the vertex's first radial disk, a coincident pinch duplicate on any further disk (the line/point-kiss
// split, ADR-0047). p is the demanding coordinate, so the first demand's exact point is stored (as the
// retired welder did).
func (m *radialMinter) vertexFor(gi, v int, p math.Point3) *topo.Vertex {
	di := m.groupDisk[[2]int{gi, v}]
	if tv, ok := m.diskVert[[2]int{v, di}]; ok {
		return tv
	}
	tv := m.mintVertex(p, m.plan.sew.disks[v][di].copy > 0)
	m.diskVert[[2]int{v, di}] = tv
	return tv
}

// mintVertex mints a shared vertex (the naming hook's lineage, else curvedbool:v#N) or a pinch
// duplicate (curvedbool:pinch#N — a fresh coincident copy is new topology either way).
func (m *radialMinter) mintVertex(p math.Point3, pinch bool) *topo.Vertex {
	if pinch {
		tv := m.bld.AddVertex(p, topo.NewLineage(topo.Tok("curvedbool", "pinch", m.npinch)))
		m.npinch++
		return tv
	}
	if m.vertexLin != nil {
		return m.bld.AddVertex(p, m.vertexLin(p))
	}
	tv := m.bld.AddVertex(p, topo.NewLineage(topo.Tok("curvedbool", "v", m.nv)))
	m.nv++
	return tv
}

// curvedFaceBox bounds the faces being stitched — the geometry whose Resolution sets the stitch weld
// grid (#1602). It is faceLoopBox over the set, so an edge is bounded over its OWN span: a cap and its
// lid are each bounded by ONE closed circle whose two ends are the same seam point, and an
// endpoint-only box degenerates to that point, flooring the weld grid to the degeneracy resolution and
// leaving the circle's two seam copies (1.2e-15 apart) unmerged (ADR-0042, ADR-0061 stage 4).
func curvedFaceBox(faces []curvedFace) math.Box {
	box := math.EmptyBox()
	for _, f := range faces {
		box = box.Union(faceLoopBox(f))
	}
	return box
}

// loopSpecs turns a face's curved loops into builder loop specs (outer first, the rest holes). When
// outerless, EVERY loop is a hole — a face on a closed surface that wraps the whole surface minus its holes
// (the torus complement, #1406), which has no outer loop.
func (m *radialMinter) loopSpecs(fi int, loops []curvedLoop, outerless bool) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(loops))
	for li, loop := range loops {
		uses := make([]topo.Use, 0, len(loop.edges))
		for ei, le := range loop.edges {
			slot := m.plan.slots[fi][li][ei]
			uses = append(uses, topo.Use{Edge: m.edgeFor(slot.gi), Reversed: m.useReversedFor(slot, le)})
		}
		specs = append(specs, loopSpecOf(li == 0 && !outerless, uses))
	}
	return specs
}

// loopSpecOf wraps a use list as the outer or an inner loop.
func loopSpecOf(outer bool, uses []topo.Use) topo.LoopSpec {
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// edgeFor returns the minted edge for group gi, minting it on first demand: the canonical
// representative's restricted curve between its disk-resolved endpoint vertices, oriented along the
// representative's traversal (so the creating loop uses it forward, except a reversed-sweep closed
// circle). An OPEN spiric branch is stored in its native direction (V0<V1, see geom.SubCurve) and
// anchored to the curve's own endpoints, so the reversed flag — not a flipped curve — orients it.
func (m *radialMinter) edgeFor(gi int) *topo.Edge {
	if e := m.edges[gi]; e != nil {
		return e
	}
	rep := m.plan.rep[gi]
	curve := edgeCurveFor(rep)
	vs, ve, repFlip := m.edgeEnds(gi, rep, curve)
	e := m.bld.AddEdge(curve, vs, ve, m.edgeLineage(gi))
	m.edges[gi] = e
	m.repFlip[gi] = repFlip
	return e
}

// edgeLineage is a group's edge lineage: the naming hook's when supplied, else a curvedbool ordinal.
func (m *radialMinter) edgeLineage(gi int) topo.Lineage {
	if m.edgeLin != nil {
		return m.edgeLin[gi]
	}
	lin := topo.NewLineage(topo.Tok("curvedbool", "e", m.ne))
	m.ne++
	return lin
}

// edgeEnds resolves a group's endpoint vertices (disk-aware, welds cached from pass 1) plus whether
// the stored curve runs opposite the representative's traversal (repFlip): a spiric branch stored in
// its native direction, and a CLOSED loop, which is always stored whole and forward (geom.SubCurve)
// and so runs opposite a representative that walked it backwards.
func (m *radialMinter) edgeEnds(gi int, rep loopEdge, curve geom.Curve3) (vs, ve *topo.Vertex, repFlip bool) {
	ends := m.plan.repEnds[gi]
	if sa, ok := curve.(geom.SpiricArc); ok && !m.plan.closed[gi] {
		ca, cb := sa.PointAt(0), sa.PointAt(1)
		ks, ke := m.pw.add(ca), m.pw.add(cb)
		return m.vertexFor(gi, ks, ca), m.vertexFor(gi, ke, cb), ks != ends[0]
	}
	a := rep.start()
	if m.plan.closed[gi] {
		v := m.vertexFor(gi, ends[0], a)
		return v, v, rep.t1 < rep.t0
	}
	return m.vertexFor(gi, ends[0], a), m.vertexFor(gi, ends[1], rep.end()), false
}

// useReversedFor reports whether a loop's use of its slot's group traverses the stored edge curve
// backwards: its pass-1 rep-relative direction XOR the group's stored-curve flip — one rule for open
// and closed edges alike, no welder re-probe. A closed use used to be read from its own parameter
// direction, which assumed the stored curve ran the way its own curve did; see closedRunsOppose.
func (m *radialMinter) useReversedFor(slot stitchSlot, _ loopEdge) bool {
	return slot.rev != m.repFlip[slot.gi]
}

// edgeCurveFor returns the curve to store on the topo edge so its WHOLE domain is exactly the loop
// edge's [t0, t1] segment. The restriction itself is geom.SubCurve: what a curve kind's parameter means
// is geom's business, and a switch over kinds here would have to be found and taught again every time a
// kind is added (#2188).
func edgeCurveFor(le loopEdge) geom.Curve3 {
	return geom.SubCurve(le.curve, le.t0, le.t1)
}

// isFullDomain reports whether [t0, t1] spans a curve's whole [0, 1] domain (a closed seam circle),
// in either direction.
func isFullDomain(t0, t1 float64) bool {
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	return lo < 1e-9 && hi > 1-1e-9
}
