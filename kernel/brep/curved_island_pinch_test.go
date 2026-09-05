// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane tangent to a torus's inner equator cuts a FIGURE-EIGHT: two spiric lobes that both begin and
// end at the pinch, exactly — (0,3,0) on both arcs, to the last bit. The meeting solve must not move
// that point. It did: converging on a parameter a millionth of a span from each arc's end, it evaluated
// the ill-conditioned u(v) = Φ ± arccos w there and put the two ends 1.07e-07 apart, past the
// arrangement's vertex weld. The two lobes then reached the pinch at two vertices joined by a step,
// which cancelled as a shared edge and merged them into one self-touching loop — a single lid where the
// cut has two (ADR-0061).
func TestIslandTouchKeepsTheExactPinchOfAFigureEight(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 3, 0), math.V3(0, 1, 0)) // offset = R−r: tangent to the inner equator
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	arcs, ok := geom.TorusPlaneSection(tor, plane)
	if !ok || len(arcs) != 2 {
		t.Fatalf("section: ok=%v, %d arcs, want 2 lobes", ok, len(arcs))
	}
	islands := make([]imprintCycle, 0, 2)
	for _, a := range arcs {
		lo, hi := a.Domain()
		islands = append(islands, imprintCycle{{curve: a, t0: lo, t1: hi}})
	}
	pinch := arcs[0].PointAt(0) // the arcs' own shared end: exact
	out, _ := splitIslandsAtTouches(islands, nil)
	if len(out) != 2 {
		t.Fatalf("solved islands = %d, want 2 (the split must not join the lobes)", len(out))
	}
	// tol:calibrated — a few ulps of the torus's own radius: the solve must return the exact end, not a
	// converged neighbour of it.
	const ulps = 8 * 2.220446049250313e-16 * 5
	for i, cyc := range out {
		for ai, arc := range cyc {
			for k, m := range []*math.Point3{arc.meet0, arc.meet1} {
				if m == nil {
					t.Errorf("island %d arc %d meet%d not solved: the lobes meet at both ends", i, ai, k)
					continue
				}
				if d := float64(m.DistanceTo(pinch)); d > ulps {
					t.Errorf("island %d arc %d meet%d is %.3g from the exact pinch %v, want ≤ %.3g", i, ai, k, d, pinch, ulps)
				}
			}
		}
	}
}

// bestTouchParams must never return a pair the arcs agree on LESS well than either candidate it chose
// between — that is the whole of its contract, and it is what lets one rule serve both figure-eight
// sections. The axis-parallel one ends at the pinch exactly and the ends win; the oblique one comes
// from a numeric root of |w| = 1 and the solve can win. Measured on the corpus, the certification is
// what took every torus figure-eight row from an exactly-WRONG body to the right volume (ADR-0061).
func TestBestTouchParamsIsNeverWorseThanEitherCandidate(t *testing.T) {
	t.Parallel()
	for _, row := range []struct {
		name   string
		axis   math.Vector3
		origin math.Point3
		normal math.Vector3
	}{
		{"axis-parallel pinch", math.V3(0, 0, 1), math.P3(0, 3, 0), math.V3(0, 1, 0)},
		{"oblique transition", math.V3(0, 0.6, 0.8), math.P3(0, 0, 1), math.V3(0, 0, 1)},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			a, b := twoLobeSectionArcs(t, row.axis, row.origin, row.normal)
			hits := geom.CurveTouches(a.curve, b.curve, false, islandTouchWeld(a.curve))
			if len(hits) == 0 {
				t.Fatal("the two lobes must be solved as meeting")
			}
			for _, hit := range hits {
				ta, tb := bestTouchParams(a, b, hit[0], hit[1])
				ea, _ := arcEndAt(a, hit[0])
				eb, _ := arcEndAt(b, hit[1])
				chosen := touchGap(a, b, ta, tb)
				if ends, solve := touchGap(a, b, ea, eb), touchGap(a, b, hit[0], hit[1]); chosen > ends || chosen > solve {
					t.Errorf("chosen gap %.3g beats neither candidate (ends %.3g, solve %.3g)", chosen, ends, solve)
				}
			}
		})
	}
}

// twoLobeSectionArcs is the two-lobe spiric section of a torus cut by the given plane, as imprint arcs.
func twoLobeSectionArcs(t *testing.T, axis math.Vector3, origin math.Point3, normal math.Vector3) (imprintArc, imprintArc) {
	t.Helper()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), axis, 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	plane, err := geom.NewPlane(origin, normal)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	arcs, ok := geom.TorusPlaneSection(tor, plane)
	if !ok || len(arcs) != 2 {
		t.Fatalf("section: ok=%v, %d arcs, want 2", ok, len(arcs))
	}
	lo0, hi0 := arcs[0].Domain()
	lo1, hi1 := arcs[1].Domain()
	return imprintArc{curve: arcs[0], t0: lo0, t1: hi0}, imprintArc{curve: arcs[1], t0: lo1, t1: hi1}
}
