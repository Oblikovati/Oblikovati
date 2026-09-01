// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// sketchPlaneAt returns the host coordinate frame of the sketch at index i on the active part.
func sketchPlaneAt(t *testing.T, s *app.Session, i int) (x, y, n math.UnitVector3) {
	t.Helper()
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	pl := part.Sketches().Item(i).Plane()
	return pl.XAxis(), pl.YAxis(), pl.Normal()
}

// TestSketchOrientationPinsFrameToAxis is the #1920 core: a sketch created on a plane built
// through the Z axis at 30° to XZ, oriented with its Y pinned to +Z, must come out with
// YAxis = +Z (axial) and XAxis radial (in-plane, perpendicular to Z) — so an add-in's ordinary
// (radius, axial) meridian drops onto that tilted plane unchanged. Without orientation the
// host-chosen frame is arbitrary (here X = −Z), which is what the feature fixes.
func TestSketchOrientationPinsFrameToAxis(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var wp wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"line-plane-angle","refs":["origin/axis/z","origin/plane/xz"],"angle":"30 deg"}`, &wp)
	if !wp.Healthy {
		t.Fatalf("angled plane not healthy: %+v", wp)
	}
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create",
		`{"workPlaneIndex":3,"orientation":{"axis":"origin/axis/z","axisIsX":false}}`, &sk)

	x, y, n := sketchPlaneAt(t, s, sk.SketchIndex)
	tol := math.Scalar(1e-9)
	if !y.IsEqualTo(math.V3(0, 0, 1).AsUnit(), tol) {
		t.Errorf("oriented sketch YAxis = %v, want +Z (axial)", y)
	}
	if zc := x.AsVector().Dot(math.V3(0, 0, 1)); stdmath.Abs(zc) > tol {
		t.Errorf("oriented sketch XAxis = %v has Z component %g, want radial (⟂ Z)", x, zc)
	}
	// The plane's facing (normal) must be preserved: still horizontal (no Z), rotated 30° about Z.
	if nz := n.AsVector().Dot(math.V3(0, 0, 1)); stdmath.Abs(nz) > tol {
		t.Errorf("oriented sketch normal = %v gained a Z component %g, want the plane facing preserved", n, nz)
	}
}

// TestSketchOrientationReverseFlipsAxis checks Reverse:true pins Y to −Z instead of +Z
// (Inventor NaturalAxisDirection=false).
func TestSketchOrientationReverseFlipsAxis(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var wp wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"line-plane-angle","refs":["origin/axis/z","origin/plane/xz"],"angle":"30 deg"}`, &wp)
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create",
		`{"workPlaneIndex":3,"orientation":{"axis":"origin/axis/z","axisIsX":false,"reverse":true}}`, &sk)
	_, y, _ := sketchPlaneAt(t, s, sk.SketchIndex)
	if !y.IsEqualTo(math.V3(0, 0, -1).AsUnit(), math.Scalar(1e-9)) {
		t.Errorf("reversed sketch YAxis = %v, want -Z", y)
	}
}

// TestSketchOrientationAxisIsXPinsX pins the projected axis as X instead of Y.
func TestSketchOrientationAxisIsXPinsX(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var wp wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"line-plane-angle","refs":["origin/axis/z","origin/plane/xz"],"angle":"30 deg"}`, &wp)
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create",
		`{"workPlaneIndex":3,"orientation":{"axis":"origin/axis/z","axisIsX":true}}`, &sk)
	x, _, _ := sketchPlaneAt(t, s, sk.SketchIndex)
	if !x.IsEqualTo(math.V3(0, 0, 1).AsUnit(), math.Scalar(1e-9)) {
		t.Errorf("axisIsX sketch XAxis = %v, want +Z", x)
	}
}

// TestSketchOrientationPerpendicularAxisErrors rejects an axis perpendicular to the plane:
// the Z axis is normal to the XY plane, so its in-plane projection is degenerate and the
// create must fail with a clear message rather than silently pick a frame.
func TestSketchOrientationPerpendicularAxisErrors(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	_, err := r.Handle(s, "sketch.create",
		[]byte(`{"plane":"XY","orientation":{"axis":"origin/axis/z","axisIsX":true}}`))
	if err == nil {
		t.Fatal("sketch.create with an axis perpendicular to the plane should error, got nil")
	}
}
