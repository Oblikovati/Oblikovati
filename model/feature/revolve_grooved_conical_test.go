// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// housingMeridian builds the 532xx self-aligning HOUSING washer meridian (cm, radius=x, axial=y): a
// grooved ball-facing front (inner land, a groove arc cradling the ball, outer land) closed by a bore
// edge, an OD edge, and a CONICAL back (a straight oblique edge, OD-high to bore-low). These are the
// solved coordinates the live add-in produces. When backHi==backLo the back is FLAT (the 511xx shaft
// washer) — the control. The mixed line/arc loop is exactly the shape whose region-extractor Reversed
// flags were inconsistent, mis-placing the groove arc and collapsing the revolved volume (#54).
func housingMeridian(backHi, backLo float64) *sketch.Sketch {
	const bore, od, pit, land = 1.5, 2.35, 1.925, 0.1836
	w := 0.1874 // half the groove-arc chord in x (pit±w are the groove/land junctions)
	s := sketch.NewSketches().Add(sketch.XYPlane())
	pBoreBack := s.Points().Add(math.P2(bore, backLo))
	pBoreFront := s.Points().Add(math.P2(bore, land))
	pGrooveIn := s.Points().Add(math.P2(pit-w, land))
	pGrooveOut := s.Points().Add(math.P2(pit+w, land))
	pOdFront := s.Points().Add(math.P2(od, land))
	pOdBack := s.Points().Add(math.P2(od, backHi))
	center := s.Points().Add(math.P2(pit, 0))
	s.Lines().Add(pBoreBack, pBoreFront)              // bore edge
	s.Lines().Add(pBoreFront, pGrooveIn)              // inner land
	s.Arcs().Add(center, pGrooveOut, pGrooveIn, true) // groove arc (apex toward +y, into the washer)
	s.Lines().Add(pGrooveOut, pOdFront)               // outer land
	s.Lines().Add(pOdFront, pOdBack)                  // OD edge
	s.Lines().Add(pOdBack, pBoreBack)                 // back edge (conical when backHi≠backLo, else flat)
	return s
}

// pappusVolume is the true solid-of-revolution volume of a meridian profile by Pappus: V = 2π·A·R̄,
// with the profile's area A and centroid radius R̄ from the loop polygon (shoelace). An independent
// oracle for the revolved mesh volume — insensitive to how the kernel tessellates, so a mesh whose
// groove torus was built on the wrong span (inverted/cancelling triangles) fails against it.
func pappusVolume(p *sketch.Profile) float64 {
	poly := p.OuterLoop().Polygon()
	var a2, cx float64
	for i := range poly {
		j := (i + 1) % len(poly)
		cross := float64(poly[i].X*poly[j].Y - poly[j].X*poly[i].Y)
		a2 += cross
		cx += (float64(poly[i].X) + float64(poly[j].X)) * cross
	}
	return 2 * stdmath.Pi * (stdmath.Abs(a2) / 2) * stdmath.Abs(cx/3/a2)
}

func revolvedGroovedWasherVolumes(t *testing.T, sk *sketch.Sketch) (mesh, pappus float64) {
	t.Helper()
	fs := NewPartFeatures(nil)
	NewRevolveFeatures(fs).Add(sk, 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("revolve produced %d bodies, want 1", len(res))
	}
	prof, err := resolveSingleProfile(sk, 0, "test")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	return ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume, pappusVolume(prof)
}

// TestGroovedFlatBackWasherVolume is the CONTROL: the 511xx grooved shaft washer (FLAT back) revolves
// to a mesh whose volume matches Pappus.
func TestGroovedFlatBackWasherVolume(t *testing.T) {
	t.Parallel()
	mesh, pappus := revolvedGroovedWasherVolumes(t, housingMeridian(0.55, 0.55))
	if relErr(mesh, pappus) > 0.02 {
		t.Fatalf("flat-back grooved washer mesh volume %.4f vs Pappus %.4f (%.1f%%)", mesh, pappus, relErr(mesh, pappus)*100)
	}
}

// TestGroovedConicalBackWasherVolume guards the #54 self-aligning HOUSING fix: the grooved washer with
// a CONICAL back must revolve to a mesh whose volume matches Pappus. Before the fix meridianVertsFromProfile
// trusted the loop's Reversed flags — which the region extractor set inconsistently for this mixed
// line/arc loop — so the groove torus was built on the inner-land span (minor radius 0.463 instead of
// 0.262) and the mesh volume collapsed to ~28% of true. Chaining by shared endpoints holds it exact.
func TestGroovedConicalBackWasherVolume(t *testing.T) {
	t.Parallel()
	mesh, pappus := revolvedGroovedWasherVolumes(t, housingMeridian(0.55, 0.3819))
	if relErr(mesh, pappus) > 0.02 {
		t.Fatalf("conical-back grooved washer mesh volume %.4f vs Pappus %.4f (%.1f%%) — the groove torus "+
			"was built on the wrong span", mesh, pappus, relErr(mesh, pappus)*100)
	}
}
