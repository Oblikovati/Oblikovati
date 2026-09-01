// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Body-mode mirror, remove-original and the join operation (Oblikovati#1890).

// TestMirrorOfBodyReflectsTheWholeSolid checks body mode does not consult the recipe: it reflects
// the running solid, which is how a symmetric part is normally built.
func TestMirrorOfBodyReflectsTheWholeSolid(t *testing.T) {
	t.Parallel()
	fs, mirror := mirrorFixture(t)
	mirror.Definition().OfBody = true
	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 2 {
		t.Fatalf("body mirror produced %d bodies, want 2 (the original and its reflection)", len(bodies))
	}
	// The cube spans x∈[0,1]; reflecting in the YZ plane through the origin puts the copy at
	// x∈[-1,0].
	centres := centresX(t, fs)
	if want := []float64{-0.5, 0.5}; !floatsNear(centres, want) {
		t.Fatalf("mirrored centres = %v, want %v", centres, want)
	}
}

// TestMirroredBodyKeepsPositiveVolume is the correctness guard on the reflection itself. A
// reflection has a negative determinant and turns a solid inside out; if the face senses are not
// flipped, the copy tessellates inward and its volume comes out negative.
func TestMirroredBodyKeepsPositiveVolume(t *testing.T) {
	t.Parallel()
	fs, mirror := mirrorFixture(t)
	mirror.Definition().OfBody = true
	fs.Recompute()
	for i, b := range fs.Result() {
		if r := ops.Validate(b); !r.Valid || !b.IsSolid() || !r.Manifold {
			t.Fatalf("body %d is not a valid manifold solid: %+v", i, r)
		}
		v := query.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
		if stdmath.Abs(v-1) > 1e-9 {
			t.Fatalf("body %d volume = %g, want +1 (a reflected solid must not be inside out)", i, v)
		}
	}
}

// TestMirrorRemoveOriginalKeepsOnlyTheReflection covers the handed-part case.
func TestMirrorRemoveOriginalKeepsOnlyTheReflection(t *testing.T) {
	t.Parallel()
	fs, mirror := mirrorFixture(t)
	mirror.Definition().OfBody = true
	mirror.Definition().RemoveOriginal = true
	fs.Recompute()
	if n := len(fs.Result()); n != 1 {
		t.Fatalf("removeOriginal left %d bodies, want 1", n)
	}
	if got := centresX(t, fs); !floatsNear(got, []float64{-0.5}) {
		t.Fatalf("surviving centre = %v, want [-0.5] (the reflection, not the original)", got)
	}
}

// TestMirrorJoinUnionsTheHalves checks the join operation makes ONE solid of the two halves rather
// than two that touch — the volume must be the sum, in a single body.
func TestMirrorJoinUnionsTheHalves(t *testing.T) {
	t.Parallel()
	fs, mirror := mirrorFixture(t)
	mirror.Definition().OfBody = true
	mirror.Definition().JoinToOriginal = true
	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("join produced %d bodies, want 1 united solid", len(bodies))
	}
	if r := ops.Validate(bodies[0]); !r.Valid || !bodies[0].IsSolid() || !r.Manifold {
		t.Fatalf("joined body is not a valid manifold solid: %+v", r)
	}
	v := query.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(v-2) > 1e-6 {
		t.Fatalf("joined volume = %g, want 2 (both halves in one solid)", v)
	}
}

// TestMirrorRefusesBodyOnlyOptionsInFeatureMode pins the contract's restriction. Accepting
// removeOriginal in feature mode and ignoring it would hand back a symmetric part where a handed
// one was asked for — the failure the caller would notice last.
func TestMirrorRefusesBodyOnlyOptionsInFeatureMode(t *testing.T) {
	t.Parallel()
	if err := ValidateMirrorMode(false, true, false); err == nil {
		t.Error("removeOriginal in feature mode should be refused")
	}
	if err := ValidateMirrorMode(false, false, true); err == nil {
		t.Error("a join operation in feature mode should be refused")
	}
	if err := ValidateMirrorMode(true, true, true); err != nil {
		t.Errorf("both options are valid in body mode, got %v", err)
	}
	if err := ValidateMirrorMode(false, false, false); err != nil {
		t.Errorf("plain feature mode should be accepted, got %v", err)
	}
}

// TestMirrorFeatureModeStillReplicatesFeatures guards the pre-#1890 behaviour: the default mode
// must keep re-applying the source features, not silently become a body mirror.
func TestMirrorFeatureModeStillReplicatesFeatures(t *testing.T) {
	t.Parallel()
	fs, mirror := mirrorFixture(t)
	if mirror.Definition().OfBody {
		t.Fatal("feature mode must be the default")
	}
	fs.Recompute()
	if n := len(fs.Result()); n != 2 {
		t.Fatalf("feature mirror produced %d bodies, want 2", n)
	}
}

// mirrorFixture is a unit cube at x∈[0,1] with a mirror across the YZ plane through the origin.
func mirrorFixture(t *testing.T) (*PartFeatures, *MirrorFeature) {
	t.Helper()
	fs := NewPartFeatures(nil)
	src := NewBaseFeatures(fs).AddBase(prismBody())
	m := NewPatternFeatures(fs).AddMirror([]ID{src.ID()}, nil, math.P3(0, 0, 0), math.V3(1, 0, 0))
	return fs, m
}

// TestMirrorBodyModeRoundTrip keeps the mode and both body-only options across an .obk
// save/load, so a handed part reopens handed (#1890).
func TestMirrorBodyModeRoundTrip(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	src := NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	m := NewPatternFeatures(fs).AddMirror([]ID{src.ID()}, nil, math.P3(0, 0, 0), math.V3(1, 0, 0))
	m.Definition().OfBody, m.Definition().RemoveOriginal = true, true

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	back := fresh.Item(1).Definition().(*MirrorFeature).Definition()
	if !back.OfBody || !back.RemoveOriginal || back.JoinToOriginal {
		t.Errorf("restored mirror = ofBody %v removeOriginal %v join %v, want true/true/false",
			back.OfBody, back.RemoveOriginal, back.JoinToOriginal)
	}
}
