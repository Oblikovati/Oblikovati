// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Crossing-cylinder intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). The split/classify/stitch
// stage following the imprint: it assembles the exact A∩B of two crossing cylinders — a rod through a
// fatter cylinder — straight from the traced imprint loops, with no general face splitting. The
// intersection's boundary is just three analytic faces: the rod's wall BAND inside the fat cylinder
// (between the two loops) plus the two LENS caps of the fat wall the rod pokes through (one per loop).
// Classification is analytic — the rod is the cylinder both loops fully encircle — so no tessellation
// winding number (the H3 oracle) is needed. Scope: the clean thin-through-fat case the imprint traces as
// two clean loops; equal-radius (Steinmetz) and partial penetrations defer to the caller's fallback.

// crossLin tags the assembled body's topology (one entity per role, so the index is always 0).
func crossLin(role string) topo.Lineage { return topo.NewLineage(topo.Tok("crosscyl", role, 0)) }

// CrossingCylinderIntersect builds the exact intersection of two bare crossing cylinders as a watertight
// analytic B-rep (rod band + two fat-wall lens caps), or ok=false when the configuration is outside the
// wired thin-through-fat case (not exactly two imprint loops, or neither cylinder is the rod both loops
// encircle) so kernel/ops keeps its fallback.
//
// Example — a radius-1.5 rod on x crossing a radius-3 cylinder on z gives a three-face solid:
//
//	fat, _  := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	thin, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 1.5, 12)
//	res, ok := brep.CrossingCylinderIntersect(fat, thin) // rod band capped by two lenses
func CrossingCylinderIntersect(a, b *topo.Body) (*topo.Body, bool) {
	loops, ok := crossingCylinderImprint(a, b)
	if !ok || len(loops) != 2 {
		return nil, false
	}
	rod, fat, ok := rodAndFatCylinders(a, b, loops)
	if !ok {
		return nil, false
	}
	lo, hi, ok := assignRimLoops(rod, fat, loops)
	if !ok {
		return nil, false
	}
	return stitchRodWithSaddleCaps(rod, fat, lo, hi), true
}

// rodAndFatCylinders picks which cylinder is the rod (the band-bearing one both imprint loops fully
// encircle) and which is the fat cylinder (the lens-capped one). ok=false when neither or both qualify.
func rodAndFatCylinders(a, b *topo.Body, loops []geom.Polyline) (rod, fat geom.Cylinder, ok bool) {
	ca, _, _, okA := cylinderSolidParams(facesOfAny(a))
	cb, _, _, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB {
		return geom.Cylinder{}, geom.Cylinder{}, false
	}
	aWraps, bWraps := allLoopsEncircle(loops, ca), allLoopsEncircle(loops, cb)
	if aWraps && !bWraps {
		return ca, cb, true
	}
	if bWraps && !aWraps {
		return cb, ca, true
	}
	return geom.Cylinder{}, geom.Cylinder{}, false
}

// allLoopsEncircle reports whether every loop winds a full turn about the cylinder's axis — the test that
// distinguishes the rod (each loop rings right around it, ≈ ±2π) from the fat cylinder (each loop is a
// local lens off to one side that barely turns, ≈ 0).
func allLoopsEncircle(loops []geom.Polyline, cyl geom.Cylinder) bool {
	for _, lp := range loops {
		if stdmath.Abs(loopTurnAboutAxis(lp, cyl)) < stdmath.Pi {
			return false
		}
	}
	return true
}

// loopTurnAboutAxis sums the signed angle a loop's spokes sweep about the cylinder axis (≈ ±2π for a loop
// encircling the axis, ≈ 0 for a local lens).
func loopTurnAboutAxis(lp geom.Polyline, cyl geom.Cylinder) float64 {
	axis := cyl.AxisDir.AsVector()
	vs := lp.Vertices
	total := 0.0
	for i := 0; i+1 < len(vs); i++ {
		total += signedAngleAround(radialOf(vs[i], cyl), radialOf(vs[i+1], cyl), axis)
	}
	return total
}

// radialOf returns the spoke from the cylinder axis out to p — the component of (axis origin → p)
// perpendicular to the axis, whose turning measures the angle about the axis.
func radialOf(p math.Point3, cyl geom.Cylinder) math.Vector3 {
	axis := cyl.AxisDir.AsVector()
	v := cyl.Origin.VectorTo(p)
	return v.Sub(axis.Scale(v.Dot(axis)))
}

// assignRimLoops orients both imprint loops CCW about the rod axis and assigns them to the lower (lo) and
// upper (hi) rim of the band by which way the fat wall faces there: the loop whose fat-outward normal
// points along +rodaxis is the upper rim (the band traverses it reversed, its cap forward), the other the
// lower. The hi loop is rotated to start at the lo loop's seam angle so their joining seam is a clean
// constant-angle ruling of the rod. ok=false unless there is exactly one loop of each end.
func assignRimLoops(rod, fat geom.Cylinder, loops []geom.Polyline) (lo, hi geom.Polyline, ok bool) {
	rodAxis := rod.AxisDir.AsVector()
	var los, his []geom.Polyline
	for _, lp := range loops {
		ccw := orientLoopCCW(lp, rod)
		if fatOutwardAlongAxis(ccw, fat, rodAxis) >= 0 {
			his = append(his, ccw)
		} else {
			los = append(los, ccw)
		}
	}
	if len(los) != 1 || len(his) != 1 {
		return geom.Polyline{}, geom.Polyline{}, false
	}
	return los[0], alignLoopStart(his[0], los[0], rod), true
}

// orientLoopCCW returns the loop oriented so it winds CCW about the rod axis (positive turn), reversing
// its vertex order when the traced loop runs the other way.
func orientLoopCCW(lp geom.Polyline, rod geom.Cylinder) geom.Polyline {
	if loopTurnAboutAxis(lp, rod) >= 0 {
		return lp
	}
	return reverseLoop(lp)
}

// reverseLoop returns the loop with its vertex order reversed (still closed: the first vertex still
// repeats as the last).
func reverseLoop(lp geom.Polyline) geom.Polyline {
	vs := lp.Vertices
	out := make([]math.Point3, len(vs))
	for i, p := range vs {
		out[len(vs)-1-i] = p
	}
	rev, _ := geom.NewPolyline(out)
	return rev
}

// fatOutwardAlongAxis returns the component along the rod axis of the fat cylinder's outward normal at the
// loop's centroid — its sign places the loop's lens on the +rodaxis (≥0) or −rodaxis (<0) end of the band.
func fatOutwardAlongAxis(lp geom.Polyline, fat geom.Cylinder, rodAxis math.Vector3) float64 {
	outward := unit(radialOf(loopCentroid(lp), fat))
	return float64(outward.Dot(rodAxis))
}

// loopCentroid averages a loop's distinct vertices (skipping the repeated closing vertex).
func loopCentroid(lp geom.Polyline) math.Point3 {
	core := lp.Vertices[:len(lp.Vertices)-1]
	var sx, sy, sz math.Scalar
	for _, p := range core {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := math.Scalar(len(core))
	return math.P3(sx/n, sy/n, sz/n)
}

// alignLoopStart rotates the hi loop so it begins at the vertex nearest, in rod-axis angle, to the lo
// loop's start — so the seam joining the two starts is a near-constant-angle ruling of the rod (lying on
// the surface), the clean seam the band's periodic face needs.
func alignLoopStart(hi, lo geom.Polyline, rod geom.Cylinder) geom.Polyline {
	target := axisAngleOf(lo.Vertices[0], rod)
	core := hi.Vertices[:len(hi.Vertices)-1]
	best, bestErr := 0, stdmath.Inf(1)
	for i, p := range core {
		if d := angleDelta(axisAngleOf(p, rod), target); d < bestErr {
			best, bestErr = i, d
		}
	}
	return rotateClosedLoop(core, best)
}

// axisAngleOf returns p's angle about the cylinder axis, measured from the cylinder's reference direction.
func axisAngleOf(p math.Point3, cyl geom.Cylinder) float64 {
	return signedAngleAround(cyl.Ref.AsVector(), radialOf(p, cyl), cyl.AxisDir.AsVector())
}

// angleDelta is the absolute difference between two angles, wrapped into [0, π].
func angleDelta(a, b float64) float64 {
	d := stdmath.Abs(a - b)
	if d > stdmath.Pi {
		d = 2*stdmath.Pi - d
	}
	return d
}

// rotateClosedLoop returns a closed polyline starting at index start of the distinct (de-duplicated)
// vertices, wrapping around and repeating the new start vertex at the end to keep the loop closed.
func rotateClosedLoop(core []math.Point3, start int) geom.Polyline {
	out := make([]math.Point3, 0, len(core)+1)
	for i := 0; i < len(core); i++ {
		out = append(out, core[(start+i)%len(core)])
	}
	out = append(out, core[start]) // repeat the start vertex so the loop closes
	rot, _ := geom.NewPolyline(out)
	return rot
}

// stitchRodWithSaddleCaps welds the three analytic faces of the intersection into one solid: the rod-wall
// band (a periodic cylinder side — seam up, hi rim, seam down, lo rim, the SolidCylinder pattern) plus the
// two fat-wall lens caps, each sharing one rim edge with the band in the OPPOSITE orientation, so every
// edge is used exactly twice (a closed manifold). The rims stay closed saddle-polyline edges so the band
// tessellates via the saddle-band loft and each lens via the metric-patch mesher.
func stitchRodWithSaddleCaps(rod, fat geom.Cylinder, lo, hi geom.Polyline) *topo.Body {
	loPts, hiPts := lo.Vertices, hi.Vertices
	bld := topo.NewBuilder(true, crossLin("body"))
	vLo := bld.AddVertex(loPts[0], crossLin("vlo"))
	vHi := bld.AddVertex(hiPts[0], crossLin("vhi"))
	eLo := bld.AddEdge(lo, vLo, vLo, crossLin("elo"))
	eHi := bld.AddEdge(hi, vHi, vHi, crossLin("ehi"))
	eSeam := bld.AddEdge(geom.NewLineSegment(loPts[0], hiPts[0]), vLo, vHi, crossLin("seam"))
	bld.AddFace(rod, crossLin("band"),
		topo.OuterLoop(topo.Fwd(eSeam), topo.Rev(eHi), topo.Rev(eSeam), topo.Fwd(eLo)))
	bld.AddFace(fat, crossLin("caplo"), topo.OuterLoop(topo.Rev(eLo)))
	bld.AddFace(fat, crossLin("caphi"), topo.OuterLoop(topo.Fwd(eHi)))
	return bld.Build()
}
