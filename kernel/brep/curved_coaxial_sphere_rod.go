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
// SCOPE — every extent of the rod, because the classification is one-dimensional (see
// curved_coaxial_sphere_rod_spans.go) and so needs no case table:
//
//	the rod ENDS inside the ball          ∪ the ball stud · − a blind bore or a dimpled stub · ∩ the plug
//	the rod passes THROUGH                ∪ a ball on an axle · − a bead (genus 1) or two severed stubs
//	a cap stops in the ball's SHOULDER    the ball survives in two pieces and the rod's end cap as an
//	                                      ANNULUS, since a plane∩sphere circle joins the seam circle
//	the rod is BURIED entirely            ∪ the untouched ball · − the ball with the rod as an interior void
//
// What still declines is only what would produce a face of no size: a rod cap sitting ON a seam station
// (a zero-width band) or ON a pole (a zero-radius disc), and a rod that never reaches the ball. Those
// keep the guarded fallback.

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

// coaxialRod is a recognised ball + coaxial rod, reduced to the axial numbers every result is built
// from: the rod's two cap stations measured from the ball centre along `out`, and OCCT's seam offset.
// `out` runs from the LO cap to the HI one, so sLo < sHi always.
type coaxialRod struct {
	ball, rod  *topo.Body
	sph        geom.Sphere
	cyl        geom.Cylinder
	out        math.Vector3 // unit rod axis, lo cap → hi cap
	sLo, sHi   float64      // the rod's cap stations, from the ball centre along out
	seamOffset float64      // ±this is where the rod's wall crosses the ball — OCCT's √(R²−r_c²)
	frame      geom.Circle  // the angle-0 reference every rim copies, so the wall seams weld
	ballLin    topo.Lineage
	wallLin    topo.Lineage
	loLin      topo.Lineage
	hiLin      topo.Lineage
}

// coaxialRodOf resolves either argument order into a coaxialRod, so a caller never has to know which
// operand is the ball. ok=false when the pair is not the coaxial family.
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
	return coaxialRodEnds(coaxialRod{ball: ball, rod: rod, sph: sph, cyl: rs.cyl, seamOffset: d,
		ballLin: ballLin, wallLin: rs.wall}, rs)
}

// coaxialRodEnds orients the rod lo→hi and records its cap stations, declining the extents whose result
// would carry a degenerate face. A cap sitting ON a seam station leaves a zero-width band; a cap ON a
// pole leaves a zero-radius disc; a rod that does not reach the ball at all has no boolean to do here.
// Everything else is buildable — the rod may end inside the ball, clear it entirely, or stop part way
// through its shoulder, and the span arithmetic in curved_coaxial_sphere_rod_spans.go handles all of
// them without a case table.
func coaxialRodEnds(r coaxialRod, rs rodSolid) (coaxialRod, bool) {
	axis := rs.cyl.AxisDir.AsVector()
	sBase := axialCoord(r.sph.Center, rs.base, axis)
	sTop := axialCoord(r.sph.Center, rs.top, axis)
	if sBase > sTop {
		return r.withEnds(rs.top, rs.base, rs.topCapLin, rs.baseCap, sTop, sBase)
	}
	return r.withEnds(rs.base, rs.top, rs.baseCap, rs.topCapLin, sBase, sTop)
}

// withEnds completes the frame and runs the degeneracy gate.
func (r coaxialRod) withEnds(lo, hi math.Point3, loLin, hiLin topo.Lineage, sLo, sHi float64) (coaxialRod, bool) {
	r.out = unit(lo.VectorTo(hi))
	r.sLo, r.sHi, r.loLin, r.hiLin = sLo, sHi, loLin, hiLin
	frame, err := geom.NewCircle(r.sph.Center, r.out, r.cyl.Radius)
	if err != nil || !r.capStationsClear() {
		return coaxialRod{}, false
	}
	r.frame = frame
	return r, true
}

// capStationsClear reports whether both rod caps sit clear of every station that would make a face
// degenerate — the seam stations ±d and the poles ±R — and whether the rod reaches the ball at all.
func (r coaxialRod) capStationsClear() bool {
	tol := r.stationTol()
	R := r.sph.Radius
	if r.sHi <= -R+tol || r.sLo >= R-tol {
		return false // the rod stops short of the ball entirely: no boolean of this family
	}
	for _, s := range []float64{r.sLo, r.sHi} {
		for _, bad := range []float64{-r.seamOffset, r.seamOffset, -R, R} {
			if stdmath.Abs(s-bad) <= tol {
				return false
			}
		}
	}
	return true
}

// circleAt is a rim of the result: the circle at the given axial station and radius, carrying the
// frame's angle-0 reference so every rim's PointAt(0) lands on one ruling — what lets the wall seams
// weld. Its normal is always +out, so a "forward" walk means the same thing on every rim.
func (r coaxialRod) circleAt(station, radius float64) geom.Circle {
	return geom.Circle{Center: r.stationPoint(station), Normal: r.frame.Normal,
		RefDir: r.frame.RefDir, Radius: radius}
}
