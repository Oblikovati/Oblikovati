// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// fakeCloudPointSource is a named fake: it reports a settable model-space position, and can report
// "gone" to exercise the freeze path.
type fakeCloudPointSource struct {
	id  string
	pos math.Point3
	ok  bool
}

func (f *fakeCloudPointSource) SourceID() string              { return f.id }
func (f *fakeCloudPointSource) Position() (math.Point3, bool) { return f.pos, f.ok }

// TestCloudPointFollowsAndFreezes: the work point tracks the source as it moves, and freezes the
// last good position when the source is gone (#645).
func TestCloudPointFollowsAndFreezes(t *testing.T) {
	src := &fakeCloudPointSource{id: "Scan", pos: math.P3(1, 2, 3), ok: true}
	points := NewWorkGeometry().WorkPoints()
	wp := points.AddByCloudPoint(src)

	wp.recompute(nil)
	if wp.Point() != math.P3(1, 2, 3) {
		t.Errorf("initial point = %v, want (1,2,3)", wp.Point())
	}
	src.pos = math.P3(4, 5, 6) // the cloud moves
	wp.recompute(nil)
	if wp.Point() != math.P3(4, 5, 6) {
		t.Errorf("re-derived point = %v, want (4,5,6) (should follow the cloud)", wp.Point())
	}
	src.ok = false // source gone → freeze at the last good position
	wp.recompute(nil)
	if wp.Point() != math.P3(4, 5, 6) {
		t.Errorf("frozen point = %v, want (4,5,6)", wp.Point())
	}
}

// TestCloudPointSerializeRoundTrip: the cloud id and frozen position round-trip, and a relink
// (carrying the frozen position) restores associativity (#645).
func TestCloudPointSerializeRoundTrip(t *testing.T) {
	src := &fakeCloudPointSource{id: "Scan", pos: math.P3(7, 8, 9), ok: true}
	g := NewWorkGeometry()
	g.WorkPoints().AddByCloudPoint(src)

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	var pt *WorkFeatureData
	for i := range data {
		if data[i].Kind == "point-cloud-point" {
			pt = &data[i]
		}
	}
	if pt == nil || pt.CloudID != "Scan" || len(pt.Position) != 3 || pt.Position[0] != 7 {
		t.Fatalf("serialized point-cloud-point = %+v, want cloud Scan at (7,8,9)", pt)
	}

	g2 := NewWorkGeometry()
	if err := ApplyWork(g2, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	wp := cloudPointOf(t, g2.WorkPoints())
	wp.recompute(nil) // no source yet → frozen
	if wp.Point() != math.P3(7, 8, 9) {
		t.Errorf("restored frozen point = %v, want (7,8,9)", wp.Point())
	}

	var gotFrozen math.Point3
	moved := &fakeCloudPointSource{id: "Scan", pos: math.P3(70, 80, 90), ok: true}
	n := g2.WorkPoints().RelinkCloudPoints(func(id string, frozen math.Point3) (PointFromCloudSource, bool) {
		gotFrozen = frozen
		if id == "Scan" {
			return moved, true
		}
		return nil, false
	})
	if n != 1 || gotFrozen != math.P3(7, 8, 9) {
		t.Fatalf("relinked %d (frozen %v), want 1 carrying (7,8,9)", n, gotFrozen)
	}
	wp.recompute(nil)
	if wp.Point() != math.P3(70, 80, 90) {
		t.Errorf("after relink point = %v, want (70,80,90)", wp.Point())
	}
}

// TestCloudPointEvalAndRelinkEdges covers the no-source/no-position error, CloudID, FrozenPosition,
// and a relink id-mismatch (#645).
func TestCloudPointEvalAndRelinkEdges(t *testing.T) {
	d := &pointCloudPointDef{cloudID: "X"}
	if _, err := d.eval(nil); err == nil {
		t.Error("eval with no source and no prior position should error")
	}
	if d.CloudID() != "X" || d.FrozenPosition() != (math.Point3{}) {
		t.Errorf("CloudID/FrozenPosition = %q/%v", d.CloudID(), d.FrozenPosition())
	}
	if d.relink(&fakeCloudPointSource{id: "other"}) {
		t.Error("relink with a mismatched id should report false")
	}
}

func cloudPointOf(t *testing.T, points *WorkPoints) *WorkPoint {
	t.Helper()
	for i := 0; i < points.Count(); i++ {
		if _, ok := points.Item(i).def.(*pointCloudPointDef); ok {
			return points.Item(i)
		}
	}
	t.Fatal("no point-cloud-point in the restored part")
	return nil
}
