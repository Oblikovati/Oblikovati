//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// Two ways the Format panel's work never reached the screen. Selecting a centerline redrew it
// solid, so the dash pattern that identifies it vanished exactly when the user pointed at it. And
// the overlay only ever asked for an entity's dash pattern, never its colour — #2015 shipped the
// model, the persistence, the wire surface and the three ribbon lists, but SketchEntityStyle had
// no caller outside its own unit test, so a colour set in the panel changed nothing.

// formattedSketch returns a sketch holding one centerline and one plain line.
func formattedSketch(t *testing.T) (*sketch.Sketch, *sketch.Line, *sketch.Line) {
	t.Helper()
	sc := sketch.NewSketches()
	sk := sc.Add(sketch.XYPlane())
	center := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	center.SetCenterline(true)
	plain := sk.Lines().AddByTwoPoints(math.P2(0, 2), math.P2(4, 2))
	return sk, center, plain
}

// vertexCount totals the vertices across every draw item — a dashed line yields more than a solid
// one because each dash is its own segment.
func vertexCount(items []renderer.DrawItem) int {
	n := 0
	for _, it := range items {
		n += len(it.Positions)
	}
	return n
}

// TestSelectedCenterlineKeepsItsDashes is the reported bug: selecting a centerline collapsed it
// from a dashed run to a single solid segment.
func TestSelectedCenterlineKeepsItsDashes(t *testing.T) {
	sk, center, _ := formattedSketch(t)
	only := func(target sketch.Entity) func(sketch.Entity) bool {
		return func(e sketch.Entity) bool { return e == target }
	}
	unselected := vertexCount(sketchOverlay(sk, nil, nil, false))
	selected := vertexCount(sketchOverlay(sk, only(center), nil, false))
	if selected != unselected {
		t.Fatalf("selecting the centerline changed its vertex count %d→%d; the dash pattern was dropped",
			unselected, selected)
	}
}

// TestHoveredCenterlineKeepsItsDashes: the hover-candidate lane had the same defect.
func TestHoveredCenterlineKeepsItsDashes(t *testing.T) {
	sk, center, _ := formattedSketch(t)
	unhovered := vertexCount(sketchOverlay(sk, nil, nil, false))
	hovered := vertexCount(sketchOverlay(sk, nil, center, false))
	if hovered != unhovered {
		t.Fatalf("hovering the centerline changed its vertex count %d→%d; the dash pattern was dropped",
			unhovered, hovered)
	}
}

// TestEntityColourOverrideReachesTheDrawItems is the other reported bug: a colour set in the
// Format panel never coloured anything.
func TestEntityColourOverrideReachesTheDrawItems(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	red := types.NewColor(255, 0, 0)
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{Color: red})

	if !hasColor(sketchOverlay(sk, nil, nil, false), red.Rgba().Array()) {
		t.Fatal("no draw item carries the override colour; the Format panel's colour never renders")
	}
}

// TestShowFormatSuppressesTheOverrideColour: with Show Format on, the entity draws in the default
// colour instead — the button's documented (name-inverted) behaviour, now actually observable.
func TestShowFormatSuppressesTheOverrideColour(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	red := types.NewColor(255, 0, 0)
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{Color: red})

	if hasColor(sketchOverlay(sk, nil, nil, true), red.Rgba().Array()) {
		t.Fatal("Show Format did not suppress the override colour")
	}
}

// TestUnformattedGeometryKeepsTheThemeColour guards the common case: a sketch with no overrides
// still draws in one theme-coloured item, not fragmented into per-entity ones.
func TestUnformattedGeometryKeepsTheThemeColour(t *testing.T) {
	sk, _, _ := formattedSketch(t)
	items := sketchOverlay(sk, nil, nil, false)
	if len(items) != 1 {
		t.Fatalf("unformatted sketch produced %d draw items, want 1 batched by colour", len(items))
	}
	if items[0].Color != chromeTheme.sketchColor {
		t.Errorf("colour = %v, want the theme sketch colour %v", items[0].Color, chromeTheme.sketchColor)
	}
}

// TestFormatRevisionMovesOnEveryFormatEdit: the finished-sketch overlay cache keys on this, so a
// recolour that left it unchanged would leave stale colours on screen until something unrelated
// invalidated the cache.
func TestFormatRevisionMovesOnEveryFormatEdit(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	before := sk.FormatRevision()
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{Color: types.NewColor(0, 0, 255)})
	afterSet := sk.FormatRevision()
	if afterSet == before {
		t.Fatalf("FormatRevision unchanged (%d) after setting a format", before)
	}
	sk.ClearEntityFormat(plain.EntityID())
	if sk.FormatRevision() == afterSet {
		t.Fatalf("FormatRevision unchanged (%d) after clearing a format", afterSet)
	}
}

// TestSketchOverlayKeyChangesOnFormatEdit: recolouring an entity changes neither the geometry
// version nor the entity count, so without the format revision in the key the cached
// finished-sketch overlay would keep serving the old colours.
func TestSketchOverlayKeyChangesOnFormatEdit(t *testing.T) {
	s, sk := partWithSketch(t)
	before, ok := sketchOverlayKey(s)
	if !ok {
		t.Fatal("sketchOverlayKey not available for an active part")
	}
	sk.SetEntityFormat(sk.Lines().Item(0).EntityID(), sketch.EntityFormat{Color: types.NewColor(255, 0, 0)})
	if after, _ := sketchOverlayKey(s); after == before {
		t.Fatal("key unchanged after a format edit; the cache would serve stale colours")
	}
}

// TestSketchOverlayKeyChangesOnShowFormat: the toggle changes what is drawn without touching the
// model at all, so it has to be in the key too.
func TestSketchOverlayKeyChangesOnShowFormat(t *testing.T) {
	s, _ := partWithSketch(t)
	before, _ := sketchOverlayKey(s)
	s.ToggleShowFormat()
	if after, _ := sketchOverlayKey(s); after == before {
		t.Fatal("key unchanged after toggling Show Format; the cache would ignore it")
	}
}

// TestLineWeightReachesTheDrawItems is the third way the Format panel did nothing: a line weight
// was stored and persisted but the viewport had no width to put it in, so every line drew as a
// hairline whatever the panel said.
func TestLineWeightReachesTheDrawItems(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{LineWeight: 2}) // 2 mm

	var widest float32
	for _, it := range sketchOverlay(sk, nil, nil, false) {
		if it.Width > widest {
			widest = it.Width
		}
	}
	if widest <= 1 {
		t.Fatalf("widest stroke = %v; a 2 mm line weight still draws as a hairline", widest)
	}
}

// TestLineWeightSplitsFromUnweightedGeometry: one draw item carries one width, so a weighted entity
// cannot share an item with an unweighted one.
func TestLineWeightSplitsFromUnweightedGeometry(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{LineWeight: 2})
	items := sketchOverlay(sk, nil, nil, false)
	if len(items) < 2 {
		t.Fatalf("got %d draw items; the weighted line was batched with the hairline one", len(items))
	}
}

// TestSelectedWeightedLineKeepsItsWeight: selection recolours, it must not restyle. A heavy line
// that thinned on selection would read as the selection having changed the geometry.
func TestSelectedWeightedLineKeepsItsWeight(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{LineWeight: 2})
	only := func(e sketch.Entity) bool { return e == plain }

	var got float32
	for _, it := range sketchOverlay(sk, only, nil, false) {
		if it.Color == chromeTheme.sketchSelectedColor && it.Width > got {
			got = it.Width
		}
	}
	if got <= 1 {
		t.Fatalf("selected weighted line drew at width %v, want its 2 mm stroke kept", got)
	}
}

// TestSketchStrokeWidthConvertsAndClamps: a weight becomes a pixel stroke, an unset weight stays on
// the hairline path, and an absurd weight (a DWG layer table can carry anything) is capped rather
// than painting over the viewport.
func TestSketchStrokeWidthConvertsAndClamps(t *testing.T) {
	if w := sketchStrokeWidth(app.EntityStyle{}); w != 0 {
		t.Errorf("unset weight gave width %v, want 0 (hairline path)", w)
	}
	w := sketchStrokeWidth(app.EntityStyle{LineWeight: 1})
	if w <= 1 || w > 8 {
		t.Errorf("1 mm gave %v px, want a small multi-pixel stroke", w)
	}
	if w := sketchStrokeWidth(app.EntityStyle{LineWeight: 1e6}); w != maxSketchStrokePixels {
		t.Errorf("absurd weight gave %v, want the %v cap", w, float32(maxSketchStrokePixels))
	}
}

// TestShowFormatSuppressesTheLineWeight: the toggle hides weight along with colour and line type.
func TestShowFormatSuppressesTheLineWeight(t *testing.T) {
	sk, _, plain := formattedSketch(t)
	sk.SetEntityFormat(plain.EntityID(), sketch.EntityFormat{LineWeight: 2})
	for _, it := range sketchOverlay(sk, nil, nil, true) {
		if it.Width > 1 {
			t.Fatalf("Show Format did not suppress the line weight (width %v)", it.Width)
		}
	}
}

// hasColor reports whether any draw item is drawn in c.
func hasColor(items []renderer.DrawItem, c [4]float32) bool {
	for _, it := range items {
		if it.Color == c {
			return true
		}
	}
	return false
}
