// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// filletVerticalsOf rounds the four vertical edges of the session's active block via the Fillet
// TOOL (the real interactive path), leaving a body whose top rim is a tangent loop of 4 straight
// sides + 4 corner arcs — the state rampam is in before trying to fillet "all around".
func filletVerticalsOf(t *testing.T, s *Session, block *topo.Body) *topo.Body {
	t.Helper()
	f := NewFilletTool()
	s.StartTool(f)
	for _, e := range block.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == c.X && a.Y == c.Y {
			f.Pick(s, EdgeHandle{Edge: e})
		}
	}
	f.SetRadius(0.5)
	if err := s.OK(); err != nil {
		t.Fatalf("vertical fillet: %v", err)
	}
	return activePartDef(t, s).SurfaceBodies().Item(0)
}

// oneTopRimEdge returns one edge handle lying on the body's top plane (z≈max).
func oneTopRimEdge(t *testing.T, b *topo.Body) EdgeHandle {
	t.Helper()
	maxZ := 0.0
	for _, v := range b.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	for _, e := range b.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			return EdgeHandle{Edge: e}
		}
	}
	t.Fatal("no top-rim edge")
	return EdgeHandle{}
}

func toriOf(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
}

// TestFilletAllAroundShiftClickExpandsTangentLoop is the intended #1798 gesture, end-to-end through
// the TOOL: after rounding the verticals, ONE Shift+click on a top-rim edge must expand to the whole
// 8-edge tangent loop and fillet all around, yielding a valid solid with 4 torus corners.
func TestFilletAllAroundShiftClickExpandsTangentLoop(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	rounded := filletVerticalsOf(t, s, block)

	f := NewFilletTool()
	s.StartTool(f)
	f.PickWithMods(s, oneTopRimEdge(t, rounded), ShiftMod)
	if n := len(f.Edges()); n != 8 {
		t.Fatalf("Shift+click on the rounded rim selected %d edges, want 8 (the whole tangent loop)", n)
	}
	f.SetRadius(0.25)
	if err := s.OK(); err != nil {
		t.Fatalf("all-around fillet OK: %v", err)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("all-around fillet body invalid: %+v", r)
	}
	if tori := toriOf(body); tori != 4 {
		t.Errorf("all-around fillet tori = %d, want 4", tori)
	}
}

// TestFilletAllAroundPlainClickIsSingleEdge documents rampam's likely experience (#1 "unable to
// fillet all around, it does not detect tangency"): a PLAIN click selects only the one clicked edge,
// so a user who does not know the (undiscoverable) Shift+click gets a single-edge fillet, not an
// all-around one. This is the interaction gap — the kernel/feature layers are correct (they round
// all 8 when handed them), but nothing surfaces the tangent loop on a normal click.
func TestFilletAllAroundPlainClickIsSingleEdge(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	rounded := filletVerticalsOf(t, s, block)

	f := NewFilletTool()
	s.StartTool(f)
	f.Pick(s, oneTopRimEdge(t, rounded)) // plain click — no modifier
	if n := len(f.Edges()); n != 1 {
		t.Fatalf("plain click selected %d edges, want 1 (no tangent-chain on a normal click)", n)
	}
}
