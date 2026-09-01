// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// constVal is a settable scalar closure for building feature definitions in tests.
func constVal(v float64) func() float64 { return func() float64 { return v } }

// TestEditableParamsExposeAndWriteScalars checks the generic edit mechanism across several
// feature kinds: each lists the expected labels/units, Get reads the current value, and Set
// replaces the closure (so a later Get sees the new value) — the contract app.BeginEditFeature
// and the head dialog rely on.
func TestEditableParamsExposeAndWriteScalars(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		feat      Editable
		wantLabel string
		wantUnit  param.Unit
		start     float64
	}{
		{"revolve", &RevolveFeature{def: &RevolveDefinition{Angle: constVal(1.5)}}, "Angle", param.Angle, 1.5},
		{"fillet", &FilletFeature{def: &FilletDefinition{Radius: constVal(0.5)}}, "Radius", param.Length, 0.5},
		{"chamfer", &ChamferFeature{def: &ChamferDefinition{Distance: constVal(0.3)}}, "Distance", param.Length, 0.3},
		{"shell", &ShellFeature{def: &ShellDefinition{Thickness: constVal(0.2)}}, "Thickness", param.Length, 0.2},
		{"draft", &FaceDraftFeature{def: &FaceDraftDefinition{Angle: constVal(0.1)}}, "Angle", param.Angle, 0.1},
		{"emboss", &EmbossFeature{def: &EmbossDefinition{Depth: constVal(0.4)}}, "Depth", param.Length, 0.4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ps := c.feat.EditableParams()
			if len(ps) == 0 {
				t.Fatalf("%s exposed no editable params", c.name)
			}
			if ps[0].Label != c.wantLabel || ps[0].Unit != c.wantUnit {
				t.Fatalf("%s param0 = (%q,%v), want (%q,%v)", c.name, ps[0].Label, ps[0].Unit, c.wantLabel, c.wantUnit)
			}
			if got := ps[0].Get(); got != c.start {
				t.Fatalf("%s Get = %v, want %v", c.name, got, c.start)
			}
			ps[0].Set(c.start + 1)
			if got := ps[0].Get(); got != c.start+1 {
				t.Fatalf("%s after Set, Get = %v, want %v", c.name, got, c.start+1)
			}
		})
	}
}

// TestHoleEditableParamsVaryByType checks a hole's editable set tracks its type and through-all
// flag: a through drill exposes only the diameter; a blind drill adds depth; a counterbore adds
// its two recess inputs.
func TestHoleEditableParamsVaryByType(t *testing.T) {
	t.Parallel()
	through := &HoleFeature{def: &HoleDefinition{Diameter: constVal(0.4), ThroughAll: true, Type: DrilledHole}}
	if ps := through.EditableParams(); len(ps) != 1 || ps[0].Label != "Diameter" {
		t.Fatalf("through hole params = %v, want [Diameter]", labelsOf(through.EditableParams()))
	}
	blind := &HoleFeature{def: &HoleDefinition{Diameter: constVal(0.4), Depth: constVal(0.8), Type: DrilledHole}}
	if got := labelsOf(blind.EditableParams()); len(got) != 2 {
		t.Fatalf("blind hole params = %v, want [Diameter Depth]", got)
	}
	cbore := &HoleFeature{def: &HoleDefinition{
		Diameter: constVal(0.4), Depth: constVal(0.8), CounterDiameter: constVal(0.8), CounterDepth: constVal(0.2), Type: CounterboreHole,
	}}
	if got := labelsOf(cbore.EditableParams()); len(got) != 4 {
		t.Fatalf("counterbore params = %v, want 4", got)
	}
}

// TestPatternFeaturesAreEditable pins that rectangular/circular patterns expose editable inputs
// (they were not editable — "cannot edit regular-pattern"): integer counts, per-direction
// spacing (the step vector's magnitude, direction preserved), and the circular total angle.
func TestPatternFeaturesAreEditable(t *testing.T) {
	t.Parallel()
	rect := &RectangularPatternFeature{def: &RectangularPatternDefinition{
		CountX: func() int { return 3 }, CountY: func() int { return 2 },
		StepX: math.V3(2, 0, 0), StepY: math.V3(0, 1.5, 0),
	}}
	ps := rect.EditableParams()
	if len(ps) != 4 {
		t.Fatalf("rect pattern params = %v, want 4 (CountX,CountY,SpacingX,SpacingY)", labelsOf(ps))
	}
	if !ps[0].Integer || ps[0].Get() != 3 {
		t.Fatalf("CountX param integer=%v get=%v, want true/3", ps[0].Integer, ps[0].Get())
	}
	ps[0].Set(5)
	if rect.def.CountX() != 5 {
		t.Fatalf("after Set, CountX = %d, want 5", rect.def.CountX())
	}
	if ps[2].Unit != param.Length || ps[2].Get() != 2 {
		t.Fatalf("SpacingX = (%v,%v), want (Length,2)", ps[2].Unit, ps[2].Get())
	}
	ps[2].Set(4) // re-space; direction (+X) must be preserved
	if rect.def.StepX.Length() != 4 || rect.def.StepX.X != 4 {
		t.Fatalf("after Set, StepX = %v, want length 4 along +X", rect.def.StepX)
	}

	circ := &CircularPatternFeature{def: &CircularPatternDefinition{Count: func() int { return 6 }, Angle: func() float64 { return 1.0 }}}
	cs := circ.EditableParams()
	if len(cs) != 2 || !cs[0].Integer || cs[1].Unit != param.Angle {
		t.Fatalf("circ pattern params = %v, want [Count(int), Angle(angle)]", labelsOf(cs))
	}
}

func labelsOf(ps []EditableParam) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Label
	}
	return out
}

// TestEditableRefsKeyBased checks the edge/face slot mechanism: a fillet exposes a multi edge
// slot, a hole a single face slot; Add appends/replaces keys, Snapshot restores, Clear empties.
func TestEditableRefsKeyBased(t *testing.T) {
	t.Parallel()
	e1, e2, f1 := []byte("edge-1"), []byte("edge-2"), []byte("face-1")

	fil := &FilletFeature{def: &FilletDefinition{EdgeKeys: [][]byte{e1}}}
	slot := fil.EditableRefs()[0]
	if slot.Label != "Edges" || slot.Kind != RefEdges || !slot.Multi || slot.Count() != 1 {
		t.Fatalf("fillet slot = %+v, want one multi Edges slot with count 1", slot)
	}
	undo := slot.Snapshot()
	slot.Add(PickedRef{Key: e2}) // re-pick adds a second edge
	if slot.Count() != 2 || len(fil.def.EdgeKeys) != 2 {
		t.Fatalf("after Add, fillet edges = %d, want 2", slot.Count())
	}
	undo() // Cancel restores
	if len(fil.def.EdgeKeys) != 1 {
		t.Fatalf("after restore, fillet edges = %d, want 1", len(fil.def.EdgeKeys))
	}

	hole := &HoleFeature{def: &HoleDefinition{PlacementFaceKey: f1, Diameter: constVal(0.4), ThroughAll: true}}
	hs := hole.EditableRefs()[0]
	if hs.Kind != RefFace || hs.Multi || hs.Count() != 1 {
		t.Fatalf("hole slot = %+v, want one single RefFace slot count 1", hs)
	}
	hs.Add(PickedRef{Key: []byte("face-2")}) // re-pick the placement face (replace)
	if string(hole.def.PlacementFaceKey) != "face-2" {
		t.Fatalf("after Add, hole face = %q, want face-2", hole.def.PlacementFaceKey)
	}
	hs.Clear()
	if hole.def.PlacementFaceKey != nil || hs.Count() != 0 {
		t.Fatalf("after Clear, hole face = %q count %d, want empty", hole.def.PlacementFaceKey, hs.Count())
	}
}

// TestEditableRefsProfileAndPlane checks the profile and plane slots: a revolve re-binds to a
// re-picked profile (sketch + index); a mirror re-binds to a re-picked plane (origin/normal),
// and a plane slot is not clearable (a mirror always needs a plane).
func TestEditableRefsProfileAndPlane(t *testing.T) {
	t.Parallel()
	rev := &RevolveFeature{def: &RevolveDefinition{ProfileIndex: 0}}
	rslot := rev.EditableRefs()[0]
	if rslot.Kind != RefProfile || rslot.Count() != 0 { // no sketch yet ⇒ 0
		t.Fatalf("revolve profile slot = %+v, want RefProfile count 0", rslot)
	}
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	rslot.Add(PickedRef{Sketch: sk, Profile: 2})
	if rev.def.Sketch != sk || rev.def.ProfileIndex != 2 || rslot.Count() != 1 {
		t.Fatalf("after Add, revolve profile = (%v,%d) count %d, want (sk,2) count 1", rev.def.Sketch, rev.def.ProfileIndex, rslot.Count())
	}

	mir := &MirrorFeature{def: &MirrorDefinition{Normal: math.V3(1, 0, 0)}}
	mslot := mir.EditableRefs()[0]
	if mslot.Kind != RefPlane || mslot.Clear != nil {
		t.Fatalf("mirror plane slot = %+v, want RefPlane, not clearable", mslot)
	}
	undo := mslot.Snapshot()
	mslot.Add(PickedRef{Origin: math.P3(1, 2, 3), Normal: math.V3(0, 1, 0), PlaneKey: []byte("pl")})
	if mir.def.Normal != (math.V3(0, 1, 0)) || mir.def.Origin != math.P3(1, 2, 3) {
		t.Fatalf("after Add, mirror plane = (%v,%v), want origin(1,2,3) normal(0,1,0)", mir.def.Origin, mir.def.Normal)
	}
	undo()
	if mir.def.Normal != (math.V3(1, 0, 0)) {
		t.Fatalf("after restore, mirror normal = %v, want (1,0,0)", mir.def.Normal)
	}
}
