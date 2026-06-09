// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"os"
	"sort"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// discThenCutPatterned models the wheel core: a Ø60 disc, one Ø6 bolt-hole cut, then a 5-up
// circular pattern of that cut. Under OBK_ANALYTIC_CURVES the disc and bolt are analytic
// cylinders; the pattern boolean must re-facet the curved tool (#129) or it explodes (the raw
// periodic-cylinder face used to blow the body up to tens of thousands of edges and hang).
func discThenCutPatterned(t *testing.T) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	disc := sketch.NewSketches().Add(sketch.XYPlane())
	disc.Circles().AddByCenterRadius(math.P2(0, 0), 30)
	NewExtrudeFeatures(fs).AddByDistanceExtent(disc, 0, ops.NewBody, func() float64 { return 10 })

	hole := sketch.NewSketches().Add(sketch.XYPlane())
	hole.Circles().AddByCenterRadius(math.P2(20, 0), 3)
	cut := NewExtrudeFeatures(fs).AddExtrude(hole, []int{0}, ops.Cut, Extent{Type: DistanceExtent, Distance: func() float64 { return 10 }}, 0)

	NewPatternFeatures(fs).AddCircular([]ID{cut.ID()}, func() int { return 5 }, func() float64 { return 2 * 3.141592653589793 },
		math.P3(0, 0, 0), math.V3(0, 0, 1))

	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("disc+cut = %d bodies, want 1", len(bodies))
	}
	return bodies[0]
}

func sortedEdgeKeys(b *topo.Body) []string {
	es := b.Edges()
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = string(e.ReferenceKey())
	}
	sort.Strings(out)
	return out
}

// TestAnalyticPatternedCutMatchesFaceted is the decisive #129 check: the patterned-cut body must
// have the SAME sorted edge reference keys with OBK_ANALYTIC_CURVES on as off, so topology()-style
// "smallest key" selection picks the same physical edge (the invariant the wheel/box-bosses suites
// depend on) AND so the pattern boolean does not explode on a curved tool. The equal-count check
// alone is the regression for the tens-of-thousands-of-edges blow-up.
func TestAnalyticPatternedCutMatchesFaceted(t *testing.T) {
	os.Unsetenv("OBK_ANALYTIC_CURVES")
	off := sortedEdgeKeys(discThenCutPatterned(t))

	os.Setenv("OBK_ANALYTIC_CURVES", "1")
	defer os.Unsetenv("OBK_ANALYTIC_CURVES")
	on := sortedEdgeKeys(discThenCutPatterned(t))

	if len(off) != len(on) {
		t.Fatalf("edge count: off=%d on=%d", len(off), len(on))
	}
	for i := range off {
		if off[i] != on[i] {
			t.Fatalf("edge key %d differs:\n off=%q\n  on=%q", i, off[i], on[i])
		}
	}
}
