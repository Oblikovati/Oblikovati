// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Coaxial sphere ∩ cylinder — the ball-stud family (Oblikovati#2036). A rod whose axis passes through
// a ball's centre meets it in a CIRCLE, and that is the one sphere∩cylinder configuration with a
// closed-form answer: OCCT solves exactly this case analytically in
// IntAna_QuadQuadGeo::Perform(const gp_Cylinder&, const gp_Sphere&) — axis through the centre gives
// IntAna_Circle at ±√(R_s²−R_c²) along the axis — and returns IntAna_NoGeometricSolution for every
// off-axis pair, handing those to the numeric marcher. This file ports that recognizer; the four
// solids its single circle bounds are assembled in curved_coaxial_sphere_rod_build.go.
//
// KIND (ADR-0045): the contact is a 1-D curve, so this is TRANSVERSAL — but it does NOT ride the (u,v)
// arrangement, for the same reason the drill and the boss don't. The arrangement earns its keep when an
// imprint subdivides a periodic band non-trivially; here the imprint is one PLANAR circle and a sphere
// is not a rim-bounded band at all. A circle on a sphere splits it into exactly two caps by
// construction, so there is nothing for a cell classifier to decide: naming which cap survives IS the
// split (compare DrillThroughHole, where "the circle is an inner loop" is likewise the whole answer).
//
// SCOPE — the ONE-CIRCLE configurations, where the rod has one cap strictly INSIDE the ball and one
// strictly OUTSIDE it:
//
//	∪  ball + protruding stub          (the ball stud / rod end)
//	−  ball − rod  = a blind spherical bore with a flat bottom (a socket)
//	−  rod − ball  = the free stub with a spherical dimple in its base
//	∩  the plug: the buried length of the rod, domed by the ball
//
// A rod that passes RIGHT THROUGH the ball (two circles) is deliberately NOT handled. Its result needs
// a spherical ZONE face — the belt between the two circles — and that belt straddles the equator of its
// own band axis, which is exactly the shape kernel/ops has no analytic mesh for (revolution.go's
// sphereZoneAnalytic gates equator-crossing bands out for the same reason, so no revolve emits one
// either): measured, such a face tessellates to a quarter of its true area. Declining keeps the
// faceted-but-honest CSG fallback until that mesh path exists (#2036 follow-up).

// coaxialSphereCircleOffset returns the axial distance from the sphere centre to each circle in which
// an INFINITE cylinder cuts the sphere, when the cylinder's axis passes through that centre. It is the
// port of OCCT IntAna_QuadQuadGeo::Perform(gp_Cylinder, gp_Sphere): the two circles sit at
// centre ± √(R_s²−R_c²)·axis with radius R_c. ok=false reproduces OCCT's two declines — an off-axis
// cylinder (IntAna_NoGeometricSolution: a quartic space curve, not a circle) and a sphere no larger
// than the cylinder (IntAna_Empty) — plus the equal-radius case OCCT reports as a single circle, which
// is an internal TANGENCY and so not a transversal boolean this file can build.
//
// Example: a Ø10 ball and a coaxial Ø6 rod give 4 — the seam circle sits 4 mm off the centre.
func coaxialSphereCircleOffset(sph geom.Sphere, cyl geom.Cylinder) (float64, bool) {
	tol := geom.ResolutionForSize(sph.Radius).Plane()
	if pointAxisDistance(sph.Center, cyl.Origin, cyl.AxisDir.AsVector()) > tol {
		return 0, false // OCCT: the axis misses the sphere centre → IntAna_NoGeometricSolution
	}
	if sph.Radius <= cyl.Radius {
		return 0, false // OCCT: IntAna_Empty — the ball fits inside the rod, no intersection curve
	}
	d := stdmath.Sqrt(sph.Radius*sph.Radius - cyl.Radius*cyl.Radius)
	if d <= tol {
		return 0, false // the two circles have collapsed onto one: a tangential, non-transversal contact
	}
	return d, true
}

// pointAxisDistance is the perpendicular distance from p to the line through origin along unit dir.
func pointAxisDistance(p, origin math.Point3, dir math.Vector3) float64 {
	v := origin.VectorTo(p)
	return float64(v.Sub(dir.Scale(v.Dot(dir))).Length())
}

// axialCoord is p's coordinate along dir measured from from — the axis parameter every extent test in
// this file uses, with the sphere centre as the origin.
func axialCoord(from, p math.Point3, dir math.Vector3) float64 {
	return float64(from.VectorTo(p).Dot(dir))
}

// sphereSolidOf resolves a body as a bare ball: one boundary-less geom.Sphere face, which is what both
// SolidSphere and a 360° revolve of an on-axis semicircle build. It returns the face's lineage too, so
// the trimmed ball keeps its reference key across the boolean (ADR-0043 K1a). ok=false once the sphere
// carries any boundary — an already-trimmed ball is a different, unhandled shape.
func sphereSolidOf(b *topo.Body) (geom.Sphere, topo.Lineage, bool) {
	faces := b.Faces()
	if len(faces) != 1 || len(faces[0].Loops()) != 0 {
		return geom.Sphere{}, topo.Lineage{}, false
	}
	sph, ok := faces[0].Geometry().(geom.Sphere)
	return sph, faces[0].Lineage(), ok
}

// rodSolid is a bare SolidCylinder parsed once: its surface, its two cap centres, and each face's
// lineage, so every face that survives the boolean whole keeps its reference key (ADR-0043 K1a).
type rodSolid struct {
	cyl       geom.Cylinder
	base, top math.Point3 // cap centres, base the lower one along cyl.AxisDir
	wall      topo.Lineage
	baseCap   topo.Lineage
	topCapLin topo.Lineage
}

// rodSolidOf parses a bare cylinder solid — one cylindrical side plus two planar caps — or declines.
func rodSolidOf(b *topo.Body) (rodSolid, bool) {
	faces := facesOfAny(b)
	cyl, base, height, ok := cylinderSolidParams(faces)
	if !ok {
		return rodSolid{}, false
	}
	r := rodSolid{cyl: cyl, base: base, top: base.TranslateBy(cyl.AxisDir.AsVector().Scale(math.Scalar(height)))}
	r.wall, r.baseCap, r.topCapLin = rodFaceLineages(faces, r.base)
	return r, true
}

// rodFaceLineages splits the parsed faces' lineages into the side and the two caps, telling the caps
// apart by which plane sits on the base centre. cylinderSolidParams has already guaranteed exactly one
// cylinder and two planes, so the pair is complete by construction.
func rodFaceLineages(faces []curvedFace, base math.Point3) (wall, baseCap, topCap topo.Lineage) {
	var caps [2]struct {
		lineage topo.Lineage
		dist    float64
	}
	n := 0
	for _, f := range faces {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane {
			wall = f.lineage
			continue
		}
		caps[n].lineage, caps[n].dist = f.lineage, float64(base.VectorTo(pl.Origin).Length())
		n++
	}
	if caps[0].dist <= caps[1].dist {
		return wall, caps[0].lineage, caps[1].lineage
	}
	return wall, caps[1].lineage, caps[0].lineage
}

// coaxialRod is a recognised ball + coaxial rod whose surfaces meet in exactly ONE circle: the rod's
// buried cap lies strictly inside the ball and its free cap strictly outside. Every assembly in
// curved_coaxial_sphere_rod_build.go is three faces drawn from this one frame.
type coaxialRod struct {
	ball, rod *topo.Body
	sph       geom.Sphere
	cyl       geom.Cylinder
	out       math.Vector3 // unit rod axis, buried cap → free cap
	buried    geom.Circle  // the rod rim inside the ball
	free      geom.Circle  // the rod rim outside the ball
	seam      geom.Circle  // cylinder ∩ sphere — OCCT's IntAna_Circle
	ballLin   topo.Lineage
	wallLin   topo.Lineage
	buriedLin topo.Lineage
	freeLin   topo.Lineage
}

// coaxialRodOf resolves either argument order into a coaxialRod, so a caller never has to know which
// operand is the ball. ok=false when the pair is not the single-circle coaxial family.
func coaxialRodOf(a, b *topo.Body) (coaxialRod, bool) {
	if r, ok := coaxialRodOrdered(a, b); ok {
		return r, true
	}
	return coaxialRodOrdered(b, a)
}

// coaxialRodOrdered resolves ball as the sphere and rod as the cylinder, or declines.
func coaxialRodOrdered(ball, rod *topo.Body) (coaxialRod, bool) {
	sph, ballLin, okS := sphereSolidOf(ball)
	rs, okR := rodSolidOf(rod)
	if !okS || !okR {
		return coaxialRod{}, false
	}
	d, okD := coaxialSphereCircleOffset(sph, rs.cyl)
	if !okD {
		return coaxialRod{}, false
	}
	return coaxialRodEnds(coaxialRod{ball: ball, rod: rod, sph: sph, cyl: rs.cyl, ballLin: ballLin,
		wallLin: rs.wall}, rs, d)
}

// coaxialRodEnds picks which rod cap is buried and which is free, and declines every other extent. A
// cap counts as BURIED only when its whole disc is inside the ball (|s| < d, since the disc's far edge
// sits at √(s²+R_c²)) and FREE only when its whole disc is outside (|s| > R_s). A cap landing in the
// band between — the rod stopping part way through the ball's shoulder — leaves an annular cap bounded
// by a plane∩sphere circle, a different solid; a rod clearing BOTH sides is the two-circle case the
// file header rules out.
func coaxialRodEnds(r coaxialRod, rs rodSolid, d float64) (coaxialRod, bool) {
	axis := rs.cyl.AxisDir.AsVector()
	tol := geom.ResolutionForSize(r.sph.Radius).Plane()
	sBase, sTop := axialCoord(r.sph.Center, rs.base, axis), axialCoord(r.sph.Center, rs.top, axis)
	buried := func(s float64) bool { return stdmath.Abs(s) < d-tol }
	free := func(s float64) bool { return stdmath.Abs(s) > r.sph.Radius+tol }
	switch {
	case buried(sBase) && free(sTop):
		return r.withEnds(rs.base, rs.top, rs.baseCap, rs.topCapLin, d)
	case buried(sTop) && free(sBase):
		return r.withEnds(rs.top, rs.base, rs.topCapLin, rs.baseCap, d)
	}
	return coaxialRod{}, false
}

// withEnds completes the frame: the seam circle at OCCT's offset, and one circle per rod rim sharing
// the seam's angle-0 reference direction so all three rims' PointAt(0) line up on a single ruling —
// what lets the wall seams weld.
func (r coaxialRod) withEnds(buried, free math.Point3, buriedLin, freeLin topo.Lineage, d float64) (coaxialRod, bool) {
	r.out = unit(buried.VectorTo(free))
	seam, err := geom.NewCircle(r.sph.Center.TranslateBy(r.out.Scale(math.Scalar(d))), r.out, r.cyl.Radius)
	if err != nil {
		return coaxialRod{}, false
	}
	r.seam, r.buried, r.free = seam, coaxialCircleAt(seam, buried), coaxialCircleAt(seam, free)
	r.buriedLin, r.freeLin = buriedLin, freeLin
	return r, true
}

// coaxialCircleAt copies a circle's whole frame (normal, angle-0 reference, radius) to another centre
// on its axis, so both circles' PointAt(t) agree in azimuth.
func coaxialCircleAt(base geom.Circle, center math.Point3) geom.Circle {
	return geom.Circle{Center: center, Normal: base.Normal, RefDir: base.RefDir, Radius: base.Radius}
}
