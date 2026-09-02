// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// padClosure sums a cage's analytic face terms: the outward vector area a CLOSED shell owes the
// divergence theorem (zero, whatever the shape), the volume it bounds, and its area.
func padClosure(t *testing.T, b *topo.Body) (residual, volume, area float64) {
	t.Helper()
	var ax, ay, az float64
	for _, f := range b.Faces() {
		ft, ok := query.FaceTerms(f)
		if !ok {
			t.Fatalf("a wrapped pad face declined analytic integration: %T with %d loops",
				f.Geometry(), len(f.Loops()))
		}
		ax, ay, az = ax+ft.Ax, ay+ft.Ay, az+ft.Az
		volume, area = volume+ft.Vol, area+ft.Area
	}
	return stdmath.Sqrt(ax*ax + ay*ay + az*az), volume, area
}

// conePadFixture builds one wrapped pad on the chamfer cone the emboss corpus uses: a 1x1 cm profile
// at slant 21, raised 0.3 cm. Its segments WARP over the cone, so the cage takes the split-triangle
// path this test exists for.
func conePadFixture(t *testing.T) *topo.Body {
	t.Helper()
	fr, plane := coneWrapFixture(t, 0.5, 21)
	inner, outer, err := fr.offsets(0.3, false)
	if err != nil {
		t.Fatalf("offsets: %v", err)
	}
	poly := []math.Point2{math.P2(-0.5, -0.5), math.P2(0.5, -0.5), math.P2(0.5, 0.5), math.P2(-0.5, 0.5)}
	pad, err := wrappedSkin(poly, plane, fr, inner, outer, "pad")
	if err != nil {
		t.Fatalf("wrappedSkin: %v", err)
	}
	return pad
}

// TestWrappedPadCageClosesItsVectorArea pins Oblikovati/Oblikovati#3503: addWarpedWall named each
// split triangle's plane in its own loop's order, which is the REVERSE of the sense the planar-quad
// wall and the two caps carry, so the four split faces ended up with a material side opposite every
// other face of the pad.
//
// Nothing downstream could see it. ops.Validate checks loop TRAVERSAL, which stayed manifold, and the
// mesh reads a face's orientation off its loop rather than off the stored flag — so the cage passed
// as a valid solid and tessellated correctly while its analytic faces disagreed about which side held
// material. The vector-area closure is what catches it: the residual was 7.1e-3 of the pad's area and
// the volume read 0.5438 cm³ against a true 0.3735.
func TestWrappedPadCageClosesItsVectorArea(t *testing.T) {
	t.Parallel()
	pad := conePadFixture(t)
	if r := ops.Validate(pad); !r.Valid || !pad.IsSolid() {
		t.Fatalf("the wrapped pad is not a valid solid: %+v", r.Issues)
	}
	residual, volume, area := padClosure(t, pad)

	// Two error scales meet here and the bounds sit between them. The cage's boundary is still
	// CHORDED — its loop edges are straight segments between points of a cone, so they leave the
	// surface by the resampling sagitta — which leaves an irreducible residual around 9e-5 of the area
	// and puts the analytic volume about 0.6% off the mesh. The orientation defect was three orders
	// louder: 0.206 of the area, and a volume of 4.930 cm³ against a true 0.352. Closing the remaining
	// gap needs exact section edges on the caps, which the mixed boolean cannot yet consume (#3503).
	if rel := residual / area; rel > 1e-3 {
		t.Errorf("the pad's outward vector area misses closing by %g of its area (residual %g on %g); "+
			"a closed shell owes the divergence theorem zero, so some face's material side is flipped",
			rel, residual, area)
	}
	mesh := query.BodyGeometryProperties(pad, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(volume-mesh) / mesh; rel > 2e-2 {
		t.Errorf("the pad integrates to %g cm³ analytically but %g meshed (rel %g); the two measure the "+
			"same cage, so they disagree only when a face is counted with the wrong sign", volume, mesh, rel)
	}
}

// TestWrappedPadOnAConeSplitsWarpedWalls keeps the fixture honest: the closure test above proves
// nothing about the split-triangle path unless that path is actually taken.
func TestWrappedPadOnAConeSplitsWarpedWalls(t *testing.T) {
	t.Parallel()
	pad := conePadFixture(t)
	if n := len(pad.Faces()); n <= 6 {
		t.Errorf("the pad has %d faces; a 4-segment profile whose walls all stayed planar would have 6 "+
			"(2 caps + 4 quads), so this fixture no longer exercises addWarpedWall", n)
	}
}
