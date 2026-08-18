// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Lofted-flange END-BEND RADIUS (#2086, carved from #1966). Without a bend radius the transition
// wall leaves each profile perpendicular to its plane, meeting it at a sharp knuckle. A bend radius
// rounds that end bend: the wall gains a short flat LIP lying in the profile plane, then a true
// circular FOLD of the radius turns the lip up into the transition, exactly as a flange bends off a
// face. Each corresponding band point therefore traces a composite path — lipA → foldA → the
// die-formed / press-brake middle → foldB → lipB — all sharing one sample partition so the points
// still transpose into corresponding loft sections.

// endBendFoldSamples is how many segments each quarter-fold arc is sampled into.
const endBendFoldSamples = 6

// endBendSections builds the loft sections for a lofted flange with a rounded end bend of radius r.
// The middle keeps the die-formed / press-brake behaviour (its section count comes from the same
// tolerance logic), sampled between the fold ends rather than the profiles.
func endBendSections(bandA, bandB []math.Point3, nA, nB math.UnitVector3, r float64,
	output LoftedFlangeOutputType, tol float64) [][]math.Point3 {
	lipLen := r // a short lip, as long as the bend radius
	cA, cB := centroid(bandA), centroid(bandB)
	ends := make([]endBend, len(bandA))
	for i := range bandA {
		a1A := inPlaneOutward(nA, bandNeighbourTangent(bandA, i), cA.VectorTo(bandA[i]))
		a1B := inPlaneOutward(nB, bandNeighbourTangent(bandB, i), cB.VectorTo(bandB[i]))
		ends[i] = newEndBend(bandA[i], bandB[i], nA, nB, r, lipLen, a1A, a1B)
	}
	mid := loftSectionCount(middleCurves(ends), output, tol)
	return assembleEndBendSections(ends, mid)
}

// endBend holds one band point's composite-path geometry: the flat-lip outer ends, the two sampled
// fold arcs, and the fold ends where each fold hands off to the middle transition.
type endBend struct {
	lipOutA, lipOutB math.Point3
	arcA, arcB       []math.Point3 // arcA: p0→ea; arcB: eb→p1 (already oriented for the path)
	ea, eb           math.Point3
	nA, nB           math.UnitVector3
}

// newEndBend resolves one band point's composite path. The wall leaves A along u0 (its plane normal
// oriented toward B) and folds up from the in-plane lip over radius r; B mirrors it, with the fold
// built by running it backward from the profile so it joins the middle without a kink.
func newEndBend(p0, p1 math.Point3, nA, nB math.UnitVector3, r, lipLen float64,
	a1A, a1B math.Vector3) endBend {
	chord := p0.VectorTo(p1)
	u0 := unitOr(orientedTangent(nA, chord), nA.AsVector())
	w1 := unitOr(orientedTangent(nB, chord), nB.AsVector())
	arcA, ea := foldArc(p0, a1A.Negate(), u0, r, endBendFoldSamples)
	// B's fold from the profile up into the transition, run backward (p1→eb), then reversed so the
	// path enters it heading +w1 to match the middle's arrival.
	arcBback, eb := foldArc(p1, a1B.Negate(), w1.Negate(), r, endBendFoldSamples)
	return endBend{
		lipOutA: p0.TranslateBy(a1A.Scale(math.Scalar(lipLen))),
		lipOutB: p1.TranslateBy(a1B.Scale(math.Scalar(lipLen))),
		arcA:    arcA, arcB: reversePoints(arcBback),
		ea: ea, eb: eb, nA: nA, nB: nB,
	}
}

// middleCurves are the die-formed Hermite curves between the two fold ends — the same construction
// as the profile-to-profile bundle, so the middle's faceting logic is unchanged.
func middleCurves(ends []endBend) []hermiteCurve {
	if len(ends) == 0 {
		return nil
	}
	ea := make([]math.Point3, len(ends))
	eb := make([]math.Point3, len(ends))
	for i, e := range ends {
		ea[i], eb[i] = e.ea, e.eb
	}
	return hermiteBundle(ea, eb, ends[0].nA, ends[0].nB)
}

// assembleEndBendSections samples every band point's composite path at the shared partition and
// transposes the samples into loft sections: lip A, fold A, the mid-1 middle interiors, fold B, lip
// B. The fold ends ea/eb are shared with the middle, so they are not sampled twice.
func assembleEndBendSections(ends []endBend, mid int) [][]math.Point3 {
	paths := make([][]math.Point3, len(ends))
	middle := middleCurves(ends)
	for i, e := range ends {
		path := []math.Point3{e.lipOutA}
		path = append(path, e.arcA...) // p0 … ea
		for j := 1; j < mid; j++ {
			path = append(path, middle[i].at(float64(j)/float64(mid)))
		}
		path = append(path, e.arcB...) // eb … p1
		paths[i] = append(path, e.lipOutB)
	}
	return transposeToSections(paths)
}

// transposeToSections turns per-band-point paths (all the same length) into per-parameter sections.
func transposeToSections(paths [][]math.Point3) [][]math.Point3 {
	if len(paths) == 0 {
		return nil
	}
	m := len(paths[0])
	sections := make([][]math.Point3, m)
	for s := 0; s < m; s++ {
		section := make([]math.Point3, len(paths))
		for i := range paths {
			section[i] = paths[i][s]
		}
		sections[s] = section
	}
	return sections
}

// foldArc samples a circular arc of radius r starting at pStart tangent to tStart and turning until
// it is tangent to tEnd (a quarter turn when the two are perpendicular). It returns the sample points
// (both ends included) and the arc's end point, so callers do not have to place the end by hand. The
// centre sits r from pStart along the component of tEnd perpendicular to tStart.
func foldArc(pStart math.Point3, tStart, tEnd math.Vector3, r float64, samples int) ([]math.Point3, math.Point3) {
	ts, e1 := math.UnitVector3FromVector(tStart)
	te, e2 := math.UnitVector3FromVector(tEnd)
	if e1 != nil || e2 != nil {
		return []math.Point3{pStart}, pStart
	}
	nu, err := math.UnitVector3FromVector(te.AsVector().Sub(ts.AsVector().Scale(te.AsVector().Dot(ts.AsVector()))))
	if err != nil {
		return []math.Point3{pStart}, pStart
	}
	center := pStart.TranslateBy(nu.AsVector().Scale(math.Scalar(r)))
	r0 := center.VectorTo(pStart)
	axis, err := math.UnitVector3FromVector(ts.AsVector().Cross(nu.AsVector()))
	if err != nil {
		return []math.Point3{pStart}, pStart
	}
	sweep := angleBetween(ts.AsVector(), te.AsVector())
	pts := make([]math.Point3, samples+1)
	for k := 0; k <= samples; k++ {
		pts[k] = center.TranslateBy(rotateAbout(r0, axis, sweep*float64(k)/float64(samples)))
	}
	return pts, pts[samples]
}

// rotateAbout rotates v about the unit axis by angle (Rodrigues). Used to sweep a fold's radius
// vector from its start to its end.
func rotateAbout(v math.Vector3, axis math.UnitVector3, angle float64) math.Vector3 {
	a := axis.AsVector()
	cos, sin := stdmath.Cos(angle), stdmath.Sin(angle)
	return v.Scale(math.Scalar(cos)).
		Add(a.Cross(v).Scale(math.Scalar(sin))).
		Add(a.Scale(math.Scalar(float64(a.Dot(v)) * (1 - cos))))
}

// inPlaneOutward is the unit in-plane direction, perpendicular to the local profile edge, oriented
// away from the band's centroid (the direction the flat lip extends). The perpendicular keeps the
// lip square to the profile; the centroid reference orients it consistently for both thickness
// offsets — unlike the chord, which vanishes where the two profiles coincide. It falls back to the
// pure radial direction when the edge is degenerate (the thickness caps).
func inPlaneOutward(n math.UnitVector3, edge, fromCentroid math.Vector3) math.Vector3 {
	radial := fromCentroid.Sub(n.AsVector().Scale(fromCentroid.Dot(n.AsVector())))
	u, err := math.UnitVector3FromVector(n.AsVector().Cross(edge))
	if err != nil {
		if ru, err2 := math.UnitVector3FromVector(radial); err2 == nil {
			return ru.AsVector()
		}
		return math.V3(0, 0, 0)
	}
	if u.AsVector().Dot(radial) < 0 {
		return u.AsVector().Negate()
	}
	return u.AsVector()
}

// bandNeighbourTangent is the local edge direction at band point i, from its two neighbours on the
// closed band loop.
func bandNeighbourTangent(band []math.Point3, i int) math.Vector3 {
	n := len(band)
	return band[(i-1+n)%n].VectorTo(band[(i+1)%n])
}

// centroid is the mean of a band's points — a robust interior reference for orienting the flat lip
// outward.
func centroid(band []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range band {
		sx, sy, sz = sx+float64(p.X), sy+float64(p.Y), sz+float64(p.Z)
	}
	n := float64(len(band))
	return math.P3(sx/n, sy/n, sz/n)
}

// reversePoints returns the points in reverse order (a fresh slice).
func reversePoints(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// unitOr returns v as a unit vector, or fallback (unit) if v is degenerate.
func unitOr(v, fallback math.Vector3) math.Vector3 {
	u, err := math.UnitVector3FromVector(v)
	if err != nil {
		f, _ := math.UnitVector3FromVector(fallback)
		return f.AsVector()
	}
	return u.AsVector()
}
