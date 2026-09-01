// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// tangentTwoBoxUnion builds the exact edge-tangent union of two 2×2×2 boxes that meet ONLY along
// the vertical line x=2,y=2 (zero volumetric overlap): the shared line is bordered by four faces
// (two per box) and the boolean splits it into two coincident manifold edges with EXACT coordinates
// (#1600). The two coincident edges share their endpoint vertices, so only carried edge identity —
// not coordinate — keeps them apart through a downstream re-weld.
func tangentTwoBoxUnion(t *testing.T) *topo.Body {
	t.Helper()
	a, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "a")
	if err != nil {
		t.Fatalf("SolidBlock a: %v", err)
	}
	b, err := brep.SolidBlock(math.P3(2, 2, 0), math.P3(4, 4, 2), "b")
	if err != nil {
		t.Fatalf("SolidBlock b: %v", err)
	}
	body, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatalf("tangent union: %v", err)
	}
	return body
}

// TestFilletOnExactTangentStaysManifold is #1600's empirical blocker: filleting an edge of the body
// that carries an exact edge-tangent seam (two coincident edges sharing endpoints) must yield a
// VALID 2-manifold solid. Before the identity-preserving re-weld, assembleBody keyed edges by
// welded vertex-pair alone, so the two coincident seam edges collapsed into one edge used by FOUR
// faces ("non-manifold edge used by 4 faces") and the fillet failed. The carried source-edge id
// (method C) keeps them distinct, so the tangency survives the re-weld.
func TestFilletOnExactTangentStaysManifold(t *testing.T) {
	t.Parallel()
	body := tangentTwoBoxUnion(t)
	top := highestEdge(t, body)
	out, err := blend.FilletEdges(body, [][]byte{top.ReferenceKey()}, 0.1)
	if err != nil {
		t.Fatalf("fillet on exact tangent seam failed (coincident edges collapsed?): %v", err)
	}
	r := validate.Validate(out)
	if !r.Valid {
		t.Fatalf("filleted tangent body is not a valid manifold: %v", r.Issues)
	}
	if !r.Manifold {
		t.Errorf("filleted tangent body is non-manifold: %v", r.Issues)
	}
	if !r.EulerConsistent {
		t.Errorf("filleted tangent body has an inadmissible Euler characteristic (χ=%d): %v", r.EulerCharacteristic, r.Issues)
	}
}

// highestEdge returns the body's topmost edge (by midpoint z) — the rim far from the tangent seam,
// so the fillet exercises the re-weld of the seam without touching it directly.
func highestEdge(t *testing.T, b *topo.Body) *topo.Edge {
	t.Helper()
	var top *topo.Edge
	bz := math.Scalar(-1e18)
	for _, e := range b.Edges() {
		vs := e.Vertices()
		if z := (vs[0].Point().Z + vs[len(vs)-1].Point().Z) / 2; z > bz {
			bz, top = z, e
		}
	}
	if top == nil {
		t.Fatal("no edges on body")
	}
	return top
}

// TestTangentContactChainedRecomputeDeterministic pins #1600's determinism criterion: a tangent
// union followed by a dependent coplanar feature (a fillet on the seam-bearing body) must produce a
// bit-identical result on every recompute. A residual 1e-5 nudge made downstream coplanar/imprint
// classification flip nondeterministically between runs; the exact, undisplaced result removes that.
func TestTangentContactChainedRecomputeDeterministic(t *testing.T) {
	t.Parallel()
	var want string
	for i := range 10 {
		body := tangentTwoBoxUnion(t)
		out, err := blend.FilletEdges(body, [][]byte{highestEdge(t, body).ReferenceKey()}, 0.1)
		if err != nil {
			t.Fatalf("run %d: fillet failed: %v", i, err)
		}
		got := bodyGeometryHash(out)
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Fatalf("run %d produced a different geometry hash than run 0 (nondeterministic tangent recompute): %s != %s", i, got, want)
		}
	}
}

// bodyGeometryHash summarises a body's vertex coordinates in a canonical (sorted) order, so two
// runs with identical geometry hash identically regardless of build-order vertex numbering.
func bodyGeometryHash(b *topo.Body) string {
	pts := make([]string, 0, len(b.Vertices()))
	for _, v := range b.Vertices() {
		p := v.Point()
		pts = append(pts, fmt.Sprintf("%0.17g,%0.17g,%0.17g", float64(p.X), float64(p.Y), float64(p.Z)))
	}
	sort.Strings(pts)
	return strings.Join(pts, ";")
}
