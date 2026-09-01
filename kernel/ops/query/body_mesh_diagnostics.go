// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// BodyMeshDiagnostics tessellates b at q and returns what the tessellator recorded while meshing its
// faces, collapsed to one entry per code.
//
// The mesher already records its degradations — a saturated interior cap, an uncovered CDT domain, a
// wall wrap that fell to the flat patch — onto the [Mesh] it returns (see [Mesh.Diagnose]). But a mesh
// is a value most callers throw away, so until Oblikovati/Oblikovati#2058 every one of those records
// died at this package's boundary: an add-in saw an empty diagnostics array while the tessellator knew
// the mesh did not cover its own face (#2038, where that cost 77% of a body's volume, silently). This
// is the harvest a caller holding a [diag.Recorder] forwards into it, the way
// [BooleanWithDiagnostics] reports a fallback.
//
// It meshes FACES only (edge polylines carry no diagnostics) and is a pure query — nothing is cached
// and the mesh is discarded, so the cost is one tessellation. A nil body yields nil.
//
// Example:
//
//	for _, d := range query.BodyMeshDiagnostics(body, query.DefaultQuality()) {
//		rec.Record(d)
//	}
func BodyMeshDiagnostics(b *topo.Body, q Quality) []diag.Diagnostic {
	if b == nil {
		return nil
	}
	_, meshes := tessellate.TessellateBodyFaces(b, q)
	tallies := newMeshDiagTallies()
	for _, m := range meshes {
		tallies.addMesh(m)
	}
	return tallies.collapsed()
}

// meshDiagTally is one code's running total: the first record seen (its Detail names that face's own
// numbers) raised to the worst severity any face reported it at, plus how many faces reported it.
type meshDiagTally struct {
	first diag.Diagnostic
	count int
}

// meshDiagTallies accumulates per-face records by code while preserving first-appearance order, so a
// body's report reads in meshing order rather than in map order.
type meshDiagTallies struct {
	order  []diag.Code
	byCode map[diag.Code]*meshDiagTally
}

func newMeshDiagTallies() *meshDiagTallies {
	return &meshDiagTallies{byCode: map[diag.Code]*meshDiagTally{}}
}

// addMesh folds one face mesh's records in. A nil mesh (a face the mesher declined) contributes none.
func (ts *meshDiagTallies) addMesh(m *Mesh) {
	if m == nil {
		return
	}
	for _, d := range m.Diagnostics {
		ts.add(d)
	}
}

func (ts *meshDiagTallies) add(d diag.Diagnostic) {
	t, seen := ts.byCode[d.Code]
	if !seen {
		t = &meshDiagTally{first: d}
		ts.byCode[d.Code], ts.order = t, append(ts.order, d.Code)
	}
	t.count++
	if d.Severity > t.first.Severity {
		t.first.Severity = d.Severity // the worst face wins; a body is as degraded as its worst face
	}
}

// collapsed renders one diagnostic per code. Collapsing is not cosmetic: one malformed trim usually
// repeats on every face that shares it — a whole imported wall, every instance of a pattern — and a
// few hundred identical lines on a feature reply hide the report instead of being it.
func (ts *meshDiagTallies) collapsed() []diag.Diagnostic {
	if len(ts.order) == 0 {
		return nil
	}
	out := make([]diag.Diagnostic, 0, len(ts.order))
	for _, code := range ts.order {
		out = append(out, ts.byCode[code].rendered())
	}
	return out
}

// rendered is the tally as one diagnostic: the first face's Detail verbatim when only that face hit
// it, with the face count appended when more did.
func (t *meshDiagTally) rendered() diag.Diagnostic {
	if t.count == 1 {
		return t.first
	}
	d := t.first
	d.Detail = fmt.Sprintf("%s [recorded while meshing %d of the body's faces]", d.Detail, t.count)
	return d
}
