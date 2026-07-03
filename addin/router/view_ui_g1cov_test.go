// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// This file raises new-code coverage on the DRAWING / VIEWS / UI router handlers converted in
// refactor/m40-g1-typed-router (M40 G1). It targets the whole-0% handlers first —
// drawingViewsAddSection, updateTriad, listStyleLibraries, getEnvironment,
// setClientGraphicsVisible — driving each with real DTO-field assertions, then a few error
// paths. Reuses the package harness (call/mustJSON/seededSession) and the fixtures already in
// drawing_views_test.go (drawingViewSession) and graphics_test.go (heatmapArgs).

// TestVucovDrawingViewsAddSection covers drawingViewsAddSection (wire drawingViews.addSection),
// the last untested drawing-view constructor: a section cut off a base view.
func TestVucovDrawingViewsAddSection(t *testing.T) {
	r, s := drawingViewSession(t)
	call(t, r, s, "drawingViews.addBase", `{"name":"FRONT","orientation":"front","scale":2,"centerXmm":120,"centerYmm":100}`, nil)
	var sec wire.ViewResult
	call(t, r, s, "drawingViews.addSection",
		`{"name":"SEC","parentView":"FRONT","x1":80,"y1":100,"x2":160,"y2":100,"centerXmm":120,"centerYmm":260}`, &sec)
	if sec.View.Name != "SEC" || sec.View.Type != "section" || sec.View.BaseView != "FRONT" {
		t.Fatalf("section view = %+v, want SEC of type section off FRONT", sec.View)
	}
}

// TestVucovDrawingViewsAddSectionNoParent covers the error path when the referenced parent view
// does not exist on the sheet.
func TestVucovDrawingViewsAddSectionNoParent(t *testing.T) {
	r, s := drawingViewSession(t)
	args := `{"name":"SEC","parentView":"MISSING","x1":0,"y1":0,"x2":10,"y2":0}`
	if _, err := r.Handle(s, "drawingViews.addSection", []byte(args)); err == nil {
		t.Fatal("expected error adding a section off a missing parent view")
	}
}

// TestVucovUpdateTriad covers updateTriad (wire triad.update): unlike triad.show it does not
// force Visible, so the caller-supplied spec (position + visibility) must round-trip through
// triad.get verbatim.
func TestVucovUpdateTriad(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "triad.update", `{"triad":{"position":[4,5,6],"visible":true,"allowed":[1]}}`, nil)
	var spec wire.TriadSpec
	call(t, r, s, "triad.get", "{}", &spec)
	if !spec.Visible || spec.Position.X != 4 || spec.Position.Z != 6 || len(spec.Allowed) != 1 {
		t.Fatalf("triad = %+v, want visible at (4,5,6) with one allowed segment", spec)
	}
}

// TestVucovUpdateTriadInvalid covers the update error path: an out-of-range segment is rejected.
func TestVucovUpdateTriadInvalid(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "triad.update", []byte(`{"triad":{"allowed":[42]}}`)); err == nil {
		t.Fatal("expected error for an out-of-range triad segment")
	}
}

// TestVucovListStyleLibraries covers listStyleLibraries + styleLibrariesResult (wire
// styles.listLibraries): a fresh document loads no library cascade, so the result decodes to an
// empty list.
func TestVucovListStyleLibraries(t *testing.T) {
	r, s := seededSession(t)
	var res wire.StyleLibrariesResult
	call(t, r, s, "styles.listLibraries", "{}", &res)
	if len(res.Libraries) != 0 {
		t.Fatalf("libraries = %+v, want none on a fresh document", res.Libraries)
	}
}

// TestVucovGetEnvironment covers getEnvironment + environmentView (wire environment.get): after
// activating a built-in preset it echoes that preset with its display parameters and an empty
// file path.
func TestVucovGetEnvironment(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "environment.set", `{"preset":"Studio","rotation":0.25,"intensity":0.5,"showImage":true}`, nil)
	var env wire.EnvironmentView
	call(t, r, s, "environment.get", "{}", &env)
	if env.Preset != "Studio" || env.FilePath != "" || !env.ShowImage || env.Intensity != 0.5 || env.Rotation != 0.25 {
		t.Fatalf("environment = %+v, want the Studio preset shown", env)
	}
}

// TestVucovSetClientGraphicsVisible covers setClientGraphicsVisible (wire
// clientGraphics.setVisible): toggling a submitted group off flips only its Visible flag,
// leaving its geometry (node/primitive counts) intact.
func TestVucovSetClientGraphicsVisible(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "clientGraphics.set", mustJSON(t, heatmapArgs()), nil)
	call(t, r, s, "clientGraphics.setVisible", `{"clientId":"fea","visible":false}`, nil)
	var list wire.ListClientGraphicsResult
	call(t, r, s, "clientGraphics.list", "{}", &list)
	if len(list.Groups) != 1 || list.Groups[0].ClientId != "fea" || list.Groups[0].Visible {
		t.Fatalf("groups = %+v, want the fea group hidden", list.Groups)
	}
	if list.Groups[0].PrimitiveCount != 1 {
		t.Errorf("primitiveCount = %d, want geometry preserved", list.Groups[0].PrimitiveCount)
	}
}

// TestVucovSetClientGraphicsVisibleUnknown covers the error path: toggling a group that was
// never submitted fails.
func TestVucovSetClientGraphicsVisibleUnknown(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "clientGraphics.setVisible", []byte(`{"clientId":"ghost","visible":true}`)); err == nil {
		t.Fatal("expected error toggling visibility on an unknown group")
	}
}
