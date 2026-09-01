// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Spherical ZONE (belt) tessellation — Oblikovati#2061. A sphere bounded by TWO coaxial full-circle
// rims is a belt: the shape a rod passing right through a ball leaves of the ball, and the shape a
// revolved meridian arc with both endpoints off the axis sweeps. Neither existing sphere mesher covers
// it — sphereCapFan wants ONE planar rim, sphereZoneCapFan wants one rim plus an enclosed pole,
// sphereSeamedCapFan wants a seam ending at a pole — so the face fell through to spherePatchMesh's
// gnomonic CDT. That chart is centred on the patch and covers less than a hemisphere, which is fine
// for a belt that stays on one side of its own equator and hopeless for one that straddles it: a
// R=0.5 belt between y=±0.4 (true area 2.513) meshed to 0.617 through the seamed loop and to 2.809
// through outer+inner loops, the inner rim silently ignored. revolution.go's sphereZoneAnalytic gated
// equator-crossing bands out for the same reason.
//
// The fix is the same one sphereCapFan applies to a cap: sweep LATITUDE RINGS about the rims' own
// axis, which is a surface of revolution regardless of how geom.Sphere happens to be parameterised
// (its seam and poles are artefacts fixed to world +Z). Both rims are kept at their own edge
// discretization, so the belt welds to whatever shares them.

// sphereZoneBandFan meshes a sphere face bounded by two coaxial full-circle rims by sweeping latitude
// rings between them. ok=false unless that exact shape holds, so every other sphere face keeps its
// existing path.
//
// Example: a ball with an axle bored right through keeps a belt whose area is exactly 2πR·h, where h
// is the axial distance between the two rim planes.
func sphereZoneBandFan(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	sph, ok := s.(geom.Sphere)
	if !ok {
		return nil, false
	}
	near, far, fr, ok := zoneBandRims(f, sph, q)
	if !ok {
		return nil, false
	}
	return buildSphereZoneBand(sph, near, far, fr, q), true
}

// zoneRing is one rim of the belt: its own edge discretization ordered by azimuth about the band axis,
// those azimuths, and the polar angle its points sit at from the axis.
type zoneRing struct {
	pts   []math.Point3
	ang   []float64
	polar float64
}

// zoneFrame is the belt's own orthonormal frame: the rims' shared axis plus a reference pair spanning
// the plane it is normal to, so every point has a well-defined azimuth about THAT axis rather than
// about geom.Sphere's world-Z parameterisation.
type zoneFrame struct {
	axis, ref, bin math.Vector3
}

// zoneBandRims recognises the belt: exactly two rim edges (a doubled seam edge, which some producers
// use to bridge the rims into one loop, borders only this face and is skipped), each a full circle on
// the sphere, with parallel plane normals and distinct axial stations. The rims come back ordered
// near-to-far in polar angle from the frame's axis.
func zoneBandRims(f *topo.Face, sph geom.Sphere, q Quality) (near, far zoneRing, fr zoneFrame, ok bool) {
	rims := zoneRimRings(f, q)
	if len(rims) != 2 {
		return near, far, fr, false
	}
	axis, ok := parallelRimAxis(sph, rims)
	if !ok {
		return near, far, fr, false
	}
	fr = newZoneFrame(axis, sph, rims[0][0])
	a, b := newZoneRing(sph, rims[0], fr), newZoneRing(sph, rims[1], fr)
	if stdmath.Abs(a.polar-b.polar) < zoneRimSeparationTol {
		return near, far, fr, false // the two rims coincide: no belt to sweep
	}
	if a.polar > b.polar {
		a, b = b, a
	}
	return a, b, fr, true
}

// zoneRimSeparationTol is the least polar-angle separation (radians) two rims must have to bound a
// belt. Scale-free: it is an angle on the unit sphere, so it means the same at any model size.
const zoneRimSeparationTol = 1e-9

// zoneRimRings returns the discretization of each rim edge of the face — every edge except a seam,
// which is used twice by this same face and bridges the rims rather than bounding the belt. A face
// carrying any OPEN (non-closed) edge is not a two-rim belt and returns nothing.
func zoneRimRings(f *topo.Face, q Quality) [][]math.Point3 {
	seam := seamEdgesOf(f)
	var rims [][]math.Point3
	for _, e := range f.Edges() {
		if seam[e] {
			continue
		}
		if e.StartVertex() != e.EndVertex() {
			return nil // an open rim: an arc-bounded patch, not a belt
		}
		if ring := dropClosingDup(discretizeEdge(e, q)); len(ring) >= 3 {
			rims = append(rims, ring)
		}
	}
	return rims
}

// parallelRimAxis returns the belt's axis when both rims lie in parallel planes — the condition for
// them to be coaxial circles on the sphere, since a circle on a sphere is always centred on the
// diameter normal to its own plane. ok=false when the normals diverge (two obliquely cut rims bound
// a lune, not a belt).
func parallelRimAxis(sph geom.Sphere, rims [][]math.Point3) (math.Vector3, bool) {
	a, b := probe.NewellUnit(rims[0]), probe.NewellUnit(rims[1])
	if !rimIsCircle(sph, rims[0], a) || !rimIsCircle(sph, rims[1], b) {
		return math.Vector3{}, false
	}
	if stdmath.Abs(float64(a.Cross(b).Length())) > zoneAxisParallelTol {
		return math.Vector3{}, false
	}
	return a, true
}

// zoneAxisParallelTol is how far the two rim normals may diverge (the sine of their angle) and still
// count as coaxial. Scale-free: a sine, not a length.
const zoneAxisParallelTol = 1e-7

// newZoneFrame builds the belt's frame from the axis, taking the reference direction from a rim point
// so the azimuths of that rim start at a real vertex rather than an arbitrary place.
func newZoneFrame(axis math.Vector3, sph geom.Sphere, seed math.Point3) zoneFrame {
	ref := meridianPerp(sph, seed, axis)
	return zoneFrame{axis: axis, ref: ref, bin: axis.Cross(ref)}
}

// azimuth is p's angle about the belt axis, in [0, 2π).
func (fr zoneFrame) azimuth(sph geom.Sphere, p math.Point3) float64 {
	d := sph.Center.VectorTo(p)
	a := stdmath.Atan2(float64(d.Dot(fr.bin)), float64(d.Dot(fr.ref)))
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}

// pointAt is the sphere point at the given azimuth and polar angle from the belt axis; its outward
// unit normal is the same direction.
func (fr zoneFrame) pointAt(sph geom.Sphere, azimuth, polar float64) (math.Point3, math.Vector3) {
	ca, sa := stdmath.Cos(azimuth), stdmath.Sin(azimuth)
	perp := fr.ref.Scale(math.Scalar(ca)).Add(fr.bin.Scale(math.Scalar(sa)))
	dir := capNormal(fr.axis, perp, polar)
	return sph.Center.TranslateBy(dir.Scale(math.Scalar(sph.Radius))), dir
}

// newZoneRing orders a rim's points by azimuth about the belt axis and records its polar angle, so
// the two rims and every interior ring share one angular ordering for the stitcher.
func newZoneRing(sph geom.Sphere, pts []math.Point3, fr zoneFrame) zoneRing {
	ordered := append([]math.Point3(nil), pts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return fr.azimuth(sph, ordered[i]) < fr.azimuth(sph, ordered[j])
	})
	ang := make([]float64, len(ordered))
	for i, p := range ordered {
		ang[i] = fr.azimuth(sph, p)
	}
	return zoneRing{pts: ordered, ang: ang, polar: rimPolarAngle(sph, ordered, fr.axis)}
}

// buildSphereZoneBand sweeps latitude rings from the near rim to the far one. Row 0 and the last row
// are the rims' OWN discretizations, kept exactly so the belt welds to its neighbours; the interior
// rows sit at the near rim's azimuths, which makes every strip but the last an aligned quad grid.
func buildSphereZoneBand(sph geom.Sphere, near, far zoneRing, fr zoneFrame, q Quality) *Mesh {
	m := &Mesh{}
	rows := []bandRow{addZoneRow(m, sph, near.pts, near.ang)}
	span := far.polar - near.polar
	for k := 1; k < zoneRingCount(span, q); k++ {
		polar := near.polar + span*float64(k)/float64(zoneRingCount(span, q))
		rows = append(rows, addZoneRing(m, sph, fr, near.ang, polar))
	}
	rows = append(rows, addZoneRow(m, sph, far.pts, far.ang))
	for i := 0; i+1 < len(rows); i++ {
		stitchBandRows(m, rows[i], rows[i+1])
	}
	return m
}

// zoneRingCount picks how many latitude steps the belt is swept in, so each step stays inside the
// quality's angle tolerance — the same rule capRingCount applies to a cap.
func zoneRingCount(span float64, q Quality) int {
	n := int(stdmath.Ceil(stdmath.Abs(span) / q.AngleTol()))
	if n < 1 {
		return 1
	}
	return n
}

// addZoneRow adds a rim's own points as a row, with the exact outward radial normal at each.
func addZoneRow(m *Mesh, sph geom.Sphere, pts []math.Point3, ang []float64) bandRow {
	idx := make([]int, len(pts))
	for i, p := range pts {
		idx[i] = m.AddVertex(p, sph.Center.VectorTo(p).Scale(math.Scalar(1/sph.Radius)))
	}
	return bandRow{idx: idx, ang: ang}
}

// addZoneRing adds an interior latitude ring at the given polar angle, on the supplied azimuths.
func addZoneRing(m *Mesh, sph geom.Sphere, fr zoneFrame, ang []float64, polar float64) bandRow {
	idx := make([]int, len(ang))
	for i, a := range ang {
		p, n := fr.pointAt(sph, a, polar)
		idx[i] = m.AddVertex(p, n)
	}
	return bandRow{idx: idx, ang: ang}
}
