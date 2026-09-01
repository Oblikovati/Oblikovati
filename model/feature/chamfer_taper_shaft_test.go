// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// stepShaftSketch is the #1689 meridian (cm): (0,0)→(0.8,0)→(1.0,1.0)→(1.0,2.0)→(0,2.0) — a bottom
// cap + oblique cone + straight cylinder wall + top cap, closed on the Y axis. Revolved it is a
// stepped taper shaft whose top rim (r=1 at y=2) is shared by the cylinder wall and the top cap.
func stepShaftSketch() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(0.8, 0))
	p2 := s.Points().Add(math.P2(1.0, 1.0))
	p3 := s.Points().Add(math.P2(1.0, 2.0))
	p4 := s.Points().Add(math.P2(0, 2.0))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	s.Lines().Add(p2, p3)
	s.Lines().Add(p3, p4)
	s.Lines().Add(p4, p0)
	return s
}

// TestChamferOfTaperShaftRimDoesNotCollapse pins the #1689 regression: chamfering the top rim of a
// revolved stepped shaft (a body with a cone face, so the analytic fast path declines and the
// general wedge-cut boolean runs) must yield a VALID, non-collapsed solid — not the empty body the
// pre-#1600/#1601/#1726 boolean shipped while still reporting healthy. Guards both halves of the bug:
// the geometric collapse AND the silent gate (a real collapse now records a volume-reject Defect and
// falls through the guarded path rather than passing as OK).
func TestChamferOfTaperShaftRimDoesNotCollapse(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~20s): `make test-corpus`")
	}
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewRevolveFeatures(fs).Add(stepShaftSketch(), 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	body := fs.Result()[0]
	volBefore := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume

	var rim []byte
	for _, e := range body.Edges() {
		if c, ok := e.Geometry().(geom.Circle); ok {
			if cy := float64(c.Center.Y); cy > 1.9 && cy < 2.1 && float64(c.Radius) > 0.9 {
				rim = e.ReferenceKey()
			}
		}
	}
	if rim == nil {
		t.Fatalf("no top rim circle edge (r≈1 at y≈2) among %d edges", len(body.Edges()))
	}

	pf := NewDressUpFeatures(fs).AddChamfer([][]byte{rim}, func() float64 { return 0.06 }) // 0.6 mm
	fs.Recompute()

	res := fs.Result()
	if len(res) == 0 {
		t.Fatalf("#1689: chamfer destroyed the body (empty result) while health=%+v", pf.Health())
	}
	out := res[0]
	if v := ops.Validate(out); !v.Valid || !out.IsSolid() {
		t.Fatalf("#1689: chamfered shaft is not a valid solid: %+v", v.Issues)
	}
	volAfter := query.BodyGeometryProperties(out, ops.DefaultQuality()).Volume
	// The general path re-facets the analytic body, so exact volume shifts; a chamfer must still
	// leave the bulk of the shaft (well over half) — a collapse drops it toward zero.
	if volAfter < volBefore*0.9 {
		t.Errorf("#1689: chamfer removed too much — %g → %g cm^3 (want ≳ %g); health=%+v",
			volBefore, volAfter, volBefore*0.9, pf.Health())
	}
	if !pf.Health().OK() {
		t.Errorf("#1689: chamfer of a valid taper-shaft rim reported sick: %+v", pf.Health())
	}
}
