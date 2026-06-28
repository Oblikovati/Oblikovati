// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestChamferKeepsUntouchedEdgeIdentity is ADR-0043 P2′: a boolean (here a chamfer's wedge cut)
// must leave the edges it passes through WHOLE with their original identity, not renumber them to a
// build-order ordinal. Chamfering one box edge leaves the diagonally-opposite vertical edge
// untouched, so a selection on it stays bound to Extrusion1:side-edge#3 across the operation —
// where the boolean's brep:edge#N fallback would have lost it.
func TestChamferKeepsUntouchedEdgeIdentity(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 3}, {X: 0, Y: 3}},
		sketch.XYPlane(), span{near: 0, far: 5}, 0, "Extrusion1")
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	NewDressUpFeatures(fs).AddChamfer([][]byte{edgeByKeySuffix1536(t, box, "Extrusion1:side-edge#1")}, func() float64 { return 0.6 })
	fs.Recompute()

	untouched, anyOrdinal := false, false
	for _, e := range fs.Result()[0].Edges() {
		switch k := keyString1536(e.ReferenceKey()); {
		case k == "Extrusion1:side-edge#3":
			untouched = true
		case strings.HasPrefix(k, "brep:edge#"):
			anyOrdinal = true // a split fragment near the chamfer keeps the fallback (a later phase)
		}
	}
	if !untouched {
		t.Error("the untouched opposite edge lost its Extrusion1:side-edge#3 identity through the chamfer")
	}
	_ = anyOrdinal // documented: split fragments are not yet provenance-named (epic #1539)
}
