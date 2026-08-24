// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// grazingSphereOnCylinder is a small sphere set just inside a cylinder's wall so their
// surfaces graze — the near-tangent contact the analytic/planar/CSG boolean cannot
// stitch into a valid solid, which is what the mesh-arrangement fallback rescues. The
// operands are coarse so the primary path's (failing) CSG attempt stays cheap.
func grazingSphereOnCylinder(t *testing.T) (*topo.Body, *topo.Body) {
	t.Helper()
	a, err := brep.SolidCylinder(math.P3(0, 0, -2), math.V3(0, 0, 1), 1, 4)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	b, err := brep.SolidSphere(math.P3(1.4, 0, 0), 0.5, "s")
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	return a, b
}

// TestBooleanMeshArrangementFallbackRescue is the routing regression: the
// analytic/planar/CSG path alone leaves the grazing sphere-on-cylinder join INVALID,
// and booleanGeneral rescues it with the exact mesh-arrangement engine. The fallback
// diagnostic fires ONLY when the primary path left an invalid result, so its presence
// proves both that the primary tore and that the mesh engine recovered a valid solid.
func TestBooleanMeshArrangementFallbackRescue(t *testing.T) {
	a, b := grazingSphereOnCylinder(t)
	rec := &diag.Recorder{}
	res, err := BooleanWithDiagnostics(Join, a, b, rec)
	if err != nil {
		t.Fatalf("boolean returned error: %v", err)
	}
	if !rec.Has(CodeBooleanMeshArrangementFallback) {
		t.Fatal("expected the mesh-arrangement fallback to fire (the primary path should tear here)")
	}
	if res == nil || !validBooleanSolid(res) || len(res.Faces()) == 0 {
		t.Fatal("mesh-arrangement fallback did not produce a valid non-empty solid")
	}
}

// TestMeshArrangementFallbackDeclines covers the guards under which the fallback does
// not apply: a non-set operation, an empty result (a small sphere wholly inside a
// larger one cut to nothing), and operands too large to justify the expensive engine.
func TestMeshArrangementFallbackDeclines(t *testing.T) {
	a, b := grazingSphereOnCylinder(t)
	if got := meshArrangementFallback(NewBody, a, b, nil); got != nil {
		t.Error("NewBody is not a set operation; fallback must decline")
	}

	small, err := brep.SolidSphere(math.P3(0, 0, 0), 1, "small")
	if err != nil {
		t.Fatalf("small sphere: %v", err)
	}
	big, err := brep.SolidSphere(math.P3(0, 0, 0), 3, "big")
	if err != nil {
		t.Fatalf("big sphere: %v", err)
	}
	if got := meshArrangementFallback(Cut, small, big, nil); got != nil {
		t.Error("small minus enclosing big is empty; the fallback must not adopt a faceless body")
	}

	oversize := soupToBody(bodyToSoup(big, DefaultQuality()), "oversize") // faceted: hundreds of faces
	if len(oversize.Faces()) <= csgFallbackFaceLimit {
		t.Fatalf("oversize fixture has only %d faces; expected > %d", len(oversize.Faces()), csgFallbackFaceLimit)
	}
	if got := meshArrangementFallback(Join, oversize, oversize, nil); got != nil {
		t.Error("operands above the face limit must skip the expensive fallback")
	}
}
