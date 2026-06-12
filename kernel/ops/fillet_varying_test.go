// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
)

// The variable-radius fillet (M09-F01 PBI-099, #323): the blend is a generalized
// cone built as exactly planar trapezoids between rulings, so the result's
// tessellated volume is closed-form in the chord count.

// varyingNotchFactor is the cross-section fraction of r² a chorded 90° fillet
// removes: the square corner r² minus the K-chord fan (K·r²/2·sin(π/2/K)),
// with K = filletChordsPerTurn/4 chords over the quarter turn.
func varyingNotchFactor(k int) float64 {
	return 1 - float64(k)/2*stdmath.Sin(stdmath.Pi/2/float64(k))
}

// TestFilletVaryingRadiusVolume rounds one vertical edge of a 2×2×2 box from
// r=0.3 at one end to r=0.6 at the other. The removed volume is the exact
// chord-geometry integral c·∫r(z)²dz = c·L·(r0²+r0·r1+r1²)/3 — the strip faces
// and chorded end fans are all planar, so the tessellation adds no error.
func TestFilletVaryingRadiusVolume(t *testing.T) {
	box := shellBox(2, 2, 2)
	pick := ops.EdgeFilletRadii{Key: verticalEdgeKey(t, box), R0: 0.3, R1: 0.6}
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{pick})
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("variable-filleted box not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != 0 {
		t.Errorf("variable fillet produced %d cylinder faces, want 0 (planar ruling strips)", n)
	}
	const k = 8 // ceil((π/2) / (2π/32)) chords across the 90° wedge
	removed := varyingNotchFactor(k) * 2 * (0.3*0.3 + 0.3*0.6 + 0.6*0.6) / 3
	want := 8 - removed
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("variable fillet volume = %g, want %g (exact chord geometry)", got, want)
	}
}

// TestFilletVaryingCollapsesToConstant: equal end radii through the varying API
// must reproduce the constant cylinder fillet exactly.
func TestFilletVaryingCollapsesToConstant(t *testing.T) {
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

// TestFilletVaryingStripsAreTrulyPlanar: every blend face's vertices lie on its
// reported plane (the exactness claim the volume test builds on).
func TestFilletVaryingStripsAreTrulyPlanar(t *testing.T) {
	box := shellBox(2, 2, 2)
	pick := ops.EdgeFilletRadii{Key: verticalEdgeKey(t, box), R0: 0.2, R1: 0.7}
	res, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{pick})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			t.Fatalf("non-planar face %T in a variable fillet result", f.Geometry())
		}
		for _, v := range f.Vertices() {
			if d := stdmath.Abs(float64(pl.Origin.VectorTo(v.Point()).Dot(pl.Normal()))); d > 1e-9 {
				t.Fatalf("face vertex %.12g off its plane by %g", v.Point(), d)
			}
		}
	}
}
