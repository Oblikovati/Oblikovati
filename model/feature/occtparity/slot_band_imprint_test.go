// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The SLOT family's per-face oracle: simple/Y1..Y4 are one 100³ box with a 10×10 through-slot at
// x ∈ [90,100], z ∈ [80,90], filleted along its y = 0 ∧ z = 100 edge at four radii. The band runs into
// the slot at three of them, and the band∩obstacle imprint (kernel/ops/fillet_band_imprint.go) has to
// re-trim every face the band's own boundary crosses — TOGETHER, because they share edges.
//
// Every expectation here is a CLOSED FORM in the body's own dimensions. That matters twice over:
//
//   - Y2 and Y4 were already scored PASS before the imprint existed, at +0.159 % and +0.645 % body
//     area — inside the corpus's 1 % gate. An area gate cannot see that; a per-face closed form can.
//   - Y4's largest single defect (+100 on the host plane, whose loop back-tracked along a COLLINEAR
//     sibling) is invisible to knownSelfCrossingLoops too, because simpleLoop2D scores a collinear
//     overlap as zero crossings. Neither existing ratchet guards these faces; this file does.
//
// MEASURING FUNCTION: each face's OWN BOUNDARY — Newell over the loop sampled through every edge's
// carried curve for a plane, the developed shoelace in the cylinder's metric chart for the band. Not a
// mesh: the tessellation's chord error is an order above the residuals these targets carry.

// slotFaceSamples is the per-edge sampling of a measured boundary. A quarter arc of radius 25 sampled
// 4096 times has a Newell-area discretization error ~1e-8 relative, two decades under slotAreaTol.
const slotFaceSamples = 4096

// slotAreaTol is the relative agreement demanded between a measured face boundary and its closed form.
const slotAreaTol = 1e-6

// TestSlotFamilyFacesMatchTheirClosedForms is the acceptance: every face the imprint touches, on the
// SHIPPED body of each case, against the closed form its geometry has.
func TestSlotFamilyFacesMatchTheirClosedForms(t *testing.T) {
	t.Parallel()
	for _, c := range slotCases() {
		body := slotShippedBody(t, c.name)
		for lineage, want := range c.faces {
			got, ok := slotFaceArea(body, lineage)
			if !ok {
				t.Errorf("%s: no face with lineage %s on the shipped body", c.name, lineage)
				continue
			}
			if rel := stdmath.Abs(got-want) / want; rel > slotAreaTol {
				t.Errorf("%s %s: boundary area %.7f, closed form %.7f (rel %.3g)", c.name, lineage, got, want, rel)
			}
		}
	}
}

// TestSlotFamilyBodiesAreWatertightAndWholeAtTheOracleFaceCount is the atomicity check the area targets
// cannot make on their own: a partial re-trim of these five/seven shared-edge faces leaves the shell
// OPEN, and an open shell is what the imprint must never ship. The face count is the oracle's own —
// the imprint re-trims faces, it never adds or removes one.
func TestSlotFamilyBodiesAreWatertightAndWholeAtTheOracleFaceCount(t *testing.T) {
	t.Parallel()
	for _, c := range slotCases() {
		body := slotShippedBody(t, c.name)
		rep := ops.Validate(body)
		if !rep.Valid || !rep.Closed || !rep.Manifold || !rep.HolesContained || !body.IsSolid() {
			t.Errorf("%s: not a watertight solid (valid=%v closed=%v manifold=%v holes=%v solid=%v) %v",
				c.name, rep.Valid, rep.Closed, rep.Manifold, rep.HolesContained, body.IsSolid(), rep.Issues)
		}
		if len(body.Faces()) != 11 {
			t.Errorf("%s: %d faces, want 11 (10 box faces + the band)", c.name, len(body.Faces()))
		}
	}
}

// TestSlotFamilyBodyAreaMatchesOCCT closes the loop on the oracle: the closed forms sum to OCCT's own
// reference area for each case, so the per-face targets above are not a private theory.
func TestSlotFamilyBodyAreaMatchesOCCT(t *testing.T) {
	t.Parallel()
	for _, c := range slotCases() {
		body := slotShippedBody(t, c.name)
		total := 0.0
		for _, f := range body.Faces() {
			total += slotBoundaryArea(f)
		}
		if rel := stdmath.Abs(total-c.occt) / c.occt; rel > 1e-5 {
			t.Errorf("%s: body boundary area %.6f, OCCT %.6f (rel %.3g)", c.name, total, c.occt, rel)
		}
	}
}

// slotCase is one radius of the slot family with the closed forms of every face the imprint changes.
type slotCase struct {
	name  string
	occt  float64            // the corpus's own OCCT reference area, to 6 figures
	faces map[string]float64 // face lineage → closed-form boundary area
}

// slotCases builds all four radii's closed forms from the body's dimensions alone.
//
// Geometry: the band is a quarter cylinder of radius r about (y = r, z = 100 − r) running along x, so
// the fillet removes {y < r, z > 100 − r} outside it. Writing t = z − (100 − r), the material removed
// from a plane of constant x or constant z is governed by y < r − √(r² − t²), which integrates in
// closed form (slotCircularSegment). The slot occupies z ∈ [80,90] for x ∈ [90,100].
func slotCases() []slotCase {
	return []slotCase{
		{"Y1", 61327.9, map[string]float64{}}, // r=10: the contact lands ON the slot roof — nothing to imprint
		{"Y3", 60738.4, slotFaces(20, 8000)},
		{"Y2", 61050.1, slotFaces(15, 8450)},
		{"Y4", 60393.2, slotFaces(25, 7500)},
	}
}

// slotFaces is the closed form of every face the band's imprint crosses at radius r. host is the y = 0
// plane's own area, which is the rectangle 100 × (100 − r) less whatever of the slot's y = 0 profile
// still survives below the contact line.
func slotFaces(r, host float64) map[string]float64 {
	lo, hi := 80-(100-r), 90-(100-r) // the slot's floor / roof in the band's own t = z − (100 − r)
	out := map[string]float64{
		"import:step#16:face#0":              host,
		"import:step#16:face#4":              1000 - slotBite(r, stdmath.Max(0, lo), stdmath.Min(r, hi)),
		"import:step#16:face#6":              1000 - 10*(r-stdmath.Sqrt(r*r-hi*hi)),
		"import:step#16:face#8":              1000 - slotBite(r, stdmath.Min(r, hi), r),
		"import:step#16:edge#3/fillet:cyl#0": r*stdmath.Pi/2*100 - r*slotSweep(r, lo, hi)*10,
	}
	if lo > 0 { // the slot's FLOOR is above the contact line: the band bites it, and the wall below too
		out["import:step#16:face#2"] = 1000 - 10*(r-stdmath.Sqrt(r*r-lo*lo))
		out["import:step#16:face#1"] = 8000 - slotBite(r, 0, lo)
	}
	return out
}

// slotSweep is the band's swept angle inside the slot — the part of the quarter turn whose z lies in
// the slot's own [80,90], clamped to the band.
func slotSweep(r, lo, hi float64) float64 {
	a := stdmath.Asin(stdmath.Max(0, lo) / r)
	return stdmath.Asin(stdmath.Min(r, hi)/r) - a
}

// slotBite is ∫ₐᵇ (r − √(r² − t²)) dt — the area the band takes out of a plane over t ∈ [a,b].
func slotBite(r, a, b float64) float64 {
	return r*(b-a) - slotCircularSegment(r, a, b)
}

// slotCircularSegment is ∫ₐᵇ √(r² − t²) dt = [t√(r² − t²)/2 + (r²/2)·asin(t/r)]ₐᵇ, exact.
func slotCircularSegment(r, a, b float64) float64 {
	f := func(t float64) float64 {
		return t*stdmath.Sqrt(stdmath.Max(0, r*r-t*t))/2 + r*r/2*stdmath.Asin(stdmath.Max(-1, stdmath.Min(1, t/r)))
	}
	return f(b) - f(a)
}

// slotShippedBody runs one case's real fillet exactly as the corpus scores it.
func slotShippedBody(t *testing.T, name string) *topo.Body {
	t.Helper()
	for _, r := range Corpus() {
		if r.Grid != "simple" || r.Case != name {
			continue
		}
		body, ok := shippedCaseBody(r, CorpusFixtureDir())
		if !ok {
			t.Fatalf("%s: no healthy shipped body", name)
		}
		return body
	}
	t.Fatalf("%s: not in the corpus", name)
	return nil
}

// slotFaceArea is the boundary area of the face carrying the given lineage.
func slotFaceArea(body *topo.Body, lineage string) (float64, bool) {
	for _, f := range body.Faces() {
		if f.Lineage().String() == lineage {
			return slotBoundaryArea(f), true
		}
	}
	return 0, false
}

// slotBoundaryArea measures a face's OWN boundary: Newell for a plane (outer loop less its holes), the
// developed shoelace in the cylinder's metric chart for the band.
func slotBoundaryArea(f *topo.Face) float64 {
	switch s := f.Geometry().(type) {
	case geom.Plane:
		a := 0.0
		for _, l := range f.Loops() {
			sign := 1.0
			if !l.IsOuter() {
				sign = -1
			}
			a += sign * slotNewellArea(slotLoopPolyline(l))
		}
		return a
	case geom.Cylinder:
		return slotDevelopedArea(slotLoopPolyline(f.Loops()[0]), s)
	}
	return 0
}

// slotLoopPolyline develops a loop through each edge's own carried curve, in traversal order.
func slotLoopPolyline(l *topo.Loop) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		a, b := u.Edge().StartVertex().Point(), u.Edge().EndVertex().Point()
		if u.Reversed() {
			a, b = b, a
		}
		pts = append(pts, slotEdgePoints(u.Edge().Geometry(), a, b, u.Reversed())...)
	}
	return pts
}

// slotEdgePoints samples one edge use from its own start toward its own end.
func slotEdgePoints(c geom.Curve3, a, b math.Point3, reversed bool) []math.Point3 {
	pts := make([]math.Point3, 0, slotFaceSamples)
	for i := range slotFaceSamples {
		t := float64(i) / slotFaceSamples
		if c == nil {
			pts = append(pts, a.Lerp(b, math.Scalar(t)))
			continue
		}
		lo, hi := c.Domain()
		if reversed {
			pts = append(pts, c.PointAt(hi-t*(hi-lo)))
			continue
		}
		pts = append(pts, c.PointAt(lo+t*(hi-lo)))
	}
	return pts
}

// slotNewellArea is the enclosed area of a planar 3D ring by Newell's method.
func slotNewellArea(pts []math.Point3) float64 {
	var nx, ny, nz float64
	for i, a := range pts {
		b := pts[(i+1)%len(pts)]
		nx += float64((a.Y - b.Y) * (a.Z + b.Z))
		ny += float64((a.Z - b.Z) * (a.X + b.X))
		nz += float64((a.X - b.X) * (a.Y + b.Y))
	}
	return stdmath.Sqrt(nx*nx+ny*ny+nz*nz) / 2
}

// slotDevelopedArea is the enclosed area of a ring on a cylinder, in the cylinder's own metric chart
// (arc length around the section, distance along the axis) — the band's true developed area.
func slotDevelopedArea(pts []math.Point3, c geom.Cylinder) float64 {
	ax, ref := c.AxisDir.AsVector(), c.Ref.AsVector()
	bi := ax.Cross(ref)
	chart := func(p math.Point3) (float64, float64) {
		w := c.Origin.VectorTo(p)
		along := float64(w.Dot(ax))
		rad := w.Sub(ax.Scale(math.Scalar(along)))
		return c.Radius * stdmath.Atan2(float64(rad.Dot(bi)), float64(rad.Dot(ref))), along
	}
	sum := 0.0
	for i := range pts {
		x0, y0 := chart(pts[i])
		x1, y1 := chart(pts[(i+1)%len(pts)])
		sum += x0*y1 - x1*y0
	}
	return stdmath.Abs(sum) / 2
}
