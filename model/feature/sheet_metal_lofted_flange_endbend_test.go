// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Lofted-flange end-bend radius (#2086). The fold is a TRUE circular arc, so the arc sampler is
// measured against the radius directly, and the finished wall must be a valid watertight solid that
// gains the lip + fold material the sharp end bend did not have.

// TestFoldArcHasRadiusAndTangents the fold is a true circular arc of the given radius: from the
// origin tangent to +Y, turning to +X, every point sits on the circle of radius r about (r,0,0), and
// the sampled tangents match +Y at the start and +X at the end.
func TestFoldArcHasRadiusAndTangents(t *testing.T) {
	const r = 0.5
	pts, end := foldArc(math.P3(0, 0, 0), math.V3(0, 1, 0), math.V3(1, 0, 0), r, endBendFoldSamples)
	if len(pts) != endBendFoldSamples+1 {
		t.Fatalf("arc has %d points, want %d", len(pts), endBendFoldSamples+1)
	}
	center := math.P3(r, 0, 0)
	for k, p := range pts {
		if d := float64(center.DistanceTo(p)); stdmath.Abs(d-r) > 1e-9 {
			t.Errorf("arc point %d at radius %.6f, want %.6f", k, d, r)
		}
	}
	if d := float64(pts[0].DistanceTo(math.P3(0, 0, 0))); d > 1e-9 {
		t.Errorf("arc does not start at pStart: off by %.6f", d)
	}
	if d := float64(end.DistanceTo(math.P3(r, r, 0))); d > 1e-9 {
		t.Errorf("arc end %v is not the expected quarter point (%.1f,%.1f,0)", end, r, r)
	}
	assertDir(t, pts[0].VectorTo(pts[1]), math.V3(0, 1, 0), "start tangent +Y")
	assertDir(t, pts[len(pts)-2].VectorTo(pts[len(pts)-1]), math.V3(1, 0, 0), "end tangent +X")
}

// assertDir checks that got points the same way as want (unit-dot ≈ 1).
func assertDir(t *testing.T, got, want math.Vector3, label string) {
	t.Helper()
	gu, err := math.UnitVector3FromVector(got)
	if err != nil {
		t.Errorf("%s: zero vector", label)
		return
	}
	wu, _ := math.UnitVector3FromVector(want)
	// A sampled chord sits half a segment angle (~7.5° at 6 samples) off the true tangent, so accept
	// anything within that of the intended direction.
	if d := float64(gu.AsVector().Dot(wu.AsVector())); d < 0.99 {
		t.Errorf("%s: direction dot %.4f (got %v, want %v)", label, d, got, want)
	}
}

// TestEndBendLipLiesInProfilePlane the first and last loft sections are the flat lips: they lie IN
// each profile's plane and sit outboard of the profile by the lip length, so the wall really gains a
// flat lip before it folds up.
func TestEndBendLipLiesInProfilePlane(t *testing.T) {
	up := math.V3(0, 0, 1).AsUnit()
	bandA := []math.Point3{math.P3(-1, -1, 0), math.P3(1, -1, 0), math.P3(1, 1, 0), math.P3(-1, 1, 0)}
	bandB := []math.Point3{math.P3(-2, -2, 3), math.P3(2, -2, 3), math.P3(2, 2, 3), math.P3(-2, 2, 3)}
	const r = 0.3
	sections := endBendSections(bandA, bandB, up, up, r, r, DieFormedLoftedFlange, 0)
	lipA := sections[0]
	for i, p := range lipA {
		if stdmath.Abs(float64(p.Z)) > 1e-9 {
			t.Errorf("lip A point %d is off plane A (z=%.4f, want 0)", i, float64(p.Z))
		}
		// The centroid of a symmetric square band is the origin; the lip sits farther out than the
		// profile by the lip length (= r).
		out := float64(math.P3(0, 0, 0).DistanceTo(math.P3(p.X, p.Y, 0)))
		prof := float64(math.P3(0, 0, 0).DistanceTo(math.P3(bandA[i].X, bandA[i].Y, 0)))
		if got := out - prof; stdmath.Abs(got-r) > 1e-6 {
			t.Errorf("lip A point %d flares %.4f, want the lip length %.4f", i, got, r)
		}
	}
	if z := float64(sections[len(sections)-1][0].Z); stdmath.Abs(z-3) > 1e-9 {
		t.Errorf("lip B is off plane B (z=%.4f, want 3)", z)
	}
}

// loftedFlangeWall builds a square-to-larger-square lofted flange with the given end-bend radius and
// returns its solid and feature. It gates on the SELF-INTERSECTION scan (CheckGeometry), not just
// topology — a rounded end bend can be topologically valid yet interpenetrate, which the live test
// caught and a topology-only check would miss.
func loftedFlangeWall(t *testing.T, radius float64) (*topo.Body, *PartFeature) {
	t.Helper()
	planeB, _ := sketch.NewPlane(math.P3(0, 0, 3), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	def := &SheetMetalLoftedFlangeDefinition{
		ProfileA: lProfileOnPlane(sketch.XYPlane(), 1, 1), ProfileB: lProfileOnPlane(planeB, 2, 2),
		Operation: ops.NewBody,
	}
	if radius > 0 {
		def.Radius = constFloat(radius)
	}
	fs := NewPartFeatures(thicknessParams(t))
	pf := NewSheetMetalLoftedFlangeFeatures(fs).Add(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("lofted flange (radius %.2f) sick: %+v", radius, pf.Health())
	}
	body := fs.Result()[0]
	if ok, probs := ops.ValidateBodyEntities(body, ops.CheckGeometry, ops.DefaultQuality()); !ok {
		t.Fatalf("lofted flange (radius %.2f) self-intersects: %d problem(s)", radius, len(probs))
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Errorf("lofted flange (radius %.2f) not watertight: %d boundary edges", radius, len(open))
	}
	return body, pf
}

// TestLoftedFlangeEndBendKeepsLipWhenItFits at a small radius the flat lip does not self-intersect,
// so the wall keeps it: the solid is self-intersection-free, reports no lip-dropped fallback, and
// flares OUTWARD past the sharp reference (the lip really extends the wall in the profile plane).
func TestLoftedFlangeEndBendKeepsLipWhenItFits(t *testing.T) {
	reach := func(b *topo.Body) float64 { box := b.RangeBox(); return float64(box.Min.DistanceTo(box.Max)) }
	sharp, _ := loftedFlangeWall(t, 0)
	body, pf := loftedFlangeWall(t, 0.05) // small enough that the lip fits the L-profile band
	if hasDiagCode(pf.Diagnostics(), codeLoftedFlangeLipDropped) {
		t.Errorf("a lip that fits must not be dropped: %v", pf.Diagnostics())
	}
	if !(reach(body) > reach(sharp)) {
		t.Errorf("the kept lip did not flare the wall past the sharp reference (%.4f vs %.4f)", reach(body), reach(sharp))
	}
}

// TestLoftedFlangeEndBendFallsBackOnTightRadius a large radius makes the flat lip self-intersect, so
// the wall must fall back to the fold alone — still a valid (self-intersection-free) solid — and
// report the lip-dropped diagnostic rather than ship an interpenetrating wall.
func TestLoftedFlangeEndBendFallsBackOnTightRadius(t *testing.T) {
	_, pf := loftedFlangeWall(t, 0.3) // fatals inside if the solid self-intersects
	if !hasDiagCode(pf.Diagnostics(), codeLoftedFlangeLipDropped) {
		t.Errorf("a self-intersecting lip must report %q so the fallback is visible: %v",
			codeLoftedFlangeLipDropped, pf.Diagnostics())
	}
	if hasDiagCode(pf.Diagnostics(), codeLoftedFlangeUnmodeled) {
		t.Errorf("the end-bend radius is modelled, not deferred: %v", pf.Diagnostics())
	}
}
