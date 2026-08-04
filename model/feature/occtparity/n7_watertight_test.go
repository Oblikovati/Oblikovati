// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestN7WholeBodyWatertight is the F3 kernel-level topology gate on the REAL N7 STEP body (the canal
// corner + geometric far-runout whole-body weld): it asserts the assembled solid is watertight WITHOUT
// tessellating (fast), so it complements the tessellation-based area gate TestOCCTBlendSimple/N7. The
// checks are the assembly's watertightness crux: every edge is used by EXACTLY two faces (manifold +
// closed = no crack, no non-manifold seam), the body is a valid closed orientable solid, and it carries
// the oracle's 12 faces. A regression that cracks a seam (e.g. the wall foot-locus sampled differently
// from the corner patch, F3's reconciliation #1) fails the 2-incidence assertion here loud and fast.
func TestN7WholeBodyWatertight(t *testing.T) {
	t.Parallel()
	body := n7ResultBody(t)
	if got := len(body.Faces()); got != 12 {
		t.Fatalf("N7 result has %d faces, want the oracle's 12", got)
	}
	for _, e := range body.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Fatalf("N7 result edge %d is %d-incident (%v→%v), want exactly 2 (a watertight manifold solid)",
				e.ID(), n, e.StartVertex().Point(), e.EndVertex().Point())
		}
	}
	rep := ops.Validate(body)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !body.IsSolid() {
		t.Fatalf("N7 result not a watertight solid: valid=%v closed=%v manifold=%v holes=%v solid=%v issues=%v",
			rep.Valid, rep.Closed, rep.Manifold, rep.HolesContained, body.IsSolid(), rep.Issues)
	}
}

// n7ResultBody imports the N7 STEP fixture and runs the real fillet feature, returning the single result
// solid (the canal-weld whole body). Skips if the case is not resolvable on this build.
func n7ResultBody(t *testing.T) *topo.Body {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == "N7" {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Skipf("N7 import-divergence (not a fillet defect): %v", err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Skip("N7 picks could not be located on the imported body")
	}
	res, okFillet, reason := runFillet(body, sets)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("N7 fillet unhealthy: ok=%v reason=%q results=%d", okFillet, reason, len(res))
	}
	return res[0]
}
