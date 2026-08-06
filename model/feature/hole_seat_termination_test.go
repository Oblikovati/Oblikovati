// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The hole's seat/tap/clearance axes (#1862) and its terminations (#1863).

// TestClearanceHoleIsSizedByItsFastener is the point of a clearance hole: the FASTENER is the
// authored thing and the diameter follows it, so swapping the screw resizes the bore. Recording the
// resolved diameter — which is what an import used to do — breaks exactly that.
func TestClearanceHoleIsSizedByItsFastener(t *testing.T) {
	for _, tc := range []struct {
		fastener, fit string
		wantCM        float64
	}{
		{"M6", "close", 0.64}, {"M6", "medium", 0.66}, {"M6", "free", 0.70},
		{"M6", "", 0.66},       // medium is the default fit
		{"M8", "close", 0.84},  // the same fit on a bigger fastener
		{"M22", "medium", 2.4}, // a size outside the ISO coarse-pitch table still resolves
	} {
		got, err := HoleClearanceInfo{Fastener: tc.fastener, Fit: tc.fit}.ClearanceDiameter()
		if err != nil {
			t.Fatalf("%s %q: %v", tc.fastener, tc.fit, err)
		}
		if stdmath.Abs(got-tc.wantCM) > 1e-9 {
			t.Errorf("%s %q clearance = %g cm, want %g (ISO 273)", tc.fastener, tc.fit, got, tc.wantCM)
		}
	}
}

// TestClearanceHoleRefusesWhatItCannotSize: a clearance hole that quietly fell back to some other
// diameter would produce a part that does not assemble, so every unknown input is named.
func TestClearanceHoleRefusesWhatItCannotSize(t *testing.T) {
	for name, info := range map[string]HoleClearanceInfo{
		"unknown standard": {Standard: "ANSI B18.2.8", Fastener: "M6"},
		"unknown fit":      {Fastener: "M6", Fit: "sliding"},
		"unknown size":     {Fastener: "M7"},
		"unparseable":      {Fastener: "sixmil"},
	} {
		if d, err := info.ClearanceDiameter(); err == nil {
			t.Errorf("%s resolved to Ø%g cm; it must be refused", name, d)
		}
	}
}

// TestClearanceHoleDrivesTheBore wires the table through the feature: the definition carries the
// fastener, not a diameter, and the cut comes out at the table size.
func TestClearanceHoleDrivesTheBore(t *testing.T) {
	block, top := holeBlock()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 0 }, func() float64 { return 1 })
	hole.Definition().(*HoleFeature).Definition().Clearance = HoleClearanceInfo{Fastener: "M6", Fit: "free"}
	fs.Recompute()

	if !hole.Health().OK() {
		t.Fatalf("clearance hole sick: %+v", hole.Health())
	}
	want := 32 - holeCylinderArea(0.7/2)*1 // ISO 273 free fit for M6 is Ø7 mm
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("clearance-bored volume = %g, want %g (Ø7 mm free fit, not the zero Diameter)", got, want)
	}
}

// TestSpotFaceCutsTheSeatWithoutBecomingACounterbore: a spotface is the same flat-bottomed recess a
// counterbore cuts, so it must remove the same material — while staying a DISTINCT seat type, since
// collapsing it into a counterbore on import loses the callout.
func TestSpotFaceCutsTheSeatWithoutBecomingACounterbore(t *testing.T) {
	spot := seatedHoleVolume(t, SpotFaceHole)
	bore := seatedHoleVolume(t, CounterboreHole)
	if stdmath.Abs(spot-bore) > 1e-9 {
		t.Errorf("spotface removed %g, counterbore %g — the same flat-bottomed seat must cut the same", spot, bore)
	}
	if SpotFaceHole == CounterboreHole {
		t.Error("SpotFaceHole and CounterboreHole are the same value; the seats must stay distinguishable")
	}
}

// seatedHoleVolume drills a Ø1 × 1.5 bore with a Ø2 × 0.5 seat of the given type and returns the
// remaining volume.
func seatedHoleVolume(t *testing.T, seat HoleType) float64 {
	t.Helper()
	block, top := holeBlock()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddCounterbore(top,
		func() float64 { return 1 }, func() float64 { return 1.5 },
		func() float64 { return 2 }, func() float64 { return 0.5 })
	hole.Definition().(*HoleFeature).Definition().Type = seat
	fs.Recompute()
	if !hole.Health().OK() {
		t.Fatalf("seat %v sick: %+v", seat, hole.Health())
	}
	return ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume
}

// TestTapIsOrthogonalToTheSeat: in Inventor the seat (drilled/counterbore/countersink/spotface) and
// the tap are two independent axes, so a counterbored TAPPED hole is an ordinary thing. Modelling
// "tapped" as a seat made it impossible to say.
func TestTapIsOrthogonalToTheSeat(t *testing.T) {
	block, top := holeBlock()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddCounterbore(top,
		func() float64 { return 1 }, func() float64 { return 1.5 },
		func() float64 { return 2 }, func() float64 { return 0.5 })
	def := hole.Definition().(*HoleFeature).Definition()
	def.Tap = HoleTapInfo{Tapped: true, Designation: "M10x1.5", Class: "6H", LeftHanded: true}
	fs.Recompute()

	if !hole.Health().OK() {
		t.Fatalf("tapped counterbore sick: %+v", hole.Health())
	}
	if def.Type != CounterboreHole || !def.Tap.Tapped {
		t.Errorf("seat=%v tapped=%v — a counterbore must stay a counterbore while ALSO being tapped", def.Type, def.Tap.Tapped)
	}
	if !def.Tap.LeftHanded || def.Tap.Class != "6H" {
		t.Errorf("tap = %+v, want the class and handedness carried alongside the designation", def.Tap)
	}
}

// TestZeroTapIsRightHanded pins the field's naming decision: a definition built as a literal — which
// several call sites and every restored recipe do — must come out an ORDINARY right-hand thread.
func TestZeroTapIsRightHanded(t *testing.T) {
	if (HoleTapInfo{Tapped: true, Designation: "M6"}).LeftHanded {
		t.Error("a zero-valued tap is left-handed; the ordinary thread must be the zero value")
	}
}

// TestHoleTerminatesOnAFace: the bore runs from the placement face down to a named plane, so the
// depth comes from the model instead of being measured by hand into the args.
func TestHoleTerminatesOnAFace(t *testing.T) {
	got := terminatedHoleVolume(t, func(d *HoleDefinition) {
		d.Termination, d.ToPlane = ToFaceExtent, holeStopPlane(1.25)
	})
	want := 32 - holeCylinderArea(1)*0.75 // from the z=2 face down to z=1.25
	if stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("to-face bore left %g, want %g (0.75 deep, measured to the plane)", got, want)
	}
}

// TestHoleTerminatesBetweenTwoFaces: from-to moves the bore's START too, which no depth alone can
// express — the hole begins at the from-plane, not at the face it was placed on.
func TestHoleTerminatesBetweenTwoFaces(t *testing.T) {
	got := terminatedHoleVolume(t, func(d *HoleDefinition) {
		d.Termination = FromToExtent
		d.FromPlane, d.ToPlane = holeStopPlane(1.5), holeStopPlane(0.5)
	})
	want := 32 - holeCylinderArea(1)*1.0 // an internal slot from z=1.5 down to z=0.5
	if stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("from-to bore left %g, want %g (1.0 deep, starting below the placement face)", got, want)
	}
}

// TestHoleTerminatorMustBeSquareToTheDrill is the honest-limitation guard: a bore bottoms at ONE
// depth, so a slanted terminator has no single answer and is refused rather than approximated.
func TestHoleTerminatorMustBeSquareToTheDrill(t *testing.T) {
	block, top := holeBlock()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 1 })
	def := hole.Definition().(*HoleFeature).Definition()
	def.Termination, def.ToPlane = ToFaceExtent, NewFixedWorkPlane(sketch.XZPlane())
	fs.Recompute()

	if hole.Health().OK() {
		t.Fatal("a terminator parallel to the drill axis was accepted; it names no single depth")
	}
	if reason := hole.Health().Reason; !strings.Contains(reason, "square") {
		t.Errorf("reason = %q, want it to say the terminator is not square to the drill axis", reason)
	}
}

// terminatedHoleVolume drills a Ø2 hole into the block under the given termination and returns the
// remaining volume.
func terminatedHoleVolume(t *testing.T, set func(*HoleDefinition)) float64 {
	t.Helper()
	block, top := holeBlock()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 0 })
	set(hole.Definition().(*HoleFeature).Definition())
	fs.Recompute()
	if !hole.Health().OK() {
		t.Fatalf("terminated hole sick: %+v", hole.Health())
	}
	return ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume
}

// holeStopPlane is a plane parallel to the block's top face at height z — square to a bore drilled
// down the −Z axis, which is what a terminator must be.
func holeStopPlane(z float64) *WorkPlane {
	xAxis, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	yAxis, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	pl, _ := sketch.NewPlane(math.P3(0, 0, math.Scalar(z)), xAxis, yAxis)
	return NewFixedWorkPlane(pl)
}
