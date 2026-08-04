// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// formatTestSketch opens a part with one line in an active sketch, returning the line's entity id
// — what every Format-panel wire method addresses.
func formatTestSketch(t *testing.T, s *app.Session) (*sketch.Sketch, uint64) {
	t.Helper()
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	sk := part.Sketches().Add(sketch.XYPlane())
	if err := s.EditSketch(sk); err != nil {
		t.Fatalf("edit sketch: %v", err)
	}
	l := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 0))
	return sk, uint64(l.EntityID())
}

// An entity's formatting round-trips over the wire, and HasFormat distinguishes "inherits
// everything" from "explicitly default" — two states that look the same but write back
// differently (#2015).
func TestSketchEntityFormatRoundTrip(t *testing.T) {
	r, s := emptyPartSession(t)
	_, id := formatTestSketch(t, s)

	var before wire.SketchEntityFormatView
	call(t, r, s, wire.MethodSketchGetEntityFormat, `{"entityId":`+itoa(id)+`}`, &before)
	if before.HasFormat {
		t.Error("a new entity must report no format")
	}

	var set wire.SketchEntityFormatView
	call(t, r, s, wire.MethodSketchSetEntityFormat,
		`{"entityId":`+itoa(id)+`,"format":{"lineType":"dashed","lineWeight":0.35,`+
			`"overrideColor":{"r":255,"g":0,"b":0,"opacity":1,"source":79105}}}`, &set)
	if !set.HasFormat || set.Format.LineType != types.SketchLineDashed || set.Format.LineWeight != 0.35 {
		t.Fatalf("set result = %+v, want the dashed 0.35 override", set)
	}
	if !set.Format.OverrideColor.IsOverride() {
		t.Error("the colour must come back as an override")
	}

	var reread wire.SketchEntityFormatView
	call(t, r, s, wire.MethodSketchGetEntityFormat, `{"entityId":`+itoa(id)+`}`, &reread)
	if reread != set {
		t.Errorf("re-read = %+v, want the stored %+v", reread, set)
	}
}

// Writing a format that overrides nothing clears the entity's overrides.
func TestSketchEntityFormatDefaultClears(t *testing.T) {
	r, s := emptyPartSession(t)
	_, id := formatTestSketch(t, s)

	var set wire.SketchEntityFormatView
	call(t, r, s, wire.MethodSketchSetEntityFormat,
		`{"entityId":`+itoa(id)+`,"format":{"lineType":"dashed"}}`, &set)
	if !set.HasFormat {
		t.Fatal("the override must be stored")
	}
	call(t, r, s, wire.MethodSketchSetEntityFormat, `{"entityId":`+itoa(id)+`,"format":{}}`, &set)
	if set.HasFormat {
		t.Error("writing a default format must clear the overrides")
	}
}

// The armed creation modes round-trip, including the Show Format toggle.
func TestSketchFormatModesRoundTrip(t *testing.T) {
	r, s := emptyPartSession(t)

	var got wire.SketchFormatModesView
	call(t, r, s, wire.MethodSketchGetFormatModes, "{}", &got)
	if got.Modes.Construction || got.Modes.SuppressFormatOverrides {
		t.Fatalf("defaults = %+v, want everything off", got.Modes)
	}

	var set wire.SketchFormatModesView
	call(t, r, s, wire.MethodSketchSetFormatModes,
		`{"modes":{"construction":true,"centerline":false,"centerPoint":true,`+
			`"drivenDimension":false,"suppressFormatOverrides":true}}`, &set)
	if !set.Modes.Construction || !set.Modes.CenterPoint || !set.Modes.SuppressFormatOverrides {
		t.Errorf("set result = %+v, want construction, centre point and show format on", set.Modes)
	}

	var reread wire.SketchFormatModesView
	call(t, r, s, wire.MethodSketchGetFormatModes, "{}", &reread)
	if reread != set {
		t.Errorf("re-read = %+v, want the stored %+v", reread, set)
	}
}
