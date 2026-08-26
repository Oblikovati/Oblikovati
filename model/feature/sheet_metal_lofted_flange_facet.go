// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/math"
)

// Lofted-flange die-formed curve and press-brake faceting (#1966). Each pair of corresponding band
// points is joined by a cubic Hermite that leaves both profiles perpendicular to their planes — the
// natural die-forming direction — so the transition wall is a genuinely curved surface rather than a
// straight ruling. A die-formed output samples every curve finely; a press-brake output samples them
// at the COARSEST shared count that still meets the facet tolerance, so the wall becomes flat plates
// joined by bend lines. The sections must share one parameter partition or the loft sections would
// stop corresponding point-for-point, so the faceting is one shared count, not per-curve.

const (
	// dieFormedSections is the fine sampling that stands in for the smooth die-formed wall.
	dieFormedSections = 24
	// maxLoftSections caps the press-brake refinement (and is the finest a press-brake wall gets).
	maxLoftSections = 48
	// defaultPressBrakeSections is used when a press-brake output gives no usable tolerance.
	defaultPressBrakeSections = 6
	// hermiteTangentAlpha scales the end tangents by this fraction of the chord length. The advisor's
	// cusp-free bound is α≤1; 0.7 (Catmull-Rom-like) gives a clean bulge without looping.
	hermiteTangentAlpha = 0.7
)

// hermiteCurve joins one corresponding band-point pair with a cubic Hermite whose end tangents run
// along the two profile-plane normals.
type hermiteCurve struct {
	p0, p1 math.Point3
	m0, m1 math.Vector3
}

// at evaluates the Hermite at t∈[0,1]. h00+h01≡1, so the two endpoint terms are an affine point
// combination and the tangent terms add a displacement — no point-from-components needed.
func (c hermiteCurve) at(t float64) math.Point3 {
	t2 := t * t
	t3 := t2 * t
	h01 := -2*t3 + 3*t2
	h10 := t3 - 2*t2 + t
	h11 := t3 - t2
	base := c.p0.TranslateBy(c.p0.VectorTo(c.p1).Scale(math.Scalar(h01)))
	return base.TranslateBy(c.m0.Scale(math.Scalar(h10)).Add(c.m1.Scale(math.Scalar(h11))))
}

// loftedFlangeSections builds the loft sections between the two bands. Die-formed samples the smooth
// wall finely; a press-brake output samples it at the coarsest count that meets the facet tolerance.
func loftedFlangeSections(bandA, bandB []math.Point3, nA, nB math.UnitVector3,
	output LoftedFlangeOutputType, tol float64) [][]math.Point3 {
	curves := hermiteBundle(bandA, bandB, nA, nB)
	return sampleSections(curves, loftSectionCount(curves, output, tol))
}

// hermiteBundle builds the die-formed curve for every corresponding band-point pair.
func hermiteBundle(bandA, bandB []math.Point3, nA, nB math.UnitVector3) []hermiteCurve {
	curves := make([]hermiteCurve, len(bandA))
	for i := range bandA {
		d := bandA[i].VectorTo(bandB[i])
		scale := math.Scalar(hermiteTangentAlpha) * d.Length()
		curves[i] = hermiteCurve{
			p0: bandA[i], p1: bandB[i],
			m0: orientedTangent(nA, d).Scale(scale), m1: orientedTangent(nB, d).Scale(scale),
		}
	}
	return curves
}

// orientedTangent returns the plane normal flipped, if needed, to advance along the chord d, so the
// die-formed curve heads from one profile toward the other instead of looping back on itself.
func orientedTangent(n math.UnitVector3, d math.Vector3) math.Vector3 {
	v := n.AsVector()
	if v.Dot(d) < 0 {
		return v.Negate()
	}
	return v
}

// sampleSections samples every curve at m+1 shared parameters, one section per parameter.
func sampleSections(curves []hermiteCurve, m int) [][]math.Point3 {
	sections := make([][]math.Point3, m+1)
	for j := 0; j <= m; j++ {
		t := float64(j) / float64(m)
		section := make([]math.Point3, len(curves))
		for i, c := range curves {
			section[i] = c.at(t)
		}
		sections[j] = section
	}
	return sections
}

// loftSectionCount is how many segments the loft is faceted into. Die-formed is a fixed fine count;
// a press-brake output refines from one segment up to the coarsest count that meets the tolerance.
func loftSectionCount(curves []hermiteCurve, output LoftedFlangeOutputType, tol float64) int {
	if !output.IsPressBrake() {
		return dieFormedSections
	}
	if tol <= 0 {
		return defaultPressBrakeSections
	}
	for m := 1; m < maxLoftSections; m++ {
		if facetError(curves, m, output) <= tol {
			return m
		}
	}
	return maxLoftSections
}

// facetError is the worst facet error across the whole curve bundle at m segments, measured the way
// the output type demands: chord deviation (sagitta), turning angle, or facet width.
func facetError(curves []hermiteCurve, m int, output LoftedFlangeOutputType) float64 {
	worst := 0.0
	for _, c := range curves {
		if e := curveFacetError(c, m, output); e > worst {
			worst = e
		}
	}
	return worst
}

// curveFacetError is one curve's facet error at m segments for the output mode.
func curveFacetError(c hermiteCurve, m int, output LoftedFlangeOutputType) float64 {
	pts := make([]math.Point3, m+1)
	for j := 0; j <= m; j++ {
		pts[j] = c.at(float64(j) / float64(m))
	}
	switch output {
	case PressBrakeFacetAngleLoftedFlange:
		return maxTurningAngle(pts)
	case PressBrakeFacetDistanceLoftedFlange:
		return maxSegmentLength(pts)
	default: // PressBrakeChordToleranceLoftedFlange
		return maxSagitta(c, pts, m)
	}
}

// sagittaSubSamples is how many interior points per facet the sagitta is measured at. A die-formed
// transition is an S-curve, antisymmetric about its own midpoint — so a single midpoint sample reads
// ZERO deviation while the curve bows hardest near the quarter points; several samples catch it.
const sagittaSubSamples = 5

// maxSagitta is the largest deviation of the true curve from its facet chords, sampled at several
// interior points of each segment so the S-curve's off-centre bow is not missed.
func maxSagitta(c hermiteCurve, pts []math.Point3, m int) float64 {
	worst := 0.0
	for j := range m {
		for k := 1; k < sagittaSubSamples; k++ {
			t := (float64(j) + float64(k)/float64(sagittaSubSamples)) / float64(m)
			if d := pointSegmentDistance(c.at(t), pts[j], pts[j+1]); d > worst {
				worst = d
			}
		}
	}
	return worst
}

// maxSegmentLength is the longest facet chord.
func maxSegmentLength(pts []math.Point3) float64 {
	worst := 0.0
	for j := 0; j < len(pts)-1; j++ {
		if l := float64(pts[j].VectorTo(pts[j+1]).Length()); l > worst {
			worst = l
		}
	}
	return worst
}

// maxTurningAngle is the largest angle (radians) between adjacent facet directions.
func maxTurningAngle(pts []math.Point3) float64 {
	worst := 0.0
	for j := 1; j < len(pts)-1; j++ {
		a := pts[j-1].VectorTo(pts[j])
		b := pts[j].VectorTo(pts[j+1])
		if ang := angleBetween(a, b); ang > worst {
			worst = ang
		}
	}
	return worst
}

// pointSegmentDistance is the distance from q to the line through a and b (the chord), used as the
// sagitta of a curve midpoint against its facet chord.
func pointSegmentDistance(q, a, b math.Point3) float64 {
	ab := a.VectorTo(b)
	l := float64(ab.Length())
	if l == 0 {
		return float64(a.VectorTo(q).Length())
	}
	return float64(a.VectorTo(q).Cross(ab).Length()) / l
}
