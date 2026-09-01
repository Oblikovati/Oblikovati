// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// The boundary walk's endpoint grading (edgeGradedT, M48/C3 Oblikovati/Oblikovati#3453 follow-up).
//
// An edge's parametrization is not guaranteed regular at its ends: a spiric arc TURNS at the oval's
// v-extremes, where its two branches meet, so |dP/dt| diverges like 1/√(t−t0) and every boundary
// integrand carries that speed. A fixed-order rule in the curve parameter resolved none of it —
// measured, the torus ∩ box oval cap integrated 23.4960 against a true patch area of 24.0889 and its
// planar lid 19.7881 against 20.2302 — and because the two misses differ, the cap's vector area
// disagreed with its own lid by 0.26% of it, which the closure post-condition then declined although
// the volume was right. These hold the fix to the two things that were wrong: an INDEPENDENT area
// oracle for the patch, and the exact cancellation the two faces owe each other.

// spiricCapTorus is the corpus's oval-cap geometry: torus R=5, r=2 about +z, kept where y ≥ 6.
const (
	spiricCapMajor = 5.0
	spiricCapMinor = 2.0
	spiricCapY     = 6.0
)

// spiricCapAreaOracle is the area of the torus patch {y ≥ yCut} reduced to ONE smooth integral in the
// tube angle, independent of the B-rep and of the uv walk under test. At tube angle v the patch's
// azimuth window is where (R + r·cos v)·sin u ≥ yCut, an arc of π − 2·asin(yCut/ρ), and the torus area
// element is r·ρ du dv. The window closes like √(vMax − v) at the tube extreme, so the integral is
// taken in w with v = vMax − w², which makes the integrand a smooth multiple of w².
func spiricCapAreaOracle(major, minor, yCut float64) float64 {
	vMax := stdmath.Acos((yCut - major) / minor)
	at := func(w float64) float64 {
		rho := major + minor*stdmath.Cos(vMax-w*w)
		window := stdmath.Pi - 2*stdmath.Asin(stdmath.Min(yCut/rho, 1))
		return minor * rho * window * 2 * w
	}
	return 2 * simpsonOracle(at, 0, stdmath.Sqrt(vMax), 4000)
}

// simpsonOracle is a composite Simpson rule for the closed-form oracles in this file. It is here so
// the oracle shares no code with the engine it gates.
func simpsonOracle(at func(float64) float64, a, b float64, cells int) float64 {
	h := (b - a) / float64(cells)
	sum := at(a) + at(b)
	for i := 1; i < cells; i++ {
		weight := 2.0
		if i%2 == 1 {
			weight = 4
		}
		sum += weight * at(a+float64(i)*h)
	}
	return sum * h / 3
}

// spiricOvalCapBody is the corpus case "torus ∩ box (axis-∥ oval cap)": one torus patch bounded by a
// single spiric oval, plus the planar lid that closes it.
func spiricOvalCapBody(t *testing.T) *topo.Body {
	t.Helper()
	tor, err := brep.SolidTorus(m.P3(0, 0, 0), m.V3(0, 0, 1), spiricCapMajor, spiricCapMinor, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	box, err := brep.SolidBlock(m.P3(-20, spiricCapY, -20), m.P3(20, 20, 20), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	res, err := Boolean(Intersect, tor, box)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	return res
}

// soleCurvedFace returns the body's only non-planar face, failing when the count is not one.
func soleCurvedFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	var found []*topo.Face
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			found = append(found, f)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly 1 curved face, got %d across %d faces", len(found), len(b.Faces()))
	}
	return found[0]
}

// solePlanarFace returns the body's only planar face, failing when the count is not one.
func solePlanarFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	var found []*topo.Face
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); planar {
			found = append(found, f)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly 1 planar face, got %d across %d faces", len(found), len(b.Faces()))
	}
	return found[0]
}

// TestSpiricCapAreaMatchesSectionOracle holds the turning-edge patch to an oracle built from the
// torus section itself, not from the walk that integrates it.
func TestSpiricCapAreaMatchesSectionOracle(t *testing.T) {
	t.Parallel()
	body := spiricOvalCapBody(t)
	got, ok := AnalyticFaceArea(soleCurvedFace(t, body))
	if !ok {
		t.Fatalf("the torus patch declined analytic integration")
	}
	want := spiricCapAreaOracle(spiricCapMajor, spiricCapMinor, spiricCapY)
	if rel := relErrMP(got, want); rel > 1e-9 {
		t.Errorf("spiric cap area %.9f vs section oracle %.9f (rel %.3e > 1e-9)", got, want, rel)
	}
}

// TestSpiricCapVectorAreaCancelsItsLid asserts what Stokes requires and what the 0.26% defect broke:
// the vector area of a patch depends only on its boundary, so a cap and the planar lid sharing that
// boundary must come out exactly equal and opposite. The lid is the trustworthy half — a plane's
// |A| IS its area — so this pins the curved patch to it.
func TestSpiricCapVectorAreaCancelsItsLid(t *testing.T) {
	t.Parallel()
	body := spiricOvalCapBody(t)
	cap, capOK := faceTerms(soleCurvedFace(t, body))
	lid, lidOK := faceTerms(solePlanarFace(t, body))
	if !capOK || !lidOK {
		t.Fatalf("analytic integration declined: cap=%v lid=%v", capOK, lidOK)
	}
	if rel := relErrMP(vectorMagnitude(lid.ax, lid.ay, lid.az), lid.area); rel > 1e-12 {
		t.Fatalf("the planar lid's |A| %.9f is not its area %.9f (rel %.3e)", vectorMagnitude(lid.ax, lid.ay, lid.az), lid.area, rel)
	}
	residual := vectorMagnitude(cap.ax+lid.ax, cap.ay+lid.ay, cap.az+lid.az)
	if rel := residual / lid.area; rel > 1e-9 {
		t.Errorf("cap A=(%.9f,%.9f,%.9f) does not cancel lid A=(%.9f,%.9f,%.9f): residual %.3e of %.6f",
			cap.ax, cap.ay, cap.az, lid.ax, lid.ay, lid.az, residual, lid.area)
	}
}

// TestSpiricCapVectorAreaMatchesOvalSection is the absolute anchor on the magnitude the cancellation
// test can only pin relatively: a patch's vector area is the vector area of the FLAT region its
// boundary bounds, so the cap's |A| is the plane area of the spiric oval, taken from the section's own
// closed form.
func TestSpiricCapVectorAreaMatchesOvalSection(t *testing.T) {
	t.Parallel()
	terms, ok := faceTerms(soleCurvedFace(t, spiricOvalCapBody(t)))
	if !ok {
		t.Fatalf("the torus patch declined analytic integration")
	}
	got := vectorMagnitude(terms.ax, terms.ay, terms.az)
	want := spiricOvalAreaOracle(spiricCapMajor, spiricCapMinor, spiricCapY)
	if rel := relErrMP(got, want); rel > 1e-9 {
		t.Errorf("spiric cap |A| %.9f vs oval section area %.9f (rel %.3e > 1e-9)", got, want, rel)
	}
}

// spiricOvalAreaOracle is the plane area the spiric oval encloses in the cutting plane y = yCut. At
// tube angle v the oval spans |x| ≤ √(ρ² − yCut²) with ρ = major + minor·cos v, and z = minor·sin v is
// monotone over the tube range, so the region integrates as ∫ 2√(ρ² − yCut²) dz. The width closes like
// √(vMax − v) at the tube extreme, so v = vMax − w² again makes the integrand smooth.
func spiricOvalAreaOracle(major, minor, yCut float64) float64 {
	vMax := stdmath.Acos((yCut - major) / minor)
	at := func(w float64) float64 {
		v := vMax - w*w
		rho := major + minor*stdmath.Cos(v)
		return 2 * stdmath.Sqrt(stdmath.Max(rho*rho-yCut*yCut, 0)) * minor * stdmath.Cos(v) * 2 * w
	}
	return 2 * simpsonOracle(at, 0, stdmath.Sqrt(vMax), 4000)
}

// vectorMagnitude is |(x, y, z)| — the length of a face's outward vector area.
func vectorMagnitude(x, y, z float64) float64 {
	return stdmath.Sqrt(x*x + y*y + z*z)
}
