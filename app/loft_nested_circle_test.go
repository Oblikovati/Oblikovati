// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// #1526: lofting "the circle in the first sketch" to a circle in the second produced no preview and
// no body. The first sketch had a circle drawn inside a rectangle, and the profile finder absorbed the
// circle as a hole of the rectangle (one annulus region) — so there was no disk profile to loft, and
// picking the circle gave the annulus, whose hole count (1) cannot pair with the plain circle's (0).
// The fix offers the disk as its own region; these lock the end-to-end loft.

// yzPlaneAtX is the YZ plane (normal +X) shifted to x — the bug's two parallel circle planes.
func yzPlaneAtX(x float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(math.Scalar(x), 0, 0), math.V3(0, 1, 0).AsUnit(), math.V3(0, 0, 1).AsUnit())
	return p
}

func circleSketchOn(def *compdef.PartComponentDefinition, plane sketch.Plane, r float64) *sketch.Sketch {
	sk := def.Sketches().Add(plane)
	sk.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(r))
	return sk
}

// rectWithInnerCircle mirrors #1526's Sketch1: a rectangle that encloses a centred circle.
func rectWithInnerCircle(def *compdef.PartComponentDefinition, plane sketch.Plane, r float64) *sketch.Sketch {
	sk := def.Sketches().Add(plane)
	p0 := sk.Points().Add(math.P2(-3.354, -3.367))
	p1 := sk.Points().Add(math.P2(3.545, -3.367))
	p2 := sk.Points().Add(math.P2(3.545, 3.668))
	p3 := sk.Points().Add(math.P2(-3.354, 3.668))
	sk.Lines().Add(p0, p1)
	sk.Lines().Add(p1, p2)
	sk.Lines().Add(p2, p3)
	sk.Lines().Add(p3, p0)
	sk.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(r))
	return sk
}

func loftPartSession(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	return s, def
}

// diskProfileIndex returns the index of the sketch's hole-less (disk) region.
func diskProfileIndex(t *testing.T, sk *sketch.Sketch) int {
	t.Helper()
	ps := sk.Profiles()
	for i := 0; i < ps.Count(); i++ {
		if len(ps.Item(i).InnerLoops()) == 0 {
			return i
		}
	}
	t.Fatalf("sketch offers no hole-less disk region (regions=%d)", ps.Count())
	return -1
}

// TestLoftCircleInsideRectangleToCircle is the #1526 regression: the disk of a circle drawn inside a
// rectangle lofts to a plain circle on a parallel plane, with a live preview and a valid solid.
func TestLoftCircleInsideRectangleToCircle(t *testing.T) {
	t.Parallel()
	s, def := loftPartSession(t)
	sk1 := rectWithInnerCircle(def, yzPlaneAtX(0), 2.539)
	sk2 := circleSketchOn(def, yzPlaneAtX(10), 3.549)
	if sk1.Profiles().Count() != 2 {
		t.Fatalf("Sketch1 offers %d regions, want 2 (the disk and the annulus)", sk1.Profiles().Count())
	}

	l := NewLoftTool()
	s.StartTool(l)
	l.Pick(s, ProfileHandle{Sketch: sk1, ProfileIndex: diskProfileIndex(t, sk1)})
	l.Pick(s, ProfileHandle{Sketch: sk2, ProfileIndex: 0})
	if _, ok := l.DraftFeature(s); !ok {
		t.Error("the loft showed no preview (the #1526 symptom)")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("commit loft: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("loft produced %d bodies, want 1 (#1526)", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("lofted body is not a valid solid: %+v", r)
	}
}
