// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// Loft FACE-section matrix (Slice 2c): a section can be an existing body face, and Tangent/Smooth
// make the loft leave that face tangent to its surface. For a planar source face this is exact G1
// continuity — the loft flares in the face plane rather than ruling straight to the next section.

// topPlanarFaceKey returns the reference key of the highest planar face of a body (its top cap).
func topPlanarFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	var key []byte
	bestZ := -1e30
	for _, f := range b.Faces() {
		bb := f.RangeBox()
		if float64(bb.Max.Z-bb.Min.Z) > 1e-6 {
			continue // not horizontal-planar
		}
		if zc := float64(bb.Min.Z+bb.Max.Z) / 2; zc > bestZ {
			bestZ, key = zc, f.ReferenceKey()
		}
	}
	if key == nil {
		t.Fatal("no top planar face found")
	}
	return key
}

// loftFromCylinderTop extrudes a cylinder (r=2, h=3), then lofts from its top face up to a small
// circle (r=1, z=6) with the given start (face) condition, returning the lofted body (asserted a
// valid solid). The loft is a new body so the assertion isolates the loft shape from the coplanar
// JOIN-union path.
func loftFromCylinderTop(t *testing.T, faceEnd LoftEnd) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(circleOn(sketch.XYPlane(), 2), 0, ops.NewBody, func() float64 { return 3 })
	fs.Recompute()
	key := topPlanarFaceKey(t, fs.Result()[0])

	pf := NewLoftFeatures(fs).addConditioned(
		[]LoftSection{{FaceKey: key}, sec(circleOn(planeAtZ(6), 1))},
		false, ops.NewBody, faceEnd, LoftEnd{},
	)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("face-section loft went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	b := bodies[len(bodies)-1] // the loft body (after the cylinder)
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("face-section loft not a valid solid: valid=%v solid=%v issues=%v", r.Valid, b.IsSolid(), capIssues3(r.Issues))
	}
	return b
}

// TestLoftFaceFreeIsRuled: a Free loft from the cylinder top is a straight frustum — its max
// radius equals the face radius (no flare). The baseline tangent beats.
func TestLoftFaceFreeIsRuled(t *testing.T) {
	t.Parallel()
	b := loftFromCylinderTop(t, LoftEnd{})
	if maxX := float64(b.RangeBox().Max.X); maxX > 2.03 {
		t.Errorf("Free face loft flared: max x = %.3f, want ~2.0 (ruled)", maxX)
	}
}

// TestLoftFaceTangentFlares: a Tangent condition leaves the planar top face tangent to its plane,
// so the loft flares OUT past the face radius (a smooth trumpet continuation) — exact G1.
func TestLoftFaceTangentFlares(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	b := loftFromCylinderTop(t, LoftEnd{Condition: LoftTangent})
	if maxX := float64(b.RangeBox().Max.X); maxX < 2.15 {
		t.Errorf("Tangent face loft did not flare: max x = %.3f, want > 2.15 (ruled would be 2.0)", maxX)
	}
}

// TestLoftFaceSmoothIsRealCurvature (M36-F06): Smooth (G2) now imposes the adjacent face's real
// curvature via the quintic end blend — it is no longer an alias of Tangent (G1), so it produces a
// genuinely different, valid solid (against the planar cap it leaves with zero curvature, so the
// takeoff stays in-plane longer than G1's cubic). The numeric curvature-continuity proof against a
// CURVED face is TestLoftG2MatchesFaceCurvature.
func TestLoftFaceSmoothIsRealCurvature(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~12s): `make test-corpus`")
	}
	t.Parallel()
	tan := query.BodyGeometryProperties(loftFromCylinderTop(t, LoftEnd{Condition: LoftTangent}), ops.DefaultQuality()).Volume
	smooth := query.BodyGeometryProperties(loftFromCylinderTop(t, LoftEnd{Condition: LoftSmooth}), ops.DefaultQuality()).Volume
	if relErr(tan, smooth) < 1e-3 {
		t.Errorf("Smooth (%.4f) should differ from Tangent (%.4f) now that G2 imposes real curvature, not an alias", smooth, tan)
	}
}

// TestLoftFaceImpactScalesFlare: a larger impact flares the tangent continuation more.
func TestLoftFaceImpactScalesFlare(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~7s): `make test-corpus`")
	}
	t.Parallel()
	soft := float64(loftFromCylinderTop(t, LoftEnd{Condition: LoftTangent, Impact: 1}).RangeBox().Max.X)
	hard := float64(loftFromCylinderTop(t, LoftEnd{Condition: LoftTangent, Impact: 2}).RangeBox().Max.X)
	if hard <= soft {
		t.Errorf("higher impact did not flare more: impact1 max x = %.3f, impact2 = %.3f", soft, hard)
	}
}

// TestLoftFaceSectionRoundTrip: a face section (FaceKey + a Tangent condition) survives a recipe
// save/restore — the key and condition come back so a reopened .obk rebuilds the tangent loft.
func TestLoftFaceSectionRoundTrip(t *testing.T) {
	t.Parallel()
	top := centeredSquareOn(planeAtZ(6), 1)
	idx := sketchList{sks: []*sketch.Sketch{top}}
	fs := NewPartFeatures(nil)
	key := []byte("face/abc123")
	NewLoftFeatures(fs).addConditioned(
		[]LoftSection{{FaceKey: key}, {Sketch: top, ProfileIndex: 0}},
		false, ops.NewBody, LoftEnd{Condition: LoftTangent, Impact: 1.5}, LoftEnd{},
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
	if string(got.Sections[0].FaceKey) != string(key) {
		t.Errorf("face key round-trip = %q, want %q", got.Sections[0].FaceKey, key)
	}
	if got.First.Condition != LoftTangent || got.First.Impact != 1.5 {
		t.Errorf("face condition round-trip = %+v, want Tangent impact 1.5", got.First)
	}
}

// TestLoftFaceLostReference: a face key that no body carries fails cleanly (not a panic).
func TestLoftFaceLostReference(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewLoftFeatures(fs).addConditioned(
		[]LoftSection{{FaceKey: []byte("does-not-exist")}, sec(circleOn(planeAtZ(6), 1))},
		false, ops.NewBody, LoftEnd{Condition: LoftTangent}, LoftEnd{},
	)
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("loft with a lost face reference should go sick, not succeed")
	}
}
