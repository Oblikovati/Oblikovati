// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TessellateBody's post-condition: a closed solid meshes closed, and a tear is REPORTED.

// TestAWatertightSolidRecordsNothing is the control. A box meshes closed, so the post-condition must
// stay silent — a diagnostic that fires on the ordinary case is noise, not a signal.
func TestAWatertightSolidRecordsNothing(t *testing.T) {
	t.Parallel()
	body := subd.ToBody(subd.Box(2, 2, 2), "box")
	mesh, _ := TessellateBody(body, DefaultQuality())
	if n := FreeEdgeCount(mesh); n != 0 {
		t.Fatalf("the control body meshes with %d free edges; it is not the watertight case", n)
	}
	for _, d := range mesh.Diagnostics {
		if d.Code == CodeMeshNotWatertight {
			t.Errorf("a watertight body reported %q: %s", d.Code, d.Detail)
		}
	}
}

// TestATornMeshOfAClosedSolidIsReported drives the post-condition itself: a closed solid whose per-face
// meshes do not join into a closed surface (one triangle removed) must come back with the named defect,
// the count, and the face.
func TestATornMeshOfAClosedSolidIsReported(t *testing.T) {
	t.Parallel()
	body := subd.ToBody(subd.Box(2, 2, 2), "box")
	faces, fm := TessellateBodyFaces(body, DefaultQuality())
	fm[0].Indices = fm[0].Indices[3:] // tear the first face's mesh: one triangle short of its own rim
	fm[0].Diagnostics = nil
	recordBodyMeshTear(body, faces, fm)
	assertTearReported(t, fm[0], string(faces[0].ReferenceKey()))
}

// TestATearReachesTheWholeBodyMesh: the tear is recorded on a FACE mesh, and MergeMesh carries face
// diagnostics up, so a caller holding only the whole-body mesh still sees it. That is the route
// TessellateBody's own callers take.
func TestATearReachesTheWholeBodyMesh(t *testing.T) {
	t.Parallel()
	body := subd.ToBody(subd.Box(2, 2, 2), "box")
	faces, fm := TessellateBodyFaces(body, DefaultQuality())
	fm[0].Indices, fm[0].Diagnostics = fm[0].Indices[3:], nil
	recordBodyMeshTear(body, faces, fm)
	whole := &Mesh{}
	for _, m := range fm {
		MergeMesh(whole, m)
	}
	assertTearReported(t, whole, string(faces[0].ReferenceKey()))
}

// assertTearReported checks the mesh carries the named Defect and that it points at the face.
func assertTearReported(t *testing.T, m *Mesh, face string) {
	t.Helper()
	for _, d := range m.Diagnostics {
		if d.Code != CodeMeshNotWatertight {
			continue
		}
		if d.Severity != diag.Defect {
			t.Errorf("the tear is reported at severity %v, want a Defect", d.Severity)
		}
		if !strings.Contains(d.Detail, face) {
			t.Errorf("the tear does not name the face it touches (%q); detail: %s", face, d.Detail)
		}
		return
	}
	t.Errorf("a torn mesh of a closed solid reported nothing; got %v", m.Diagnostics)
}

// TestAnOpenSheetIsNotReportedAsTorn: the closure test reads the B-REP, so a body that is genuinely
// open — a sheet, an unstitched import — is not a tear and must stay silent whatever its mesh does.
func TestAnOpenSheetIsNotReportedAsTorn(t *testing.T) {
	t.Parallel()
	if bodyIsClosedSolid(openSheetBody(t)) {
		t.Fatal("a one-face sheet is reported as a closed solid; the post-condition's gate is wrong")
	}
}

// openSheetBody builds a single planar face with a boundary — an open sheet, not a solid.
func openSheetBody(t *testing.T) *topo.Body {
	t.Helper()
	lin := func(i int) topo.Lineage { return topo.NewLineage(topo.Tok("sheet", "e", i)) }
	bld := topo.NewBuilder(false, lin(9))
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	vs := make([]*topo.Vertex, len(pts))
	for i, p := range pts {
		vs[i] = bld.AddVertex(p, lin(i))
	}
	uses := make([]topo.Use, len(pts))
	for i := range pts {
		j := (i + 1) % len(pts)
		e := bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), vs[i], vs[j], lin(4+i))
		uses[i] = topo.Fwd(e)
	}
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	bld.AddFace(pl, lin(8), topo.OuterLoop(uses...))
	return bld.Build()
}
