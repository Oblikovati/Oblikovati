// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
	"oblikovati/model/health"
	"oblikovati/model/sketch"
)

// squareWithHoleSketch returns a sketch with an outer square and a concentric inner
// square (a hole), so the detected profile has one outer and one inner loop.
func squareWithHoleSketch(side, hole float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	o0 := s.Points().Add(math.P2(0, 0))
	o1 := s.Points().Add(math.P2(side, 0))
	o2 := s.Points().Add(math.P2(side, side))
	o3 := s.Points().Add(math.P2(0, side))
	s.Lines().Add(o0, o1)
	s.Lines().Add(o1, o2)
	s.Lines().Add(o2, o3)
	s.Lines().Add(o3, o0)
	lo := (side - hole) / 2
	hi := lo + hole
	i0 := s.Points().Add(math.P2(lo, lo))
	i1 := s.Points().Add(math.P2(hi, lo))
	i2 := s.Points().Add(math.P2(hi, hi))
	i3 := s.Points().Add(math.P2(lo, hi))
	s.Lines().Add(i0, i1)
	s.Lines().Add(i1, i2)
	s.Lines().Add(i2, i3)
	s.Lines().Add(i3, i0)
	return s
}

// plateWithHolesSketch returns a w×h rectangle (corner at the origin) with one square hole
// of the given side at each requested centre, so the detected profile carries one outer and
// len(centres) inner loops — the multi-hole cap case.
func plateWithHolesSketch(w, h, hole float64, centres [][2]float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	addLoop := func(pts [][2]float64) {
		ids := make([]*sketch.Point, len(pts))
		for i, p := range pts {
			ids[i] = s.Points().Add(math.P2(p[0], p[1]))
		}
		for i := range ids {
			s.Lines().Add(ids[i], ids[(i+1)%len(ids)])
		}
	}
	addLoop([][2]float64{{0, 0}, {w, 0}, {w, h}, {0, h}})
	r := hole / 2
	for _, c := range centres {
		addLoop([][2]float64{{c[0] - r, c[1] - r}, {c[0] + r, c[1] - r}, {c[0] + r, c[1] + r}, {c[0] - r, c[1] + r}})
	}
	return s
}

func TestBoundaryPatchFillsClosedBoundary(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewBoundaryPatchFeatures(fs).Add(squareSketch(4), 0, PatchTangent)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("boundary patch went unhealthy: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("result has %d bodies, want 1", len(bodies))
	}
	body := bodies[0]
	if body.IsSolid() {
		t.Error("boundary patch should be a surface body, not a solid")
	}
	if got := len(body.Faces()); got != 1 {
		t.Errorf("patch has %d faces, want 1", got)
	}
	// An open surface patch: every boundary edge is open (used by one face).
	if got := len(ops.BoundaryEdges(body)); got != 4 {
		t.Errorf("patch has %d boundary edges, want 4 (the square perimeter)", got)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("patch failed validation: %+v", r)
	}
}

func TestBoundaryPatchHonorsConditionPerLoop(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewBoundaryPatchFeatures(fs).Add(squareSketch(2), 0, PatchCurvature)
	def := pf.Definition().(*BoundaryPatchFeature).Definition()
	if def.Loops.Count() != 1 {
		t.Fatalf("definition has %d loops, want 1", def.Loops.Count())
	}
	if c := def.Loops.Item(0).Condition; c != PatchCurvature {
		t.Errorf("loop condition = %v, want PatchCurvature (G2)", c)
	}
}

func TestBoundaryPatchWithHoleCutsInnerLoop(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewBoundaryPatchFeatures(fs).Add(squareWithHoleSketch(6, 2), 0, PatchFree)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("patch-with-hole went unhealthy: %+v", pf.Health())
	}
	body := fs.Result()[0]
	// One trimmed face; its boundary edges are the outer + inner loops (4 + 4).
	if got := len(body.Faces()); got != 1 {
		t.Errorf("patch has %d faces, want 1", got)
	}
	if got := len(ops.BoundaryEdges(body)); got != 8 {
		t.Errorf("patch-with-hole has %d boundary edges, want 8 (outer+inner)", got)
	}
}

func TestBoundaryPatchGoesSickOnOpenProfile(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	c := s.Points().Add(math.P2(2, 1))
	s.Lines().Add(a, b)
	s.Lines().Add(b, c) // open chain
	fs := NewPartFeatures(nil, nil)
	pf := NewBoundaryPatchFeatures(fs).Add(s, 0, PatchFree)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("patch of an open profile = %v, want sick", pf.Health().Status)
	}
}

func TestRuledSurfaceBuildsBand(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewRuledSurfaceFeatures(fs).AddByDistance(squareSketch(2), 0, RuledNormal, func() float64 { return 3 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("ruled surface went unhealthy: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if body.IsSolid() {
		t.Error("ruled surface should be a surface body, not a solid")
	}
	// A square ruled by distance → 4 side faces, no caps.
	if got := len(body.Faces()); got != 4 {
		t.Errorf("ruled band has %d faces, want 4", got)
	}
	// The band is open: its top and bottom loops are boundary edges (4 + 4).
	if got := len(ops.BoundaryEdges(body)); got != 8 {
		t.Errorf("ruled band has %d boundary edges, want 8 (top+bottom loops)", got)
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("ruled band failed validation: %+v", r)
	}
	if z := body.RangeBox().Diagonal().Z; !approxEq(z, 3) {
		t.Errorf("ruled band height = %v, want 3", z)
	}
}

func TestRuledSurfaceTangentDefers(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewRuledSurfaceFeatures(fs).AddByDistance(squareSketch(2), 0, RuledTangent, func() float64 { return 3 })
	fs.Recompute()
	// Inputs resolved, geometry deferred (tangent ruling needs adjacent-face data).
	if pf.Health().Status != health.Warning {
		t.Errorf("tangent ruled surface = %v, want warning (deferred)", pf.Health().Status)
	}
	if len(fs.Result()) != 0 {
		t.Errorf("deferred ruled surface produced %d bodies, want 0 (passthrough)", len(fs.Result()))
	}
}

func TestRuledSurfaceGoesSickOnOpenProfile(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	s.Lines().Add(a, b) // single open segment
	fs := NewPartFeatures(nil, nil)
	pf := NewRuledSurfaceFeatures(fs).AddByDistance(s, 0, RuledNormal, func() float64 { return 3 })
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("ruled surface of an open profile = %v, want sick", pf.Health().Status)
	}
}

func TestRuledSurfaceZeroDistanceErrors(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewRuledSurfaceFeatures(fs).AddByDistance(squareSketch(2), 0, RuledNormal, nil)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("zero-distance ruled surface = %v, want sick", pf.Health().Status)
	}
}
