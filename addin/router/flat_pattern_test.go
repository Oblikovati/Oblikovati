// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestFlatPatternMapEntityOverWire a folded face maps to a flat face (and back) by reference
// key over the wire — the correspondence a flat drawing dimension survives recompute by.
func TestFlatPatternMapEntityOverWire(t *testing.T) {
	r, s := flangedSheet(t)

	var keys wire.ReferenceKeysResult
	call(t, r, s, wire.MethodModelReferenceKeys, "{}", &keys)
	if len(keys.Bodies) == 0 || len(keys.Bodies[0].Faces) == 0 {
		t.Fatal("no face reference keys")
	}
	faceKey := keys.Bodies[0].Faces[0].Key

	toFlat, _ := json.Marshal(map[string]any{"key": faceKey, "toFlat": true})
	var flat wire.MapEntityResult
	call(t, r, s, wire.MethodFlatPatternMapEntity, string(toFlat), &flat)
	if !flat.Found || flat.Kind != "face" || flat.Key == "" {
		t.Fatalf("folded→flat map = %+v, want a found face", flat)
	}

	toFolded, _ := json.Marshal(map[string]any{"key": flat.Key})
	var folded wire.MapEntityResult
	call(t, r, s, wire.MethodFlatPatternMapEntity, string(toFolded), &folded)
	if !folded.Found {
		t.Errorf("flat→folded map = %+v, want a found face", folded)
	}

	// An unknown key reports not-found (not an error).
	unknown, _ := json.Marshal(map[string]any{"key": "bogus", "toFlat": true})
	var miss wire.MapEntityResult
	call(t, r, s, wire.MethodFlatPatternMapEntity, string(unknown), &miss)
	if miss.Found {
		t.Error("an unknown key should report not-found")
	}
}

// TestFlatPatternOrientationsOverWire drives the M13-F05 orientation surface: a flanged sheet
// reports the default orientation with a positive length/width; adding a vertical orientation
// swaps them; activate/delete behave; the default is undeletable.
func TestFlatPatternOrientationsOverWire(t *testing.T) {
	r, s := flangedSheet(t)

	var list wire.OrientationsResult
	call(t, r, s, wire.MethodFlatPatternListOrientations, "{}", &list)
	if len(list.Orientations) != 1 || !list.Orientations[0].Active {
		t.Fatalf("default list = %+v, want one active orientation", list.Orientations)
	}
	def := list.Orientations[0]
	if def.Length <= 0 || def.Width <= 0 {
		t.Errorf("default length/width = %g×%g, want positive", def.Length, def.Width)
	}

	var added wire.OrientationResult
	call(t, r, s, wire.MethodFlatPatternAddOrientation, `{"name":"Vertical","alignmentType":"vertical","activate":true}`, &added)
	if !added.Orientation.Active || added.Orientation.AlignmentType != "vertical" {
		t.Errorf("added = %+v, want active vertical", added.Orientation)
	}
	if added.Orientation.Length != def.Width || added.Orientation.Width != def.Length {
		t.Errorf("vertical (%g×%g) should swap default (%g×%g)", added.Orientation.Length, added.Orientation.Width, def.Length, def.Width)
	}

	// Activate the default again, then delete the custom one.
	var act wire.OrientationResult
	call(t, r, s, wire.MethodFlatPatternActivateOrientation, `{"name":"Flat Pattern Default"}`, &act)
	if !act.Orientation.Active || act.Orientation.Name != "Flat Pattern Default" {
		t.Errorf("activated = %+v, want the default active", act.Orientation)
	}
	var afterDelete wire.OrientationsResult
	call(t, r, s, wire.MethodFlatPatternDeleteOrientation, `{"name":"Vertical"}`, &afterDelete)
	if len(afterDelete.Orientations) != 1 {
		t.Errorf("after delete = %d orientations, want 1", len(afterDelete.Orientations))
	}

	// Error paths.
	if _, err := r.Handle(s, wire.MethodFlatPatternDeleteOrientation, []byte(`{"name":"Flat Pattern Default"}`)); err == nil {
		t.Error("deleting the default orientation must error")
	}
	if _, err := r.Handle(s, wire.MethodFlatPatternActivateOrientation, []byte(`{"name":"Nope"}`)); err == nil {
		t.Error("activating an unknown orientation must error")
	}
}

// TestFlatPatternEdgesAndFaces a flanged sheet reports its fold line as a bend-up edge (a
// plain flange folds up), filterable by type, and its front/back faces with equal areas.
func TestFlatPatternEdgesAndFaces(t *testing.T) {
	r, s := flangedSheet(t)

	var all wire.EdgesResult
	call(t, r, s, wire.MethodFlatPatternEdgesOfType, "{}", &all)
	if len(all.Edges) != 1 || all.Edges[0].Type != "bendUp" {
		t.Fatalf("edges = %+v, want one bendUp fold line", all.Edges)
	}
	if all.Edges[0].Angle <= 0 {
		t.Errorf("fold line angle = %v, want positive", all.Edges[0].Angle)
	}

	var up, down wire.EdgesResult
	call(t, r, s, wire.MethodFlatPatternEdgesOfType, `{"type":"bendUp"}`, &up)
	call(t, r, s, wire.MethodFlatPatternEdgesOfType, `{"type":"bendDown"}`, &down)
	if len(up.Edges) != 1 || len(down.Edges) != 0 {
		t.Errorf("filtered up=%d down=%d, want 1 and 0", len(up.Edges), len(down.Edges))
	}

	var faces wire.FacesResult
	call(t, r, s, wire.MethodFlatPatternFaces, "{}", &faces)
	if len(faces.Faces) != 2 || faces.Faces[0].Type != "front" || faces.Faces[1].Type != "back" {
		t.Fatalf("faces = %+v, want front+back", faces.Faces)
	}
	if faces.Faces[0].Area <= 0 || faces.Faces[0].Area != faces.Faces[1].Area {
		t.Errorf("front/back areas = %v/%v, want equal positive", faces.Faces[0].Area, faces.Faces[1].Area)
	}

	if _, err := r.Handle(s, wire.MethodFlatPatternEdgesOfType, []byte(`{"type":"nope"}`)); err == nil {
		t.Error("a bad edge type must error")
	}
}

// TestFlatPatternPlatesAndSettings a flanged sheet reports one plate and round-trips the
// deferred-update setting over the wire.
func TestFlatPatternPlatesAndSettings(t *testing.T) {
	r, s := flangedSheet(t)

	var plates wire.PlatesResult
	call(t, r, s, wire.MethodFlatPatternListPlates, "{}", &plates)
	if len(plates.Plates) != 1 || plates.Plates[0].Length <= 0 || plates.Plates[0].Area <= 0 {
		t.Fatalf("plates = %+v, want one plate with positive extents/area", plates.Plates)
	}

	var got wire.SettingsResult
	call(t, r, s, wire.MethodFlatPatternGetSettings, "{}", &got)
	if got.Settings.DeferUpdate {
		t.Error("default deferUpdate should be false")
	}
	call(t, r, s, wire.MethodFlatPatternSetSettings, `{"deferUpdate":true}`, &got)
	if !got.Settings.DeferUpdate {
		t.Error("setSettings did not enable deferUpdate")
	}
	call(t, r, s, wire.MethodFlatPatternGetSettings, "{}", &got)
	if !got.Settings.DeferUpdate {
		t.Error("deferUpdate did not persist on the part")
	}
}

// TestFlatPatternBendOrderOverWire a flanged sheet lists its bend order; setting the order by
// feature name round-trips; an unknown name errors.
func TestFlatPatternBendOrderOverWire(t *testing.T) {
	r, s := flangedSheet(t)

	var list wire.BendOrderResult
	call(t, r, s, wire.MethodFlatPatternListBendOrder, "{}", &list)
	if len(list.Bends) != 1 || list.Bends[0].Order != 1 || list.Bends[0].Feature == "" {
		t.Fatalf("bend order = %+v, want one bend at order 1", list.Bends)
	}
	feature := list.Bends[0].Feature

	var set wire.BendOrderResult
	toFlat, _ := json.Marshal(map[string]any{"order": []string{feature}})
	call(t, r, s, wire.MethodFlatPatternSetBendOrder, string(toFlat), &set)
	if len(set.Bends) != 1 || set.Bends[0].Feature != feature {
		t.Errorf("set bend order = %+v, want %s", set.Bends, feature)
	}

	bad, _ := json.Marshal(map[string]any{"order": []string{"NoSuchBend"}})
	if _, err := r.Handle(s, wire.MethodFlatPatternSetBendOrder, bad); err == nil {
		t.Error("setBendOrder with an unknown bend must error")
	}
}

// TestFlatPatternCenterlinesOverWire add/list/delete cosmetic centerlines over the wire.
func TestFlatPatternCenterlinesOverWire(t *testing.T) {
	r, s := flangedSheet(t)

	var res wire.CenterlinesResult
	call(t, r, s, wire.MethodFlatPatternListCenterlines, "{}", &res)
	if len(res.Centerlines) != 0 {
		t.Fatalf("fresh centerlines = %d, want 0", len(res.Centerlines))
	}
	call(t, r, s, wire.MethodFlatPatternAddCenterline, `{"start":[0,0],"end":[4,0]}`, &res)
	call(t, r, s, wire.MethodFlatPatternAddCenterline, `{"start":[2,-1],"end":[2,1]}`, &res)
	if len(res.Centerlines) != 2 || res.Centerlines[1].Index != 1 {
		t.Fatalf("after adds = %+v, want 2 centerlines", res.Centerlines)
	}

	call(t, r, s, wire.MethodFlatPatternDeleteCenterline, `{"index":0}`, &res)
	if len(res.Centerlines) != 1 || res.Centerlines[0].Start.X != 2 {
		t.Errorf("after delete = %+v, want the vertical centerline", res.Centerlines)
	}
	if _, err := r.Handle(s, wire.MethodFlatPatternDeleteCenterline, []byte(`{"index":9}`)); err == nil {
		t.Error("deleting an out-of-range centerline must error")
	}
}

// TestFlatPatternRejectsPlainPart every flat-pattern method errors on an ordinary part.
func TestFlatPatternRejectsPlainPart(t *testing.T) {
	r, s := seededSession(t)
	for _, m := range []string{
		wire.MethodFlatPatternListOrientations,
		wire.MethodFlatPatternEdgesOfType,
		wire.MethodFlatPatternFaces,
		wire.MethodFlatPatternMapEntity,
		wire.MethodFlatPatternListPlates,
		wire.MethodFlatPatternGetSettings,
		wire.MethodFlatPatternSetSettings,
		wire.MethodFlatPatternListBendOrder,
		wire.MethodFlatPatternSetBendOrder,
		wire.MethodFlatPatternAddCenterline,
		wire.MethodFlatPatternListCenterlines,
		wire.MethodFlatPatternDeleteCenterline,
	} {
		if _, err := r.Handle(s, m, []byte("{}")); err == nil {
			t.Errorf("%s on a plain part must error", m)
		}
	}
}

// punchedSheet builds a flat sheet with a two-circle punch sketch stamped through it — two punch
// instances from one feature, which is what the read-model has to report separately.
func punchedSheet(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := newSheetMetalPart(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[8,6]]}`, &struct{}{})
	var face featureResult
	call(t, r, s, "features.add", `{"kind":"sheetMetalFace","args":{"sketchIndex":0}}`, &face)
	if !face.Healthy {
		t.Fatal("base Face unhealthy")
	}
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"circle","points":[[2,2]],"radius":"5 mm"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"circle","points":[[6,4]],"radius":"5 mm"}`, &struct{}{})
	var punch featureResult
	call(t, r, s, "features.add", `{"kind":"sheetMetalPunch","args":{"sketchIndex":1}}`, &punch)
	if !punch.Healthy {
		t.Fatal("punch unhealthy")
	}
	return r, s
}

// TestFlatPatternListPunches (#1963): the punch geometry was already developed for the flat and
// had no way out of the host. Each instance must come back with its own PLACEMENT — a list that
// reported outlines alone would not tell a nest or a punch note where the tool goes.
func TestFlatPatternListPunches(t *testing.T) {
	r, s := punchedSheet(t)
	var res wire.PunchesResult
	call(t, r, s, wire.MethodFlatPatternListPunches, "{}", &res)
	if len(res.Punches) != 2 {
		t.Fatalf("listPunches returned %d punches, want 2 (one per sketched circle)", len(res.Punches))
	}
	seen := map[string]bool{}
	for _, p := range res.Punches {
		if len(p.Outline) < 3 {
			t.Errorf("punch %q outline has %d points, want a closed profile", p.ID, len(p.Outline))
		}
		if !p.DirectionUp {
			t.Errorf("punch %q reports directionUp=false, want the default punch side", p.ID)
		}
		if p.HasDepth {
			t.Errorf("punch %q reports a depth; a punch through all the material has none", p.ID)
		}
		seen[centreKey(p.Position.X, p.Position.Y)] = true
	}
	// The two circles were sketched at (2,2) and (6,4); their developed positions must be those
	// centres, not both the same point or the sheet's origin.
	for _, want := range []string{centreKey(2, 2), centreKey(6, 4)} {
		if !seen[want] {
			t.Errorf("no punch placed at %s; got %v", want, seen)
		}
	}
}

// centreKey rounds a position to 0.01 so a centroid comparison is not chasing facet noise.
func centreKey(x, y float64) string {
	return fmt.Sprintf("%.2f,%.2f", math.Round(x*100)/100, math.Round(y*100)/100)
}

// TestPunchDepthIsReportedOnlyWhenItHasOne (#1963): a punch that stops short of the far face has a
// depth, and one that goes clean through does not. Reporting a through punch as depth 0 would read
// as a zero-deep punch — the opposite of what it is — so the two are told apart by the flag.
func TestPunchDepthIsReportedOnlyWhenItHasOne(t *testing.T) {
	r, s := newSheetMetalPart(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"rectangle","points":[[0,0],[8,6]]}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"sheetMetalFace","args":{"sketchIndex":0}}`, &featureResult{})
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"circle","points":[[3,3]],"radius":"5 mm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"sheetMetalPunch","args":{"sketchIndex":1,"depth":"0.5 mm"}}`, &featureResult{})

	var res wire.PunchesResult
	call(t, r, s, wire.MethodFlatPatternListPunches, "{}", &res)
	if len(res.Punches) != 1 {
		t.Fatalf("listPunches returned %d punches, want 1", len(res.Punches))
	}
	p := res.Punches[0]
	if !p.HasDepth {
		t.Fatal("a punch given a depth reports hasDepth=false")
	}
	if p.Depth < 0.0499 || p.Depth > 0.0501 {
		t.Errorf("punch depth = %g cm, want 0.05 (0.5 mm)", p.Depth)
	}
}

// TestFlatInfoCarriesPunches: unfold's own reply carries the same list, so a caller that already
// asked for the flat does not need a second round trip to find its punches.
func TestFlatInfoCarriesPunches(t *testing.T) {
	r, s := punchedSheet(t)
	var res wire.UnfoldResult
	call(t, r, s, wire.MethodSheetMetalUnfold, "{}", &res)
	if len(res.Flat.Punches) != 2 {
		t.Errorf("unfold reported %d punches, want 2", len(res.Flat.Punches))
	}
}
