// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestExtrudedBSplineImportsClosed regresses the STEP open-shell import defect (corpus resurvey
// 2026-07-24 §4): a MANIFOLD_SOLID_BREP whose swept side wall is a SURFACE_OF_LINEAR_EXTRUSION of a
// B-spline profile used to drop that one face, importing a 3-face OPEN shell (Valid=F). DRAWEXE 8.0.0
// reads the SAME file as a valid 4-face closed solid, area 108534 (oracle). The fixture is the corpus
// G3 base body; we now recover the swept face and match OCCT to ~1e-5. See NewExtrudedBSplineSurface.
func TestExtrudedBSplineImportsClosed(t *testing.T) {
	t.Parallel()
	body := importOneSolid(t, "extruded_bspline_solid.step")
	rep := ops.Validate(body)
	if !rep.Valid || !rep.Closed {
		t.Fatalf("imported body Valid=%v Closed=%v, want valid closed solid (open-shell regression)", rep.Valid, rep.Closed)
	}
	if nf := len(body.Faces()); nf != 4 {
		t.Fatalf("imported face count = %d, want 4 (one B-spline extrusion wall + three planes; OCCT SOLID:1 FACE:4)", nf)
	}
	const occtArea = 108534.0 // DRAWEXE stepread G3.step -> checkshape valid, sprops area
	area := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Area
	if rel := (area - occtArea) / occtArea; rel < -1e-3 || rel > 1e-3 {
		t.Errorf("surface area = %.6g, want OCCT %.6g (rel %.4f%% > 0.1%%)", area, occtArea, rel*100)
	}
}
