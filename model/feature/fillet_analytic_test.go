// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
)

func bodyTorusCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
}

// TestAnalyticFilletOfCylinderRimIsATorus is the #127 fix for fillet: filleting the rim of an
// extruded circle (an analytic cylinder) yields a TRUE toroidal fillet —
// one geom.Torus face on a valid watertight solid — rather than a faceted rolling-ball blend.
func TestAnalyticFilletOfCylinderRimIsATorus(t *testing.T) {
	const r, h, f = 5.0, 10.0, 2.0
	fs, rim := extrudedCylinderTopRim(t, r, h)
	pf := NewDressUpFeatures(fs).AddFillet([][]byte{rim}, func() float64 { return f })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("analytic fillet sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if vr := ops.Validate(body); !vr.Valid || !body.IsSolid() {
		t.Fatalf("filleted cylinder is not a valid solid: %+v", vr.Issues)
	}
	if n := bodyTorusCount(body); n != 1 {
		t.Fatalf("filleted rim has %d torus faces, want 1 (a true toroidal fillet, #127)", n)
	}
	if n := bodyConeCount(body); n != 0 {
		t.Errorf("fillet produced %d cone faces, want 0", n)
	}
	sqMoment := (r - f/2) * (f * f)
	qdMoment := ((r - f) + 4*f/(3*stdmath.Pi)) * (stdmath.Pi * f * f / 4)
	want := stdmath.Pi*r*r*h - 2*stdmath.Pi*(sqMoment-qdMoment)
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.03 {
		t.Errorf("filleted cylinder volume = %g, want ≈%g", got, want)
	}
}
