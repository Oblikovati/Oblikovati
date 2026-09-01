// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
)

// topFaceKey returns the reference key of the body's highest face (greatest range-box centre z) —
// a planar cap to use as a to-face termination target.
func topFaceKey(b *topo.Body) string {
	var best *topo.Face
	for _, f := range b.Faces() {
		if best == nil || f.RangeBox().Center().Z > best.RangeBox().Center().Z {
			best = f
		}
	}
	return string(best.ReferenceKey())
}

// TestExtrudeToFaceApply exercises the to-face extent in-package: extrude a profile up to an
// existing body's top face (given by key), proving the op resolves the termination plane and
// produces a healthy body. This is the coverage twin of the router/wire test (#1226 → API).
func TestExtrudeToFaceApply(t *testing.T) {
	t.Parallel()
	s := profiledPart(t)
	if _, err := apply(t, s, "extrude", `{"sketchIndex":0,"distance":"10 mm"}`); err != nil {
		t.Fatalf("seed extrude: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	top := topFaceKey(def.SurfaceBodies().Item(0))

	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 1, "profileIndex": 0, "extent": "to-face", "toFace": top, "operation": "new",
	})
	out, err := apply(t, s, "extrude", string(args))
	if err != nil {
		t.Fatalf("to-face extrude: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !res.Healthy || res.Bodies != 2 {
		t.Fatalf("to-face result = %+v, want healthy with 2 bodies", res)
	}
}

// TestExtrudeToFaceApplyErrors covers the to-face guard rails: a missing target and an
// unresolvable target reference are both clear errors, not silent successes.
func TestExtrudeToFaceApplyErrors(t *testing.T) {
	t.Parallel()
	if _, err := apply(t, profiledPart(t), "extrude", `{"sketchIndex":0,"extent":"to-face"}`); err == nil {
		t.Error("to-face without a toFace target should error")
	}
	if _, err := apply(t, profiledPart(t), "extrude", `{"sketchIndex":0,"extent":"to-face","toFace":"plane/99"}`); err == nil {
		t.Error("to-face with an unresolvable target should error")
	}
}
