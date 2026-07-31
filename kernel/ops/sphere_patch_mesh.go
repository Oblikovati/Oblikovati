// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Multi-arc sphere-patch tessellation (M2 Phase 1, Oblikovati/Oblikovati#1334). A sphere face bounded
// by SEVERAL arcs — the curved face a box (or any multi-plane) cut leaves on a sphere — meshed wrong
// through the lat/long (u,v) path: near a pole every column collapses, so the constrained-Delaunay in
// (u,v) folds and over/under-fills. The fix is to triangulate in a conformal-ish chart centred on the
// patch instead, where there is no pole/seam degeneracy: a GNOMONIC chart for a patch within a
// hemisphere (least distortion), a STEREOGRAPHIC chart for a larger patch (it stays finite for every
// point except the patch's antipode, so a patch up to nearly the whole sphere maps to a bounded region).
// Interior Steiner points lift back to the sphere exactly, so the patch carries its true curvature; the
// boundary keeps the exact edge samples so it welds with neighbour faces. Self-gates: a patch that wraps
// past the stereographic limit (its boundary reaches the antipode) defers to the caller.

// Chart axis-dot thresholds: every boundary point's direction·axis must clear the threshold so the chart
// is well-conditioned. Gnomonic diverges at 90° (dot 0); stereographic diverges only at 180° (dot −1),
// so it is used down to ~150° (dot −0.85) where the gnomonic chart has already bowed out.
const (
	gnomonicMinDot = 0.15  // ~81° from the axis: within a hemisphere → gnomonic (least distortion)
	stereoMinDot   = -0.85 // ~148° from the axis: a >hemisphere patch clear of the antipode → stereographic
)

// spherePatchMesh meshes a multi-arc sphere patch in a patch-centred chart. ok=false unless the surface
// is a sphere whose boundary clears the chart's antipode — then the caller keeps its existing path.
func spherePatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	sph, ok := s.(geom.Sphere)
	if !ok || len(outer3D) < 3 {
		return nil, false
	}
	chart, ok := chooseSphereChart(sph, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	uv, pos, nrm, loops, loops2D := projectPatchBoundary(sph, chart, outer3D, holes3D)
	clamp := addPatchInterior(chart, &uv, &pos, &nrm, loops2D, q)
	tris := constrainedDelaunay(uv, loops)
	if len(tris) == 0 {
		return nil, false
	}
	m := patchMeshFrom(pos, nrm, tris)
	repairFolds(m, 8)
	diagnosePatchGridClamp(m, clamp, q)
	return m, true
}

// patchGridClamp records what the chart's chord-tolerance spacing ASKED for versus what
// patchGridCellBudget GRANTED, so the saturation diagnostic can state the deficit factor instead of a
// bare flag — a census/debug consumer needs requested-vs-laid to judge how far below tolerance the
// mesh sits (patchgridcap-report.md §1: 25 shipped cases rode the old silent clamp).
type patchGridClamp struct {
	reqU, reqV   int // intervals the tolerance-derived spacing asked for, per chart axis
	laidU, laidV int // intervals actually laid after the budget
}

func (c patchGridClamp) capped() bool {
	return c.laidU < c.reqU || c.laidV < c.reqV
}

// diagnosePatchGridClamp records the budget scaling on the mesh when the interior Steiner grid was
// denied the spacing the chord tolerance asked for — the same honest degradation record the NURBS
// interior refinement emits (CodeTessellateCapSaturated, Oblikovati#1412). Before this the clamp was
// SILENT, and a large sphere patch (a hemisphere is the worst case: chart bbox 2R × 2R) under-reported
// its area at every swept tolerance with nothing to see — the S6/S7 "sphere notch" that never was
// (sphere-notch-report.md).
func diagnosePatchGridClamp(m *Mesh, c patchGridClamp, q Quality) {
	if !c.capped() {
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeTessellateCapSaturated,
		Severity: diag.Warning,
		Detail: fmt.Sprintf("sphere-patch interior grid clamped to %dx%d of the %dx%d steps chord tol %g asked",
			c.laidU, c.laidV, c.reqU, c.reqV, q.tol()),
	})
}

// sphereChart projects a sphere point to a 2D chart and lifts a 2D point back to the sphere (point +
// outward normal). Both the gnomonic and stereographic charts satisfy it.
type sphereChart interface {
	project(p math.Point3) [2]float64
	lift(a, b float64) (math.Point3, math.Vector3)
	gridSpacing(q Quality) float64 // interior Steiner spacing whose lifted chord error meets q
}

// chooseSphereChart picks the chart for a patch: gnomonic within a hemisphere, stereographic for a larger
// patch that still clears the antipode, else ok=false. The axis is the boundary's mean direction.
func chooseSphereChart(sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3) (sphereChart, bool) {
	axis, minDot, ok := patchAxis(sph, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	e1, e2 := planeBasis(axis)
	switch {
	case minDot >= gnomonicMinDot:
		return gnomonicChart{sph: sph, axis: axis, e1: e1, e2: e2}, true
	case minDot >= stereoMinDot:
		return stereoChart{sph: sph, axis: axis, e1: e1, e2: e2}, true
	default:
		return nil, false
	}
}

// patchAxis returns the chart axis — a direction INSIDE the patch — and the minimum direction·axis over
// the boundary (how far the patch reaches). The boundary's mean direction points at the patch for a small
// patch but at the REMOVED region for a >hemisphere patch (e.g. a 7/8-sphere's boundary hugs the missing
// octant), so the axis is whichever of ±mean the boundary loop winds positively around — the interior.
func patchAxis(sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3) (math.Vector3, float64, bool) {
	pts := append(append([]math.Point3{}, outer3D...), flattenLoops(holes3D)...)
	var sum math.Vector3
	for _, p := range pts {
		sum = sum.Add(sphereDir(sph, p))
	}
	mean, err := math.UnitVector3FromVector(sum)
	if err != nil {
		return math.Vector3{}, 0, false
	}
	axis := interiorAxis(sph, outer3D, mean.AsVector())
	minDot := 1.0
	for _, p := range pts {
		minDot = stdmath.Min(minDot, float64(sphereDir(sph, p).Dot(axis)))
	}
	return axis, minDot, true
}

// interiorAxis chooses, between the mean direction and its opposite, the one the boundary loop winds
// positively (≈ +2π) around — the side of the separating boundary that is the patch interior. Falls back
// to the mean if neither reads clearly inside (a degenerate/non-convex boundary).
func interiorAxis(sph geom.Sphere, outer3D []math.Point3, mean math.Vector3) math.Vector3 {
	for _, cand := range []math.Vector3{mean, mean.Scale(-1)} {
		center := sph.Center.TranslateBy(cand.Scale(math.Scalar(sph.Radius)))
		if loopWindingAround(outer3D, center, cand) > stdmath.Pi {
			return cand
		}
	}
	return mean
}

// loopWindingAround returns the signed turning angle of the boundary points seen from p, summed around
// the axis normal (the geodesic winding number): ≈ +2π when the loop winds CCW around p, else ≈ 0/−2π.
func loopWindingAround(pts []math.Point3, p math.Point3, normal math.Vector3) float64 {
	if len(pts) < 3 {
		return 0
	}
	sum := 0.0
	for i := range pts {
		a := tangentPart(p.VectorTo(pts[i]), normal)
		b := tangentPart(p.VectorTo(pts[(i+1)%len(pts)]), normal)
		if a.LengthSquared() == 0 || b.LengthSquared() == 0 {
			continue
		}
		sum += stdmath.Atan2(float64(a.Cross(b).Dot(normal)), float64(a.Dot(b)))
	}
	return sum
}

// tangentPart projects v onto the plane perpendicular to the unit normal.
func tangentPart(v, normal math.Vector3) math.Vector3 {
	return v.Sub(normal.Scale(v.Dot(normal)))
}

// gnomonicChart is the central projection of a sphere onto the tangent plane at axis: a unit direction u
// maps to ((R/(u·axis))·u − R·axis) resolved in the orthonormal in-plane basis (e1, e2).
type gnomonicChart struct {
	sph    geom.Sphere
	axis   math.Vector3
	e1, e2 math.Vector3
}

func (c gnomonicChart) project(p math.Point3) [2]float64 {
	u := sphereDir(c.sph, p)
	s := float64(u.Dot(c.axis))
	local := u.Scale(math.Scalar(c.sph.Radius / s)).Add(c.axis.Scale(math.Scalar(-c.sph.Radius)))
	return [2]float64{float64(local.Dot(c.e1)), float64(local.Dot(c.e2))}
}

func (c gnomonicChart) lift(a, b float64) (math.Point3, math.Vector3) {
	planeDir := c.axis.Scale(math.Scalar(c.sph.Radius)).Add(c.e1.Scale(math.Scalar(a))).Add(c.e2.Scale(math.Scalar(b)))
	dir := planeDir.Scale(math.Scalar(1 / float64(planeDir.Length())))
	return c.sph.Center.TranslateBy(dir.Scale(math.Scalar(c.sph.Radius))), dir
}

// gridSpacing: a gnomonic step h near the axis lifts to ≈ h on the sphere, so the chord error R(1−cos
// h/R) ≈ h²/2R ≤ tol gives h = √(2R·tol).
func (c gnomonicChart) gridSpacing(q Quality) float64 {
	return stdmath.Sqrt(2 * c.sph.Radius * q.tol())
}

// stereoChart is the stereographic projection of a sphere from the antipode of axis (−axis) onto the
// equatorial plane: u maps to R·(u − (u·axis)·axis)/(1 + u·axis), finite for every u except −axis. It
// maps a >hemisphere patch around axis to a bounded region (gnomonic would diverge past 90°).
type stereoChart struct {
	sph    geom.Sphere
	axis   math.Vector3
	e1, e2 math.Vector3
}

func (c stereoChart) project(p math.Point3) [2]float64 {
	u := sphereDir(c.sph, p)
	den := 1 + float64(u.Dot(c.axis))
	k := c.sph.Radius / den
	return [2]float64{k * float64(u.Dot(c.e1)), k * float64(u.Dot(c.e2))}
}

func (c stereoChart) lift(a, b float64) (math.Point3, math.Vector3) {
	r := c.sph.Radius
	rho2 := a*a + b*b
	s := (r*r - rho2) / (r*r + rho2) // u·axis recovered from the projection radius
	k := 2 * r / (r*r + rho2)        // perpendicular scale
	u := c.axis.Scale(math.Scalar(s)).Add(c.e1.Scale(math.Scalar(k * a))).Add(c.e2.Scale(math.Scalar(k * b)))
	dir := u.Scale(math.Scalar(1 / float64(u.Length()))) // unit to first order; normalise for exactness
	return c.sph.Center.TranslateBy(dir.Scale(math.Scalar(r))), dir
}

// gridSpacing: stereographic magnifies most at the axis (a step h there lifts to ≈ 2h on the sphere), so
// halve the gnomonic spacing to keep the worst-case (central) chord error within tol.
func (c stereoChart) gridSpacing(q Quality) float64 {
	return stdmath.Sqrt(2*c.sph.Radius*q.tol()) / 2
}

// projectPatchBoundary maps every boundary loop to the chart, keeping the EXACT 3D points and radial
// normals (so the patch welds to neighbour faces). Returns the parallel 2D/3D/normal arrays, the loop
// index sequences for the CDT, and the loops as math.Point2 for clearOfTrim.
func projectPatchBoundary(sph geom.Sphere, chart sphereChart, outer3D []math.Point3, holes3D [][]math.Point3) (uv [][2]float64, pos []math.Point3, nrm []math.Vector3, loops [][]int, loops2D [][]math.Point2) {
	for _, loop := range append([][]math.Point3{outer3D}, holes3D...) {
		idx := make([]int, len(loop))
		ring2D := make([]math.Point2, len(loop))
		for i, p := range loop {
			g := chart.project(p)
			idx[i] = len(uv)
			uv = append(uv, g)
			pos = append(pos, p)
			nrm = append(nrm, sphereDir(sph, p))
			ring2D[i] = math.P2(math.Scalar(g[0]), math.Scalar(g[1]))
		}
		loops = append(loops, idx)
		loops2D = append(loops2D, ring2D)
	}
	return uv, pos, nrm, loops, loops2D
}

// patchGridCellBudget bounds the interior Steiner grid's TOTAL candidate count (nu·nv), replacing the
// silent per-axis patchGridCap=80 (introduced 65a4b0e21 at 60, raised aa08a7d2c — never
// tolerance-derived) that floored 25 shipped corpus faces at PropertyQuality with deficits up to −200
// area on simple/D6's R=150 host sphere, 6–70× outside PropertyQuality's own ~0.01% contract
// (patchgridcap-report.md). The budget is a COST bound, not a quality bound: the CDT pipeline measures
// ~20 µs per laid node (zz cost probe: 2.8 M nodes = 60 s), so 2^18 cells keeps the worst single face
// at a few seconds. When it binds, BOTH axes scale by the same factor (minimal, even degradation) and
// the mesh carries tessellate.cap-saturated stating requested-vs-granted — never a silent floor.
//
// SWEEP DOCTRINE (for the next sweep-runner; patchgridcap-report.md fix wave I3): a budget-BOUND
// sweep DRIFTS as the chord tolerance tightens — the boundary sampling keeps refining under a fixed
// interior grid — it does NOT sit at a labeled plateau (measured, D2's R=150 host: −0.211 → −0.246
// area vs DRAWEXE across the bound band). And the budget-HONOURED band is not monotone either: D2
// reads −0.022 at ct 8e-3 but −0.191 at 4e-3, both honoured, no diagnostic (different grid/boundary
// phase, both inside their own chord contracts). So never read a step like −0.02 → −0.19 as a
// regression: check tessellate.cap-saturated and the requested-vs-laid record before interpreting
// any sweep step.
const patchGridCellBudget = 1 << 18

// addPatchInterior lays a grid of Steiner points across the chart bbox at the chart's spacing, keeps
// those strictly inside the trim, and lifts each to the sphere — the interior curvature the boundary
// alone would chord flat. Returns the requested-vs-laid grid record so the caller can diagnose a
// budget-scaled (coarser-than-tolerance) grid on the mesh.
func addPatchInterior(chart sphereChart, uv *[][2]float64, pos *[]math.Point3, nrm *[]math.Vector3, loops2D [][]math.Point2, q Quality) patchGridClamp {
	umin, umax, vmin, vmax := bounds2D(loops2D[0])
	h := chart.gridSpacing(q)
	if h <= 0 {
		return patchGridClamp{}
	}
	clamp := budgetGridSteps(umax-umin, vmax-vmin, h)
	scan := newTrimScan(loops2D[0], loops2D[1:], h*0.25)
	for i := 1; i < clamp.laidU; i++ {
		u := umin + (umax-umin)*float64(i)/float64(clamp.laidU)
		layPatchGridColumn(chart, scan, clamp, u, vmin, vmax, uv, pos, nrm)
	}
	return clamp
}

// layPatchGridColumn lays one constant-u column of the interior grid: each candidate strictly clear
// of the trim is lifted to the sphere and appended. The u ordinate is the same expression the fused
// loop computed (bit-identical positions; extraction is the fix-wave m5 nesting flatten).
func layPatchGridColumn(chart sphereChart, scan *trimScan, clamp patchGridClamp, u, vmin, vmax float64, uv *[][2]float64, pos *[]math.Point3, nrm *[]math.Vector3) {
	for j := 1; j < clamp.laidV; j++ {
		p := [2]float64{u, vmin + (vmax-vmin)*float64(j)/float64(clamp.laidV)}
		if !scan.clear(p) {
			continue
		}
		pt, n := chart.lift(p[0], p[1])
		*uv = append(*uv, p)
		*pos = append(*pos, pt)
		*nrm = append(*nrm, n)
	}
}

// budgetGridSteps sizes the interior grid: the tolerance-derived intervals per axis, scaled down
// EVENLY only when their product exceeds patchGridCellBudget — so the chord tolerance is honoured
// whenever the budget allows, and degradation (diagnosed by the caller) is minimal and isotropic.
func budgetGridSteps(uExt, vExt, h float64) patchGridClamp {
	reqU, reqV := int(uExt/h), int(vExt/h)
	c := patchGridClamp{reqU: reqU, reqV: reqV, laidU: reqU, laidV: reqV}
	if cells := float64(reqU) * float64(reqV); cells > patchGridCellBudget {
		scale := stdmath.Sqrt(cells / patchGridCellBudget)
		c.laidU, c.laidV = int(float64(reqU)/scale), int(float64(reqV)/scale)
	}
	return c
}

// sphereDir returns the outward unit direction from the sphere centre to a point on it.
func sphereDir(sph geom.Sphere, p math.Point3) math.Vector3 {
	return sph.Center.VectorTo(p).Scale(math.Scalar(1 / sph.Radius))
}

// flattenLoops concatenates loop point rings.
func flattenLoops(loops [][]math.Point3) []math.Point3 {
	var out []math.Point3
	for _, l := range loops {
		out = append(out, l...)
	}
	return out
}
