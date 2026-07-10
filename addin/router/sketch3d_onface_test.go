// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketch3DOnFaceConstraintOverWire pins a 3D sketch point onto a part face by reference key and
// checks the constraint lands and enumerates as onFace (#1839). This is the dedicated end-to-end
// test the coverage guard points at (onFace needs a face ref + a solid, unlike the fixture kinds).
func TestSketch3DOnFaceConstraintOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	faceRef := blockFaceRef(t, r, s)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var pt wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[10,0,0]]}`, &pt)

	var res wire.AddSketch3DConstraintResult
	args, _ := json.Marshal(wire.AddSketch3DConstraintArgs{
		SketchIndex: 0, Kind: "onFace", Entities: []uint64{pt.EntityID}, FaceRef: faceRef,
	})
	call(t, r, s, "sketch3d.addConstraint", string(args), &res)
	if res.Kind != "onFace" {
		t.Errorf("addConstraint kind = %q, want onFace", res.Kind)
	}

	var cons wire.ListConstraints3DResult
	call(t, r, s, "sketch3d.constraints", `{"sketchIndex":0}`, &cons)
	if len(cons.Constraints) != 1 || cons.Constraints[0].Kind != "onFace" {
		t.Errorf("enumerated constraints = %+v, want one onFace", cons.Constraints)
	}
}

// TestSketch3DOnFaceUnresolvedFaceErrors: an onFace with a face reference that does not resolve is a
// clean error (#1839).
func TestSketch3DOnFaceUnresolvedFaceErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	blockFaceRef(t, r, s) // a solid exists, but the ref below is not one of its faces
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	var pt wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[10,0,0]]}`, &pt)

	args, _ := json.Marshal(wire.AddSketch3DConstraintArgs{
		SketchIndex: 0, Kind: "onFace", Entities: []uint64{pt.EntityID}, FaceRef: "no-such-face",
	})
	if err := tryCall(t, r, s, "sketch3d.addConstraint", string(args)); err == nil {
		t.Error("onFace with an unresolved face should error")
	}
}
