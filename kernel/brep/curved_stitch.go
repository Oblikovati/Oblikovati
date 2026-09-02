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

// curvedFaceBox bounds the loop-edge endpoints of the faces being stitched — the geometry whose
// Resolution sets the stitch weld grid (#1602).
func curvedFaceBox(faces []curvedFace) math.Box {
	box := math.EmptyBox()
	for _, f := range faces {
		for _, loop := range f.loops {
			for _, le := range loop.edges {
				box = box.ExtendPoint(le.start()).ExtendPoint(le.end())
			}
		}
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
// circle). An OPEN spiric branch is stored in its native direction (V0<V1, see spiricArcOf) and
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
// the stored curve runs opposite the representative's traversal (repFlip — true only for a spiric
// branch stored in its native direction, see spiricArcOf).
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
		return v, v, false
	}
	return m.vertexFor(gi, ends[0], a), m.vertexFor(gi, ends[1], rep.end()), false
}

// useReversedFor reports whether a loop's use of its slot's group traverses the stored edge curve
// backwards: a closed seam edge by its sweep sign (the stored closed curve runs forward), an open
// edge by its pass-1 rep-relative direction XOR the group's stored-curve flip — no welder re-probe.
func (m *radialMinter) useReversedFor(slot stitchSlot, le loopEdge) bool {
	if m.plan.closed[slot.gi] {
		return le.t1 < le.t0
	}
	return slot.rev != m.repFlip[slot.gi]
}

// edgeCurveFor returns the curve to store on the topo edge so its WHOLE domain is exactly the loop
// edge's [t0, t1] segment: a circle/arc sub-range becomes an Arc3d (else TessellateEdge would walk the
// full circle), a line sub-range a LineSegment between the endpoints, a full closed curve is kept.
func edgeCurveFor(le loopEdge) geom.Curve3 {
	switch c := le.curve.(type) {
	case geom.Circle:
		if isFullDomain(le.t0, le.t1) {
			return c
		}
		return arcOfCircle(c, le.t0, le.t1)
	case geom.Arc3d:
		return subArc(c, le.t0, le.t1)
	case geom.LineSegment:
		return geom.NewLineSegment(le.start(), le.end())
	case geom.Line:
		return geom.NewLineSegment(le.start(), le.end())
	default:
		return conicEdgeCurveFor(le)
	}
}

// conicEdgeCurveFor restricts the analytic conic edges (the oblique cone-cut sections) to their loop
// sub-range: a hyperbola/parabola to its bounded arc, an elliptical arc as-is, a full ellipse to the
// elliptical arc over [t0, t1]. Any other curve is stored whole.
func conicEdgeCurveFor(le loopEdge) geom.Curve3 {
	// Both hyperbola forms restrict through geom, which knows what each one's parameter means: a
	// Hyperbola's is θ, an already-bounded HyperbolicArc's is its own [0,1]. Storing a pre-clipped
	// arc unsliced would leave the edge's curve spanning more than its two vertices (#3459).
	if arc, ok := geom.ConicSubArc(le.curve, le.t0, le.t1); ok {
		return arc
	}
	switch c := le.curve.(type) {
	case geom.Parabola:
		return c.Arc(le.t0, le.t1) // a parabola loop edge's params are the cross coordinate t; store the bounded arc
	case geom.EllipticalArc:
		return ellipticalSubArc(c, le.t0, le.t1) // restrict/re-anchor to the run's [t0,t1] (the reversed lobe walk)
	case geom.EllipseFull:
		return ellipseArcOf(c, le.t0, le.t1) // a section sub-arc of a full ellipse (the (u,v) cone split)
	case geom.SpiricArc:
		return spiricArcOf(c, le.t0, le.t1) // a torus-cut spiric branch, oriented to the loop's traversal
	default:
		return c
	}
}

// spiricArcOf restricts a SpiricArc to its loop sub-range [t0, t1], stored in its NATIVE tube-angle direction
// (V0 < V1) regardless of how this loop walks it — orientation is carried by the edge's reversed flag, not by
// flipping V0/V1. A reversed-range edge (V0 > V1) would mesh as a DIFFERENT region in the direction-sensitive
// spiric loft (the two branches of a bigon must both be native so the cap patch comes out the right size,
// #1406); newEdge anchors the edge to this native arc's endpoints so the reversed flag stays correct.
func spiricArcOf(sa geom.SpiricArc, t0, t1 float64) geom.Curve3 {
	v0 := sa.V0 + t0*(sa.V1-sa.V0)
	v1 := sa.V0 + t1*(sa.V1-sa.V0)
	if v0 > v1 {
		v0, v1 = v1, v0
	}
	sa.V0, sa.V1 = v0, v1
	return sa
}

// ellipticalSubArc restricts a partial EllipticalArc to its loop sub-range [t0, t1], re-anchored so the
// stored curve's PointAt(0) is the edge's StartVertex and PointAt(1) its EndVertex (EllipticalArc.PointAt(t)
// walks StartAngle+t·SweepAngle over t∈[0,1]). A lobe of the equal-radius Steinmetz bicylinder walks its
// shared arc in the arc's DECREASING-parameter direction (t0=1, t1=0); keeping the arc's original forward
// parameterisation left PointAt(0) at the FAR pinch, 2R from the edge's StartVertex, so the face's
// discretised boundary crossed the solid and the (u,v) trim loop self-intersected (#1403). For a run that
// already spans the whole arc forward (t0=0, t1=1, the oblique cone-cut rim/lid) this returns an identical
// arc, so those paths are unchanged.
func ellipticalSubArc(e geom.EllipticalArc, t0, t1 float64) geom.Curve3 {
	a, _ := geom.NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius,
		e.StartAngle+t0*e.SweepAngle, (t1-t0)*e.SweepAngle)
	return a
}

// ellipseArcOf builds the EllipticalArc covering a full ellipse's parameter sub-range [t0, t1]
// (EllipseFull.PointAt(t) is the point at angle 2πt), so the edge tessellates over that arc alone.
func ellipseArcOf(e geom.EllipseFull, t0, t1 float64) geom.Curve3 {
	const twoPi = 2 * stdmath.Pi
	a, _ := geom.NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius, twoPi*t0, twoPi*(t1-t0))
	return a
}

// isFullDomain reports whether [t0, t1] spans a curve's whole [0, 1] domain (a closed seam circle),
// in either direction.
func isFullDomain(t0, t1 float64) bool {
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	return lo < 1e-9 && hi > 1-1e-9
}

// arcOfCircle builds the Arc3d covering a circle's parameter sub-range [t0, t1] (Circle.PointAt(t) is
// the point at angle 2πt), so the edge tessellates over that arc alone.
func arcOfCircle(c geom.Circle, t0, t1 float64) geom.Curve3 {
	const twoPi = 2 * stdmath.Pi
	a, _ := geom.NewArc3d(c.Center, c.Normal.AsVector(), c.RefDir.AsVector(), c.Radius, twoPi*t0, twoPi*(t1-t0))
	return a
}

// subArc restricts an Arc3d to a parameter sub-range [t0, t1].
func subArc(a geom.Arc3d, t0, t1 float64) geom.Curve3 {
	return geom.Arc3d{
		Center: a.Center, Normal: a.Normal, RefDir: a.RefDir, Radius: a.Radius,
		StartAngle: a.StartAngle + t0*a.SweepAngle, SweepAngle: (t1 - t0) * a.SweepAngle,
	}
}
