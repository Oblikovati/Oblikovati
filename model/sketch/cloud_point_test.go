// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// fakeCloudAnchor is a named fake scan-point source: a settable model position (and "gone" flag),
// plus the persisted id and cloud-local anchor.
type fakeCloudAnchor struct {
	id    string
	local math.Point3
	pos   math.Point3
	ok    bool
}

func (f *fakeCloudAnchor) SourceID() string                   { return f.id }
func (f *fakeCloudAnchor) LocalAnchor() math.Point3           { return f.local }
func (f *fakeCloudAnchor) ModelPosition() (math.Point3, bool) { return f.pos, f.ok }

// TestCloudAnchoredPointFollows: a cloud-anchored sketch point projects onto the sketch plane and
// re-projects when the source moves, on UpdateProjections (#645).
func TestCloudAnchoredPointFollows(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := &fakeCloudAnchor{id: "Scan", local: math.P3(1, 1, 5), pos: math.P3(3, 4, 7), ok: true}
	p := s.AddCloudAnchoredPoint(src)
	if p.Position() != math.P2(3, 4) { // (3,4,7) onto XY → (3,4)
		t.Errorf("initial sketch point = %v, want (3,4)", p.Position())
	}
	src.pos = math.P3(8, 9, 2) // the cloud moves
	s.UpdateProjections()
	if p.Position() != math.P2(8, 9) {
		t.Errorf("re-projected point = %v, want (8,9) (should follow the cloud)", p.Position())
	}
}

// TestCloudAnchorSerializeRoundTrip: the provenance round-trips, and a relink re-projects the
// restored point to the re-attached cloud (#645).
func TestCloudAnchorSerializeRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	src := &fakeCloudAnchor{id: "CapstanScan", local: math.P3(2, 3, 4), pos: math.P3(5, 6, 7), ok: true}
	s.AddCloudAnchoredPoint(src)

	out := roundTrip(t, sc)
	anchors := out.CloudAnchors()
	if len(anchors) != 1 || anchors[0].CloudID != "CapstanScan" || anchors[0].Local != math.P3(2, 3, 4) {
		t.Fatalf("restored anchors = %+v, want one CapstanScan at local (2,3,4)", anchors)
	}
	restored := pointByID(out, anchors[0].PointID)
	if restored == nil || restored.Position() != math.P2(5, 6) {
		t.Fatalf("restored point = %v, want (5,6)", restored)
	}

	moved := &fakeCloudAnchor{id: "CapstanScan", local: math.P3(2, 3, 4), pos: math.P3(50, 60, 7), ok: true}
	n := out.RelinkCloudAnchors(func(id string, local math.Point3) (CloudPointAnchor, bool) {
		if id == "CapstanScan" && local == math.P3(2, 3, 4) {
			return moved, true
		}
		return nil, false
	})
	if n != 1 {
		t.Fatalf("relinked %d, want 1", n)
	}
	if restored.Position() != math.P2(50, 60) {
		t.Errorf("after relink point = %v, want (50,60) (follows the re-attached cloud)", restored.Position())
	}
}

// TestRelinkCloudAnchorsNoMatch: a relink that finds no matching cloud leaves the point detached.
func TestRelinkCloudAnchorsNoMatch(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddCloudAnchoredPoint(&fakeCloudAnchor{id: "Scan", local: math.P3(0, 0, 0), pos: math.P3(1, 1, 0), ok: true})
	if n := s.RelinkCloudAnchors(func(string, math.Point3) (CloudPointAnchor, bool) { return nil, false }); n != 0 {
		t.Errorf("relinked %d, want 0 (no match)", n)
	}
}

func pointByID(s *Sketch, id ID) *Point {
	for _, p := range s.AllPoints() {
		if p.id == id {
			return p
		}
	}
	return nil
}

// TestCloudAnchorDetachedAndUnfit covers the unfit seed (source can't fit → origin) and the
// detached-anchor skip during re-projection (a restored anchor before relink) (#645).
func TestCloudAnchorDetachedAndUnfit(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.AddCloudAnchoredPoint(&fakeCloudAnchor{id: "S", ok: false}) // ModelPosition !ok → seeded at origin
	if p.Position() != math.P2(0, 0) {
		t.Errorf("unfit seed = %v, want (0,0)", p.Position())
	}
	q := s.Points().Add(math.P2(5, 5))
	s.RestoreCloudAnchor(q, "S2", math.P3(0, 0, 0)) // a restored anchor: source is nil until relinked
	s.UpdateProjections()                           // must skip the detached anchor, leaving q put
	if q.Position() != math.P2(5, 5) {
		t.Errorf("detached anchor moved to %v, want (5,5)", q.Position())
	}
}
