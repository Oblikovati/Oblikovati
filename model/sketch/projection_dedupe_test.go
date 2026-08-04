// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// Projecting the same source twice must not stack two coincident reference points on top of
// each other — two overlapping snap targets the user cannot tell apart or select between. It
// matters most at the origin, which every new sketch already projects automatically (#2016).

func TestProjectPointReusesAnExistingProjection(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := &identifiedPointSource{id: "origin/point/center", kind: "workPoint", pos: math.P3(0, 0, 0)}

	first := s.ProjectPoint(src)
	again := s.ProjectPoint(src)

	if first != again {
		t.Error("projecting the same source twice returned two projections, want the existing one")
	}
	if got := countProjectedPoints(s); got != 1 {
		t.Errorf("projected points = %d, want 1", got)
	}
}

func TestProjectCurveReusesAnExistingProjection(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	src := &identifiedCurveSource{id: "origin/axis/x", kind: "workAxis",
		pts: []math.Point3{math.P3(-1, 0, 0), math.P3(1, 0, 0)}}

	if first, again := s.ProjectCurve(src), s.ProjectCurve(src); first != again {
		t.Error("projecting the same curve source twice returned two projections")
	}
	if got := countProjectedCurves(s); got != 1 {
		t.Errorf("projected curves = %d, want 1", got)
	}
}

// Distinct sources still project separately — the dedupe keys on identity, not on position.
func TestProjectPointKeepsDistinctSourcesApart(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.ProjectPoint(&identifiedPointSource{id: "V1", kind: "vertex", pos: math.P3(1, 0, 0)})
	s.ProjectPoint(&identifiedPointSource{id: "V2", kind: "vertex", pos: math.P3(2, 0, 0)})

	if got := countProjectedPoints(s); got != 2 {
		t.Errorf("projected points = %d, want 2 (distinct sources)", got)
	}
}

// A source with no stable identity must never match another: two anonymous sources are two
// projections, not one, because nothing says they are the same thing.
func TestProjectPointDoesNotMergeAnonymousSources(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.ProjectPoint(&identifiedPointSource{id: "", kind: "", pos: math.P3(1, 0, 0)})
	s.ProjectPoint(&identifiedPointSource{id: "", kind: "", pos: math.P3(2, 0, 0)})

	if got := countProjectedPoints(s); got != 2 {
		t.Errorf("projected points = %d, want 2 (anonymous sources stay distinct)", got)
	}
}

func countProjectedPoints(s *Sketch) int {
	n := 0
	for _, e := range s.Entities() {
		if _, ok := e.(*ProjectedPoint); ok {
			n++
		}
	}
	return n
}

func countProjectedCurves(s *Sketch) int {
	n := 0
	for _, e := range s.Entities() {
		if _, ok := e.(*ProjectedCurve); ok {
			n++
		}
	}
	return n
}

// identifiedPointSource is a point source that carries the stable (kind, id) identity the
// dedupe keys on — unlike the bare fakePointSource, which is deliberately anonymous.
type identifiedPointSource struct {
	id   string
	kind string
	pos  math.Point3
}

func (s *identifiedPointSource) SourceID() string   { return s.id }
func (s *identifiedPointSource) SourceKind() string { return s.kind }
func (s *identifiedPointSource) Position() (math.Point3, bool) {
	return s.pos, true
}

// identifiedCurveSource is [identifiedPointSource] for a curve.
type identifiedCurveSource struct {
	id   string
	kind string
	pts  []math.Point3
}

func (s *identifiedCurveSource) SourceID() string   { return s.id }
func (s *identifiedCurveSource) SourceKind() string { return s.kind }
func (s *identifiedCurveSource) SamplePoints() ([]math.Point3, bool) {
	return s.pts, true
}
