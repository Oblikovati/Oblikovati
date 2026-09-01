// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// TestDraftTapersSideFace drafts the +X face of a 2×2×2 box inward by atan(0.25) about +Z:
// the face hinges at its base (z=0) and leans in, removing a 4·tan(angle)=1 wedge ⇒ vol 7.
func TestDraftTapersSideFace(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	var side []byte
	for _, f := range box.Faces() {
		if f.Geometry().NormalAt(0, 0).X > 0.99 {
			side = f.ReferenceKey()
		}
	}
	res, err := blend.DraftFaces(box, [][]byte{side}, math.V3(0, 0, 1), -stdmath.Atan(0.25))
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("drafted body not a valid solid: %+v", r)
	}
	if got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-7) > 1e-6 {
		t.Errorf("draft volume = %g, want 7", got)
	}
}

// TestDraftPreservesFaceIdentity is ADR-0043: drafting moves geometry but preserves every
// face/edge's identity (a 1:1 rebuild), so a selection on ANY face — the drafted one or an
// untouched neighbour — survives. Before, the rebuild renamed everything to draft:f#N / draft:e#N.
func TestDraftPreservesFaceIdentity(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	var side, top []byte
	for _, f := range box.Faces() {
		if n := f.Geometry().NormalAt(0, 0); n.X > 0.99 {
			side = f.ReferenceKey()
		} else if n.Z > 0.99 {
			top = f.ReferenceKey()
		}
	}
	res, err := blend.DraftFaces(box, [][]byte{side}, math.V3(0, 0, 1), -stdmath.Atan(0.25))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.FindFaceByKey(side); !ok {
		t.Error("the drafted side face lost its identity")
	}
	if _, ok := res.FindFaceByKey(top); !ok {
		t.Error("the untouched top face lost its identity — the rebuild renamed it")
	}
}

// TestDraftLostKeyErrors reports a vanished face key so the feature can go Sick.
func TestDraftLostKeyErrors(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	if _, err := blend.DraftFaces(box, [][]byte{[]byte("ghost")}, math.V3(0, 0, 1), 0.1); err == nil {
		t.Error("draft with a lost key should error")
	}
}

// TestDraftFilletedBodyTapers is the real #1802 fix (ADR-0050 Phase 7, #1809): drafting a planar
// face of a body that carries a fillet's curved face produces a VALID tapered solid — the modifier
// tilts the plane and re-intersects the cylinder (the shared edge becoming an arc), preserving the
// fillet face — instead of panicking in the plane-only rebuild (the Phase-0 stopgap it supersedes).
func TestDraftFilletedBodyTapers(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	filleted, err := blend.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0.5)
	if err != nil {
		t.Fatalf("fillet setup: %v", err)
	}
	if hasCylinderFaces(filleted) == 0 {
		t.Fatal("fillet setup produced no curved face — cannot exercise the modifier")
	}
	var side []byte
	for _, f := range filleted.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok && pl.NormalAt(0, 0).X > 0.99 {
			side = f.ReferenceKey()
		}
	}
	if side == nil {
		t.Fatal("no planar +X face to draft on the filleted body")
	}
	res, err := blend.DraftFaces(filleted, [][]byte{side}, math.V3(0, 0, 1), -stdmath.Atan(0.2))
	if err != nil {
		t.Fatalf("draft on a filleted body must taper it, not fail: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("drafted filleted body is not a valid solid: %+v", r)
	}
	if hasCylinderFaces(res) == 0 {
		t.Error("the fillet's cylinder face was lost by the draft — it should be re-trimmed, not dropped")
	}
}
