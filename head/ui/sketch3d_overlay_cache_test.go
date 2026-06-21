//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// partWithSketch3D returns a session whose active part holds one 3D sketch with a couple of
// line segments and a standalone point — the fixture for the 3D-overlay cache.
func partWithSketch3D(t *testing.T) (*app.Session, *sketch.Sketch3D) {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "p.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches3D().Add()
	sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 1))
	sk.AddLine3D(math.P3(1, 0, 1), math.P3(1, 1, 2))
	sk.AddPoint3D(math.P3(5, 5, 5))
	sketch3DOverlayCache.key = "" // isolate from other tests sharing the package cache
	return s, sk
}

// TestSketch3DOverlayKeyStableThenInvalidates pins the cache key: identical geometry yields an
// identical key; adding an entity or a sketch changes it so the overlay rebuilds.
func TestSketch3DOverlayKeyStableThenInvalidates(t *testing.T) {
	s, sk := partWithSketch3D(t)
	k1, ok := sketch3DOverlayKey(s)
	if !ok {
		t.Fatal("sketch3DOverlayKey not available for an active part")
	}
	if k2, _ := sketch3DOverlayKey(s); k1 != k2 {
		t.Fatalf("key not stable across calls: %q vs %q", k1, k2)
	}
	sk.AddLine3D(math.P3(1, 1, 2), math.P3(0, 1, 0))
	if k3, _ := sketch3DOverlayKey(s); k3 == k1 {
		t.Error("key did not change after adding a curve (cache would go stale)")
	}
}

// TestSketch3DCacheHoldsAcrossCameraAndPick is the core property: once built, the cached curve
// bulk is identical across repeated calls (a camera move / pick does not bump the geometry key),
// so the expensive sampling does not repeat every frame.
func TestSketch3DCacheHoldsAcrossCameraAndPick(t *testing.T) {
	s, _ := partWithSketch3D(t)
	first := cachedSketch3DCurves(s)
	if len(first) == 0 {
		t.Fatal("expected curve overlay items for a 3D sketch with lines")
	}
	keyAfterBuild := sketch3DOverlayCache.key
	// A second call must serve the cache unchanged (same key, same item count).
	if again := cachedSketch3DCurves(s); len(again) != len(first) || sketch3DOverlayCache.key != keyAfterBuild {
		t.Errorf("cache rebuilt without a geometry change: items %d->%d", len(first), len(again))
	}
}

// TestSketch3DLiveOverlayAddsPointsAndHighlight checks the per-frame tail: standalone-point
// crosses are drawn (from the cached positions) and re-sizing them does not touch the cached
// curves.
func TestSketch3DLiveOverlayAddsPointsAndHighlight(t *testing.T) {
	s, _ := partWithSketch3D(t)
	cachedSketch3DCurves(s) // populate cache (incl. the standalone point)
	if len(sketch3DOverlayCache.points) != 1 {
		t.Fatalf("cached standalone points = %d, want 1", len(sketch3DOverlayCache.points))
	}
	live := sketch3DLiveOverlay(s, 0.25)
	if len(live) == 0 {
		t.Error("expected a point-cross overlay item from the standalone point")
	}
}

// TestSketch3DBoundsCoverCurvesAndPoints checks the cached bounds enclose both the curve work
// and a standalone point sitting apart from the lines.
func TestSketch3DBoundsCoverCurvesAndPoints(t *testing.T) {
	s, _ := partWithSketch3D(t)
	min, max, ok := cachedSketch3DBounds(s)
	if !ok {
		t.Fatal("expected bounds for a non-empty 3D sketch")
	}
	if max[0] < 5 || max[1] < 5 || max[2] < 5 {
		t.Errorf("bounds max = %v, want to enclose the standalone point at (5,5,5)", max)
	}
	if min[0] > 0 || min[1] > 0 || min[2] > 0 {
		t.Errorf("bounds min = %v, want to enclose the origin line end", min)
	}
}

// TestSketch3DOverlaysCombinesCachedAndLive checks the public overlay is the cached curves plus
// the live tail (the per-frame point cross), so geometry stays visible while points size to zoom.
func TestSketch3DOverlaysCombinesCachedAndLive(t *testing.T) {
	s, _ := partWithSketch3D(t)
	curves := cachedSketch3DCurves(s)
	all := sketch3DOverlays(s, 0.1)
	if len(all) <= len(curves) {
		t.Errorf("overlay items %d should exceed cached curves %d (point cross missing)", len(all), len(curves))
	}
}
