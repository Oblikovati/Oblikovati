// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The mixed per-face dispatch (ADR-0058) declines rather than guesses: a configuration it does not
// model is refused at classification, before any geometry is built, and the caller falls to the
// curved/CSG paths. The retirement converts each such decline into a positive case as the
// configuration lands (ADR-0061); this file holds one that has.

// TestBooleanMixedUnionsADisjointSphereExactly was a DECLINE case: a whole sphere carried no boundary
// point to classify the face as a whole, so the dispatch refused it and the caller fell to the CSG
// fallback. ADR-0061 stage 3 gives a sphere its own loop-framed chart, so it is now an ordinary face
// with an ordinary answer — the positive corpus case the retirement converts each decline into.
//
// A sphere disjoint from a block unions to both lumps, exactly, with every surface analytic.
func TestBooleanMixedUnionsADisjointSphereExactly(t *testing.T) {
	t.Parallel()
	block, err := SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	ball, err := SolidSphere(math.P3(30, 30, 30), 2, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	res, err := Boolean(Union, block, ball)
	if err != nil {
		t.Fatalf("Boolean(Union, block, disjoint ball) = %v; a sphere is a charted face now", err)
	}
	if len(res.Shells()) != 2 {
		t.Errorf("a union of two disjoint solids has %d shell(s), want 2", len(res.Shells()))
	}
	spheres := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			spheres++
		}
	}
	if spheres != 1 {
		t.Errorf("the ball contributes %d analytic sphere face(s), want 1 — it must not be faceted", spheres)
	}
	for _, e := range res.Edges() {
		if len(e.Uses()) != 2 {
			t.Errorf("edge %v has %d uses; the union must be closed", e.Lineage(), len(e.Uses()))
		}
	}
}
