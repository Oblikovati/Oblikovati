// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Large spherical-zone tessellation (fixes OCCT blend/simple J2). A sphere cut by ONE plane that
// MISSES the centre keeps a large zone that reaches an enclosed pole: its outer loop is one
// full-circle rim PLUS the parametric seam running down to that pole vertex. sphereCapFan's capAxis
// derives the cap axis from newellUnit(outer3D) — the Newell normal of that seamed loop, which the
// seam+pole samples BIAS — so the fan meshes the wrong (small, ~4600) side instead of the true zone
// (~26815). A sphere∩plane is a surface of revolution about the RIM CIRCLE's own axis regardless of
// the sphere's parameterization (the seam/pole are parametric artifacts), so sphereZoneCapFan rebuilds
// the fan on the rim circle's EXACT geometric normal with the apex taken from the off-rim pole vertex,
// meshing the correct zone. It self-gates to exactly that shape; anything else defers to
// spherePatchMesh, keeping every coplanar-rim cap byte-identical.

// zoneOffPlaneRelTol: a loop vertex counts as off the rim plane (the cap's enclosed pole, not a rim
// sample) when its signed distance from the rim plane exceeds this fraction of the sphere radius
// (model-relative, ADR-0042). A rim vertex sits on the plane (distance 0); a pole is a full radius off.
const zoneOffPlaneRelTol = 1e-3

// zoneFullCircleTol: a swept edge is a full circle when its sweep is within this of 2π (radians;
// scale-free, an angle).
const zoneFullCircleTol = 1e-6

// sphereZoneCapFan meshes a sphere face whose outer boundary is one full-circle rim plus a meridian
// seam to an enclosed POLE off the rim plane — the large zone one off-centre plane cuts from a sphere
// (OCCT blend/simple J2: psphere -90..45, the kept part reaches the south pole). The fan is built on
// the rim circle's exact normal and the pole vertex, so it sweeps the true zone where newellUnit is
// biased. ok=false unless that exact shape holds, so the caller defers to spherePatchMesh.
//
// Example: a sphere cut by z=+35.36 (R=50) keeps the zone from the south pole up to that rim → a
// watertight fan of area 2πR·h from rim to pole, not the small north cap capAxis would have meshed.
func sphereZoneCapFan(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	sph, ok := s.(geom.Sphere)
	if !ok {
		return nil, false
	}
	rim, axis, ok := zoneRimAxis(f, sph, q)
	if !ok || len(rim) < 3 {
		return nil, false
	}
	return buildSphereCap(sph, rim, axis, q), true
}

// zoneRimAxis returns the rim's shared discretization (fan row 0) and the unit cap axis (centre →
// enclosed pole) for a pole-reaching sphere zone, or ok=false when the outer loop is not one
// full-circle rim plus a single off-plane pole vertex.
func zoneRimAxis(f *topo.Face, sph geom.Sphere, q Quality) (rim []math.Point3, axis math.Vector3, ok bool) {
	loop := outerLoopOf(f)
	if loop == nil {
		return nil, math.Vector3{}, false
	}
	rimEdge, rimCenter, rimNormal, ok := singleFullCircleRim(loop)
	if !ok {
		return nil, math.Vector3{}, false
	}
	n, err := math.UnitVector3FromVector(rimNormal)
	if err != nil {
		return nil, math.Vector3{}, false
	}
	pole, ok := lonePoleVertex(loop, rimCenter, n.AsVector(), sph.Radius)
	if !ok {
		return nil, math.Vector3{}, false
	}
	return rimRing(rimEdge, q), poleAxis(sph.Center, pole, n.AsVector()), true
}

// singleFullCircleRim returns the loop's rim edge, its circle centre and plane normal when the loop
// has EXACTLY ONE full-circle edge (a plane∩sphere rim), else ok=false. Zero (a multi-arc box cut) or
// several (a lens/two-plane patch) both defer to spherePatchMesh.
func singleFullCircleRim(l *topo.Loop) (rim *topo.Edge, center math.Point3, normal math.Vector3, ok bool) {
	count := 0
	for _, u := range l.EdgeUses() {
		if c, nrm, isCircle := fullCircleRimGeom(u.Edge()); isCircle {
			count++
			rim, center, normal = u.Edge(), c, nrm
		}
	}
	if count != 1 {
		return nil, math.Point3{}, math.Vector3{}, false
	}
	return rim, center, normal, true
}

// fullCircleRimGeom returns an edge's circle centre and plane normal when the edge is one closed full
// circle — a geom.Circle, or a geom.Arc3d sweeping ~2π back to its own start vertex (an imported rim is
// a full-sweep Arc3d). A plane∩sphere rim is always circular, so only circular geometry qualifies.
// The closed-full-circle test itself is isClosedCircularEdge (fillet_rim.go) — the SAME predicate the
// rim-fillet pick gate uses, so a full-sweep Arc3d classifies identically here and there.
func fullCircleRimGeom(e *topo.Edge) (center math.Point3, normal math.Vector3, ok bool) {
	if !isClosedCircularEdge(e) {
		return math.Point3{}, math.Vector3{}, false
	}
	switch c := e.Geometry().(type) {
	case geom.Circle:
		return c.Center, c.Normal.AsVector(), true
	case geom.Arc3d:
		return c.Center, c.Normal.AsVector(), true
	}
	return math.Point3{}, math.Vector3{}, false
}

// lonePoleVertex returns the loop's single vertex lying off the rim plane (the cap's enclosed pole).
// ok=false for zero off-plane vertices (a coplanar rim — sphereCapFan's clean cap) or more than one (a
// plane through the sphere axis, or a multi-plane patch — spherePatchMesh), so only the true zone fans.
func lonePoleVertex(l *topo.Loop, rimCenter math.Point3, normal math.Vector3, radius float64) (math.Point3, bool) {
	offTol := zoneOffPlaneRelTol * radius
	var pole math.Point3
	found := 0
	seen := map[*topo.Vertex]bool{}
	for _, u := range l.EdgeUses() {
		for _, v := range u.Edge().Vertices() {
			if seen[v] {
				continue
			}
			seen[v] = true
			if stdmath.Abs(float64(rimCenter.VectorTo(v.Point()).Dot(normal))) > offTol {
				pole, found = v.Point(), found+1
			}
		}
	}
	if found != 1 {
		return math.Point3{}, false
	}
	return pole, true
}

// poleAxis returns the unit cap axis from the sphere centre toward the enclosed pole: the rim normal
// flipped to the side the pole vertex sits on (σ = sign((pole−centre)·n̂)). For an axis-perpendicular
// cut the pole is on the axis, but rim-normal + pole-sign is the invariant that also handles oblique cuts.
func poleAxis(center, pole math.Point3, normal math.Vector3) math.Vector3 {
	if center.VectorTo(pole).Dot(normal) < 0 {
		return normal.Scale(-1)
	}
	return normal
}

// rimRing returns the rim circle's shared edge discretization (the SAME samples the adjacent cap-plane
// lid welds to — never re-sampled) as an OPEN ring, dropping the closing duplicate a full-circle edge's
// start==end vertex leaves so the fan's wrapped columns are not degenerate.
func rimRing(rim *topo.Edge, q Quality) []math.Point3 {
	pts := discretizeEdge(rim, q)
	if n := len(pts); n > 1 && pts[0].DistanceTo(pts[n-1]) < ResolutionForPoints(pts).Weld() {
		return pts[:n-1]
	}
	return pts
}
