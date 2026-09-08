// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	"sort"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corpus test that a first-fit ladder could never pass (ADR-0061 stage 5, #3409). A ladder's
// entries are allowed to overlap — the earlier rung simply wins — so "two meshers both claim this
// face" is its mechanism, not a defect. A classification's predicates may not overlap, and this is
// the proof: every predicate is evaluated on every curved face of every corpus body, and two answers
// for one face fail the test. Reordering classifyCurvedTrim therefore changes no answer.

// classificationCorpus is the bodies whose curved faces exercise the classification: the trims each
// arm exists for, plus the controls that must reach none of them.
func classificationCorpus() []struct {
	name  string
	build func(t *testing.T) *topo.Body
} {
	return []struct {
		name  string
		build func(t *testing.T) *topo.Body
	}{
		{"bare ring", mustTorus},
		{"bare ball", func(t *testing.T) *topo.Body { return mustSphere(t, math.P3(0, 0, 0), 2) }},
		{"bare rod", func(t *testing.T) *topo.Body {
			return mustCylinder(t, math.P3(0, 0, -2), math.V3(0, 0, 1), 1, 4)
		}},
		{"ring − coaxial shaft", func(t *testing.T) *topo.Body {
			return ringMinus(t, mustCylinder(t, math.P3(0, 0, -4), math.V3(0, 0, 1), 4, 8))
		}},
		{"ring − axial drill", func(t *testing.T) *topo.Body {
			return ringMinus(t, mustCylinder(t, math.P3(5, 0, -4), math.V3(0, 0, 1), 0.8, 8))
		}},
		{"ring − half space", func(t *testing.T) *topo.Body {
			return ringMinus(t, mustBlock(t, math.P3(5, -20, -20), math.P3(20, 20, 20)))
		}},
		{"ring ∩ ball", func(t *testing.T) *topo.Body { return ringAnd(t, ops.Intersect) }},
		{"ring − ball", func(t *testing.T) *topo.Body { return ringAnd(t, ops.Cut) }},
		{"rod ∪ ball", func(t *testing.T) *topo.Body { return rodBall(t, ops.Join) }},
		{"rod − ball", func(t *testing.T) *topo.Body { return rodBall(t, ops.Cut) }},
		{"rod ∩ ball", func(t *testing.T) *topo.Body { return rodBall(t, ops.Intersect) }},
		{"ball ∩ half space (a cap)", func(t *testing.T) *topo.Body {
			return meetWith(t, mustSphere(t, math.P3(0, 0, 0), 2), mustBlock(t, math.P3(-9, -9, 0), math.P3(9, 9, 9)))
		}},
		{"ball − a slice off the top (a zone reaching a pole)", func(t *testing.T) *topo.Body {
			return cutWith(t, mustSphere(t, math.P3(0, 0, 0), 2), mustBlock(t, math.P3(-9, -9, 1), math.P3(9, 9, 9)))
		}},
		{"ball − axial bore (a belt)", func(t *testing.T) *topo.Body {
			return cutWith(t, mustSphere(t, math.P3(0, 0, 0), 2),
				mustCylinder(t, math.P3(0, 0, -9), math.V3(0, 0, 1), 0.7, 18))
		}},
		{"rod − crossing rod (a saddle band)", func(t *testing.T) *topo.Body {
			return cutWith(t, mustCylinder(t, math.P3(0, 0, -4), math.V3(0, 0, 1), 2, 8),
				mustCylinder(t, math.P3(-9, 0, 0), math.V3(1, 0, 0), 1, 18))
		}},
		{"plate with a conical drill point", func(t *testing.T) *topo.Body {
			return cutWith(t, mustBlock(t, math.P3(-5, -5, 0), math.P3(5, 5, 3.5)), mustConeDrill(t))
		}},
		{"drilled plate", func(t *testing.T) *topo.Body {
			return cutWith(t, mustBlock(t, math.P3(-5, -5, 0), math.P3(5, 5, 3.5)),
				mustCylinder(t, math.P3(0, 0, -1), math.V3(0, 0, 1), 1.5, 6))
		}},
	}
}

// TestCurvedTrimKindsAreMutuallyExclusive is the reorder-independence proof: no curved face in the
// corpus satisfies two classification predicates, and the kind classifyCurvedTrim selects is exactly
// the one predicate that holds (or the charted/uncharted residual when none does).
func TestCurvedTrimKindsAreMutuallyExclusive(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	for _, row := range classificationCorpus() {
		body := row.build(t)
		for i, f := range body.Faces() {
			if _, planar := f.Geometry().(geom.Plane); planar {
				continue // the classification only sees curved faces
			}
			hits := tessellate.CurvedTrimPredicateHits(f, q)
			if len(hits) > 1 {
				sort.Strings(hits)
				t.Errorf("%s face %d (%T): %d predicates claim it (%s) — a classification answers at "+
					"most once, so which mesher runs would depend on the order they are written in",
					row.name, i, f.Geometry(), len(hits), strings.Join(hits, ", "))
			}
			assertClassificationAgrees(t, row.name, i, f, hits, q)
		}
	}
}

// TestTheClassificationCorpusReachesEveryArm keeps the exclusivity proof from going vacuous: an
// overlap test over faces that never reach an arm proves nothing. Every arm below is measured on this
// corpus. kindWedgeBand is the one arm no primitive boolean produces — an oblique-ended cylinder wedge
// comes off the blend engine (A1/D4), and model/feature/occtparity carries its rows — so the list
// names the arms this package can build, and a new arm has to appear here or say why not.
func TestTheClassificationCorpusReachesEveryArm(t *testing.T) {
	t.Parallel()
	want := []string{"chart", "cone-apex-fan", "ruled-band-loft", "sphere-cap-fan", "sphere-patch",
		"sphere-zone-band", "spiric-band", "two-rim-holed-band", "uncharted"}
	seen := classifiedCurvedFaces(t)
	for _, kind := range want {
		if seen[kind] == 0 {
			t.Errorf("no corpus face classifies as %s — the exclusivity proof does not cover that arm", kind)
		}
	}
}

// classifiedCurvedFaces counts the curved faces of the whole corpus by the kind they classify as.
func classifiedCurvedFaces(t *testing.T) map[string]int {
	t.Helper()
	q := ops.DefaultQuality()
	seen := map[string]int{}
	for _, row := range classificationCorpus() {
		for _, f := range row.build(t).Faces() {
			if _, planar := f.Geometry().(geom.Plane); !planar {
				seen[tessellate.ClassifyCurvedTrimName(f, q)]++
			}
		}
	}
	return seen
}

// assertClassificationAgrees ties classifyCurvedTrim to the predicates: it must select the one kind
// that holds, and when none holds it must fall to the charted or uncharted residual.
func assertClassificationAgrees(t *testing.T, body string, i int, f *topo.Face, hits []string, q tessellate.Quality) {
	t.Helper()
	got := tessellate.ClassifyCurvedTrimName(f, q)
	if len(hits) == 1 {
		if got != hits[0] {
			t.Errorf("%s face %d (%T): predicate %s holds but the classification chose %s",
				body, i, f.Geometry(), hits[0], got)
		}
		return
	}
	if len(hits) == 0 && got != tessellate.ChartedTrimKindName() && got != tessellate.UnchartedTrimKindName() {
		t.Errorf("%s face %d (%T): no predicate holds but the classification chose %s, not the residual",
			body, i, f.Geometry(), got)
	}
}

// mustTorus builds the corpus ring, failing the test rather than returning an error.
func mustTorus(t *testing.T) *topo.Body {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	return ring
}

// mustConeDrill builds the conical drill whose apex-reaching cone face is the cone-apex arm's shape.
func mustConeDrill(t *testing.T) *topo.Body {
	t.Helper()
	cone, err := brep.SolidCylinderCone(math.P3(0, 0, 3), math.P3(0, 0, 1), 1.5, 0, "drill")
	if err != nil {
		t.Fatalf("conical drill: %v", err)
	}
	return cone
}

// cutWith subtracts tool from base, failing the test rather than returning an error.
func cutWith(t *testing.T, base, tool *topo.Body) *topo.Body {
	t.Helper()
	body, err := ops.Boolean(ops.Cut, base, tool)
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	return body
}

// meetWith intersects two bodies, failing the test rather than returning an error.
func meetWith(t *testing.T, base, tool *topo.Body) *topo.Body {
	t.Helper()
	body, err := ops.Boolean(ops.Intersect, base, tool)
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	return body
}

// ringAnd applies one operation between the corpus ring and the ball that meets it.
func ringAnd(t *testing.T, op ops.PartFeatureOperation) *topo.Body {
	t.Helper()
	body, err := ops.Boolean(op, mustTorus(t), mustSphere(t, math.P3(5, 0, 0), 2))
	if err != nil {
		t.Fatalf("ring/ball: %v", err)
	}
	return body
}
