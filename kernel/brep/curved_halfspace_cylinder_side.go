// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cylinder-side split (M2 Phase 1, Oblikovati/Oblikovati#1334). The general looped split tangles on a
// FULL periodic cylinder side: its seam edge runs up one constant angle, and when that seam lands in the
// kept region the two boundary circles' kept arcs join across it into runs the imprint lines cannot
// bridge. So the full side cut by a plane PARALLEL to the axis (imprint = two axis-parallel lines) gets a
// dedicated closed-form split here: the kept band is the single arc the plane leaves, rebuilt as a clean
// seam-free arc face (bottom arc, up one cut line, top arc back, down the other). After this first cut the
// side is a seam-free arc band, so every later cut composes through the general loopedSplit.

// cylinderSideSplit splits a full periodic cylinder side f by a plane parallel to the axis, given the two
// axis-parallel imprint lines. It returns the kept arc-band face and the two section lines (reversed for
// the lid). The two lines bound the chord the plane cuts across the circular cross-section.
func cylinderSideSplit(f curvedFace, lines []geom.Curve3, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	cyl, ok := f.surface.(geom.Cylinder)
	if !ok || len(lines) != 2 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	bottom, top, ok := cylinderSideBand(f)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	u1 := cylinderAngleOf(cyl, lines[0].PointAt(0))
	u2 := cylinderAngleOf(cyl, lines[1].PointAt(0))
	start, sweep := keptCylinderArc(cyl, bottom, top, u1, u2, plane, n)
	loop, section := cylinderArcBand(cyl, bottom, top, start, sweep)
	kept := curvedFace{surface: cyl, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	return []curvedFace{kept}, section, nil
}

// cylinderSideBand returns the bottom and top cross-section circle centres of a cylinder side face (the
// two full-circle boundary edges), ordered low→high along the axis. ok=false if the face is not the
// expected two-circle periodic side.
func cylinderSideBand(f curvedFace) (bottom, top math.Point3, ok bool) {
	cyl := f.surface.(geom.Cylinder)
	axis := cyl.AxisDir.AsVector()
	var centers []math.Point3
	for _, le := range f.loops[0].edges {
		if c, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			centers = append(centers, c.Center)
		}
	}
	if len(centers) != 2 {
		return math.Point3{}, math.Point3{}, false
	}
	if float64(centers[0].VectorTo(centers[1]).Dot(axis)) < 0 {
		centers[0], centers[1] = centers[1], centers[0]
	}
	return centers[0], centers[1], true
}

// cylinderAngleOf returns the angle (about the axis, in the cylinder's Ref/binormal frame) of a point on
// the cylinder surface — the angular position of an imprint line.
func cylinderAngleOf(cyl geom.Cylinder, p math.Point3) float64 {
	axis := cyl.AxisDir.AsVector()
	r := cyl.Origin.VectorTo(p)
	r = r.Sub(axis.Scale(r.Dot(axis))) // drop the axial component → radial vector
	binormal := axis.Cross(cyl.Ref.AsVector())
	return stdmath.Atan2(float64(r.Dot(binormal)), float64(r.Dot(cyl.Ref.AsVector())))
}

// keptCylinderArc returns the start angle and signed sweep of the arc the plane leaves on the kept
// (negative) side. Of the two arcs between the crossing angles u1 and u2, it keeps the one whose midpoint
// lies on the negative side; the sweep is positive (increasing angle) from the returned start.
func keptCylinderArc(cyl geom.Cylinder, bottom, top math.Point3, u1, u2 float64, plane geom.Plane, n math.Vector3) (start, sweep float64) {
	const twoPi = 2 * stdmath.Pi
	midV := float64(bottom.VectorTo(top).Dot(cyl.AxisDir.AsVector())) / 2
	forward := stdmath.Mod(u2-u1, twoPi) // u1 → u2 increasing
	if forward < 0 {
		forward += twoPi
	}
	if cylinderArcKept(cyl, bottom, u1+forward/2, midV, plane, n) {
		return u1, forward
	}
	return u2, twoPi - forward
}

// cylinderArcKept reports whether the cylinder point at angle u and mid-height midV lies on the plane's
// negative (kept) side.
func cylinderArcKept(cyl geom.Cylinder, bottom math.Point3, u, midV float64, plane geom.Plane, n math.Vector3) bool {
	cos, sin := stdmath.Cos(u), stdmath.Sin(u)
	binormal := cyl.AxisDir.AsVector().Cross(cyl.Ref.AsVector())
	radial := cyl.Ref.AsVector().Scale(math.Scalar(cos)).Add(binormal.Scale(math.Scalar(sin)))
	p := bottom.TranslateBy(cyl.AxisDir.AsVector().Scale(math.Scalar(midV))).TranslateBy(radial.Scale(math.Scalar(cyl.Radius)))
	return signedDistance(p, plane, n) < 0
}

// cylinderArcBand builds the kept arc-band loop (bottom arc, up the far cut line, top arc back, down the
// near cut line) and the two section lines (the cut lines, reversed, for the lid). The bottom arc runs
// forward (the cylinder's outward orientation) and the top arc back, mirroring the periodic side's
// bottom-forward / top-reversed circle uses so the kept face keeps an outward radial normal.
func cylinderArcBand(cyl geom.Cylinder, bottom, top math.Point3, start, sweep float64) (loop, section []loopEdge) {
	axis := cyl.AxisDir.AsVector()
	ref := cyl.Ref.AsVector()
	bottomArc, _ := geom.NewArc3d(bottom, axis, ref, cyl.Radius, start, sweep)
	topArc, _ := geom.NewArc3d(top, axis, ref, cyl.Radius, start+sweep, -sweep)
	bEnd, bStart := bottomArc.PointAt(1), bottomArc.PointAt(0)
	tStart, tEnd := topArc.PointAt(0), topArc.PointAt(1) // tStart at β (= bEnd angle), tEnd at α
	up := loopEdge{curve: geom.NewLineSegment(bEnd, tStart), t0: 0, t1: 1}
	down := loopEdge{curve: geom.NewLineSegment(tEnd, bStart), t0: 0, t1: 1}
	loop = []loopEdge{
		{curve: bottomArc, t0: 0, t1: 1},
		up,
		{curve: topArc, t0: 0, t1: 1},
		down,
	}
	return loop, []loopEdge{reverseEdge(up), reverseEdge(down)}
}
