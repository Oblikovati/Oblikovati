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
		// The corner junction (#1738): a notched cylinder drilled by a rod that crosses the notch. Its
		// wall is a two-rim band carrying a lens hole and recording NO chart, which is the only trim
		// left on the unroll (curved_trim_classify.go).
		{"notched cylinder − crossing rod", func(t *testing.T) *topo.Body {
			return cutWith(t, notchedRod(t), mustCylinder(t, math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12))
		}},
		// The #1818 near-pinch crossing: two cylinders of ALMOST the same radius joined, whose merged
		// wall carries two lens windows whose corridor is narrower than the boundary's own chords. That
		// corridor is what the general covering cannot resolve and the unroll's bent seam is built for,
		// so this is the trim kindTwoRimHoledBand is left with (curved_trim_recognize.go).
		{"near-pinch crossing rods ∪", func(t *testing.T) *topo.Body {
			return joinWith(t, mustCylinder(t, math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12),
				mustCylinder(t, math.P3(0, 0, -6), math.V3(0, 0, 1), 3.00004, 12))
		}},
		{"drilled plate", func(t *testing.T) *topo.Body {
			return cutWith(t, mustBlock(t, math.P3(-5, -5, 0), math.P3(5, 5, 3.5)),
				mustCylinder(t, math.P3(0, 0, -1), math.V3(0, 0, 1), 1.5, 6))
		}},
	}
}

// TestCurvedTrimKindsAreMutuallyExclusive is the reorder-independence proof: no curved face in the
// corpus satisfies two classification recognizers, and the kind classifyCurvedTrim selects is exactly
// the one recognizer that holds (or the sphere family's residual, or the charted/uncharted residual,
// when none does).
func TestCurvedTrimKindsAreMutuallyExclusive(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	forEachCurvedCorpusFace(t, func(body string, i int, f *topo.Face) {
		hits := tessellate.CurvedTrimRecognizerHits(f, q)
		if len(hits) > 1 {
			sort.Strings(hits)
			t.Errorf("%s face %d (%T): %d recognizers claim it (%s) — a classification answers at "+
				"most once, so which mesher runs would depend on the order they are written in",
				body, i, f.Geometry(), len(hits), strings.Join(hits, ", "))
		}
		assertClassificationAgrees(t, body, i, f, hits, q)
	})
}

// TestSphereCapRimFormsAreDisjoint is the same proof one level down, where the ladder used to hide
// after the first pass at this task: the spherical cap's three RIM FORMS are read as an inventory, all
// three evaluated, and a face two of them claim is refused rather than resolved by position. This is
// what says the refusal never has to fire — and it is a real assertion, not a tautology, because each
// form is a separate recognizer function with its own gate.
func TestSphereCapRimFormsAreDisjoint(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	seen := 0
	forEachCurvedCorpusFace(t, func(body string, i int, f *topo.Face) {
		hits := tessellate.SphereCapRimFormHits(f, q)
		seen += len(hits)
		if len(hits) > 1 {
			sort.Strings(hits)
			t.Errorf("%s face %d: %d cap rim forms claim it (%s) — the boundary cannot be two shapes, "+
				"and taking the first is the ladder this replaced", body, i, len(hits), strings.Join(hits, ", "))
		}
	})
	if seen == 0 {
		t.Error("no corpus face presented any cap rim form — the disjointness proof covers nothing")
	}
}

// forEachCurvedCorpusFace runs visit over every CURVED face of every corpus body.
func forEachCurvedCorpusFace(t *testing.T, visit func(body string, i int, f *topo.Face)) {
	t.Helper()
	for _, row := range classificationCorpus() {
		for i, f := range row.build(t).Faces() {
			if _, planar := f.Geometry().(geom.Plane); !planar {
				visit(row.name, i, f)
			}
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

// nearPinchCorpusBody is the one corpus body whose two-rim band the arm still keeps: its two lens
// windows pass 0.031 mm apart on a boundary sampled every 0.588 mm, and no covering laid at that
// sampling separates them. Every other two-rim band in the corpus goes to the general chart-driven
// mesher (curved_trim_recognize.go's corridor gate).
const nearPinchCorpusBody = "near-pinch crossing rods ∪"

// TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe is the conditioning gate's own proof over the
// corpus: the arm must keep exactly the bands the general chart-driven mesher cannot serve — one that
// records no chart, and the near-pinch body's, whose two windows pass closer than the boundary is
// sampled — and give up every other two-rim holed band there is. Both directions are asserted and the
// test fails if the corpus stops covering either, so a gate that let go of everything — or of
// nothing — is caught.
func TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	kept, given := 0, 0
	forEachCurvedCorpusFace(t, func(body string, i int, f *topo.Face) {
		isShape, charted, toArm := tessellate.TwoRimHoledBandVerdict(f, q)
		if !isShape {
			return
		}
		wantArm := !charted || body == nearPinchCorpusBody
		if toArm {
			kept++
		} else {
			given++
		}
		if toArm != wantArm {
			t.Errorf("%s face %d (charted=%v): the corridor gate sends this two-rim band to the %s; want the %s",
				body, i, charted, armOrChart(toArm), armOrChart(wantArm))
		}
	})
	if kept == 0 || given == 0 {
		t.Errorf("the corpus presents %d bands the arm keeps and %d it gives up; it must cover both or "+
			"the gate is proved in one direction only", kept, given)
	}
}

// armOrChart names which mesher a verdict selects, for the failure message.
func armOrChart(toArm bool) string {
	if toArm {
		return "unrolled arm"
	}
	return "chart-driven mesher"
}

// classifiedCurvedFaces counts the curved faces of the whole corpus by the kind they classify as.
func classifiedCurvedFaces(t *testing.T) map[string]int {
	t.Helper()
	q := ops.DefaultQuality()
	seen := map[string]int{}
	forEachCurvedCorpusFace(t, func(_ string, _ int, f *topo.Face) {
		seen[tessellate.ClassifyCurvedTrimName(f, q)]++
	})
	return seen
}

// assertClassificationAgrees ties classifyCurvedTrim to the recognizers: it must select the one kind
// that holds, and when none holds it must fall to a residual — the sphere family's arc-bounded patch,
// or the charted/uncharted split.
func assertClassificationAgrees(t *testing.T, body string, i int, f *topo.Face, hits []string, q tessellate.Quality) {
	t.Helper()
	got := tessellate.ClassifyCurvedTrimName(f, q)
	if len(hits) == 1 {
		if got != hits[0] {
			t.Errorf("%s face %d (%T): recognizer %s holds but the classification chose %s",
				body, i, f.Geometry(), hits[0], got)
		}
		return
	}
	residual := map[string]bool{
		tessellate.SpherePatchTrimKindName(): true,
		tessellate.ChartedTrimKindName():     true,
		tessellate.UnchartedTrimKindName():   true,
	}
	if len(hits) == 0 && !residual[got] {
		t.Errorf("%s face %d (%T): no recognizer holds but the classification chose %s, not a residual",
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

// notchedRod is a cylinder with one oblique half-space bite taken out of its top rim (#1738's first cut).
func notchedRod(t *testing.T) *topo.Body {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1))
	if err != nil {
		t.Fatalf("notch plane: %v", err)
	}
	notched, err := brep.HalfSpaceCut(mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10), pl)
	if err != nil {
		t.Fatalf("notch cut: %v", err)
	}
	return notched
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

// joinWith unions two bodies, failing the test rather than returning an error.
func joinWith(t *testing.T, base, tool *topo.Body) *topo.Body {
	t.Helper()
	body, err := ops.Boolean(ops.Join, base, tool)
	if err != nil {
		t.Fatalf("join: %v", err)
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
