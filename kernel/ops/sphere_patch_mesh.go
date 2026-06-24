// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Multi-arc sphere-patch tessellation (M2 Phase 1, Oblikovati/Oblikovati#1334). A sphere face bounded
// by SEVERAL arcs — the curved face a box (or any multi-plane) cut leaves on a sphere — meshed wrong
// through the lat/long (u,v) path: near a pole every column collapses, so the constrained-Delaunay in
// (u,v) folds and over/under-fills (a hand-built octant came out ~38% over its analytic πR³/6). The
// fix is to triangulate in a GNOMONIC chart instead: central projection from the sphere centre onto
// the tangent plane at the patch's mean direction. There great circles map to straight lines and there
// is no pole/seam degeneracy within a hemisphere, so interior Steiner points lift back to the sphere
// cleanly and the patch carries its true curvature. Like sphereCapFan/coneApexFan it self-gates — a
// patch that exceeds a hemisphere (where the gnomonic projection blows up) defers to the existing path.

// minPatchAxisDot requires every boundary point to lie within ~81° of the chart axis (dir·axis ≥ this),
// i.e. inside a hemisphere, so the gnomonic projection stays well-conditioned (the projection diverges
// as a point approaches 90°). A larger patch defers to the caller.
const minPatchAxisDot = 0.15

// spherePatchMesh meshes a multi-arc sphere patch in a gnomonic chart. ok=false unless the surface is a
// sphere whose boundary lies within a hemisphere — then the caller keeps its existing path.
func spherePatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	sph, ok := s.(geom.Sphere)
	if !ok || len(outer3D) < 3 {
		return nil, false
	}
	axis, ok := patchAxis(sph, outer3D, holes3D)
	if !ok {
		return nil, false
	}
	e1, e2 := planeBasis(axis)
	c := gnomonicChart{sph: sph, axis: axis, e1: e1, e2: e2}
	uv, pos, nrm, loops, loops2D := c.projectBoundary(outer3D, holes3D)
	c.addInteriorNodes(&uv, &pos, &nrm, loops2D, q)
	tris := constrainedDelaunay(uv, loops)
	if len(tris) == 0 {
		return nil, false
	}
	m := patchMeshFrom(pos, nrm, tris)
	repairFolds(m, 8)
	return m, true
}

// gnomonicChart is the central projection of a sphere onto the tangent plane at axis: a point of unit
// direction u maps to ((R/(u·axis))·u − R·axis) resolved in the orthonormal in-plane basis (e1, e2).
type gnomonicChart struct {
	sph    geom.Sphere
	axis   math.Vector3
	e1, e2 math.Vector3
}

// patchAxis returns the chart axis (the boundary's mean direction from the sphere centre) and whether
// every boundary point lies within a hemisphere of it (so the gnomonic projection is well-conditioned).
func patchAxis(sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3) (math.Vector3, bool) {
	var sum math.Vector3
	pts := append(append([]math.Point3{}, outer3D...), flattenLoops(holes3D)...)
	for _, p := range pts {
		sum = sum.Add(sphereDir(sph, p))
	}
	axis, err := math.UnitVector3FromVector(sum)
	if err != nil {
		return math.Vector3{}, false
	}
	a := axis.AsVector()
	for _, p := range pts {
		if float64(sphereDir(sph, p).Dot(a)) < minPatchAxisDot {
			return math.Vector3{}, false
		}
	}
	return a, true
}

// projectBoundary maps every boundary loop to the gnomonic plane, keeping the EXACT 3D points and radial
// normals (so the patch welds to its neighbour faces at the shared edge samples). It returns the parallel
// 2D/3D/normal arrays, the loop index sequences for the CDT, and the loops as math.Point2 for clearOfTrim.
func (c gnomonicChart) projectBoundary(outer3D []math.Point3, holes3D [][]math.Point3) (uv [][2]float64, pos []math.Point3, nrm []math.Vector3, loops [][]int, loops2D [][]math.Point2) {
	for _, loop := range append([][]math.Point3{outer3D}, holes3D...) {
		idx := make([]int, len(loop))
		ring2D := make([]math.Point2, len(loop))
		for i, p := range loop {
			g := c.project(p)
			idx[i] = len(uv)
			uv = append(uv, g)
			pos = append(pos, p)
			nrm = append(nrm, sphereDir(c.sph, p))
			ring2D[i] = math.P2(math.Scalar(g[0]), math.Scalar(g[1]))
		}
		loops = append(loops, idx)
		loops2D = append(loops2D, ring2D)
	}
	return uv, pos, nrm, loops, loops2D
}

// project maps a 3D sphere point to its gnomonic (e1, e2) coordinates.
func (c gnomonicChart) project(p math.Point3) [2]float64 {
	u := sphereDir(c.sph, p)
	s := float64(u.Dot(c.axis))
	local := u.Scale(math.Scalar(c.sph.Radius / s)).Add(c.axis.Scale(math.Scalar(-c.sph.Radius)))
	return [2]float64{float64(local.Dot(c.e1)), float64(local.Dot(c.e2))}
}

// lift inverts project: a gnomonic point becomes the sphere point along the direction axis·R + a·e1 + b·e2,
// returning that point and its outward radial normal.
func (c gnomonicChart) lift(a, b float64) (math.Point3, math.Vector3) {
	planeDir := c.axis.Scale(math.Scalar(c.sph.Radius)).Add(c.e1.Scale(math.Scalar(a))).Add(c.e2.Scale(math.Scalar(b)))
	dir := planeDir.Scale(math.Scalar(1 / float64(planeDir.Length())))
	return c.sph.Center.TranslateBy(dir.Scale(math.Scalar(c.sph.Radius))), dir
}

// patchGridCap bounds the interior Steiner grid per axis so a large/fine patch cannot explode the node
// count; beyond it the chord tolerance is met by the cap density rather than the exact spacing.
const patchGridCap = 60

// addInteriorNodes lays a grid of Steiner points across the gnomonic bbox at a spacing whose lifted
// chord error meets the tolerance, keeping only points strictly inside the trim, and lifts each to the
// sphere. Interior points carry the patch's curvature (a boundary-only triangulation would chord flat).
func (c gnomonicChart) addInteriorNodes(uv *[][2]float64, pos *[]math.Point3, nrm *[]math.Vector3, loops2D [][]math.Point2, q Quality) {
	umin, umax, vmin, vmax := bounds2D(loops2D[0])
	h := stdmath.Sqrt(2 * c.sph.Radius * q.tol()) // chord error R(1−cos(h/R)) ≈ h²/2R ≤ tol
	if h <= 0 {
		return
	}
	nu, nv := gridSteps(umax-umin, h), gridSteps(vmax-vmin, h)
	holes2D := loops2D[1:]
	for i := 1; i < nu; i++ {
		for j := 1; j < nv; j++ {
			p := [2]float64{umin + (umax-umin)*float64(i)/float64(nu), vmin + (vmax-vmin)*float64(j)/float64(nv)}
			if !clearOfTrim(loops2D[0], holes2D, p, h*0.25) {
				continue
			}
			pt, n := c.lift(p[0], p[1])
			*uv = append(*uv, p)
			*pos = append(*pos, pt)
			*nrm = append(*nrm, n)
		}
	}
}

// gridSteps returns the number of grid intervals to span extent at spacing h, clamped to patchGridCap.
func gridSteps(extent, h float64) int {
	n := int(extent / h)
	if n > patchGridCap {
		return patchGridCap
	}
	return n
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
