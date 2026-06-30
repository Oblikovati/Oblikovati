// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
)

// realBox builds a real subdivided box solid at (px,py,pz) of size (sx,sy,sz) — the same fixture
// kernel/brep's topological-naming tests use, so these exercise the REAL boolean and the REAL
// provenance naming (fragmentLineage), not synthetic lineages.
func realBox(name string, px, py, pz, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(math.V3(px, py, pz))
	}
	return subd.ToBody(m, name)
}

// realTopFaceAt finds the +Z-facing face whose range box contains p — a top-face fragment.
func realTopFaceAt(t *testing.T, b *topo.Body, p math.Point3) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Dot(math.V3(0, 0, 1)) > 0.9 && f.RangeBox().Contains(p) {
			return f
		}
	}
	t.Fatalf("no +Z top fragment over %v (faces=%d)", p, len(b.Faces()))
	return nil
}

// grooveTop subtracts a full-width slab from the top of a 16×10×4 base, grooving it at y∈[4.5,5.5]
// down to z=2 — a REAL boolean that splits the top face into a front (y<4.5) and a back (y>5.5)
// fragment. The tool name varies the cutting feature's lineage (an authored-then-edited cut).
func grooveTop(t *testing.T, tool string) *topo.Body {
	t.Helper()
	res, err := brep.Boolean(brep.Difference, realBox("base", 0, 0, 0, 16, 10, 4), realBox(tool, -1, 4.5, 2, 18, 1, 3))
	if err != nil {
		t.Fatalf("difference (%s): %v", tool, err)
	}
	return res
}

// TestRealSplitFaceFragmentsShareParent pins the precondition the recovery tiers depend on, on REAL
// geometry: the two fragments a real boolean carves from one face are named by their bordering cut
// face as the LAST lineage token, so they share a parent (dropping that token) — exactly what makes
// them recoverable siblings. (Confirmed empirically: parent is base:face#1/brep:cut#0/brep:by#0.)
func TestRealSplitFaceFragmentsShareParent(t *testing.T) {
	split := grooveTop(t, "slab")
	front := realTopFaceAt(t, split, math.P3(8, 2, 4))
	back := realTopFaceAt(t, split, math.P3(8, 8, 4))

	if pf, pb := parentOfKey(front.ReferenceKey()), parentOfKey(back.ReferenceKey()); string(pf) != string(pb) || len(pf) == 0 {
		t.Fatalf("real split fragments must share a non-empty parent:\n front parent %q\n back  parent %q", pf, pb)
	}
	if string(front.ReferenceKey()) == string(back.ReferenceKey()) {
		t.Error("the two fragments must have DISTINCT keys (different bordering cut face)")
	}
}

// TestRealFaceReferenceHealsGeometricallyAfterCutEdit is the end-to-end real-geometry verification
// of ADR-0043 P6 face recovery: a face reference is picked on a real split (the front fragment),
// then the cutting feature is EDITED (a different lineage), which drifts the fragment's
// bordering-face token so the stored key no longer resolves exactly — yet both same-parent
// fragments survive. The geometric tier must rebind the reference to the GEOMETRICALLY correct real
// fragment using the mint-time anchor, and binding it the other way must pick the other fragment.
func TestRealFaceReferenceHealsGeometricallyAfterCutEdit(t *testing.T) {
	authored := grooveTop(t, "slab")
	front := realTopFaceAt(t, authored, math.P3(8, 2, 4))
	back := realTopFaceAt(t, authored, math.P3(8, 8, 4))
	picked := front.ReferenceKey()
	frontAnchor := topo.DescribeFace(front).Centroid
	backAnchor := topo.DescribeFace(back).Centroid

	// The user edits the cut (a fresh cutting feature, "slab2"): same geometry, different lineage,
	// so the bordering-face token differs and the picked key is gone — but both fragments survive.
	edited := grooveTop(t, "slab2")
	if n := len(edited.FacesByKey(picked)); n != 0 {
		t.Fatalf("precondition: the picked key must NOT resolve exactly after the cut edit, got %d", n)
	}

	faces, heals, err := resolveFaces(edited, [][]byte{picked}, map[string]math.Point3{string(picked): frontAnchor})
	if err != nil {
		t.Fatalf("a real broken face reference with surviving siblings did not recover: %v", err)
	}
	if len(heals) != 1 || heals[0].Match != identity.MatchGeometric {
		t.Fatalf("expected a GEOMETRIC heal on real geometry, got %+v", heals)
	}
	if got := topo.DescribeFace(faces[0]).Centroid; got.DistanceTo(frontAnchor) > 1e-6 {
		t.Errorf("geometric heal bound the wrong fragment: centroid %v, want the FRONT %v (back is %v)", got, frontAnchor, backAnchor)
	}

	// The anchor — not luck — is what disambiguates: the back anchor recovers the BACK fragment.
	faces2, _, err := resolveFaces(edited, [][]byte{picked}, map[string]math.Point3{string(picked): backAnchor})
	if err != nil {
		t.Fatalf("back-anchored recovery failed: %v", err)
	}
	if got := topo.DescribeFace(faces2[0]).Centroid; got.DistanceTo(backAnchor) > 1e-6 {
		t.Errorf("back anchor must recover the BACK fragment, got centroid %v", got)
	}

	// And with NO anchor the two surviving siblings are an honest ambiguity — recovery refuses to
	// guess (the honesty invariant), rather than silently binding one.
	if _, _, err := resolveFaces(edited, [][]byte{picked}, nil); err == nil {
		t.Error("two anchorless same-parent siblings must NOT heal — recovery guessed")
	}
}
