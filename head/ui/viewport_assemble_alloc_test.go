//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestAssembleAllocFree is the #1423 guard: assembling the per-frame instance matrices + draw records
// for a STATIC scene (unchanged geometry + culled set, only the camera orbiting) must allocate
// nothing — the buffers are reused across frames. Before #1423 this allocated a 16-float slice per
// instance plus three fresh slices per frame; a 10k-instance assembly did ~10k heap slices/frame.
func TestAssembleAllocFree(t *testing.T) {
	const sources, perSource = 4, 2500 // 10k instances total
	atlas := frameAtlas{}
	visible := map[*topo.Body][]math.Matrix4{}
	tfs := make([]math.Matrix4, perSource)
	for i := range tfs {
		tfs[i] = math.Translation4(math.V3(float64(i), 0, 0))
	}
	for s := range sources {
		b := new(topo.Body)
		atlas.recs = append(atlas.recs, [5]int32{1, 0, 36, 0, 0}) // one tri-stream template per source
		atlas.regions = append(atlas.regions, atlasRegion{source: b, start: s, end: s + 1})
		visible[b] = tfs // every source visible with perSource instances
	}

	mats, recs := atlas.assemble(visible) // warm the scratch buffers
	if got := len(mats) / 16; got != sources*perSource {
		t.Fatalf("assemble produced %d instances, want %d", got, sources*perSource)
	}
	if len(recs) != sources*7 {
		t.Fatalf("assemble produced %d record ints, want %d", len(recs), sources*7)
	}

	allocs := testing.AllocsPerRun(50, func() { _, _ = atlas.assemble(visible) })
	if allocs > 0 {
		t.Errorf("assemble allocates %.1f times/op on a static frame, want 0 (#1423)", allocs)
	}
}

// TestSelectionSeqStableThenBumps pins the reflection-free selection key: the sequence holds while the
// first selected item is unchanged (so an orbit doesn't invalidate the highlighted-source cache) and
// bumps when it changes — replacing the per-frame fmt.Sprintf("%v", …) reflection (#1423).
func TestSelectionSeqStableThenBumps(t *testing.T) {
	lastSelFirst, lastSelSeq = nil, 0 // reset package state for a deterministic check
	s := assemblyWithPlacedBox(t)
	a := selectionSeq(s)
	if b := selectionSeq(s); b != a {
		t.Errorf("selectionSeq changed on an unchanged selection: %d→%d (orbit would re-tessellate)", a, b)
	}
	// Select a body → First() changes identity → the sequence must advance.
	groups := s.VisibleInstances()
	if len(groups) == 0 {
		t.Skip("no instance to select")
	}
	s.Selection().Add(app.BodyHandle{Body: groups[0].Source})
	if c := selectionSeq(s); c == a {
		t.Errorf("selectionSeq did not advance after a selection change (stale highlight key)")
	}
}
