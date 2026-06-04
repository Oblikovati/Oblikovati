// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// fakePointSource is a movable model vertex stand-in for testing 3D include (lost ⇒ the
// reference no longer resolves).
type fakePointSource struct {
	id   string
	pos  math.Point3
	lost bool
}

func (s *fakePointSource) SourceID() string { return s.id }
func (s *fakePointSource) Position() (math.Point3, bool) {
	if s.lost {
		return math.Point3{}, false
	}
	return s.pos, true
}

// fakeCurveSource is a movable model edge stand-in.
type fakeCurveSource struct {
	id   string
	pts  []math.Point3
	lost bool
}

func (s *fakeCurveSource) SourceID() string { return s.id }
func (s *fakeCurveSource) SamplePoints() ([]math.Point3, bool) {
	if s.lost {
		return nil, false
	}
	return s.pts, true
}

// TestIncludePoint3DTracksSource checks an included point starts at its source, is
// reference geometry, re-projects on Update, and freezes on BreakLink.
func TestIncludePoint3DTracksSource(t *testing.T) {
	s := NewSketches3D().Add()
	src := &fakePointSource{id: "v1", pos: math.P3(1, 2, 3)}
	p := s.IncludePoint3D(src)

	if !p.IsConstruction() {
		t.Error("included point should be reference (construction) geometry")
	}
	if p.Position() != math.P3(1, 2, 3) || p.SourceID() != "v1" || !p.Linked() {
		t.Fatalf("included point = %+v, want (1,2,3)/v1/linked", p)
	}
	// The source moves; UpdateIncluded re-projects.
	src.pos = math.P3(4, 5, 6)
	s.UpdateIncluded()
	if p.Position() != math.P3(4, 5, 6) {
		t.Errorf("after source move + update, position = %v, want (4,5,6)", p.Position())
	}
	// Breaking the link freezes the geometry and drops the source id.
	p.BreakLink()
	src.pos = math.P3(9, 9, 9)
	s.UpdateIncluded()
	if p.Position() != math.P3(4, 5, 6) || p.SourceID() != "" || p.Linked() {
		t.Errorf("after break-link, point should freeze at (4,5,6) with no source, got %+v", p)
	}
}

// TestIncludeLostReferenceFreezes checks that when a source's reference is lost, the next
// UpdateIncluded breaks the link and freezes the last geometry (reference-lost behavior).
func TestIncludeLostReferenceFreezes(t *testing.T) {
	s := NewSketches3D().Add()
	src := &fakePointSource{id: "v", pos: math.P3(1, 1, 1)}
	p := s.IncludePoint3D(src)

	src.lost = true
	s.UpdateIncluded()
	if p.Linked() || p.SourceID() != "" {
		t.Errorf("a lost reference should break the link, got linked=%v id=%q", p.Linked(), p.SourceID())
	}
	if p.Position() != math.P3(1, 1, 1) {
		t.Errorf("lost reference should freeze the last position, got %v", p.Position())
	}
	// A curve include behaves the same.
	cs := &fakeCurveSource{id: "e", pts: []math.Point3{{X: 0}, {X: 1}}}
	c := s.IncludeCurve3D(cs)
	cs.lost = true
	s.UpdateIncluded()
	if c.Linked() || len(c.Points()) != 2 {
		t.Errorf("lost curve reference should freeze 2 points, got linked=%v pts=%d", c.Linked(), len(c.Points()))
	}
}

// TestIncludeCurve3DTracksSource checks an included curve mirrors its source polyline.
func TestIncludeCurve3DTracksSource(t *testing.T) {
	s := NewSketches3D().Add()
	src := &fakeCurveSource{id: "e1", pts: []math.Point3{{X: 0}, {X: 1, Y: 1}, {X: 2}}}
	c := s.IncludeCurve3D(src)

	if !c.IsConstruction() || c.SourceID() != "e1" {
		t.Fatalf("included curve = %+v, want reference/e1", c)
	}
	if got := c.Points(); len(got) != 3 || got[1] != math.P3(1, 1, 0) {
		t.Errorf("included curve points = %v, want the source polyline", got)
	}
	// Move the source; update re-samples.
	src.pts = []math.Point3{{X: 0}, {X: 5, Y: 5, Z: 5}}
	s.UpdateIncluded()
	if got := c.Points(); len(got) != 2 || got[1] != math.P3(5, 5, 5) {
		t.Errorf("after update, curve points = %v, want the new polyline", got)
	}
	if !c.Linked() {
		t.Error("curve should be linked before break")
	}
	c.BreakLink()
	src.pts = []math.Point3{{X: 99}}
	s.UpdateIncluded()
	if len(c.Points()) != 2 || c.SourceID() != "" || c.Linked() {
		t.Errorf("after break-link, the curve should freeze with no source; got %d pts / id %q / linked %v", len(c.Points()), c.SourceID(), c.Linked())
	}
}

// TestIncludedGeometrySkippedBySerialize checks included geometry is excluded from .obk
// (it rebinds from its source on recompute).
func TestIncludedGeometrySkippedBySerialize(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	s.IncludePoint3D(&fakePointSource{id: "v", pos: math.P3(2, 2, 2)})

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(data[0].Entities) != 1 {
		t.Errorf("serialized %d entities, want 1 (the line; included point skipped)", len(data[0].Entities))
	}
}
