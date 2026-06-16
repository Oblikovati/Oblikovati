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

// partWithSketch returns a session whose active part holds one finished sketch with
// a few line segments — the fixture for the finished-sketch overlay cache.
func partWithSketch(t *testing.T) (*app.Session, *sketch.Sketch) {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "p.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	sk.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(1, 1))
	return s, sk
}

// TestSketchOverlayKeyStableThenInvalidates pins the cache key: identical state
// yields an identical key, and a geometry change (a new line, then a new sketch)
// changes it so the overlay rebuilds.
func TestSketchOverlayKeyStableThenInvalidates(t *testing.T) {
	s, sk := partWithSketch(t)
	k1, ok := sketchOverlayKey(s)
	if !ok {
		t.Fatal("sketchOverlayKey not available for an active part")
	}
	if k2, _ := sketchOverlayKey(s); k1 != k2 {
		t.Fatalf("key not stable across calls: %q vs %q", k1, k2)
	}
	sk.Lines().AddByTwoPoints(math.P2(1, 1), math.P2(0, 1))
	k3, _ := sketchOverlayKey(s)
	if k3 == k1 {
		t.Error("key did not change after adding a line (cache would go stale)")
	}
	activePart(s).Sketches().Add(sketch.XYPlane())
	if k4, _ := sketchOverlayKey(s); k4 == k3 {
		t.Error("key did not change after adding a sketch")
	}
}

// TestCachedSketchOverlayMatchesUncached checks the cached overlay returns the same
// geometry as a direct build, and that an unchanged state keeps serving it.
func TestCachedSketchOverlayMatchesUncached(t *testing.T) {
	s, _ := partWithSketch(t)
	sketchOverlayCache.key = "" // isolate from other tests sharing the package cache
	direct := partSketchOverlays(s)
	cached := cachedPartSketchOverlays(s)
	if len(direct) != len(cached) || len(cached) == 0 {
		t.Fatalf("cached overlay has %d items, direct build %d (want equal, non-zero)", len(cached), len(direct))
	}
	if again := cachedPartSketchOverlays(s); len(again) != len(cached) {
		t.Errorf("second cached call returned %d items, want %d", len(again), len(cached))
	}
}
