// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/renderer"
)

// TestPointCloudItemsRendersVisibleClouds: a visible attached cloud yields one Lines marker batch
// (6 vertices per point); hiding it removes the batch (M17-F06, #645).
func TestPointCloudItemsRendersVisibleClouds(t *testing.T) {
	s, def := emptyPartSession(t)
	if got := s.PointCloudItems(0.5); len(got) != 0 {
		t.Fatalf("a part with no clouds yields %d items, want 0", len(got))
	}

	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("scan")})
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1), math.P3(2, 2, 2)}
	pc, err := def.PointClouds().Add("Cloud1", "c.xyz", rid, pts)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}

	items := s.PointCloudItems(0.5)
	if len(items) != 1 || items[0].Primitive != renderer.Lines {
		t.Fatalf("items = %+v, want one Lines batch", items)
	}
	if len(items[0].Positions) != len(pts)*6 {
		t.Errorf("positions = %d, want %d (6 per point)", len(items[0].Positions), len(pts)*6)
	}

	pc.SetVisible(false)
	if got := s.PointCloudItems(0.5); len(got) != 0 {
		t.Errorf("a hidden cloud still rendered %d items, want 0", len(got))
	}
}
