// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/math"

	"oblikovati.org/kernel/topo"
)

// firstSharedEdge returns the body's first edge shared by exactly two faces (every box edge is).
func firstSharedEdge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if len(e.Faces()) == 2 {
			return e
		}
	}
	t.Fatal("no two-face edge on the body")
	return nil
}

func TestContinuityCheckReportsBoxEdgeAsRightAngleCrease(t *testing.T) {
	s, body := newPartWithBlock(t, 6)
	edge := firstSharedEdge(t, body)

	tool := NewContinuityCheckTool()
	s.StartTool(tool)
	tool.Pick(s, EdgeHandle{Edge: edge})

	rep := tool.Report()
	if rep == nil {
		t.Fatal("picking a shared edge should produce a continuity report")
	}
	// Two perpendicular planar faces: a 90° tangent break, no gap, no curvature difference.
	if stdmath.Abs(rep.MaxNormalDeg-90) > 0.5 {
		t.Errorf("box-edge G1 angle = %g°, want ~90°", rep.MaxNormalDeg)
	}
	if rep.MaxGap > 1e-6 || rep.MaxCurvDiff > 1e-6 {
		t.Errorf("box edge should report no G0 gap or G2 curvature difference: gap=%g curv=%g", rep.MaxGap, rep.MaxCurvDiff)
	}
	if len(tool.Preview(s)) == 0 {
		t.Error("a measured edge should draw the coloured continuity overlay")
	}
	if !continuityScrollbackHas(s, "Continuity") {
		t.Error("the Command Window should report the continuity result")
	}
}

func TestContinuityCheckIgnoresNonEdgePick(t *testing.T) {
	s, _ := newPartWithBlock(t, 4)
	tool := NewContinuityCheckTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{}) // not an edge
	if tool.Report() != nil {
		t.Error("a non-edge pick should not produce a report")
	}
}

func TestContinuityCheckViaRibbonCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 4)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Inspect.Continuity"); err != nil {
		t.Fatalf("execute Inspect.Continuity: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Continuity Check" {
		t.Errorf("Inspect.Continuity started tool %q, want Continuity Check", got)
	}
}

func TestContinuityCheckToolInterface(t *testing.T) {
	tool := NewContinuityCheckTool()
	tool.Start(nil)
	tool.Cancel(nil)
	if err := tool.Commit(nil); err != nil {
		t.Errorf("Commit is a no-op, got %v", err)
	}
	if tool.CanCommit() {
		t.Error("CanCommit should be false (report-on-pick)")
	}
	if len(tool.AcceptedKinds()) != 1 || tool.AcceptedKinds()[0] != SelectEdge {
		t.Error("the tool should accept edge picks")
	}
	if len(tool.Picks()) != 0 || len(tool.Edges()) != 0 || tool.Preview(nil) != nil {
		t.Error("before a pick the tool has no edge, picks, or overlay")
	}
}

func TestContinuitySeverityHelpers(t *testing.T) {
	cases := []struct {
		in, want float64
	}{{-1, 0}, {0.4, 0.4}, {2, 1}}
	for _, c := range cases {
		if got := math.Clamp01(c.in); got != c.want {
			t.Errorf("math.Clamp01(%g) = %g, want %g", c.in, got, c.want)
		}
	}
	// A mid-severity sample produces a colour between green and red.
	col := severityColor(0.5)
	if col[0] <= 0.1 || col[1] <= 0 {
		t.Errorf("mid severity colour = %v, want a green/red blend", col)
	}
}

func continuityScrollbackHas(s *Session, substr string) bool {
	for _, l := range s.CommandLine().Scrollback().Lines() {
		if strings.Contains(l.Text, substr) {
			return true
		}
	}
	return false
}
