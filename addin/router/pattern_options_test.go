// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/app"
)

// TestPatternRectangularAcceptsOptions drives the full wire path for an M20-F18 pattern
// with a spacing type and a clipping boundary: three new-body occurrences 5 cm apart along
// X, clipped to a box that excludes the third, leave the seed plus one copy (#652).
func TestPatternRectangularAcceptsOptions(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t) // one 40×30 mm extrude (Extrusion1) → one solid body
	if before := bodyCount(t, r, s); before != 1 {
		t.Fatalf("fixture body count = %d, want 1", before)
	}
	call(t, r, s, "features.add", `{"kind":"patternRectangular","args":{
		"sourceFeatures":["Extrusion1"],"countX":3,"countY":1,"stepX":[5,0,0],
		"spacingType":"spacing",
		"boundary":{"planeNormal":[0,0,1],"polygon":[[-5,-5,0],[8,-5,0],[8,5,0],[-5,5,0]],"inclusion":"centroid"}
	}}`, &struct{}{})
	if after := bodyCount(t, r, s); after != 2 {
		t.Errorf("clipped pattern body count = %d, want 2 (seed + one in-bounds copy)", after)
	}
}

// TestPatternRectangularRejectsUnknownSpacing surfaces a precise error for an unknown enum.
func TestPatternRectangularRejectsUnknownSpacing(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)
	_, err := r.Handle(s, "features.add", []byte(`{"kind":"patternRectangular","args":{
		"sourceFeatures":["Extrusion1"],"countX":2,"stepX":[5,0,0],"spacingType":"zigzag"}}`))
	if err == nil {
		t.Error("expected an error for an unknown spacingType")
	}
}

func bodyCount(t *testing.T, r *Router, s *app.Session) int {
	t.Helper()
	return modelTreeOf(t, r, s).Bodies
}
