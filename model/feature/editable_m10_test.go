// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// The #704 coverage guard: every scalar-bearing surfacing/freeform/M09 feature must
// expose editable params (features.edit renders only what EditableParams returns), and
// each param's Set must round-trip through Get.

// assertParamsRoundTrip drives every param of ed through a Set(7)/Get cycle.
func assertParamsRoundTrip(t *testing.T, name string, ed Editable, want int) {
	t.Helper()
	params := ed.EditableParams()
	if len(params) != want {
		t.Fatalf("%s exposes %d params, want %d", name, len(params), want)
	}
	for _, p := range params {
		p.Set(7)
		if got := p.Get(); got != 7 {
			t.Errorf("%s param %q: Set(7) then Get() = %g, want 7", name, p.Label, got)
		}
	}
}

func TestSurfacingFeaturesExposeEditableParams(t *testing.T) {
	assertParamsRoundTrip(t, "ruled", &RuledSurfaceFeature{def: &RuledSurfaceDefinition{}}, 1)
	assertParamsRoundTrip(t, "surfaceOffset", &SurfaceOffsetFeature{def: &SurfaceOffsetDefinition{}}, 1)
	assertParamsRoundTrip(t, "midSurface", &MidSurfaceFeature{def: &MidSurfaceDefinition{}}, 1)
	assertParamsRoundTrip(t, "stitch", &StitchFeature{def: &StitchDefinition{}}, 1)
	assertParamsRoundTrip(t, "sculpt", &SculptFeature{def: &SculptDefinition{}}, 1)
	assertParamsRoundTrip(t, "extend", &ExtendFeature{def: &ExtendDefinition{}}, 1)
}

func TestPartFeaturesExposeEditableParams(t *testing.T) {
	assertParamsRoundTrip(t, "boss", &BossFeature{def: &BossDefinition{}}, 2)
	assertParamsRoundTrip(t, "coreCavity", &CoreCavityFeature{def: &CoreCavityDefinition{}}, 2)
	assertParamsRoundTrip(t, "thicken", &ThickenFeature{}, 1)
	assertParamsRoundTrip(t, "faceOffset", &FaceOffsetFeature{}, 1)
	assertParamsRoundTrip(t, "moveFace (translate)", &MoveFaceFeature{}, 1)
}

func TestFreeformLevelParamClampsAndRoundTrips(t *testing.T) {
	ff := &FreeformFeature{body: &FreeformBody{}}
	params := ff.EditableParams()
	if len(params) != 1 || !params[0].Integer {
		t.Fatalf("freeform exposes %d params (integer=%v), want 1 integer", len(params), len(params) == 1 && params[0].Integer)
	}
	params[0].Set(2.4)
	if got := ff.body.Level(); got != 2 {
		t.Errorf("Set(2.4) → level %d, want 2 (rounded)", got)
	}
	params[0].Set(-3)
	if got := ff.body.Level(); got != 0 {
		t.Errorf("Set(-3) → level %d, want 0 (clamped)", got)
	}
}

func TestDirectEditParamsFollowOperation(t *testing.T) {
	cases := map[types.DirectEditOperationType]string{
		types.DirectEditSizeOperation:   "Distance",
		types.DirectEditRotateOperation: "Angle",
		types.DirectEditScaleOperation:  "Scale factor",
		types.DirectEditMoveOperation:   "Distance",
	}
	for op, label := range cases {
		f := &DirectEditFeature{def: &DirectEditDefinition{Operation: op, Translation: math.V3(0, 0, 1)}}
		params := f.EditableParams()
		if len(params) != 1 || params[0].Label != label {
			t.Errorf("op %v exposes %v, want one %q param", op, paramLabels(params), label)
		}
	}
	if del := (&DirectEditFeature{def: &DirectEditDefinition{Operation: types.DirectEditDeleteOperation}}); len(del.EditableParams()) != 0 {
		t.Error("delete exposes params, want none (no scalar input)")
	}
}

func paramLabels(params []EditableParam) []string {
	out := make([]string, len(params))
	for i, p := range params {
		out[i] = p.Label
	}
	return out
}
