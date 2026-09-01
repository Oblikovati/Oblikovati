// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The rotate arm of the move-face direct edit (M09-F04 PBI-107, #331).

// faceWithNormal returns a face whose outward plane normal matches dir.
func faceWithNormal(t *testing.T, b *topo.Body, dir math.Vector3) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok && float64(pl.Normal().Dot(dir)) > 0.9 {
			return f
		}
	}
	t.Fatal("no face with the requested normal")
	return nil
}

// TestRotateFacesTiltsTopIntoWedge rotates a 2×2×2 box's top face by θ about
// the y-axis line along its x=0 top edge: the top plane becomes z = 2 − x·tanθ
// and the volume the analytic wedge 8 − 4·tanθ.
func TestRotateFacesTiltsTopIntoWedge(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	top := faceWithNormal(t, box, math.V3(0, 0, 1))
	const theta = 0.15
	axisDir, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	res, err := ops.RotateFaces(box, [][]byte{top.ReferenceKey()}, math.P3(0, 0, 2), axisDir, theta)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("rotated-face box not a valid solid: %+v", r)
	}
	want := 8 - 4*stdmath.Tan(theta)
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("wedge volume = %g, want %g", got, want)
	}
}

// TestRotateFacesLostKey: a missing face key errors with the key.
func TestRotateFacesLostKey(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	axisDir, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	if _, err := ops.RotateFaces(box, [][]byte{[]byte("gone")}, math.P3(0, 0, 2), axisDir, 0.1); err == nil {
		t.Fatal("a lost face key must error")
	}
}
