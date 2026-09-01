// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/math"
)

// Holed developable-wall tessellation (M2 Phase 2, Oblikovati/Oblikovati#1335). A cylinder OR cone drilled
// through — a fat cylinder/cone Cut/Join with a crossing rod — leaves the fat SIDE wall as a full-period
// cylinder/cone face carrying one or more lens HOLES where the rod broke through. periodicBandGrid and
// metricPatchMesh both reject a holed periodic face (toUVLoops can't unwrap the seam-wrapping outer loop),
// so it fell to the full-domain grid, which ignores the holes entirely. But both a cylinder and a cone are
// DEVELOPABLE (zero Gaussian curvature), so the wall unrolls and meshes through the trim-local metric-scaled
// CDT: unwrap the wrapping outer loop's angle cumulatively into a contiguous (u,v) rectangle and hand it,
// holes and all, to that CDT. The boundary and hole loops map to their exact (u,v) and the interior nodes
// lift to the exact surface, so the area is accurate regardless of the parameter metric; the metric scaling
// (√E,√G over the trim's (u,v) bbox) only keeps the triangulation well shaped — exact (R,1) for a cylinder,
// the local (v·tanα, secα) average for a cone, whose metric varies slowly over a wall away from the apex.
// The seam (the rectangle's two vertical edges, 2π apart in u) maps to one line in 3D, so the triangles on
// either side meet there with no gap.

// holedConicWallMesh meshes a full-period cylinder or cone side carrying holes by unrolling it to a
// contiguous (u,v) rectangle and delegating to the metric-scaled CDT. ok=false unless the surface is a
// developable side (cylinder/cone) with at least one hole and a genuinely seam-wrapping outer loop.
// A hole straddling the wall's own seam no longer defers: the seam is re-cut clear of the holes
// (wall_seam_recut.go), because no builder can be relied on to have placed it clear (#2038).
func holedConicWallMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	if !isDevelopableSide(s) {
		return nil, false // only a cylinder/cone unrolls (developable: zero Gaussian curvature)
	}
	if len(holes3D) == 0 || isPeriodic(s.UDomain()) == isPeriodic(s.VDomain()) {
		return nil, false // need a holed, singly-periodic side
	}
	br, holesUV, ok := wallBranchClearOfHoles(s, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	return unrolledWallCDT(s, q, br.loop, holes3D, br.uv, holesUV), true
}

// wallBranchClearOfHoles unrolls the wall into a branch every hole fits inside, with the holes already
// mapped into it. It tries the branch the wall's OWN seam defines first, and only when a hole straddles
// that seam — so no whole-period shift brings it inside — re-cuts the seam into the widest hole-free
// gap and unrolls again (#2038).
func wallBranchClearOfHoles(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (wallBranch, [][]math.Point2, bool) {
	br, ok := unrollWall(s, outer3D)
	if !ok {
		return wallBranch{}, nil, false
	}
	if holesUV, fits := holesIntoBranch(s, holes3D, br.umin, br.umax); fits {
		return br, holesUV, true
	}
	wrap, ok := rebridgedWallLoop(s, outer3D, holes3D)
	if !ok {
		return wallBranch{}, nil, false
	}
	if br, ok = unrollWall(s, wrap); !ok {
		return wallBranch{}, nil, false
	}
	holesUV, fits := holesIntoBranch(s, holes3D, br.umin, br.umax)
	return br, holesUV, fits
}

// isDevelopableSide reports whether the surface is a cylinder (circular or elliptical) or a cone — the
// developable (zero Gaussian-curvature) sides whose walls unroll, so a holed wall on them meshes
// through the unrolled CDT.
func isDevelopableSide(s geom.Surface) bool {
	switch s.(type) {
	case geom.Cylinder, geom.EllipticalCylinder, geom.Cone:
		return true
	default:
		return false
	}
}

// unrolledWallCDT triangulates the unrolled wall (the contiguous (u,v) rectangle minus the holes) with a
// metric-scaled constrained Delaunay in (u,v) — (√E,√G)≈(R,1) for a cylinder, so the parameter space is
// isometric to 3D and the triangles are well shaped — and INTERIOR Steiner refinement so the curved area
// is accurate. It uses the OCCT BRepMesh insertion order (constrainedDelaunayRefined): the frontier loops
// first, constraints recovered on that small set, then the interior nodes inserted into the constrained
// mesh (where the frontier edges are protected from the cavity). Inserting everything at once tears a large
// wall with several concave saddle holes. The exact 3D boundary points are kept so the wall welds to its
// caps and to the faces sharing its hole edges.
func unrolledWallCDT(s geom.Surface, q Quality, outer3D []math.Point3, holes3D [][]math.Point3, outerUV []math.Point2, holesUV [][]math.Point2) *Mesh {
	su, sv := trimMetricScale(s, outerUV)
	outer2D := scaleUVLoop(outerUV, su, sv)
	holes2D := make([][]math.Point2, len(holesUV))
	for i, h := range holesUV {
		holes2D[i] = scaleUVLoop(h, su, sv)
	}
	uv, pos, nrm, loops := patchLoops2D(s, outer3D, holes3D, outer2D, holes2D)
	nFrontier := len(uv)
	nodes, saturated := adaptiveInteriorNodes(s, outerUV, holesUV, q, 1, false)
	nodes = append(nodes, neckCorridorNodes(holesUV)...) // #1818: seed the bridge between near-touching holes
	uv, pos, nrm = appendInteriorNodes(s, nodes, su, sv, uv, pos, nrm)
	tris, unrecovered, leaked := constrainedDelaunayRefinedChecked(uv, loops, nFrontier)
	if len(tris) == 0 {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	m := patchMeshFrom(pos, nrm, tris)
	validate.RepairFolds(m, 8)
	recordCapSaturation(m, saturated, q)
	recordConstraintLeak(m, unrecovered, leaked) // #1410: surface non-recovery; never a silent boundary leak
	return m
}

// neckCorridorNodes seeds Steiner nodes in the BRIDGE between two near-touching lens holes (the near-pinch
// crossing neck, #1818). Where two holes come within a few chord lengths, the interior grid — kept a margin
// clear of every hole — leaves the thin corridor between them EMPTY, so the CDT chords one hole rim straight
// to the other; those chords coincide with the reversed tunnel band's own pinch chords (same shared loop
// vertices) and weld into non-manifold deg-4 edges. Seeding the corridor midline forces the wall to mesh
// hole→node→hole on the fat surface instead, breaking the coincidence. Only fires for exactly two holes that
// actually near-touch, so an ordinary well-separated drilling is untouched.
func neckCorridorNodes(holesUV [][]math.Point2) [][2]float64 {
	if len(holesUV) != 2 {
		return nil
	}
	gap := minCrossVertexDistance(holesUV[0], holesUV[1])
	chord := meanLoopChord2D(holesUV[0])
	if chord <= 0 || gap > nearNeckChords*chord {
		return nil // holes well separated: no corridor to seed
	}
	var out [][2]float64
	for _, a := range holesUV[0] {
		b, d := nearestVertex2D(a, holesUV[1])
		if d <= nearNeckChords*chord {
			out = append(out, [2]float64{float64(a.X+b.X) / 2, float64(a.Y+b.Y) / 2})
		}
	}
	return out
}

// nearNeckChords is how close (in hole-chord multiples) two holes must approach before the corridor between
// them is seeded — a dimensionless, model-scale-free threshold (#1818).
const nearNeckChords = 4.0

// minCrossVertexDistance is the smallest distance between any vertex of loop a and any of loop b.
func minCrossVertexDistance(a, b []math.Point2) float64 {
	min := stdmath.Inf(1)
	for _, pa := range a {
		if _, d := nearestVertex2D(pa, b); d < min {
			min = d
		}
	}
	return min
}

// nearestVertex2D returns the vertex of loop nearest to p and its distance.
func nearestVertex2D(p math.Point2, loop []math.Point2) (math.Point2, float64) {
	best, bestD := math.Point2{}, stdmath.Inf(1)
	for _, q := range loop {
		if d := float64(p.DistanceTo(q)); d < bestD {
			best, bestD = q, d
		}
	}
	return best, bestD
}

// meanLoopChord2D is the mean consecutive-vertex spacing of a (u,v) loop — its own length scale, the neck
// gap is measured against so the corridor gate is spacing-independent.
func meanLoopChord2D(loop []math.Point2) float64 {
	if len(loop) < 2 {
		return 0
	}
	var sum float64
	for i := 1; i < len(loop); i++ {
		sum += float64(loop[i-1].DistanceTo(loop[i]))
	}
	return sum / float64(len(loop)-1)
}

// appendInteriorNodes lifts each interior (u,v) node onto the surface — its metric-scaled (u,v), 3D point,
// and normal — appending them to the CDT's parallel uv/pos/nrm buffers.
func appendInteriorNodes(s geom.Surface, nodes [][2]float64, su, sv float64, uv [][2]float64, pos []math.Point3, nrm []math.Vector3) ([][2]float64, []math.Point3, []math.Vector3) {
	for _, g := range nodes {
		uv = append(uv, [2]float64{g[0] * su, g[1] * sv})
		pos = append(pos, s.PointAt(g[0], g[1]))
		nrm = append(nrm, s.NormalAt(g[0], g[1]))
	}
	return uv, pos, nrm
}

// scaleUVLoop scales a (u,v) loop by the per-axis metric (su, sv) so the CDT runs in a space isometric to
// 3D.
func scaleUVLoop(loop []math.Point2, su, sv float64) []math.Point2 {
	out := make([]math.Point2, len(loop))
	for i, p := range loop {
		out[i] = math.P2(p.X*math.Scalar(su), p.Y*math.Scalar(sv))
	}
	return out
}

// wrappedWallUV unwraps the wall's seam-wrapping outer loop angle (u) into contiguous values, returning
// the (u,v) loop and its u-range. ok=false unless the loop spans essentially the full period (otherwise
// it is an ordinary contractible patch toUVLoops already handles, not a wrapping wall).
func wrappedWallUV(s geom.Surface, outer3D []math.Point3) (uv []math.Point2, umin, umax float64, ok bool) {
	us := make([]float64, len(outer3D))
	vs := make([]float64, len(outer3D))
	for i, p := range outer3D {
		us[i], vs[i] = s.ParamAt(p)
	}
	cu := cumulativeUnwrap(us)
	umin, umax = minMax(cu)
	if umax-umin < 2*stdmath.Pi-0.5 {
		return nil, 0, 0, false // not a full wrap
	}
	uv = make([]math.Point2, len(outer3D))
	for i := range outer3D {
		uv[i] = math.P2(math.Scalar(cu[i]), math.Scalar(vs[i]))
	}
	return uv, umin, umax, true
}

// holesIntoBranch maps each hole loop to (u,v) and shifts it by whole periods into the wall rectangle's
// u-range. ok=false if a hole wraps the seam or cannot be brought wholly inside (it straddles the seam —
// the builder is expected to place the wall's seam clear of its holes).
func holesIntoBranch(s geom.Surface, holes3D [][]math.Point3, umin, umax float64) ([][]math.Point2, bool) {
	out := make([][]math.Point2, 0, len(holes3D))
	for _, h := range holes3D {
		uv, ok := holeUVInBranch(s, h, umin, umax)
		if !ok {
			return nil, false
		}
		out = append(out, uv)
	}
	return out, true
}

// holeUVInBranch unwraps one hole loop to contiguous (u,v) and offsets it by whole periods so it sits
// inside [umin, umax]. ok=false when the hole itself wraps the seam or still straddles a rectangle edge.
func holeUVInBranch(s geom.Surface, hole []math.Point3, umin, umax float64) ([]math.Point2, bool) {
	us := make([]float64, len(hole))
	vs := make([]float64, len(hole))
	for i, p := range hole {
		us[i], vs[i] = s.ParamAt(p)
	}
	if loopEncirclesAxis(us) {
		return nil, false // a hole that itself wraps the seam (an exact ±2π winding, not a sampled span)
	}
	cu := cumulativeUnwrap(us)
	lo, hi := minMax(cu)
	shift := branchShift((lo+hi)/2, umin, umax)
	if lo+shift < umin-seamAngularTol || hi+shift > umax+seamAngularTol {
		return nil, false // straddles a seam edge even after shifting (u is radians)
	}
	uv := make([]math.Point2, len(hole))
	for i := range hole {
		uv[i] = math.P2(math.Scalar(cu[i]+shift), math.Scalar(vs[i]))
	}
	return uv, true
}

// loopWinding returns a CLOSED loop's total winding about the periodic axis: every step folded into
// (−π, π] and summed, INCLUDING the CLOSING step from the last sample back to the first — the one
// segment an open-chain accumulation such as cumulativeUnwrap never measures.
//
// It is EXACTLY quantised: the raw steps around a closed ring telescope to zero and each fold adds a
// whole period, so the sum is 2π·k for an integer k — 0 for a contractible lens hole, ±2π for a rim
// that encircles the axis. Measured over the whole shipped population (1166 loops across the OCCT blend
// corpus and the kernel suite) the residual |W − 2πk| is 0.000e+00 on every one; there is no grey band.
//
// WHY THIS AND NOT THE SAMPLED u-RANGE (the classifier it replaced). For a rim traversed once, the
// cumulative range hi−lo is 2π minus the loop's UN-TRAVERSED closing step, so "hi−lo > 2π − 0.5" is
// really "is this loop's single largest angular sampling gap under 0.5 rad" — a sampling-density test
// wearing a topology test's clothes, and the same un-measured-closing-step blind spot that made
// unwrap() unsafe (.superpowers/sdd/unwrapperiod-report.md §2). Its tightest measured headroom on the
// shipped population was 0.0516 rad — four times SMALLER than one sampling step of the very loop it was
// classifying — so a slightly coarser faceting flips a rim to a lens and drops the wall to the flat
// best-fit-plane CDT (#1724: an inside-out band, ~140 free edges). The winding instead has a full π of
// headroom either side, which is the fold's OWN bound; the largest wrapped step measured anywhere in
// the tree is 0.211·π, a 4.7× margin.
//
// Example: loopWinding([]float64{0, 2, 4, 6}) ≈ 2π (a rim); loopWinding([]float64{0, .1, .2}) == 0.
func loopWinding(us []float64) float64 {
	if len(us) < 2 {
		return 0
	}
	w := probe.WrapPi(us[0] - us[len(us)-1]) // the CLOSING step, the one an open chain never forms
	for i := 1; i < len(us); i++ {
		w += probe.WrapPi(us[i] - us[i-1])
	}
	return w
}

// loopEncirclesAxis reports whether a closed loop of periodic surface parameters winds around the axis
// — a rim — rather than being a contractible lens hole. |W| > π separates 0 from ±2π with half a period
// of headroom on each side; see loopWinding for why that bound is structural and 2π − 0.5 was not.
func loopEncirclesAxis(us []float64) bool {
	return stdmath.Abs(loopWinding(us)) > stdmath.Pi
}

// cumulativeUnwrap makes a periodic parameter contiguous by accumulating per-step deltas (each jump
// folded into (−π, π]), WITHOUT failing on a full-period wrap — unlike unwrap(), which rejects exactly
// the wrapping wall loop this mesher is built for.
func cumulativeUnwrap(a []float64) []float64 {
	out := make([]float64, len(a))
	out[0] = a[0]
	for i := 1; i < len(a); i++ {
		d := a[i] - a[i-1]
		for d > stdmath.Pi {
			d -= 2 * stdmath.Pi
		}
		for d <= -stdmath.Pi {
			d += 2 * stdmath.Pi
		}
		out[i] = out[i-1] + d
	}
	return out
}

// branchShift returns the whole-period offset that brings angle a into [umin, umax].
func branchShift(a, umin, umax float64) float64 {
	shift := 0.0
	for a+shift < umin {
		shift += 2 * stdmath.Pi
	}
	for a+shift > umax {
		shift -= 2 * stdmath.Pi
	}
	return shift
}

// minMax returns the smallest and largest of a non-empty slice.
func minMax(xs []float64) (lo, hi float64) {
	lo, hi = xs[0], xs[0]
	for _, x := range xs {
		lo, hi = stdmath.Min(lo, x), stdmath.Max(hi, x)
	}
	return lo, hi
}
