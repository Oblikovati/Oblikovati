// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"bytes"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// richPart builds a part exercising every recipe section the undo snapshot must round-trip:
// parameters, a 2D sketch consumed by an extrude, a work plane, a standalone 3D sketch, and a
// fillet keyed on a real edge. Returns the document and its definition.
func richPart(t *testing.T) (*doc.Document, *compdef.PartComponentDefinition) {
	t.Helper()
	ws := doc.NewWorkspace(nil)
	d, err := compdef.AddPart(ws, "Part1", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := d.Content().(*compdef.PartComponentDefinition)
	if _, err := def.Parameters().AddUserParameter("w", "4 cm"); err != nil {
		t.Fatalf("param w: %v", err)
	}
	if _, err := def.Parameters().AddUserParameter("h", "w / 2"); err != nil {
		t.Fatalf("param h: %v", err)
	}
	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	s3 := def.Sketches3D().Add()
	s3.AddLine3D(math.P3(0, 0, 0), math.P3(1, 1, 1))
	def.Recompute()
	edgeKey := def.SurfaceBodies().All()[0].Edges()[0].ReferenceKey()
	feature.NewDressUpFeatures(def.Features()).AddFillet([][]byte{edgeKey}, func() float64 { return 0.2 })
	def.Recompute()
	return d, def
}

// partMetrics is the structural fingerprint a snapshot round-trip must preserve exactly (counts
// + a derived parameter value + the regenerated body count) — compared instead of raw bytes so
// the check is immune to last-ULP float drift in the geometry recompute (which affects YAML too).
type partMetrics struct {
	sketches, sketches3D, features, workPlanes, params, bodies int
	hDerived                                                   float64
}

func metricsOf(def *compdef.PartComponentDefinition) partMetrics {
	h, _ := def.Parameters().ByName("h")
	var hv float64
	if h != nil {
		hv = h.Value().Value
	}
	return partMetrics{
		sketches:   def.Sketches().Count(),
		sketches3D: def.Sketches3D().Count(),
		features:   def.Features().Count(),
		workPlanes: def.WorkPlanes().Count(),
		params:     def.Parameters().Count(),
		bodies:     len(def.SurfaceBodies().All()),
		hDerived:   hv,
	}
}

// TestPartSnapshotDeterministic: marshalling the same unchanged state twice yields identical
// bytes. The undo stream's no-op delta check (bytes.Equal) depends on this — a non-deterministic
// codec would record phantom undo steps on recomputes that change nothing.
func TestPartSnapshotDeterministic(t *testing.T) {
	_, def := richPart(t)
	a, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	b, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot 2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Error("MarshalSnapshot is not deterministic for a fixed state (breaks no-op detection)")
	}
}

// TestPartSnapshotRoundTripPreservesState: a snapshot restored onto the SAME part reproduces its
// structure exactly (the redo direction), and the geometry regenerates (a body is rebuilt).
func TestPartSnapshotRoundTripPreservesState(t *testing.T) {
	_, def := richPart(t)
	want := metricsOf(def)
	snap, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}
	if err := def.RestoreSnapshot(snap); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if got := metricsOf(def); got != want {
		t.Errorf("round-trip changed structure:\n got %+v\nwant %+v", got, want)
	}
}

// TestPartSnapshotRestoreReplacesOtherPart: restoring a snapshot onto a DIFFERENT, populated part
// yields exactly the snapshot's state — proving RestoreSnapshot is a full replace (reset+apply),
// the property undo relies on to navigate between arbitrary states.
func TestPartSnapshotRestoreReplacesOtherPart(t *testing.T) {
	_, src := richPart(t)
	want := metricsOf(src)
	snap, err := src.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot: %v", err)
	}

	// A second, differently-populated part.
	ws := doc.NewWorkspace(nil)
	d2, _ := compdef.AddPart(ws, "Part2", true)
	dst := d2.Content().(*compdef.PartComponentDefinition)
	dst.Sketches().Add(sketch.XYPlane())
	if _, err := dst.Parameters().AddUserParameter("other", "9 mm"); err != nil {
		t.Fatalf("param other: %v", err)
	}
	dst.Recompute()

	if err := dst.RestoreSnapshot(snap); err != nil {
		t.Fatalf("RestoreSnapshot onto other part: %v", err)
	}
	if got := metricsOf(dst); got != want {
		t.Errorf("restore onto populated part did not fully replace:\n got %+v\nwant %+v", got, want)
	}
	if _, ok := dst.Parameters().ByName("other"); ok {
		t.Error("foreign parameter survived the restore (merge, not replace)")
	}
}

// TestEmptyPartSnapshotRoundTrips: an empty part snapshots and restores without error, staying
// empty — the baseline a document's undo stream captures the moment it opens.
func TestEmptyPartSnapshotRoundTrips(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Empty", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	snap, err := def.MarshalSnapshot()
	if err != nil {
		t.Fatalf("MarshalSnapshot empty: %v", err)
	}
	if err := def.RestoreSnapshot(snap); err != nil {
		t.Fatalf("RestoreSnapshot empty: %v", err)
	}
	if m := metricsOf(def); m.sketches != 0 || m.features != 0 || m.bodies != 0 {
		t.Errorf("empty part not empty after round-trip: %+v", m)
	}
}

// TestParamsOnlySnapshotByteStable: a part with no geometry recompute (parameters only) is
// byte-stable across a full round-trip — there is no float drift to mask, so restoring then
// re-marshalling must reproduce the exact snapshot. Pins that the codec itself loses nothing.
func TestParamsOnlySnapshotByteStable(t *testing.T) {
	ws := doc.NewWorkspace(nil)
	d, _ := compdef.AddPart(ws, "Params", true)
	def := d.Content().(*compdef.PartComponentDefinition)
	_, _ = def.Parameters().AddUserParameter("a", "10 mm")
	_, _ = def.Parameters().AddUserParameter("b", "a * 2 + 1 mm")
	_, _ = def.Parameters().AddTextUserParameter("name", "bracket")
	def.Recompute()

	s1, _ := def.MarshalSnapshot()
	if err := def.RestoreSnapshot(s1); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	s2, _ := def.MarshalSnapshot()
	if !bytes.Equal(s1, s2) {
		t.Errorf("params-only snapshot not byte-stable across round-trip (len %d vs %d)", len(s1), len(s2))
	}
}

// BenchmarkPartSnapshot vs BenchmarkPartMarshalRecipe guard the reason this codec exists: the
// undo snapshot must be much faster than the YAML save format (#1147). Same part, two codecs.
func BenchmarkPartSnapshot(b *testing.B) {
	_, def := richPartB(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := def.MarshalSnapshot(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPartMarshalRecipe(b *testing.B) {
	_, def := richPartB(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := def.MarshalRecipe(); err != nil {
			b.Fatal(err)
		}
	}
}

// richPartB is richPart for a benchmark (testing.B has no *testing.T).
func richPartB(b *testing.B) (*doc.Document, *compdef.PartComponentDefinition) {
	b.Helper()
	ws := doc.NewWorkspace(nil)
	d, err := compdef.AddPart(ws, "Part1", true)
	if err != nil {
		b.Fatalf("AddPart: %v", err)
	}
	def := d.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	return d, def
}
