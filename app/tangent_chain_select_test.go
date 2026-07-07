// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
)

// TestChamferShiftClickSelectsTangentLoop is the #1798 acceptance at the UI seam: after rounding
// a block's four vertical edges, Shift-clicking one straight edge of the resulting tangent top
// rim selects the whole 8-edge loop in one gesture (4 sides + 4 corner arcs) — the "pick one
// edge → select tangent chain" the Chamfer/Fillet tools were missing. A plain click still adds
// only the clicked edge.
func TestChamferShiftClickSelectsTangentLoop(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	roundVerticalEdges(t, s, block, 0.5)
	rounded := activePartDef(t, s).SurfaceBodies().Item(0)
	seed := longestTopRimEdgeHandle(t, rounded, 2.0)

	ch := NewChamferTool()
	s.StartTool(ch)
	ch.PickWithMods(s, seed, 0) // plain click adds only the seed
	if got := len(ch.Edges()); got != 1 {
		t.Fatalf("plain click selected %d edges, want 1", got)
	}
	ch2 := NewChamferTool()
	s.StartTool(ch2)
	ch2.PickWithMods(s, seed, ShiftMod) // Shift-click expands to the tangent loop
	if got := len(ch2.Edges()); got != 8 {
		t.Fatalf("Shift-click selected %d edges, want the 8-edge tangent loop", got)
	}
}

// roundVerticalEdges fillets all four vertical edges of the block through the Fillet tool and
// commits, leaving the active part's body with a tangent-continuous rounded rim.
func roundVerticalEdges(t *testing.T, s *Session, block *topo.Body, r float64) {
	t.Helper()
	f := NewFilletTool()
	s.StartTool(f)
	for _, e := range block.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			f.Pick(s, EdgeHandle{Edge: e})
		}
	}
	f.SetRadius(r)
	if err := s.OK(); err != nil {
		t.Fatalf("fillet the four vertical edges: %v", err)
	}
}

// longestTopRimEdgeHandle returns a handle to the longest edge whose endpoints both lie at
// z≈top — a straight side of the rounded rim.
func longestTopRimEdgeHandle(t *testing.T, b *topo.Body, top float64) EdgeHandle {
	t.Helper()
	var best *topo.Edge
	bestLen := 0.0
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(float64(a.Z)-top) > 1e-6 || stdmath.Abs(float64(c.Z)-top) > 1e-6 {
			continue
		}
		if l := float64(a.DistanceTo(c)); l > bestLen {
			best, bestLen = e, l
		}
	}
	if best == nil {
		t.Fatalf("no top-rim edge at z=%g", top)
	}
	return EdgeHandle{Edge: best}
}
