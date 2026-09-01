// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Corner-junction exact pre-split (EPIC Oblikovati/Oblikovati#1738, ADR-0048). A SECOND curved cut whose SSI
// imprint CROSSES the surviving first-cut boundary (the "notch") — the hardest partial-rim sub-case. The
// disjoint sub-family (#1732/#1735, OCC-certified #1736) ships; this file adds the crossing case.
//
// The failure it repairs (instrumented, curved_corner_junction_probe_test.go): the imprint arrives as a
// sampled polyline; the (u,v) arrangement welds its crossing with the prior conic on the polyline CHORD, but
// re-emission re-anchors each arm to its own analytic curve, so the imprint arm terminates at a chord point
// (~sagitta INSIDE the cylinder) while the prior arm terminates exactly on the cylinder — a ~2.6e-4 gap that
// blows the 1e-7 weld, splitting the junction into two vertices (χ=1, 5 free edges, the back entry-ellipse
// hole collapses). This is the classic EGC failure: an approximate combinatorial decision corrupts the
// arrangement (Yap 1997; Kettner et al. 2008).
//
// The fix is Method C (ADR-0048): compute the crossing as the EXACT triple point where the target cylinder,
// the rod surface and the first-cut plane all meet — a single 3D vertex on ALL THREE surfaces — and make
// every coupled face (target wall, notch cap, rod tunnel) re-emit to it. Because the prior boundary conic
// lies on the target cylinder ∩ the first-cut plane already, the triple point is the root, along that conic,
// of the rod's implicit surface value; both arms stay exact (Patrikalakis & Maekawa 2002; the section conic
// is the exact carrier, so no facet error enters the shared vertex).

// cornerJunction is one exact triple point: a point lying simultaneously on the target cylinder, on a prior
// boundary arm (⊂ target cylinder ∩ first-cut plane) and on the rod's cylindrical surface. It is the shared
// vertex the target wall, the notch cap and the rod tunnel all split at, so the welded shell closes (#1738).
type cornerJunction struct {
	point math.Point3 // exact: on target cylinder ∧ on prior conic ∧ on rod surface
	edge  int         // index into prior.edges of the arm carrying it
	tArm  float64     // the arm's curve parameter at the junction
}

// rodRadialDist is the perpendicular distance from p to the rod axis (the line through axisPt along the unit
// axisDir). Its zero level set offset by the rod radius is the rod's cylindrical surface — the implicit whose
// crossing with a prior arm is the triple point (#1738).
func rodRadialDist(p, axisPt math.Point3, axisDir math.Vector3) float64 {
	d := axisPt.VectorTo(p)
	along := d.Dot(axisDir)
	return float64(d.Sub(axisDir.Scale(along)).Length())
}

// cornerSampleCount brackets the rod crossing along each prior arm. The prior conic is smooth and low
// curvature over an arm, so this resolves every transversal crossing into its own sign-change bracket while
// keeping the scan cheap; the exact root is then found by bisection, independent of the sample density.
const cornerSampleCount = 256

// cornerJunctions solves every exact triple point where the rod surface (cylinder axisPt/axisDir/radius)
// crosses the prior boundary. Along each prior edge it brackets each sign change of (rodRadialDist − radius)
// and bisects it to the exact crossing — a point on the prior conic (hence on the target cylinder ∧ the
// first-cut plane) AND on the rod surface. Method C of ADR-0048: the crossing is a 1-D root on the exact
// carrier conic, so both re-emitted arms terminate at the identical vertex with no facet error (#1738).
func cornerJunctions(prior priorTrimLoop, axisPt math.Point3, axisDir math.Vector3, radius float64) []cornerJunction {
	var js []cornerJunction
	for i, e := range prior.edges {
		js = append(js, edgeRodCrossings(e, i, axisPt, axisDir, radius)...)
	}
	return js
}

// edgeRodCrossings returns the exact rod crossings along one prior edge. It scans the signed rod value
// f(t)=rodRadialDist(edge(t))−radius, and on each strict sign change bisects the bracket to the exact root.
// A sample landing exactly on the surface (f==0) is treated as its own crossing so a crossing at a sample
// vertex is not missed (the degenerate incidence the math advisor flagged).
func edgeRodCrossings(e loopEdge, idx int, axisPt math.Point3, axisDir math.Vector3, radius float64) []cornerJunction {
	f := func(t float64) float64 { return rodRadialDist(e.curve.PointAt(t), axisPt, axisDir) - radius }
	var out []cornerJunction
	prevT, prevF := e.t0, f(e.t0)
	for i := 1; i <= cornerSampleCount; i++ {
		t := e.t0 + (e.t1-e.t0)*float64(i)/float64(cornerSampleCount)
		fv := f(t)
		if prevF*fv < 0 {
			// bisectRoot needs an ASCENDING bracket: its (hi-lo)<1e-15 guard early-returns an unrefined
			// midpoint on a descending one, which is exactly what a reversed edge (t0>t1, e.g. a notch cap's
			// section walked backwards) produces — the crossing then lands ~half a sample off the surface and
			// fails to weld with the target wall's exact point.
			lo, hi := prevT, t
			if lo > hi {
				lo, hi = hi, lo
			}
			tc := bisectRoot(f, lo, hi)
			out = append(out, cornerJunction{point: e.curve.PointAt(tc), edge: idx, tArm: tc})
		}
		prevT, prevF = t, fv
	}
	return out
}

// cylinderOutwardNormal is the unit outward radial of a cylinder (axisPt/axisDir) at a point p on it — the
// surface normal used to form the SSI tangent T=n_target×n_rod and to project the prior tangent into the
// target tangent plane for the scale-invariant tangency gate (ADR-0048 §tangency).
func cylinderOutwardNormal(p, axisPt math.Point3, axisDir math.Vector3) math.Vector3 {
	d := axisPt.VectorTo(p)
	radial := d.Sub(axisDir.Scale(d.Dot(axisDir)))
	return unitVector(radial)
}

// unitVector normalises v, returning the zero vector when v is (near) degenerate so callers can detect a
// collapsed direction rather than divide by zero.
func unitVector(v math.Vector3) math.Vector3 {
	l := float64(v.Length())
	if l < 1e-300 {
		return math.Vector3{}
	}
	return v.Scale(math.Scalar(1 / l))
}

// junctionDegeneracy returns the two ADR-0048 degeneracy measures at a triple point, kept strictly distinct:
//   - surfSurf = ‖n_target × n_rod‖, the sine of the angle between the two surface normals. →0 means the two
//     SURFACES are tangent there, so the SSI itself is singular (Patrikalakis & Maekawa §5–6).
//   - curveCurve = the sine between the imprint SSI tangent T=n_target×n_rod and the prior conic tangent,
//     both in the target tangent plane. →0 means the two boundary CURVES graze (the corner-junction cusp).
//
// It is scale-invariant WITHOUT weighting by R: on a cylinder the {circumferential, axial} tangent basis is
// orthonormal in 3D, so this 3D tangent-plane angle already IS the metric angle in the first fundamental form
// I=diag(R²,1) (a cylinder is developable — the (u,v) chart with that metric is a local isometry). Measuring
// the genuine geometric angle quotients R out by construction, as ADR-0048 requires.
func junctionDegeneracy(j cornerJunction, prior priorTrimLoop,
	tgtAxisPt math.Point3, tgtAxis math.Vector3, rodAxisPt math.Point3, rodAxis math.Vector3) (surfSurf, curveCurve float64) {
	nT := cylinderOutwardNormal(j.point, tgtAxisPt, tgtAxis)
	nR := cylinderOutwardNormal(j.point, rodAxisPt, rodAxis)
	ssiT := nT.Cross(nR)
	surfSurf = float64(ssiT.Length())
	tp := prior.edges[j.edge].curve.TangentAt(j.tArm)
	tpTan := unitVector(tp.Sub(nT.Scale(tp.Dot(nT)))) // prior tangent projected into the target tangent plane
	curveCurve = float64(unitVector(ssiT).Cross(tpTan).Length())
	return surfSurf, curveCurve
}

// junctionWeldTol is the coincidence radius for deciding an imprint loop passes through a triple point. It
// sits an order below the imprint's own facet sagitta (~2.6e-4 on the cm-scale fixture) yet well above the
// weld grid, so a genuine crossing loop is caught while a disjoint loop (the back entry ellipse) is not.
const junctionWeldTol = 1e-2

// splitPriorAtJunctions splits each prior boundary edge at the triple points that lie on it (Method C,
// ADR-0048), so the arrangement samples the prior conic with a vertex EXACTLY at each junction and the
// re-emitted prior sub-arc terminates at the exact triple point — the target half of the shared vertex. An
// edge with no junction is copied through unchanged, keeping every other boolean's prior ingest identical.
func splitPriorAtJunctions(prior priorTrimLoop, js []cornerJunction) priorTrimLoop {
	byEdge := map[int][]float64{}
	for _, j := range js {
		byEdge[j.edge] = append(byEdge[j.edge], j.tArm)
	}
	out := make([]loopEdge, 0, len(prior.edges)+len(js))
	for i, e := range prior.edges {
		if ts, ok := byEdge[i]; ok {
			out = append(out, splitLoopEdgeAtParams(e, ts)...)
		} else {
			out = append(out, e)
		}
	}
	return priorTrimLoop{edges: out}
}

// splitLoopEdgeAtParams cuts one edge into sub-edges at the given interior parameters, ordered along the
// edge's own traversal sense (t0→t1), each sub-edge re-emitting the SAME analytic curve over its own range.
func splitLoopEdgeAtParams(e loopEdge, ts []float64) []loopEdge {
	lo, hi := stdmath.Min(e.t0, e.t1), stdmath.Max(e.t0, e.t1)
	mids := make([]float64, 0, len(ts))
	for _, t := range ts {
		if t > lo+1e-12 && t < hi-1e-12 {
			mids = append(mids, t)
		}
	}
	sort.Slice(mids, func(i, j int) bool { return stdmath.Abs(mids[i]-e.t0) < stdmath.Abs(mids[j]-e.t0) })
	out := make([]loopEdge, 0, len(mids)+1)
	prev := e.t0
	for _, t := range mids {
		out = append(out, loopEdge{curve: e.curve, t0: prev, t1: t})
		prev = t
	}
	return append(out, loopEdge{curve: e.curve, t0: prev, t1: e.t1})
}

// splitImprintAtJunctions replaces each imprint loop the triple points lie on with OPEN sub-polylines split
// at those points (their endpoints are the exact triple-point vertices), so the (u,v) arrangement carries
// each junction as a shared node welding to the split prior conic; a loop clear of every junction (the
// disjoint back entry ellipse) passes through unchanged as a closed polyline (complete loop ingest, #1738).
func splitImprintAtJunctions(loops []geom.Curve3, js []cornerJunction) []geom.Curve3 {
	out := make([]geom.Curve3, 0, len(loops)+len(js))
	for i := range loops {
		pts := imprintLoopPoints(loops[i])
		on := junctionsOnLoop(pts, js)
		if len(on) == 0 {
			out = append(out, loops[i]) // unsplit: the loop passes through as it came, exact or marched
			continue
		}
		for _, arc := range splitPolylineRing(pts, on) {
			a := arc
			out = append(out, &a)
		}
	}
	return out
}

// junctionsOnLoop returns the triple points lying on the sampled loop (within junctionWeldTol of it) — the
// junctions that split THIS loop, distinguishing the crossing loop from a disjoint one.
func junctionsOnLoop(verts []math.Point3, js []cornerJunction) []math.Point3 {
	var on []math.Point3
	for _, j := range js {
		if _, _, d := nearestRingSeg(verts, j.point); d < junctionWeldTol {
			on = append(on, j.point)
		}
	}
	return on
}

// splitPolylineRing cuts a closed imprint polyline into open sub-polylines at the given on-loop points: each
// point is inserted at its nearest segment, then the ring is walked between consecutive inserted points to
// emit one arc apiece (two junctions → the below-notch arc and the above-notch arc, meeting at the shared
// triple-point vertices). Each arc's endpoints are the EXACT triple points.
func splitPolylineRing(loopVerts []math.Point3, pts []math.Point3) []geom.Polyline {
	verts := ringVertices(loopVerts)
	ring := make([]ringNode, 0, len(verts)+len(pts))
	for i, v := range verts {
		ring = append(ring, ringNode{float64(i), v, false})
	}
	for _, p := range pts {
		i, tloc, _ := nearestRingSeg(verts, p)
		ring = append(ring, ringNode{float64(i) + tloc, p, true})
	}
	sort.Slice(ring, func(a, b int) bool { return ring[a].key < ring[b].key })
	return arcsBetweenSplits(ring)
}

// ringNode is one vertex of the augmented split ring: its position key (integer for an original vertex,
// fractional for an inserted split point), the 3D point, and whether it is a split point.
type ringNode struct {
	key   float64
	p     math.Point3
	split bool
}

// arcsBetweenSplits walks the sorted ring and emits one open polyline between each consecutive pair of split
// nodes (cyclically), so each arc's endpoints are the inserted triple points.
func arcsBetweenSplits(ring []ringNode) []geom.Polyline {
	var splitIdx []int
	for i, e := range ring {
		if e.split {
			splitIdx = append(splitIdx, i)
		}
	}
	var arcs []geom.Polyline
	m := len(ring)
	for s := 0; s < len(splitIdx); s++ {
		var vs []math.Point3
		for k := splitIdx[s]; ; k = (k + 1) % m {
			vs = append(vs, ring[k].p)
			if k == splitIdx[(s+1)%len(splitIdx)] {
				break
			}
		}
		if arc, err := geom.NewPolyline(vs); err == nil {
			arcs = append(arcs, arc)
		}
	}
	return arcs
}

// ringVertices drops the duplicate closing vertex of a closed polyline (first≈last) so it is a clean cyclic
// vertex ring — the form splitPolylineRing walks.
func ringVertices(verts []math.Point3) []math.Point3 {
	n := len(verts)
	if n >= 2 && float64(verts[0].DistanceTo(verts[n-1])) < junctionWeldTol {
		return verts[:n-1]
	}
	return verts
}

// recoverNotchPlane recovers the first-cut plane from the prior boundary: the section arms are the plane∩
// target-cylinder conic, so any one of them (an EllipticalArc / EllipseFull) carries the plane exactly in its
// Center + Normal. ok=false when the prior boundary has no planar-section arm (a CURVED first cut — out of
// the corner-junction scope, declines observably; ADR-0048 §out-of-scope).
func recoverNotchPlane(prior priorTrimLoop) (geom.Plane, bool) {
	for _, e := range prior.edges {
		switch c := e.curve.(type) {
		case geom.EllipticalArc:
			pl, err := geom.NewPlane(c.Center, c.Normal.AsVector())
			return pl, err == nil
		case geom.EllipseFull:
			pl, err := geom.NewPlane(c.Center, c.Normal.AsVector())
			return pl, err == nil
		}
	}
	return geom.Plane{}, false
}

// rodNotchSection returns the tool rod's trace of the notch plane — the rod∩notch-plane ellipse — the shared
// curve that trims the rod tunnel wall AND bites the notch cap (both split at the same triple points, so all
// three faces meet there watertight; ADR-0048 §tool-side weld). ok=false when the analytic section is
// unavailable.
func rodNotchSection(notch geom.Plane, rod geom.Cylinder, res geom.Resolution) (geom.Curve3, bool) {
	curves, handled := geom.IntersectSurfacesAnalytic(notch, rod, res)
	if !handled || len(curves) == 0 {
		return nil, false
	}
	return curves[0], true
}

// rodNotchArc returns the shared rod∩notch-plane boundary as a single exact elliptical arc spanning the two
// triple points along the side INSIDE the target cylinder — the arc that trims the rod tunnel wall AND bounds
// the rod's bite of the notch cap. Both consumers split at the same T± by construction, so all three coupled
// faces meet there watertight (ADR-0048 §tool-side weld). ok=false unless there are exactly two junctions and
// the section is the expected ellipse.
func rodNotchArc(notch geom.Plane, rod geom.Cylinder, res geom.Resolution, js []cornerJunction,
	tgtAxisPt math.Point3, tgtAxis math.Vector3, tgtR float64) (geom.EllipticalArc, bool) {
	sec, ok := rodNotchSection(notch, rod, res)
	e, isEll := sec.(geom.EllipseFull)
	if !ok || !isEll || len(js) != 2 {
		return geom.EllipticalArc{}, false
	}
	a0, a1 := ellipseAngleOf(e, js[0].point), ellipseAngleOf(e, js[1].point)
	sweep := a1 - a0
	for sweep <= 0 {
		sweep += 2 * stdmath.Pi
	}
	start := a0
	// rodRadialDist to the TARGET axis is the point's distance to the target cylinder axis: the arc whose
	// midpoint lies inside the target radius is the bite boundary (the removed region is inside the rod ∧
	// inside the target); the complementary arc bulges outside the target and is discarded.
	midDist := rodRadialDist(e.PointAt((a0+sweep/2)/(2*stdmath.Pi)), tgtAxisPt, tgtAxis)
	if midDist >= tgtR {
		start, sweep = a1, 2*stdmath.Pi-sweep
	}
	arc, err := geom.NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius, start, sweep)
	return arc, err == nil
}

// ellipseAngleOf returns the ellipse angle a∈(−π,π] of a point on it: from P=C+Rmaj·cos(a)·û+Rmin·sin(a)·v̂,
// cos(a)=(P−C)·û/Rmaj and sin(a)=(P−C)·v̂/Rmin (û the major axis, v̂=n×û the minor).
func ellipseAngleOf(e geom.EllipseFull, p math.Point3) float64 {
	d := e.Center.VectorTo(p)
	major := e.MajorAxis.AsVector()
	minor := e.Normal.AsVector().Cross(major)
	return stdmath.Atan2(float64(d.Dot(minor))/e.MinorRadius, float64(d.Dot(major))/e.MajorRadius)
}

// biteNotchCap replaces the notch cap face with its rod-bitten form: every boundary edge is split at the rod
// crossings (reusing the exact triple-point solver on the cap's own section curve, so the params land on it
// exactly), the sub-edges INSIDE the rod are dropped, and the resulting gap is bridged by the shared rod∩
// notch arc — the SAME arc the rod tunnel wall carries, so cap and tunnel weld at the triple points (#1738).
// ok=false when the cap has no rod bite (no gap), so a cap the rod does not cross is left whole by the caller.
func biteNotchCap(cap curvedFace, rodArc geom.EllipticalArc, axisPt math.Point3, axisDir math.Vector3, radius float64) (curvedFace, bool) {
	if len(cap.loops) != 1 {
		return curvedFace{}, false
	}
	var kept []loopEdge
	for _, e := range cap.loops[0].edges {
		ts := junctionParams(edgeRodCrossings(e, 0, axisPt, axisDir, radius))
		for _, se := range splitLoopEdgeAtParams(e, ts) {
			if !edgeInsideRod(se, axisPt, axisDir, radius) {
				kept = append(kept, se)
			}
		}
	}
	loop, ok := bridgeGapWithArc(kept, rodArc)
	if !ok {
		return curvedFace{}, false
	}
	return curvedFace{surface: cap.surface, reversed: cap.reversed, lineage: cap.lineage, loops: []curvedLoop{{edges: loop}}}, true
}

// junctionParams extracts the arm parameters of a set of rod crossings.
func junctionParams(cs []cornerJunction) []float64 {
	ts := make([]float64, len(cs))
	for i, c := range cs {
		ts[i] = c.tArm
	}
	return ts
}

// edgeInsideRod reports whether an edge's midpoint lies inside the rod — the sub-edges of the cap boundary
// the rod removes. The midpoint is used (not an endpoint) because a bite sub-edge's endpoints are the triple
// points, exactly ON the rod surface, where the strict test would be ambiguous.
func edgeInsideRod(e loopEdge, axisPt math.Point3, axisDir math.Vector3, radius float64) bool {
	mid := e.curve.PointAt((e.t0 + e.t1) / 2)
	return rodRadialDist(mid, axisPt, axisDir) < radius
}

// bridgeGapWithArc closes the single gap the removed inside-rod sub-edges leave in the cap boundary by
// inserting the rod∩notch arc, oriented to run from the gap's open start to its open end. The kept edges are
// rotated so the chain begins right after the gap, so the arc appends cleanly as the loop's closing edge.
func bridgeGapWithArc(kept []loopEdge, arc geom.EllipticalArc) ([]loopEdge, bool) {
	n := len(kept)
	if n == 0 {
		return nil, false
	}
	gap := -1
	for i := range n {
		if float64(kept[i].end().DistanceTo(kept[(i+1)%n].start())) > 1e-6 {
			gap = i
			break
		}
	}
	if gap < 0 {
		return nil, false
	}
	rot := make([]loopEdge, 0, n+1)
	for k := range n {
		rot = append(rot, kept[(gap+1+k)%n])
	}
	return append(rot, orientArcBetween(arc, rot[n-1].end(), rot[0].start())), true
}

// orientArcBetween returns the arc as a loopEdge traversed from `from` to `to`, flipping the parameter range
// when the arc's natural start is nearer the `to` endpoint.
func orientArcBetween(arc geom.EllipticalArc, from, _ math.Point3) loopEdge {
	if float64(arc.PointAt(0).DistanceTo(from)) <= float64(arc.PointAt(1).DistanceTo(from)) {
		return loopEdge{curve: arc, t0: 0, t1: 1}
	}
	return loopEdge{curve: arc, t0: 1, t1: 0}
}

// nearestRingSeg returns the ring segment index i, the projection fraction tloc∈[0,1] along it, and the
// perpendicular distance for the point p closest to the cyclic vertex ring — where an inserted split point
// lands.
func nearestRingSeg(verts []math.Point3, p math.Point3) (int, float64, float64) {
	best, bestT, bestD := 0, 0.0, stdmath.Inf(1)
	n := len(verts)
	for i := range n {
		a, b := verts[i], verts[(i+1)%n]
		ab := a.VectorTo(b)
		l2 := float64(ab.LengthSquared())
		t := 0.0
		if l2 > 1e-300 {
			t = math.Clamp01(float64(a.VectorTo(p).Dot(ab)) / l2)
		}
		if d := float64(p.DistanceTo(a.TranslateBy(ab.Scale(math.Scalar(t))))); d < bestD {
			best, bestT, bestD = i, t, d
		}
	}
	return best, bestT, bestD
}
