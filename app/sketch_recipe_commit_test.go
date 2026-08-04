// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
)

// TestToolCommitsAreRigid is the end-to-end gate for #2014: geometry created through the
// interactive tools must come out constrained, not as a floppy loop of free points. The DOF
// figures are the shapes' intrinsic parameter counts; redundancy must stay zero, because a
// duplicated constraint reports a healthy DOF while the solver settles on a degenerate
// configuration.
func TestToolCommitsAreRigid(t *testing.T) {
	cases := []struct {
		name   string
		dof    int
		commit func(s *Session) error
	}{
		{"rectangle", 4, func(s *Session) error {
			tool := NewRectangleTool()
			tool.corners = []math.Point2{math.P2(0, 0), math.P2(10, 8)}
			return tool.Commit(s)
		}},
		{"centre rectangle", 4, func(s *Session) error {
			tool := NewCenterRectangleTool()
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(5, 4)}
			return tool.Commit(s)
		}},
		{"three-point rectangle", 5, func(s *Session) error {
			tool := NewThreePointRectangleTool()
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 8)}
			return tool.Commit(s)
		}},
		{"polygon", 4, func(s *Session) error {
			tool := NewPolygonTool(6)
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(5, 0)}
			return tool.Commit(s)
		}},
		{"slot", 5, func(s *Session) error {
			tool := NewSketchSlotTool(2)
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(10, 0)}
			return tool.Commit(s)
		}},
		{"centre-point arc slot", 6, func(s *Session) error {
			tool := NewCenterPointArcSlotTool(2)
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(0, 10)}
			return tool.Commit(s)
		}},
		{"three-point arc slot", 6, func(s *Session) error {
			tool := NewThreePointArcSlotTool(2)
			tool.pts = []math.Point2{math.P2(10, 0), math.P2(7.07, 7.07), math.P2(0, 10)}
			return tool.Commit(s)
		}},
		{"circle", 3, func(s *Session) error {
			tool := NewCircleTool()
			tool.pts = []math.Point2{math.P2(0, 0), math.P2(5, 0)}
			return tool.Commit(s)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, sk := sketchSession(t)
			if err := c.commit(s); err != nil {
				t.Fatalf("Commit: %v", err)
			}
			a := sk.AnalyzeConstraints()
			if a.DOF != c.dof {
				t.Errorf("DOF = %d, want %d (vars=%d eqs=%d rank=%d)", a.DOF, c.dof, a.Variables, a.Equations, a.Rank)
			}
			if a.Redundant != 0 {
				t.Errorf("Redundant = %d, want 0", a.Redundant)
			}
		})
	}
}
