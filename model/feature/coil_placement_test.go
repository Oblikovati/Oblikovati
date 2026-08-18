// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// placedCoil builds a coil over the square x∈[4,5], y∈[0,1] about the Y axis and recomputes it,
// letting the caller set the flavour options on the definition first.
func placedCoil(t *testing.T, revolutions float64, tune func(*CoilDefinition)) (*PartFeatures, *PartFeature) {
	t.Helper()
	fs := NewPartFeatures(nil)
	def := &CoilDefinition{
		Sketch: offsetSquareSketch(4, 1), ProfileIndex: 0, Axis: yAxis(),
		Pitch: func() float64 { return 2 }, Revolutions: func() float64 { return revolutions },
		Operation: ops.NewBody,
	}
	tune(def)
	pf := NewCoilFeatures(fs).AddDefinition(def)
	fs.Recompute()
	return fs, pf
}

// solidCoilBody asserts the coil built a valid solid and returns it.
func solidCoilBody(t *testing.T, fs *PartFeatures, pf *PartFeature, what string) *topo.Body {
	t.Helper()
	if !pf.Health().OK() {
		t.Fatalf("%s went sick: %+v", what, pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("%s is not a valid solid: %+v", what, r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 0 {
		t.Fatalf("%s volume = %g, want > 0", what, v)
	}
	return body
}

// TestLeftHandedCoilWindsOppositeToRight: handedness is observable on a PARTIAL sweep. The
// profile starts on +X and the axis is +Y, so a quarter turn by the right-hand rule about +Y
// carries it to −Z (R_y(90°): (x,0,0) → (0,0,−x)); the left-handed coil must reach +Z instead,
// while climbing the same amount. A full revolution would hide this — both cover the whole
// circle — which is why the sweep is a quarter turn (#1883).
func TestLeftHandedCoilWindsOppositeToRight(t *testing.T) {
	rightFS, rightPF := placedCoil(t, 0.25, func(*CoilDefinition) {})
	leftFS, leftPF := placedCoil(t, 0.25, func(d *CoilDefinition) { d.Handedness = LeftHandedCoil })
	right := solidCoilBody(t, rightFS, rightPF, "right-handed coil")
	left := solidCoilBody(t, leftFS, leftPF, "left-handed coil")

	rb, lb := right.RangeBox(), left.RangeBox()
	if relErr(float64(rb.Min.Z), -5) > 0.05 || float64(rb.Max.Z) > 0.1 {
		t.Errorf("right-handed quarter turn spans Z [%g, %g], want ≈[-5, 0] (winds toward -Z)",
			rb.Min.Z, rb.Max.Z)
	}
	if relErr(float64(lb.Max.Z), 5) > 0.05 || float64(lb.Min.Z) < -0.1 {
		t.Errorf("left-handed quarter turn spans Z [%g, %g], want ≈[0, 5] (winds toward +Z)",
			lb.Min.Z, lb.Max.Z)
	}
	// Mirroring the winding must not change the amount of material or the climb.
	rv := ops.BodyGeometryProperties(right, ops.DefaultQuality()).Volume
	lv := ops.BodyGeometryProperties(left, ops.DefaultQuality()).Volume
	if relErr(lv, rv) > 0.01 {
		t.Errorf("left-handed volume %g != right-handed %g — handedness must only mirror", lv, rv)
	}
	if relErr(float64(lb.Max.Y-lb.Min.Y), float64(rb.Max.Y-rb.Min.Y)) > 0.01 {
		t.Errorf("left-handed axial span %g != right-handed %g — handedness must not change the rise",
			lb.Max.Y-lb.Min.Y, rb.Max.Y-rb.Min.Y)
	}
}

// TestSpiralCoilGrowsRadiallyWithoutRise: a spiral spends the rail's pitch integral on RADIUS,
// not height (#1883). Profile x∈[4,5], pitch 2, 3 turns → the outer edge reaches 5 + 2·3 = 11,
// and the coil stays as thin axially as the profile itself (1).
func TestSpiralCoilGrowsRadiallyWithoutRise(t *testing.T) {
	fs, pf := placedCoil(t, 3, func(d *CoilDefinition) { d.Spiral = true })
	body := solidCoilBody(t, fs, pf, "spiral coil")

	b := body.RangeBox()
	if got := float64(b.Max.X); relErr(got, 11) > 0.05 {
		t.Errorf("spiral outer radius = %g, want ≈11 (start 5 + pitch 2 × 3 turns)", got)
	}
	if got := float64(b.Max.Y - b.Min.Y); relErr(got, 1) > 0.05 {
		t.Errorf("spiral axial span = %g, want ≈1 (the profile's own height — a spiral has no rise)", got)
	}
}

// TestSpiralCoilRefusesHeightAndTaper: both options describe an axial rise a spiral does not
// have, so they are refused rather than silently discarded — the failure a caller notices last.
func TestSpiralCoilRefusesHeightAndTaper(t *testing.T) {
	_, withHeight := placedCoil(t, 3, func(d *CoilDefinition) {
		d.Spiral = true
		d.Height = func() float64 { return 30 }
	})
	if withHeight.Health().OK() {
		t.Error("a spiral with a height should go sick: there is no axial rise for it to describe")
	}
	_, withTaper := placedCoil(t, 3, func(d *CoilDefinition) {
		d.Spiral = true
		d.Taper = 0.1
	})
	if withTaper.Health().OK() {
		t.Error("a spiral with a taper should go sick: taper scales the radius with the rise, which is zero")
	}
}

// TestCoilHandednessAndSpiralRoundTrip: both flavour options survive the recipe round-trip, and
// a document written before they existed (no keys) reads back as a right-handed helix.
func TestCoilHandednessAndSpiralRoundTrip(t *testing.T) {
	work := NewWorkGeometry()
	axis, ok := work.AxisByRef(OriginYAxis)
	if !ok {
		t.Fatal("origin Y axis missing from fresh work geometry")
	}
	sk := offsetSquareSketch(4, 1)
	def := &CoilDefinition{
		Sketch: sk, Axis: axis, Operation: ops.NewBody,
		Pitch: func() float64 { return 2 }, Revolutions: func() float64 { return 3 },
		Handedness: LeftHandedCoil, Spiral: true,
	}
	index := sketchList{sks: []*sketch.Sketch{sk}}
	data, err := serializeCoil(def, index)
	if err != nil {
		t.Fatalf("serializeCoil: %v", err)
	}
	if !data.LeftHanded || !data.Spiral {
		t.Fatalf("persisted coil = leftHanded %v spiral %v, want both true", data.LeftHanded, data.Spiral)
	}
	restored, err := restoreCoil(NewPartFeatures(nil), data, index, work)
	if err != nil {
		t.Fatalf("restoreCoil: %v", err)
	}
	rdef := restored.Definition().(*CoilFeature).Definition()
	if rdef.Handedness != LeftHandedCoil || !rdef.Spiral {
		t.Errorf("restored coil = handedness %v spiral %v, want LeftHandedCoil true",
			rdef.Handedness, rdef.Spiral)
	}

	data.LeftHanded, data.Spiral = false, false // a pre-#1883 document carries neither key
	old, err := restoreCoil(NewPartFeatures(nil), data, index, work)
	if err != nil {
		t.Fatalf("restoreCoil (legacy): %v", err)
	}
	if odef := old.Definition().(*CoilFeature).Definition(); odef.Handedness != RightHandedCoil || odef.Spiral {
		t.Errorf("legacy coil = handedness %v spiral %v, want RightHandedCoil false",
			odef.Handedness, odef.Spiral)
	}
}
