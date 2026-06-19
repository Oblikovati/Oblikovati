// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/model/sketch"
)

// TestSketch3DRefHelpersRejectMissingEntity covers the "no entity with id" branches of the 3D
// reference resolvers and the constraint builders that lean on them: an unknown entity id (999
// in an empty sketch) is rejected by every typed ref lookup.
func TestSketch3DRefHelpersRejectMissingEntity(t *testing.T) {
	sk := sketch.NewSketches3D().Add()
	const bad = uint64(999)

	if _, err := radiusScalar3D(sk, bad); err == nil {
		t.Error("radiusScalar3D(missing) should error")
	}
	if _, err := pointRef3D(sk, bad); err == nil {
		t.Error("pointRef3D(missing) should error")
	}
	if _, err := lineRef3D(sk, bad); err == nil {
		t.Error("lineRef3D(missing) should error")
	}
	if _, err := smoothCurveRef3D(sk, bad); err == nil {
		t.Error("smoothCurveRef3D(missing) should error")
	}
	if _, err := splineRef3D(sk, bad); err == nil {
		t.Error("splineRef3D(missing) should error")
	}
	if _, err := helixRef3D(sk, bad); err == nil {
		t.Error("helixRef3D(missing) should error")
	}
	if _, err := circleRef3D(sk, bad); err == nil {
		t.Error("circleRef3D(missing) should error")
	}
	// The bend builder resolves its arc by id first, so a missing first ref is rejected.
	if _, err := bendConstraint3D(sk, []uint64{bad, bad, bad}); err == nil {
		t.Error("bendConstraint3D(missing arc) should error")
	}
}
