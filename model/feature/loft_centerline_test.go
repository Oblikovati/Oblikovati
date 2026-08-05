// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Loft CENTERLINE matrix (Slice 4, kLoftWithCenterline): a spine curve the section centroids
// follow, so the WHOLE loft bends along it (unlike a rail, which pulls one side). The centerline
// touches the end-section centroids; the cross-sections are preserved as the spine bends.

// centerlineThroughX returns a spine (provider) for a radius-2 loft (z 0→4) that bows to x=midX at
// mid height — a banana bend.
func centerlineThroughX(midX float64) func() []math.Point3 {
	return func() []math.Point3 {
		return []math.Point3{math.P3(0, 0, 0), math.P3(math.Scalar(midX), 0, 2), math.P3(0, 0, 4)}
	}
}

// centerlinedCircles lofts two equal circles (r=2, z 0 and 4) along the given centerline,
// asserting a valid solid and returning it.
func centerlinedCircles(t *testing.T, cl func() []math.Point3) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	secs := []LoftSection{sec(circleOn(sketch.XYPlane(), 2)), sec(circleOn(planeAtZ(4), 2))}
	pf := NewLoftFeatures(fs).AddGuided(secs, false, ops.NewBody, LoftEnd{}, LoftEnd{}, LoftGuideSet{Centerline: cl})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("centerlined loft went sick: %+v", pf.Health())
	}
	b := fs.Result()[0]
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("centerlined loft not a valid solid: valid=%v solid=%v issues=%v", r.Valid, b.IsSolid(), capIssues3(r.Issues))
	}
	return b
}

// TestLoftCenterlineBendsSpine: a centerline bowing to x=2 shifts the loft's mass to +X (its
// centroid moves well off the axis) — the spine bends — whereas a straight loft is centred.
func TestLoftCenterlineBendsSpine(t *testing.T) {
	straight := ops.BodyGeometryProperties(centerlinedCircles(t, centerlineThroughX(0)), ops.DefaultQuality())
	bent := ops.BodyGeometryProperties(centerlinedCircles(t, centerlineThroughX(2)), ops.DefaultQuality())
	if float64(straight.Centroid.X) < -0.05 || float64(straight.Centroid.X) > 0.05 {
		t.Errorf("straight-centerline loft is off-axis: centroid x = %.3f, want ≈0", float64(straight.Centroid.X))
	}
	if float64(bent.Centroid.X) < 0.5 {
		t.Errorf("centerline did not bend the spine: centroid x = %.3f, want > 0.5", float64(bent.Centroid.X))
	}
}

// TestLoftCenterlinePreservesVolume: bending the spine keeps the cross-sections, so the bent loft
// holds about the same volume as the straight one (a rail, by contrast, changes the volume).
func TestLoftCenterlinePreservesVolume(t *testing.T) {
	straight := ops.BodyGeometryProperties(centerlinedCircles(t, centerlineThroughX(0)), ops.DefaultQuality()).Volume
	bent := ops.BodyGeometryProperties(centerlinedCircles(t, centerlineThroughX(2)), ops.DefaultQuality()).Volume
	if relErr(bent, straight) > 0.15 {
		t.Errorf("spine bend changed the volume too much: bent %.3f vs straight %.3f", bent, straight)
	}
}

// TestLoftCenterlineKeepsEnds: the centerline doesn't move the end sections — the body still spans
// the section heights (z 0→4).
func TestLoftCenterlineKeepsEnds(t *testing.T) {
	bb := centerlinedCircles(t, centerlineThroughX(2)).RangeBox()
	if z0, z1 := float64(bb.Min.Z), float64(bb.Max.Z); z0 < -1e-6 || z0 > 1e-6 || z1 < 4-1e-6 || z1 > 4+1e-6 {
		t.Errorf("centerline moved the loft ends in z: span [%.4f,%.4f], want [0,4]", z0, z1)
	}
}

// TestLoftCenterlineRoundTrip: a centerlined loft's spine polyline survives a recipe save/restore,
// and the restored loft reports the kLoftWithCenterline type.
func TestLoftCenterlineRoundTrip(t *testing.T) {
	bottom := circleOn(sketch.XYPlane(), 2)
	top := circleOn(planeAtZ(4), 2)
	idx := sketchList{sks: []*sketch.Sketch{bottom, top}}
	fs := NewPartFeatures(nil)
	NewLoftFeatures(fs).AddGuided(
		[]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}},
		false, ops.NewBody, LoftEnd{}, LoftEnd{},
		LoftGuideSet{Centerline: centerlineThroughX(2)},
	)
	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*LoftFeature).Definition()
	if got.Centerline == nil {
		t.Fatal("centerline round-trip lost the spine")
	}
	if pts := got.Centerline(); len(pts) != 3 || float64(pts[1].X) < 1.9 || float64(pts[1].X) > 2.1 {
		t.Errorf("centerline polyline round-trip lost the bow: %v", pts)
	}
	if got.LoftType() != "centerline" {
		t.Errorf("LoftType = %q, want \"centerline\"", got.LoftType())
	}
}
