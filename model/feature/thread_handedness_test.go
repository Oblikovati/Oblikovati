// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Thread handedness (#1892). A left-hand thread is not a callout: for a CUT thread it is the
// direction the groove actually winds, and a part that models it right-handed will not mate.
// The designation could already carry it as an "-LH" suffix, so these tests also pin that the
// flag and the suffix agree instead of cancelling.

// threadedFaceOf returns the sole threaded-cylinder face of the part's result, failing if the
// cut thread did not retype exactly one face.
func threadedFaceOf(t *testing.T, fs *PartFeatures) geom.ThreadedCylinder {
	t.Helper()
	var found []geom.ThreadedCylinder
	for _, b := range fs.Result() {
		for _, f := range b.Faces() {
			if tc, ok := f.Geometry().(geom.ThreadedCylinder); ok {
				found = append(found, tc)
			}
		}
	}
	if len(found) != 1 {
		t.Fatalf("got %d threaded faces on the result, want exactly 1", len(found))
	}
	return found[0]
}

// threadedPart builds a cylinder with one thread on its side face.
func threadedPart(t *testing.T, def *ThreadDefinition) *PartFeatures {
	t.Helper()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5, 2.0)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(cyl)
	def.FaceKey = cylinderFaceKey(t, cyl)
	th := NewDressUpFeatures(fs).AddThreadDef(def)
	fs.Recompute()
	if !th.Health().OK() {
		t.Fatalf("thread went sick: %+v", th.Health())
	}
	return fs
}

// TestLeftHandedCutThreadReversesTheGroove: the flag has to reach the modeled surface, not just
// the spec — a cut thread whose groove still winds right is the wrong solid.
func TestLeftHandedCutThreadReversesTheGroove(t *testing.T) {
	if tc := threadedFaceOf(t, threadedPart(t, &ThreadDefinition{Designation: "M8x1.25", Cut: true})); !tc.RightHanded {
		t.Error("the default cut thread should wind right-handed")
	}
	tc := threadedFaceOf(t, threadedPart(t, &ThreadDefinition{Designation: "M8x1.25", Cut: true, LeftHanded: true}))
	if tc.RightHanded {
		t.Error("leftHanded cut thread still models a right-hand groove")
	}
	if tc.Pitch <= 0 || tc.Depth <= 0 {
		t.Errorf("reversing the sense damaged the groove: pitch %g depth %g", tc.Pitch, tc.Depth)
	}
}

// helixTurnSign reports which way a display helix winds: +1 when the polar angle advances with
// the axial coordinate (right-handed about +Z), −1 when it retreats.
func helixTurnSign(pts []math.Point3) float64 {
	a0 := stdmath.Atan2(float64(pts[0].Y), float64(pts[0].X))
	a1 := stdmath.Atan2(float64(pts[1].Y), float64(pts[1].X))
	d := a1 - a0
	for d > stdmath.Pi { // the samples are far denser than half a turn, so this unwraps cleanly
		d -= 2 * stdmath.Pi
	}
	for d < -stdmath.Pi {
		d += 2 * stdmath.Pi
	}
	if d < 0 {
		return -1
	}
	return 1
}

// TestLeftHandedCosmeticThreadMirrorsItsHelix: a cosmetic thread leaves the solid alone, so its
// handedness is only ever visible in what it DRAWS. A helix that keeps winding right is the
// drawing telling the machinist the opposite of the spec.
func TestLeftHandedCosmeticThreadMirrorsItsHelix(t *testing.T) {
	right := ThreadDisplayCurves(threadedPart(t, &ThreadDefinition{Designation: "M8x1.25"}))
	left := ThreadDisplayCurves(threadedPart(t, &ThreadDefinition{Designation: "M8x1.25", LeftHanded: true}))
	if len(right) != 1 || len(left) != 1 {
		t.Fatalf("got %d/%d display curves, want 1 each", len(right), len(left))
	}
	if s := helixTurnSign(right[0]); s != 1 {
		t.Errorf("right-hand display helix winds %g, want +1", s)
	}
	if s := helixTurnSign(left[0]); s != -1 {
		t.Errorf("left-hand display helix winds %g, want −1", s)
	}
}

// TestLeftHandedFlagAndSuffixAgree: "-LH" in the designation and the flag are two spellings of
// one fact. Setting both must not cancel back to a right-hand thread, and the suffix alone must
// still work now that the flag exists.
func TestLeftHandedFlagAndSuffixAgree(t *testing.T) {
	for name, def := range map[string]*ThreadDefinition{
		"suffix only": {Designation: "M8x1.25-LH"},
		"flag only":   {Designation: "M8x1.25", LeftHanded: true},
		"both":        {Designation: "M8x1.25-LH", LeftHanded: true},
	} {
		t.Run(name, func(t *testing.T) {
			fs := threadedPart(t, def)
			spec := fs.Item(1).Definition().(*ThreadFeature).Spec()
			if spec == nil || spec.RightHanded {
				t.Errorf("%s: spec = %+v, want a left-hand thread", name, spec)
			}
		})
	}
}

// TestThreadHandednessRoundTrips: handedness is part of the thread, so a reopened document must
// not quietly serve a right-hand thread where a left-hand one was authored.
func TestThreadHandednessRoundTrips(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewDressUpFeatures(fs).AddThreadDef(&ThreadDefinition{
		FaceKey: []byte("face-1"), Designation: "M8x1.25", Cut: true, LeftHanded: true,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if !data[0].Thread.LeftHanded {
		t.Fatalf("serialized thread = %+v, want leftHanded carried", data[0].Thread)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if !fresh.Item(0).Definition().(*ThreadFeature).Definition().LeftHanded {
		t.Error("restored thread lost its handedness")
	}
	// A document written before the option existed has no such key, and must read back as the
	// ordinary right-hand thread it was — not as the zero value of a "handedness" name.
	legacy := []FeatureData{{Kind: "thread", Thread: &ThreadData{Face: "ZmFjZQ==", Designation: "M6x1"}}}
	old := NewPartFeatures(nil)
	if err := old.ApplyRecipe(legacy, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe(legacy): %v", err)
	}
	if old.Item(0).Definition().(*ThreadFeature).Definition().LeftHanded {
		t.Error("a legacy thread restored left-handed")
	}
}
