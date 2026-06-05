// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// nearXY asserts a point's X within tolerance (the trim tools move endpoints along X here).
func nearXY(t *testing.T, got, want float64) {
	t.Helper()
	if stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("X = %v, want %v", got, want)
	}
}

// TestSketchTrimToolRemovesPickedSegment drives the Trim tool end-to-end: pick the middle
// of a line between two crossings → the picked segment is removed.
func TestSketchTrimToolRemovesPickedSegment(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(6, 0))
	sk.Lines().AddByTwoPoints(math.P2(2, -1), math.P2(2, 1)) // crossing at x=2
	sk.Lines().AddByTwoPoints(math.P2(4, -1), math.P2(4, 1)) // crossing at x=4

	tool := NewSketchTrimTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{Kind: SnapOnCurve, Point: math.P2(3, 0)}) // pick the middle
	s.autoCommitAfterPick()

	if s.ActiveTool() != nil {
		t.Fatal("trim tool should deactivate after the pick")
	}
	nearXY(t, float64(l.B.Position().X), 2) // reshaped original keeps [0,2]
}

// TestSketchTrimToolCutsAtCircleCrossing proves the tool benefits from the curve-aware
// crossing engine: a line trimmed where a circle crosses it.
func TestSketchTrimToolCutsAtCircleCrossing(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(-3, 0), math.P2(3, 0))
	sk.Circles().AddByCenterRadius(math.P2(0, 0), 1) // crosses at (±1,0)

	tool := NewSketchTrimTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{Kind: SnapOnCurve, Point: math.P2(0, 0)}) // pick inside the circle
	s.autoCommitAfterPick()

	if s.ActiveTool() != nil {
		t.Fatal("trim tool should deactivate after the pick")
	}
	nearXY(t, float64(l.B.Position().X), -1) // kept stub is [-3,-1]
}

// TestSketchExtendToolLengthensNearerEnd: picking near the B end extends it to the next
// crossing of the line's support.
func TestSketchExtendToolLengthensNearerEnd(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	sk.Lines().AddByTwoPoints(math.P2(5, -1), math.P2(5, 1)) // support crossing at x=5

	tool := NewSketchExtendTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{Kind: SnapOnCurve, Point: math.P2(2, 0)}) // near the B end
	s.autoCommitAfterPick()

	if s.ActiveTool() != nil {
		t.Fatal("extend tool should deactivate after the pick")
	}
	nearXY(t, float64(l.B.Position().X), 5)
}

// TestSketchSplitToolSplitsAtPoint: the picked point splits the line into two.
func TestSketchSplitToolSplitsAtPoint(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	before := sk.Lines().Count()

	tool := NewSketchSplitTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{Kind: SnapOnCurve, Point: math.P2(2, 0)})
	s.autoCommitAfterPick()

	if s.ActiveTool() != nil {
		t.Fatal("split tool should deactivate after the pick")
	}
	if sk.Lines().Count() != before+1 {
		t.Fatalf("lines after split = %d, want %d", sk.Lines().Count(), before+1)
	}
	nearXY(t, float64(l.B.Position().X), 2) // first half ends at the split point
}

// TestSketchTrimToolTrimsCircleTarget: trimming a circle that two lines cross replaces it
// with the complementary arc.
func TestSketchTrimToolTrimsCircleTarget(t *testing.T) {
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	sk.Lines().AddByTwoPoints(math.P2(0, -3), math.P2(0, 3)) // crosses at (0,±2)

	tool := NewSketchTrimTool()
	s.StartTool(tool)
	tool.PickSnap(c, SnapResult{Kind: SnapOnCurve, Point: math.P2(2, 0)}) // pick the right half
	s.autoCommitAfterPick()

	if s.ActiveTool() != nil {
		t.Fatal("trim tool should deactivate after the pick")
	}
	if sk.Circles().Count() != 0 || sk.Arcs().Count() != 1 {
		t.Fatalf("after circle trim: circles=%d arcs=%d, want 0 and 1", sk.Circles().Count(), sk.Arcs().Count())
	}
}

// TestSketchTrimToolRejectsNonCurve: a point pick keeps the tool open with an error.
func TestSketchTrimToolRejectsNonCurve(t *testing.T) {
	s, sk := sketchSession(t)
	p := sk.Points().Add(math.P2(0, 0))

	tool := NewSketchTrimTool()
	s.StartTool(tool)
	tool.PickSnap(p, SnapResult{Kind: SnapPoint, Point: math.P2(0, 0)})
	if err := tool.Commit(s); err == nil {
		t.Error("trimming a point target should error")
	}
}

func TestSketchTrimExtendSplitCommandsRegistered(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{"Sketch.Trim", "Sketch.Extend", "Sketch.Split"} {
		if _, ok := s.Commands().ByID(id); !ok {
			t.Errorf("command %q not registered", id)
		}
	}
}
