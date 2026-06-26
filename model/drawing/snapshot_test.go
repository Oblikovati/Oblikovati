// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"bytes"
	"testing"

	"oblikovati.org/api/types"
)

// richDrawing builds a drawing exercising the recipe sections a snapshot must round-trip without a
// body resolver: an extra custom sheet, a drawing sketch with an entity + a hatch, and a couple of
// self-contained annotations (a revision cloud and a feature-control frame, whose glyphs are pure
// functions of their persisted fields, so they survive a round-trip with no model attached).
func richDrawing(t *testing.T) *Content {
	t.Helper()
	c := NewContent()
	c.SetModelReference("widget.opd")
	if _, err := c.Sheets().Add(SheetSpec{Name: "Detail", Size: types.SheetSizeCustom, WidthMM: 500, HeightMM: 350}); err != nil {
		t.Fatalf("Add sheet: %v", err)
	}
	sh := c.Sheets().Active()
	sh.Sketches().Add("Sketch1")
	if _, err := sh.Sketches().AddEntity("Sketch1", types.SketchLineEntity, [][2]float64{{0, 0}, {50, 25}}, 0); err != nil {
		t.Fatalf("AddEntity: %v", err)
	}
	if _, err := sh.Sketches().AddHatchRegion("Sketch1", 10, 10, 40, 20, types.HatchANSI31, 2); err != nil {
		t.Fatalf("AddHatchRegion: %v", err)
	}
	if _, err := sh.Annotations().AddRevisionCloud("Cloud1", 100, 100, 60, 40, "A"); err != nil {
		t.Fatalf("AddRevisionCloud: %v", err)
	}
	if _, err := sh.Annotations().AddFeatureControlFrame("FCF1", 200, 150, types.CharacteristicFlatness, "0.05", nil); err != nil {
		t.Fatalf("AddFeatureControlFrame: %v", err)
	}
	return c
}

// drawingMetrics is the structural fingerprint a snapshot round-trip must preserve: which sheet is
// active, the sheet count, and the active sheet's view/sketch/annotation/dimension counts plus the
// model reference.
type drawingMetrics struct {
	modelRef                              string
	active, sheets, views, sketches, anns int
}

func metricsOf(c *Content) drawingMetrics {
	sh := c.Sheets().Active()
	return drawingMetrics{
		modelRef: c.ModelReference(),
		active:   c.Sheets().active,
		sheets:   c.Sheets().Count(),
		views:    sh.Views().Count(),
		sketches: sh.Sketches().Count(),
		anns:     sh.Annotations().Count(),
	}
}

// TestDrawingSnapshotDeterministic: marshalling a fixed drawing twice yields identical bytes. The
// undo stream's no-op-delta check (bytes.Equal) depends on this, or recomputes that change nothing
// would record phantom undo steps.
func TestDrawingSnapshotDeterministic(t *testing.T) {
	c := richDrawing(t)
	a, err := c.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	b, err := c.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot 2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Error("MarshalSnapshot is not deterministic for a fixed drawing (breaks no-op detection)")
	}
}

// TestDrawingSnapshotRoundTripPreservesState: a snapshot restored onto the SAME drawing reproduces
// its structure exactly (the redo direction), byte-stable since there is no body to re-project.
func TestDrawingSnapshotRoundTripPreservesState(t *testing.T) {
	c := richDrawing(t)
	want := metricsOf(c)
	snap, err := c.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if err := c.RestoreSnapshot(snap); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if got := metricsOf(c); got != want {
		t.Errorf("round-trip changed structure:\n got %+v\nwant %+v", got, want)
	}
	again, err := c.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot after restore: %v", err)
	}
	if !bytes.Equal(snap, again) {
		t.Errorf("drawing snapshot not byte-stable across round-trip (len %d vs %d)", len(snap), len(again))
	}
}

// TestDrawingSnapshotRestoreReplacesOtherDrawing: restoring a snapshot onto a DIFFERENT, populated
// drawing yields exactly the snapshot's state — proving RestoreSnapshot is a full replace (the
// property undo relies on to navigate between arbitrary states), not a merge onto the default sheet.
func TestDrawingSnapshotRestoreReplacesOtherDrawing(t *testing.T) {
	src := richDrawing(t)
	want := metricsOf(src)
	snap, err := src.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}

	dst := NewContent()
	dst.SetModelReference("other.opd")
	dst.Sheets().Active().Sketches().Add("Foreign")
	if _, err := dst.Sheets().Active().Sketches().AddEntity("Foreign", types.SketchCircleEntity, [][2]float64{{0, 0}}, 5); err != nil {
		t.Fatalf("seed foreign sketch: %v", err)
	}

	if err := dst.RestoreSnapshot(snap); err != nil {
		t.Fatalf("RestoreSnapshot onto other drawing: %v", err)
	}
	if got := metricsOf(dst); got != want {
		t.Errorf("restore onto populated drawing did not fully replace:\n got %+v\nwant %+v", got, want)
	}
	if _, ok := dst.Sheets().Active().Sketches().ByName("Foreign"); ok {
		t.Error("foreign drawing sketch survived the restore (merge, not replace)")
	}
}

// TestEmptyDrawingSnapshotRoundTrips: a freshly created drawing snapshots and restores without
// error, staying at its one default sheet — the baseline a document's undo stream captures on open.
func TestEmptyDrawingSnapshotRoundTrips(t *testing.T) {
	c := NewContent()
	snap, err := c.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot empty: %v", err)
	}
	if err := c.RestoreSnapshot(snap); err != nil {
		t.Fatalf("RestoreSnapshot empty: %v", err)
	}
	if c.Sheets().Count() != 1 {
		t.Errorf("empty drawing has %d sheets after round-trip, want 1 default", c.Sheets().Count())
	}
}

// TestDrawingRestoreSnapshotRejectsBadJSON: the undo restore must reject a corrupt event payload
// rather than apply a half-state. Malformed JSON is decoded before the sheet rebuild, so the drawing
// is left untouched.
func TestDrawingRestoreSnapshotRejectsBadJSON(t *testing.T) {
	c := richDrawing(t)
	before := metricsOf(c)
	if err := c.RestoreSnapshot([]byte("{not valid json")); err == nil {
		t.Error("RestoreSnapshot accepted invalid JSON")
	}
	if got := metricsOf(c); got != before {
		t.Errorf("drawing mutated by a failed parse-restore:\n got %+v\nwant %+v", got, before)
	}
}
