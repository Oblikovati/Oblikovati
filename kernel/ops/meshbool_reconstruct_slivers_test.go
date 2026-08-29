// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"
	"testing"

	"oblikovati.org/kernel/meshbool"
)

// A near-tangent cap patch (#2247): N owns the long edge (p→q); the cap (p,m,q) bridges to
// the split side Amp,Amq whose shared vertex m sits a sub-tolerance 1e-12 off segment p-q.
// The four triangles form a locally watertight fan around the cap — every interior directed
// edge is paired, and a fixed set of outer edges is unpaired (the patch boundary).
func slivPt(x, y, z float64) meshbool.Point { return meshbool.FromCoords(x, y, z) }

func capPatch() meshbool.TaggedSoup {
	p := slivPt(0, 0, 0)
	q := slivPt(4, 0, 0)
	x := slivPt(2, 3, 0)
	y := slivPt(2, -3, 0)
	m := slivPt(2, 1e-12, 0) // interior to p-q, 1e-12 off the line — a sub-weld cap
	return meshbool.TaggedSoup{
		Tris: [][3]meshbool.Point{
			{p, q, x}, // N: owns p→q
			{p, m, q}, // cap: owns q→p, p→m, m→q
			{m, p, y}, // Amp: owns m→p
			{q, m, y}, // Amq: owns q→m
		},
		Tags: []int{0, 1, 2, 3},
	}
}

// sliverBoundaryFingerprint returns the multiset of directed edges with no reverse partner (the open
// boundary), as sorted "a->b" vertex-key strings — the watertightness fingerprint. A
// watertight-preserving edit leaves this multiset unchanged.
func sliverBoundaryFingerprint(soup meshbool.TaggedSoup) []string {
	key := func(pt meshbool.Point) string {
		return pt.X.RatString() + "," + pt.Y.RatString() + "," + pt.Z.RatString()
	}
	count := map[[2]string]int{}
	for _, t := range soup.Tris {
		for k := 0; k < 3; k++ {
			count[[2]string{key(t[k]), key(t[(k+1)%3])}]++
		}
	}
	var out []string
	for e, c := range count {
		for i := 0; i < c-count[[2]string{e[1], e[0]}]; i++ {
			out = append(out, e[0]+"->"+e[1])
		}
	}
	sort.Strings(out)
	return out
}

// TestCollapseSliversRemovesCapWatertight pins the fix: collapseSlivers drops the near-tangent
// cap and re-stitches its long-edge neighbour, so no sub-resolution sliver survives AND the
// patch boundary (its watertightness fingerprint) is exactly preserved.
func TestCollapseSliversRemovesCapWatertight(t *testing.T) {
	in := capPatch()
	before := sliverBoundaryFingerprint(in)

	out := collapseSlivers(in, 1e-9)

	// The cap (near-collinear, area² ~ (4·1e-12)² ≈ 1.6e-23) must be gone: every surviving
	// triangle has a real area. Threshold well above the cap, well below the real triangles.
	for _, tr := range out.Tris {
		if a2 := slivTriArea2(tr); a2 < 1e-18 {
			t.Errorf("a sub-resolution sliver survived: area²=%g verts=%v", a2, tr)
		}
	}
	if got := len(out.Tris); got != 4 {
		t.Errorf("cap patch collapsed to %d triangles, want 4 (drop cap, split neighbour into 2)", got)
	}
	if after := sliverBoundaryFingerprint(out); !equalStrs(before, after) {
		t.Errorf("boundary changed (watertightness broken):\n before=%v\n after =%v", before, after)
	}
}

// TestCollapseSliversLeavesCleanMeshUnchanged: a mesh with no cap passes through untouched.
func TestCollapseSliversLeavesCleanMeshUnchanged(t *testing.T) {
	p := slivPt(0, 0, 0)
	q := slivPt(4, 0, 0)
	x := slivPt(2, 3, 0)
	y := slivPt(2, -3, 0)
	clean := meshbool.TaggedSoup{
		Tris: [][3]meshbool.Point{{p, q, x}, {q, p, y}},
		Tags: []int{0, 1},
	}
	out := collapseSlivers(clean, 1e-9)
	if len(out.Tris) != 2 {
		t.Fatalf("clean mesh changed: %d triangles, want 2", len(out.Tris))
	}
}

func slivTriArea2(t [3]meshbool.Point) float64 {
	a, b, c := t[0].Round(), t[1].Round(), t[2].Round()
	ux, uy, uz := b.X-a.X, b.Y-a.Y, b.Z-a.Z
	vx, vy, vz := c.X-a.X, c.Y-a.Y, c.Z-a.Z
	cx, cy, cz := uy*vz-uz*vy, uz*vx-ux*vz, ux*vy-uy*vx
	return cx*cx + cy*cy + cz*cz
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
