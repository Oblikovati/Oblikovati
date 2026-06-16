// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
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

// TestFlatPatternRejectsPlainPart the flat-pattern surface errors on an ordinary part.
func TestFlatPatternRejectsPlainPart(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, wire.MethodFlatPatternListOrientations, []byte("{}")); err == nil {
		t.Fatal("flatPattern on a plain part must error")
	}
}
