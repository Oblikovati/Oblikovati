// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestPlanarizedDiagRecordsAnalyticFaceting pins the audit-A5 chokepoint (#1601): converting a body
// that carries an analytic curved face into a planar B-rep is a PERMANENT degradation and must be
// observable. planarizedDiag records CodeBooleanAnalyticFaceted for a curved operand, stays quiet for
// an already-planar one, and in both cases returns a body with no curved face left.
func TestPlanarizedDiagRecordsAnalyticFaceting(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 2)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}

	rec := &diag.Recorder{}
	out := planarizedDiag(cyl, "unit", rec)
	if hasCurvedFace(out) {
		t.Error("planarizedDiag left a curved face on an analytic operand")
	}
	if !rec.Has(ops.CodeBooleanAnalyticFaceted) {
		t.Errorf("faceting an analytic cylinder recorded no %q; got %v", ops.CodeBooleanAnalyticFaceted, rec.Records())
	}

	quiet := &diag.Recorder{}
	if got := planarizedDiag(block, "unit", quiet); hasCurvedFace(got) {
		t.Error("planarizedDiag altered an already-planar body's face kinds")
	}
	if quiet.Has(ops.CodeBooleanAnalyticFaceted) {
		t.Errorf("planarizing an already-planar block recorded a spurious facet defect: %v", quiet.Records())
	}
}

// TestPlanarizedDiagNilRecorderSafe confirms the chokepoint is safe on the nil recorder the legacy
// (diagnostic-less) call paths still pass.
func TestPlanarizedDiagNilRecorderSafe(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 2)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	if out := planarizedDiag(cyl, "unit", nil); hasCurvedFace(out) {
		t.Error("planarizedDiag(nil recorder) did not facet the analytic operand")
	}
}

// TestNoSilentFacetBeforeConsumer is audit A5's "grep/guard test over decline paths" (#1601): NO
// feature may facet an analytic operand into a planar-only consumer (a boolean or a face delete)
// through the plain, diagnostic-less planarized() — every such site must route through planarizedDiag
// so the permanent analytic→facet loss always surfaces as a diagnostic. (hull is exempt: its result
// is a polyhedron by definition, so it feeds ConvexHullOf, not a boolean — the facet is inherent, not
// a hidden degradation.)
func TestNoSilentFacetBeforeConsumer(t *testing.T) {
	consumer := regexp.MustCompile(`ops\.(Boolean|BooleanWithDiagnostics|DeleteFaces)\(`)
	bareFacet := regexp.MustCompile(`[^D]planarized\(`) // planarized( but not planarizedDiag(
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", name, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if consumer.MatchString(line) && bareFacet.MatchString(line) {
				t.Errorf("%s:%d facets an operand into a planar consumer via plain planarized() — "+
					"route it through planarizedDiag so the analytic→facet loss is observable:\n\t%s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
}
