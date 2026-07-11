// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
)

// TestChamferShiftClickSelectsTangentLoop is the #1798/#1947 acceptance at the UI seam: after
// rounding a block's four vertical edges, one click on a straight edge of the resulting tangent top
// rim selects the whole 8-edge loop (4 sides + 4 corner arcs). With the "Tangent chain" toggle at
// its default (on, Inventor's tangent propagation) a PLAIN click expands; toggling it off makes a
// plain click select just the clicked edge; and Shift+click always expands regardless.
func TestChamferShiftClickSelectsTangentLoop(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	roundVerticalEdges(t, s, block, 0.5)
	rounded := activePartDef(t, s).SurfaceBodies().Item(0)
	seed := longestTopRimEdgeHandle(t, rounded, 2.0)

	def := NewChamferTool()
	s.StartTool(def) // seeds tangentChain from the session default (on)
	def.PickWithMods(s, seed, 0)
	if got := len(def.Edges()); got != 8 {
		t.Fatalf("plain click with the Tangent-chain default (on) selected %d edges, want the 8-edge loop", got)
	}

	off := NewChamferTool()
	s.StartTool(off)
	off.SetTangentChain(false)
	off.PickWithMods(s, seed, 0) // toggle off ⇒ plain click adds only the seed
	if got := len(off.Edges()); got != 1 {
		t.Fatalf("plain click with Tangent chain off selected %d edges, want 1", got)
	}

	shift := NewChamferTool()
	s.StartTool(shift)
	shift.SetTangentChain(false)
	shift.PickWithMods(s, seed, ShiftMod) // Shift always expands, even with the toggle off
	if got := len(shift.Edges()); got != 8 {
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
