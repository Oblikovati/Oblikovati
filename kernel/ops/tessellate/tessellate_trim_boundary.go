// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/math"
)

// Trimmed curved-face tessellation — the TRIM-BOUNDARY IMPRINT (M48 #2224 split of tessellate_trim.go).
// A curved face is meshed over its trim region, so the boundary loops must first be mapped into (u,v)
// space (unwrapping periodic parameters into a simple polygon, or declining when a loop wraps the seam),
// then — when the region is not an iso rectangle — triangulated from its boundary in a metric-scaled (u,v)
// or a best-fit-plane projection. This file holds that boundary→(u,v) imprint and the boundary-patch mesher.

// ToUVLoops maps the boundary loops to parameter space, unwrapping periodic parameters so
// a loop reads as a contiguous polygon; ok=false if a loop wraps the full seam — including
// one that starts ON the seam and only leaps the period on its closing step (see [unwrap]).
func ToUVLoops(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (outer []math.Point2, holes [][]math.Point2, ok bool) {
	uPer, vPer := IsPeriodic(s.UDomain()), IsPeriodic(s.VDomain())
	if outer, ok = toUVLoop(s, outer3D, uPer, vPer); !ok {
		return nil, nil, false
	}
	for _, h := range holes3D {
		hu, hok := toUVLoop(s, h, uPer, vPer)
		if !hok {
			return nil, nil, false
		}
		holes = append(holes, hu)
	}
	return outer, holes, true
}

// toUVLoop inverts each 3D loop point to (u,v) and unwraps periodic parameters.
func toUVLoop(s geom.Surface, loop []math.Point3, uPer, vPer bool) ([]math.Point2, bool) {
	us := make([]float64, len(loop))
	vs := make([]float64, len(loop))
	for i, p := range loop {
		us[i], vs[i] = s.ParamAt(p)
	}
	if uPer {
		var ok bool
		if us, ok = Unwrap(us); !ok {
			return nil, false
		}
	}
	if vPer {
		var ok bool
		if vs, ok = Unwrap(vs); !ok {
			return nil, false
		}
	}
	out := make([]math.Point2, len(loop))
	for i := range loop {
		out[i] = math.P2(us[i], vs[i])
	}
	return out, true
}

// Unwrap removes 2π jumps so a periodic parameter reads continuously around the CLOSED ring; ok=false
// when the loop wraps the seam and so is not a simple polygon in this chart — either because it winds a
// whole period about the periodic axis ([seamWindingLeap]) or because its developed span reaches 2π.
func Unwrap(a []float64) ([]float64, bool) {
	out := make([]float64, len(a))
	out[0] = a[0]
	lo, hi := a[0], a[0]
	for i := 1; i < len(a); i++ {
		out[i] = out[i-1] + probe.WrapPi(a[i]-a[i-1])
		lo, hi = stdmath.Min(lo, out[i]), stdmath.Max(hi, out[i])
	}
	if seamWindingLeap(a, out) {
		return out, false
	}
	return out, hi-lo < 2*stdmath.Pi-1e-6 // tol:angular (radians)
}

// seamWindingLeap reports whether the loop's CLOSING step — last sample back to first, a polygon segment
// like any other, and the one the open chain never measures — is drawn in the chart as a leap of a whole
// period while the geometry takes the short way round.
//
// WHY THE OPEN CHAIN IS NOT ENOUGH. The span guard above sees only samples 0..n−1. A loop that STARTS on
// the seam therefore passes it by ε — simple/W1's corner sphere reads 6.2586 against 2π, 0.024 rad of
// margin — and then jumps from u ≈ 2π back to u = 0 on closure, developing a patch that encircles the
// chart's pole as if it were a contractible polygon. The quantity here is the loop's total WINDING about
// the periodic axis: Σ of the wrapped steps around the full cycle, which is 0 for every loop that has a
// development and ±2πk for every loop that has none (retrace-detector-report.md §7.1).
//
// Rejecting is the honest answer, not a fallback: a loop with nonzero winding is an object on the
// covering space, so it has no simple polygon in this chart to hand the triangulator. The caller routes
// it to meshSeamCrossingFace, which is where seam-wrapping loops already go.
func seamWindingLeap(a, out []float64) bool {
	n := len(a)
	if n < 2 {
		return false
	}
	closing := probe.WrapPi(a[0] - a[n-1])
	return stdmath.Abs(out[0]-out[n-1]-closing) > SeamAngularTol
}

// IsPeriodic reports whether a [0, 2π] parameter domain wraps.
func IsPeriodic(lo, hi float64) bool {
	return stdmath.Abs(lo) < 1e-9 && stdmath.Abs(hi-2*stdmath.Pi) < 1e-9 // tol:angular (radians)
}

// nonRectangularMesh meshes a non-iso-rectangular curved trim. Every analytic curved surface (torus,
// sphere, cylinder, cone) AND the B-spline trim go through metricPatchMesh — a trim-local metric-scaled
// (u,v) CDT WITH curvature-adaptive interior Steiner points (#585, #1323 L3) — so a larger freeform
// trim's interior is refined to the chord tolerance, not chorded flat across the boundary loops.
// Anything else keeps the best-fit-plane ear-clip (boundaryPatchMesh).
func nonRectangularMesh(s geom.Surface, q Quality, outer3D []math.Point3, holes3D [][]math.Point3, outerUV []math.Point2, holesUV [][]math.Point2) *Mesh {
	switch s.(type) {
	case geom.Torus, geom.Sphere, geom.Cylinder, geom.EllipticalCylinder, geom.Cone, geom.BSplineSurface:
		// These trims fold when flattened to a best-fit plane (boundaryPatchMesh) or meshed over a plain
		// anisotropic (u,v) (gridPatchMesh): a torus's ring-vs-tube, a sphere near its poles, a trimmed
		// cyl/cone, a freeform B-spline. metricPatchMesh triangulates in a TRIM-LOCAL metric-scaled (u,v)
		// (√E,√G over the trim's own (u,v) bbox, so even a cone — whose metric degenerates only toward
		// the far-off apex — stays well conditioned) with deflection-adaptive interior nodes kept
		// strictly inside the trim (adaptiveInteriorNodes/clearOfTrim), plus validate.RepairFolds and a
		// boundary-only fallback. This was the bulk of the EDF over-enclosure (#585) and, for B-splines,
		// removes the interior chord error of the old boundary-only triangulation (#1323 L3).
		return MetricPatchMesh(s, q, outer3D, holes3D, outerUV, holesUV)
	}
	return boundaryPatchMesh(s, outer3D, holes3D)
}

// boundaryPatchMesh triangulates a curved face from its boundary loops alone (no interior
// Steiner points): the loops are flattened onto their best-fit plane (NOT the surface's own
// (u,v), which can be degenerate — e.g. a sphere patch corner landing on the lat/long pole),
// ear-clipped there, and lifted back to their exact 3D boundary points, each triangle wound
// outward. A coarse but watertight covering of the exact trim region — right for small patches
// (corner blends); larger non-rectangular curved faces would want a refined triangulation.
func boundaryPatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) *Mesh {
	outer2D, holes2D := patchProjection(s, outer3D, holes3D)
	pos := outer3D
	var tris [][3]int
	if len(holes2D) > 0 {
		// earcut's indices address the outer loop followed by the holes concatenated, so the
		// 3D buffer is built in that same order (matches the robust planar path, see earcut.go).
		pos = append([]math.Point3(nil), outer3D...)
		for _, h := range holes3D {
			pos = append(pos, h...)
		}
		tris = earcut(outer2D, holes2D)
	} else {
		tris, _ = earClip(outer2D) // an incomplete clip is caught below by patchCoverageGate
	}
	// Ear clipping is only guaranteed on a simple polygon; a degenerate/self-touching trim makes it
	// break early (a hole) or emit count-complete but OVERLAPPING triangles — the coverage gate
	// detects both in the projection plane and retriangulates through the CDT, whose
	// split-at-vertex recovery handles the self-touching case exactly (#1605; recovery tier = #1604).
	tris, accepted := patchCoverageGate(outer2D, holes2D, tris)
	// The ear-clip output is consistently oriented in the projection plane; patchMeshFrom keeps that
	// winding and flips the whole patch once if it faces inward overall. Winding each triangle to its
	// own vertex normals instead flips slivers inconsistently — the back-facing hole walls in Normal-Debug.
	nrm := make([]math.Vector3, len(pos))
	for i, p := range pos {
		u, v := s.ParamAt(p)
		nrm[i] = s.NormalAt(u, v)
	}
	m := patchMeshFrom(pos, nrm, tris)
	validate.RepairFolds(m, 8) // a curved cap's boundary triangulation can crease; flip the folding diagonals (#585)
	diagnosePatchCoverage(m, accepted)
	return m
}

// patchProjection picks the 2D embedding to ear-clip a curved patch's boundary in. A B-spline
// patch is flattened in its OWN (u,v) parameter space, where the trim loops are a simple polygon;
// the best-fit PLANE projection (used for the analytic surfaces, whose (u,v) can be degenerate at
// a pole/seam) self-intersects for a large freeform patch and makes ear-clipping bail partway,
// tearing the surface (the jagged duct lips). Lifting uses the exact 3D boundary points either way.
func patchProjection(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) ([]math.Point2, [][]math.Point2) {
	if _, isSpline := s.(geom.BSplineSurface); !isSpline {
		return projectToPlane(outer3D, holes3D)
	}
	uv := func(pts []math.Point3) []math.Point2 {
		out := make([]math.Point2, len(pts))
		for i, p := range pts {
			u, v := s.ParamAt(p)
			out[i] = math.P2(math.Scalar(u), math.Scalar(v))
		}
		return out
	}
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		holes2D[i] = uv(h)
	}
	return uv(outer3D), holes2D
}

// projectToPlane flattens the boundary loops onto the outer loop's best-fit plane (Newell
// normal + an in-plane basis) — a non-degenerate 2D embedding of a single-valued patch.
func projectToPlane(outer3D []math.Point3, holes3D [][]math.Point3) ([]math.Point2, [][]math.Point2) {
	n := probe.NewellUnit(outer3D)
	e1, e2 := planeBasis(n)
	o := outer3D[0]
	flat := func(p math.Point3) math.Point2 {
		d := o.VectorTo(p)
		return math.P2(d.Dot(e1), d.Dot(e2))
	}
	outer2D := make([]math.Point2, len(outer3D))
	for i, p := range outer3D {
		outer2D[i] = flat(p)
	}
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		hp := make([]math.Point2, len(h))
		for j, p := range h {
			hp[j] = flat(p)
		}
		holes2D[i] = hp
	}
	return outer2D, holes2D
}

// planeBasis returns two orthonormal in-plane vectors for the plane with unit normal n.
func planeBasis(n math.Vector3) (e1, e2 math.Vector3) {
	seed := math.V3(1, 0, 0)
	if stdmath.Abs(n.X) > 0.9 {
		seed = math.V3(0, 1, 0)
	}
	a, err := math.UnitVector3FromVector(n.Cross(seed))
	if err != nil {
		return math.V3(1, 0, 0), math.V3(0, 1, 0)
	}
	return a.AsVector(), n.Cross(a.AsVector())
}

// triangleFlipped reports whether triangle abc winds against the surface normal at its
// centroid (so it should be reversed to face outward).
func triangleFlipped(s geom.Surface, a, b, c math.Point3) bool {
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	cen := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
	u, v := s.ParamAt(cen)
	return n.Dot(s.NormalAt(u, v)) < 0
}
