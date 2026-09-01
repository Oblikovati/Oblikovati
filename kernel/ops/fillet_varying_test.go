// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The variable-radius fillet (M09-F01 PBI-099, #323; exact since #1606): the blend is the
// EXACT rational ruled surface between the end cross-sections — an oblique circular cone for a
// linear taper — so the removed volume follows the smooth-blend integral, and the only error in
// the measured value is the tessellator's chordal deviation on the curved faces.

// smoothNotchFactor is the cross-section fraction of r² a smooth 90° fillet removes: the square
// corner r² minus the quarter disc.
func smoothNotchFactor() float64 { return 1 - stdmath.Pi/4 }

// TestFilletVaryingRadiusVolume rounds one vertical edge of a 2×2×2 box from
// r=0.3 at one end to r=0.6 at the other. The removed volume is the exact
// chord-geometry integral c·∫r(z)²dz = c·L·(r0²+r0·r1+r1²)/3 — the strip faces
// and chorded end fans are all planar, so the tessellation adds no error.
func TestFilletVaryingRadiusVolume(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	pick := ops.EdgeFilletRadii{Key: verticalEdgeKey(t, box), R0: 0.3, R1: 0.6}
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{pick})
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("variable-filleted box not a valid solid: %+v", r)
	}
	if n := blendSurfaceFaces(res); n != 1 {
		t.Errorf("variable fillet produced %d rational blend faces, want 1 (the exact oblique cone, #1606)", n)
	}
	removed := smoothNotchFactor() * 2 * (0.3*0.3 + 0.3*0.6 + 0.6*0.6) / 3
	want := 8 - removed
	// Measure at fine quality: the exact blend CONVERGES to the smooth integral as the chord
	// tolerance tightens — the discriminating property the old C0 strips could never satisfy
	// (they converge to the chord integral regardless of tessellation).
	if got := ops.BodyGeometryProperties(res, fineQuality()).Volume; stdmath.Abs(got-want) > 0.03*removed {
		t.Errorf("variable fillet volume = %g, want %g (smooth-blend integral, 3%%-of-notch band)", got, want)
	}
}

// fineQuality is the tight tessellation the smooth-volume assertions measure at.
func fineQuality() ops.Quality {
	return ops.Quality{ChordTolerance: 0.002, AngleTolerance: 2 * stdmath.Pi / 180}
}

// blendSurfaceFaces counts the body's rational blend faces (the exact variable-fillet surface).
func blendSurfaceFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.BSplineSurface); ok {
			n++
		}
	}
	return n
}

// TestFilletIntermediateRadiiVolume rounds one vertical edge of a 2×2×2 box with a
// piecewise-linear radius profile: 0.3 at the start, 0.7 at the mid (T=0.5), 0.4 at
// the end (#695). The removed volume is the per-segment chord integral summed — every
// ruling strip is still planar, so the tessellation adds no error.
func TestFilletIntermediateRadiiVolume(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	pick := ops.EdgeFilletRadii{
		Key: verticalEdgeKey(t, box), R0: 0.3, R1: 0.4,
		Mids: []ops.FilletRadiusPoint{{T: 0.5, R: 0.7}},
	}
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{pick})
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("intermediate-radii box not a valid solid: %+v", r)
	}
	if n := blendSurfaceFaces(res); n != 2 {
		t.Errorf("intermediate-radii fillet produced %d rational blend faces, want 2 (one exact span per radius segment, #1606)", n)
	}
	seg := func(ra, rb, length float64) float64 { return length * (ra*ra + ra*rb + rb*rb) / 3 }
	removed := smoothNotchFactor() * (seg(0.3, 0.7, 1.0) + seg(0.7, 0.4, 1.0))
	want := 8 - removed
	if got := ops.BodyGeometryProperties(res, fineQuality()).Volume; stdmath.Abs(got-want) > 0.03*removed {
		t.Errorf("intermediate-radii fillet volume = %g, want %g (smooth-blend integral, 3%%-of-notch band)", got, want)
	}
}

// TestFilletIntermediateRadiiValidation rejects out-of-range and non-increasing points (#695).
func TestFilletIntermediateRadiiValidation(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	key := verticalEdgeKey(t, box)
	cases := map[string][]ops.FilletRadiusPoint{
		"must be strictly between 0 and 1": {{T: 0, R: 0.5}},
		"radius":                           {{T: 0.5, R: 0}},
		"strictly increasing in T":         {{T: 0.6, R: 0.4}, {T: 0.6, R: 0.5}},
	}
	for want, mids := range cases {
		_, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{{Key: key, R0: 0.3, R1: 0.4, Mids: mids}})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("mids %+v: err = %v, want %q", mids, err, want)
		}
	}
}

// TestFilletVaryingCollapsesToConstant: equal end radii through the varying API
// must reproduce the constant cylinder fillet exactly.
func TestFilletVaryingCollapsesToConstant(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	key := verticalEdgeKey(t, box)
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{{Key: key, R0: 0.5, R1: 0.5}})
	if err != nil {
		t.Fatal(err)
	}
	if n := hasCylinderFaces(res); n != 1 {
		t.Errorf("equal-radii fillet has %d cylinder faces, want 1 (constant path)", n)
	}
}

// TestFilletVaryingAtSharedCornerRejected: a variable edge meeting another
// filleted edge at a corner has no watertight blend — a precise error, not a
// broken body.
func TestFilletVaryingAtSharedCornerRejected(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	vert := verticalEdgeKey(t, box)
	v, _ := box.FindEdgeByKey(vert)
	ends := map[uint64]bool{v.StartVertex().ID(): true, v.EndVertex().ID(): true}
	var neighbour []byte
	for _, e := range box.Edges() {
		if e != v && (ends[e.StartVertex().ID()] || ends[e.EndVertex().ID()]) {
			neighbour = e.ReferenceKey()
			break
		}
	}
	_, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{
		{Key: vert, R0: 0.2, R1: 0.5},
		{Key: neighbour, R0: 0.2, R1: 0.2},
	})
	if err == nil || !strings.Contains(err.Error(), "variable-radius edge") {
		t.Fatalf("err = %v, want the variable-at-shared-corner rejection", err)
	}
}

// TestFilletVaryingBlendIsExactAndG1 (premise inverted by #1606, audit A10): the variable
// blend used to ship as C0 planar strips with ~11° creases; it is now the EXACT rational ruled
// surface. Assert every boundary vertex lies exactly on the blend surface and the surface is
// G1 across its interior (machine-precision normal continuity — it is one analytic patch).
func TestFilletVaryingBlendIsExactAndG1(t *testing.T) {
	t.Parallel()
	box := shellBox(2, 2, 2)
	pick := ops.EdgeFilletRadii{Key: verticalEdgeKey(t, box), R0: 0.2, R1: 0.7}
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{pick})
	if err != nil {
		t.Fatal(err)
	}
	blends := 0
	for _, f := range res.Faces() {
		surf, ok := f.Geometry().(geom.BSplineSurface)
		if !ok {
			continue
		}
		blends++
		for _, v := range f.Vertices() {
			u, vv := surf.ParamAt(v.Point())
			if d := float64(surf.PointAt(u, vv).DistanceTo(v.Point())); d > 1e-9 {
				t.Fatalf("blend boundary vertex %.12g off the exact surface by %g", v.Point(), d)
			}
		}
		for i := 1; i < 8; i++ {
			u := float64(i) / 8
			if dot := surf.NormalAt(u-1e-9, 0.5).Dot(surf.NormalAt(u+1e-9, 0.5)); dot < 1-1e-9 {
				t.Fatalf("blend normal creases at u=%g: cos=%.12f (the old strips creased every ~11°)", u, dot)
			}
		}
	}
	if blends != 1 {
		t.Fatalf("variable fillet has %d rational blend faces, want 1", blends)
	}
}
