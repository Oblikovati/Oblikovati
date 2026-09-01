// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

// Sphere-cap tessellation (M2 Phase 1, Oblikovati/Oblikovati#1334). A sphere face trimmed by ONE
// planar circle — the cap a single plane cuts off (a hemisphere, a drilled spherical seat, the
// curved face of sphere∩half-space) — used to mesh FLAT: meshSeamCrossingFace routed it to the
// best-fit-plane CDT, whose only vertices are the coplanar rim circle, so with no interior points
// the "cap" was a flat disk in the cut plane (zero bulge → zero cap volume, wrong mass/render/
// export for every spherical cap, boolean or not). sphereCapFan instead sweeps latitude rings
// about the cut axis from the rim to the enclosed pole, so the cap carries its true spherical
// curvature. Like coneApexFan it claims only the single-circle-boundary case; an arc-bounded
// sphere patch (a box cut) keeps the existing path.

// SphereCapFan meshes a sphere face whose outer boundary is a single planar circle by fanning
// latitude rings from that rim to the enclosed pole. ok=false unless the surface is a sphere and
// the boundary is one circle (every rim point coplanar) — any other trim defers to the caller.
// A face carrying HOLES is not a cap: the fan sweeps the rim straight to the pole and would pave
// right over them, so it declines and lets a mesher that can carry the hole take the face (a
// coaxial circular hole makes it a BELT, sphereZoneBandFan; anything else the gnomonic CDT).
//
// Example: a unit sphere cut by z=0, the lower face's rim is the equator → a watertight, true
// hemisphere cap (tessellated volume → 2/3 πr³).
func SphereCapFan(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	sph, ok := s.(geom.Sphere)
	if !ok || len(outer3D) < 3 || len(holes3D) > 0 {
		return nil, false
	}
	axis, ok := capAxis(sph, outer3D)
	if !ok {
		return nil, false
	}
	return buildSphereCap(sph, outer3D, axis, q), true
}

// capAxis returns the unit direction from the sphere centre toward the pole the cap encloses (the
// rim circle's plane normal, flipped to the face's material side). ok=false when the rim is not a
// single planar circle on the sphere (an arc-bounded patch), which sphereCapFan does not handle.
func capAxis(sph geom.Sphere, outer3D []math.Point3) (math.Vector3, bool) {
	a := probe.NewellUnit(outer3D) // rim plane normal (unit)
	if !rimIsCircle(sph, outer3D, a) {
		return math.Vector3{}, false
	}
	// Interior surface direction at the first rim point: outward radial × loop tangent. The cap
	// pole lies on whichever side of the rim plane that interior direction points to.
	outward := sph.Center.VectorTo(outer3D[0])
	tangent := outer3D[0].VectorTo(outer3D[1])
	if outward.Cross(tangent).Dot(a) < 0 {
		a = a.Scale(-1)
	}
	return a, true
}

// rimCoplanarTol is the distance a rim sample may deviate from the rim's best-fit plane and still
// count as one circle. A genuine plane-cut circle's samples are coplanar to edge-tessellation
// noise (~chord error); an arc-bounded patch's far arcs leave the plane by feature scale.
const rimCoplanarTol = 1e-6

// rimIsCircle reports whether every boundary sample lies in the single plane (normal a through the
// rim centroid) — i.e. the boundary is one circle, not a chain of arcs from several cutting planes.
func rimIsCircle(sph geom.Sphere, outer3D []math.Point3, a math.Vector3) bool {
	if a.LengthSquared() == 0 {
		return false
	}
	c0 := centroidOf(outer3D)
	for _, p := range outer3D {
		if stdmath.Abs(float64(c0.VectorTo(p).Dot(a))) > rimCoplanarTol*sph.Radius {
			return false
		}
	}
	return true
}

// buildSphereCap sweeps the rim ring (outer3D, kept exactly so it welds to the cap's planar lid)
// up to the pole at centre+R·axis, emitting one latitude ring per angular step.
func buildSphereCap(sph geom.Sphere, outer3D []math.Point3, axis math.Vector3, q Quality) *Mesh {
	r := sph.Radius
	alpha := rimPolarAngle(sph, outer3D, axis) // rim's angle from the cap axis
	rows := capRingCount(alpha, q)
	m := &Mesh{}
	grid := capRingVertices(m, sph, outer3D, axis, alpha, rows)
	pole := m.AddVertex(sph.Center.TranslateBy(axis.Scale(math.Scalar(r))), axis)
	emitCapGrid(m, grid, pole)
	return m
}

// rimPolarAngle returns the rim's mean polar angle from the cap axis (acos of the rim direction's
// axial component), the angle the rings sweep down from to reach the pole (angle 0).
func rimPolarAngle(sph geom.Sphere, outer3D []math.Point3, axis math.Vector3) float64 {
	sum := 0.0
	for _, p := range outer3D {
		dir := sph.Center.VectorTo(p).Scale(math.Scalar(1 / sph.Radius))
		sum += stdmath.Acos(math.Clamp(float64(dir.Dot(axis)), -1, 1))
	}
	return sum / float64(len(outer3D))
}

// capRingCount picks the number of latitude rings so each ring's angular step stays within the
// angle tolerance (so the meridian chord error matches the rim's own faceting), at least one ring.
func capRingCount(alpha float64, q Quality) int {
	n := int(stdmath.Ceil(alpha / q.AngleTol()))
	if n < 1 {
		return 1
	}
	return n
}

// capRingVertices builds the cols×rows vertex grid: row 0 is the exact rim (outer3D), rows step the
// polar angle linearly from the rim angle down toward 0 (the pole, added separately). Each column
// keeps its own azimuth, so a column is a meridian from rim to pole.
func capRingVertices(m *Mesh, sph geom.Sphere, outer3D []math.Point3, axis math.Vector3, alpha float64, rows int) [][]int {
	grid := make([][]int, len(outer3D))
	for i, p := range outer3D {
		perp := MeridianPerp(sph, p, axis)
		grid[i] = make([]int, rows)
		grid[i][0] = m.AddVertex(p, sph.Center.VectorTo(p).Scale(math.Scalar(1/sph.Radius)))
		for k := 1; k < rows; k++ {
			ang := alpha * float64(rows-k) / float64(rows) // rows→pole as k grows; k=0 is the rim
			grid[i][k] = m.AddVertex(capPoint(sph, axis, perp, ang), capNormal(axis, perp, ang))
		}
	}
	return grid
}

// MeridianPerp returns the unit in-plane direction (perpendicular to the cap axis) of rim point p's
// meridian, so capPoint can place that column's rings at the rim's own azimuth.
func MeridianPerp(sph geom.Sphere, p math.Point3, axis math.Vector3) math.Vector3 {
	dir := sph.Center.VectorTo(p)
	perp := dir.Add(axis.Scale(math.Scalar(-dir.Dot(axis))))
	if u, err := math.UnitVector3FromVector(perp); err == nil {
		return u.AsVector()
	}
	return perp
}

// capPoint is the sphere point at polar angle ang from the cap axis along the meridian (axis, perp).
func capPoint(sph geom.Sphere, axis, perp math.Vector3, ang float64) math.Point3 {
	cos, sin := stdmath.Cos(ang), stdmath.Sin(ang)
	dir := axis.Scale(math.Scalar(cos)).Add(perp.Scale(math.Scalar(sin)))
	return sph.Center.TranslateBy(dir.Scale(math.Scalar(sph.Radius)))
}

// capNormal is the outward unit normal at the cap point — the radial direction equals (axis, perp)
// at angle ang.
func capNormal(axis, perp math.Vector3, ang float64) math.Vector3 {
	cos, sin := stdmath.Cos(ang), stdmath.Sin(ang)
	return axis.Scale(math.Scalar(cos)).Add(perp.Scale(math.Scalar(sin)))
}

// emitCapGrid triangulates the rim→pole grid (columns wrap, since the rim is a closed loop) and
// fans the last ring to the shared pole vertex, each triangle wound to agree with its normals.
func emitCapGrid(m *Mesh, grid [][]int, pole int) {
	cols, rows := len(grid), len(grid[0])
	for i := range cols {
		ni := (i + 1) % cols
		for k := 0; k+1 < rows; k++ {
			emitClosedTri(m, grid[i][k], grid[ni][k], grid[ni][k+1])
			emitClosedTri(m, grid[i][k], grid[ni][k+1], grid[i][k+1])
		}
		emitClosedTri(m, grid[i][rows-1], grid[ni][rows-1], pole)
	}
}

// centroidOf returns the average of the points.
func centroidOf(pts []math.Point3) math.Point3 {
	var x, y, z float64
	for _, p := range pts {
		x, y, z = x+float64(p.X), y+float64(p.Y), z+float64(p.Z)
	}
	n := float64(len(pts))
	return math.P3(math.Scalar(x/n), math.Scalar(y/n), math.Scalar(z/n))
}
