//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// The 3D overlay drew every curve in one colour with no dash pattern, so nothing the Format
// panel set on a 3D sketch could reach the screen — a construction curve was indistinguishable
// from a normal one (#2039).

// sketch3DOverlayFixture returns a session whose active part holds one 3D sketch with a single
// line, and clears the package overlay cache so each case builds fresh.
func sketch3DOverlayFixture(t *testing.T) (*app.Session, *sketch.Sketch3D, *sketch.Line3D) {
	t.Helper()
	sketch3DOverlayCache.key = ""
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "sketch3d-overlay.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	part := pd.Content().(*compdef.PartComponentDefinition)
	sk := part.Sketches3D().Add()
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(10, 0, 0))
	return s, sk, l
}

// overlayColors is the set of colours the 3D curve overlay draws in.
func overlayColors(items []renderer.DrawItem) map[[4]float32]bool {
	out := map[[4]float32]bool{}
	for _, it := range items {
		out[it.Color] = true
	}
	return out
}

// TestSketch3DConstructionCurveIsDashed: a construction 3D curve is split into dashes, so it
// yields more vertices than the same curve drawn solid.
func TestSketch3DConstructionCurveIsDashed(t *testing.T) {
	s, _, l := sketch3DOverlayFixture(t)
	solid := vertexCount(buildSketch3DCurvesOnly(s))

	l.SetConstruction(true)
	sketch3DOverlayCache.key = ""
	dashed := vertexCount(buildSketch3DCurvesOnly(s))

	if dashed <= solid {
		t.Errorf("construction 3D curve drew %d vertices, solid drew %d — it is not dashed", dashed, solid)
	}
}

// TestSketch3DEntityColourOverrideReachesTheDrawItems: a colour set through the Format panel's
// list is the colour the 3D overlay draws the curve in.
func TestSketch3DEntityColourOverrideReachesTheDrawItems(t *testing.T) {
	s, sk, l := sketch3DOverlayFixture(t)
	red := types.NewColor(220, 20, 20)
	sk.SetEntityFormat(l.EntityID(), sketch.EntityFormat{Color: red})

	sketch3DOverlayCache.key = ""
	if got := overlayColors(buildSketch3DCurvesOnly(s)); !got[red.Rgba().Array()] {
		t.Errorf("3D overlay colours = %v, want the per-entity red %v", got, red.Rgba().Array())
	}
}

// Show Format draws with default attributes, so the override must NOT reach the draw items.
func TestSketch3DShowFormatSuppressesTheOverride(t *testing.T) {
	s, sk, l := sketch3DOverlayFixture(t)
	red := types.NewColor(220, 20, 20)
	sk.SetEntityFormat(l.EntityID(), sketch.EntityFormat{Color: red})
	s.ToggleShowFormat()
	defer s.ToggleShowFormat() // the toggle is a persisted application option

	sketch3DOverlayCache.key = ""
	if got := overlayColors(buildSketch3DCurvesOnly(s)); got[red.Rgba().Array()] {
		t.Errorf("Show Format is on but the 3D overlay still drew the override colour: %v", got)
	}
}

// The overlay cache keys on the format revision, so recolouring an entity — which changes
// neither the geometry version nor the entity count — still redraws.
func TestSketch3DOverlayCacheSeesAFormatEdit(t *testing.T) {
	s, sk, l := sketch3DOverlayFixture(t)
	before, ok := sketch3DOverlayKey(s)
	if !ok {
		t.Fatal("the 3D overlay cache key should be resolvable for an active part")
	}
	sk.SetEntityFormat(l.EntityID(), sketch.EntityFormat{Color: types.NewColor(220, 20, 20)})
	after, _ := sketch3DOverlayKey(s)
	if after == before {
		t.Error("the overlay cache key did not change on a format edit — the viewport would show stale styling")
	}
}
