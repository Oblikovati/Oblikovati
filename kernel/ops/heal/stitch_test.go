// SPDX-License-Identifier: GPL-2.0-only

package heal

import (
	"strings"
	"testing"

	"oblikovati.org/test-utilities/brepfixture"
)

func TestStitchClosedSurfacesYieldsSolid(t *testing.T) {
	t.Parallel()
	body, err := Stitch(brepfixture.CubeFaces(), 0, false, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	if !body.IsSolid() {
		t.Error("stitching the six closed cube faces should yield a solid")
	}
	if got := len(body.Faces()); got != 6 {
		t.Errorf("stitched cube has %d faces, want 6", got)
	}
	if got := len(BoundaryEdges(body)); got != 0 {
		t.Errorf("stitched cube has %d boundary edges, want 0 (watertight)", got)
	}
	if r := Validate(body); !r.Valid || !r.Closed || !r.Manifold || !r.OrientationOK {
		t.Errorf("stitched cube validation = %+v, want fully valid", r)
	}
}

// TestStitchSeamEdgesAreProvenanceNamed is ADR-0043: a stitched seam edge is named by the two
// source faces it joins (and the body keeps each source face's identity), not by a synthesized
// weld-edge ordinal — so a selection on a sewn edge/face survives an upstream edit.
func TestStitchSeamEdgesAreProvenanceNamed(t *testing.T) {
	t.Parallel()
	body, err := Stitch(brepfixture.CubeFaces(), 0, false, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	ks := func(k []byte) string {
		if len(k) > 0 && k[0] < 0x20 {
			return string(k[1:])
		}
		return string(k)
	}
	faceKept, weldOrd, prov := false, 0, 0
	for _, f := range body.Faces() {
		if ks(f.ReferenceKey()) == "bottom:face#0" {
			faceKept = true // a source face kept its identity through the weld
		}
	}
	for _, e := range body.Edges() {
		switch k := ks(e.ReferenceKey()); {
		case strings.Contains(k, "weld-edge#"):
			weldOrd++
		case strings.Contains(k, "/stitch:x#0/"):
			prov++ // a seam named by its two joining faces
		}
	}
	if !faceKept {
		t.Error("a stitched face lost its source identity (bottom:face#0)")
	}
	if weldOrd > 0 {
		t.Errorf("%d edges still carry a synthesized weld-edge ordinal", weldOrd)
	}
	if prov == 0 {
		t.Error("no seam edge is provenance-named (face / stitch:x#0 / face) by its joining faces")
	}
}

func TestStitchMaintainAsSurfaceKeepsOpen(t *testing.T) {
	t.Parallel()
	// Even when the quilt closes, maintainSurface keeps it a surface body.
	body, err := Stitch(brepfixture.CubeFaces(), 0, true, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	if body.IsSolid() {
		t.Error("maintainSurface should keep the result a surface body")
	}
}

func TestStitchOpenQuiltStaysSurface(t *testing.T) {
	t.Parallel()
	// Drop the top face → the quilt cannot close, so the result is a surface body.
	faces := brepfixture.CubeFaces()[1:] // omit bottom? no — omit one to leave an opening
	body, err := Stitch(faces, 0, false, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	if body.IsSolid() {
		t.Error("an open quilt must not become a solid")
	}
	if len(BoundaryEdges(body)) == 0 {
		t.Error("an open quilt should report boundary edges")
	}
}

func TestStitchNoBodiesErrors(t *testing.T) {
	t.Parallel()
	if _, err := Stitch(nil, 0, false, "stitch"); err == nil {
		t.Error("stitching no bodies should error")
	}
}
