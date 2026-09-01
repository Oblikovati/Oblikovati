// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	"hash/fnv"
	"sort"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestTangentBooleanChainedRecomputeIsDeterministic pins audit A4's determinism guarantee (#1600):
// a boolean across a tangent contact — which resolves through the displaced-geometry nudge — feeding
// a dependent second boolean must produce a BIT-IDENTICAL result on every recompute. The nudge's
// direction (summed operand-B normals) and magnitude (a fixed multiple of the weld grid) are fully
// deterministic, so the chained result must be too: a downstream feature can never flip between the
// coplanar and imprint code paths from run to run. Ten runs, one canonical geometry hash.
func TestTangentBooleanChainedRecomputeIsDeterministic(t *testing.T) {
	t.Parallel()
	var want string
	for i := range 10 {
		a := guardBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2), "a")
		b := guardBlock(t, math.P3(2, 2, 0), math.P3(4, 4, 2), "b") // shares only the vertical edge x=2,y=2
		tangent, err := Boolean(Join, a, b)
		if err != nil || tangent == nil {
			t.Fatalf("run %d: tangent union: %v", i, err)
		}
		// A dependent feature on the tangent result: cut a straddling block off the far corner, so the
		// chain exercises coplanar detection on faces the tangent boolean produced.
		tool := guardBlock(t, math.P3(3, 3, 0), math.P3(5, 5, 2), "tool")
		chained, err := Boolean(Cut, tangent, tool)
		if err != nil || chained == nil {
			t.Fatalf("run %d: dependent cut: %v", i, err)
		}
		got := geometryHash(chained)
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Fatalf("run %d geometry hash %s != run 0 %s — the tangent-contact chained recompute is nondeterministic", i, got, want)
		}
	}
}

// geometryHash is a canonical, order-independent fingerprint of a body's geometry: its vertex
// coordinates quantized to the weld grid and sorted, folded into an FNV hash. Two runs that produce
// the same solid at the same coordinates hash identically regardless of build/traversal order.
func geometryHash(b *topo.Body) string {
	keys := make([]string, 0, len(b.Vertices()))
	for _, v := range b.Vertices() {
		p := v.Point()
		keys = append(keys, fmt.Sprintf("%.6f,%.6f,%.6f", p.X, p.Y, p.Z))
	}
	sort.Strings(keys)
	h := fnv.New64a()
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%016x/%dv", h.Sum64(), len(keys))
}
