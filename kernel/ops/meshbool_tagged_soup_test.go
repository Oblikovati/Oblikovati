// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestBodyToTaggedSoupProvenance proves the IN adapter tags every triangle with the
// EXACT originating surface: a capped cylinder tessellates into one cylindrical wall
// and two planar caps, and every soup triangle's tag must resolve to that face's true
// surface kind. This is the provenance reconstruction (ADR-0056) depends on — the tag
// is not a fit, it is which face the tessellator meshed.
func TestBodyToTaggedSoupProvenance(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	soup, refs := bodyToTaggedSoup(cyl, PropertyQuality(), 0)
	if len(soup.Tris) == 0 || len(soup.Tags) != len(soup.Tris) {
		t.Fatalf("tagged soup malformed: %d tris, %d tags", len(soup.Tris), len(soup.Tags))
	}

	var cylFaces, planeFaces int
	for _, r := range refs {
		switch r.surface.(type) {
		case geom.Cylinder:
			cylFaces++
		case geom.Plane:
			planeFaces++
		}
	}
	if cylFaces != 1 || planeFaces != 2 {
		t.Fatalf("capped cylinder refs = %d cylinder + %d plane, want 1 + 2", cylFaces, planeFaces)
	}

	// Every triangle's tag must index a real ref, and a wall triangle's vertices must
	// sit on that cylinder — proving the tag names the surface the facet truly lies on.
	wallTag := -1
	for i, r := range refs {
		if _, ok := r.surface.(geom.Cylinder); ok {
			wallTag = i
		}
	}
	var wallTris int
	for i, tag := range soup.Tags {
		if tag < 0 || tag >= len(refs) {
			t.Fatalf("tri %d: tag %d out of range [0,%d)", i, tag, len(refs))
		}
		if tag != wallTag {
			continue
		}
		wallTris++
		cyl := refs[tag].surface.(geom.Cylinder)
		for _, p := range soup.Tris[i] {
			if d := geom.SignedDistanceToSurface(cyl, p.Round()); absF(d) > 1e-6 {
				t.Fatalf("wall tri %d vertex off cylinder by %g", i, d)
			}
		}
	}
	if wallTris == 0 {
		t.Fatal("no triangle tagged to the cylinder wall")
	}
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
