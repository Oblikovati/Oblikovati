// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane parallel to a torus axis, offset between the inner and outer tube radii, sections it in ONE
// oval delivered as TWO spiric branches. The branches MEET at the oval's v-extremes, and that meeting
// point is where |w| = 1 — the one place arccos is infinitely steep. Evaluated without care, each
// branch put the shared point somewhere else, ~3·10⁻⁸ apart in azimuth: the two arcs then failed to
// weld into one loop, the arrangement saw an open chain dividing nothing, and a torus cut by such a
// plane kept the whole face on both sides (ADR-0062).
func TestSpiricBranchesMeetExactlyAtTheOvalExtreme(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	// A SWEEP, not one plane: the loss is in the conditioning of arccos at its end, so it shows only
	// where w rounds short of 1 — which depends on every digit of the plane. The plane a half-space cut
	// actually presents is a face of a bounding prism, built from its corners, so its normal carries
	// exactly this kind of dust. A single tidy plane can pass while the real one fails.
	for _, off := range []float64{3.5, 4, 5, 6, 6.5} {
		for _, tilt := range []float64{0, 1e-16, 1e-13, 1e-9} {
			origin := math.P3(0, math.Scalar(off), 0)
			plane, err := geom.NewPlane(origin, math.V3(math.Scalar(tilt), 1, math.Scalar(tilt)))
			if err != nil {
				t.Fatalf("geom.NewPlane(off=%g tilt=%g): %v", off, tilt, err)
			}
			arcs, ok := geom.TorusPlaneSection(tor, plane)
			if !ok || len(arcs) != 2 {
				t.Fatalf("off=%g tilt=%g: want one oval in two branches, got ok=%v n=%d", off, tilt, ok, len(arcs))
			}
			a, b := arcs[0], arcs[1]
			aLo, aHi := a.Domain()
			bLo, bHi := b.Domain()
			for _, pair := range [][2]math.Point3{
				{a.PointAt(aLo), b.PointAt(bHi)},
				{a.PointAt(aHi), b.PointAt(bLo)},
			} {
				if d := float64(pair[0].DistanceTo(pair[1])); d > 1e-12 {
					t.Errorf("off=%g tilt=%g: the branches meet %g apart at an oval extreme (%v vs %v), want one point",
						off, tilt, d, pair[0], pair[1])
				}
			}
		}
	}
}

// The snap must not move a point that is not at a limit: a spiric evaluated in its interior is the
// exact section, and must stay on both the torus and the plane.
func TestSpiricStaysOnBothSurfaces(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	arcs, _ := geom.TorusPlaneSection(tor, plane)
	n := plane.Normal()
	for _, cv := range arcs {
		lo, hi := cv.Domain()
		for k := 0; k <= 32; k++ {
			p := cv.PointAt(lo + (hi-lo)*float64(k)/32)
			if d := stdmath.Abs(float64(plane.Origin.VectorTo(p).Dot(n))); d > 1e-12 {
				t.Fatalf("a section point is %g off the plane", d)
			}
			u, v := tor.ParamAt(p)
			if d := float64(p.DistanceTo(tor.PointAt(u, v))); d > 1e-12 {
				t.Fatalf("a section point is %g off the torus", d)
			}
		}
	}
}

// At the offset where the plane is exactly tangent to the tube, the two boundary roots of w(v) = ±1
// coincide and the span between them is a POINT — so both of its branch arcs are the tangency point
// repeated. They are not lobes: they bound nothing, and fed to a boolean as imprints they are closed
// curves of zero extent that every consumer samples into a ring of identical points. Measured before
// the fix, the section returned FOUR arcs where the figure-eight has two lobes, and the cut that used
// them took 30.89 s instead of 0.05 s (ADR-0061 stage 2).
func TestTorusPlaneSectionDropsTheDegenerateTangentArcs(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	// offset = R − r: the plane grazes the inner equator, where the two ovals meet.
	tangent, err := geom.NewPlane(math.P3(0, 3, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	secs, ok := geom.TorusPlaneSection(tor, tangent)
	if !ok {
		t.Fatal("the tangent section was refused")
	}
	if len(secs) != 2 {
		t.Errorf("the tangent section returned %d curves, want 2 (one per lobe)", len(secs))
	}
	for i, cv := range secs {
		lo, hi := cv.Domain()
		reach := 0.0
		for k := 0; k <= 8; k++ {
			d := float64(cv.PointAt(lo).DistanceTo(cv.PointAt(lo + (hi-lo)*float64(k)/8)))
			if d > reach {
				reach = d
			}
		}
		if reach < 1 {
			t.Errorf("curve %d reaches only %g from its start — it is a point, not a lobe", i, reach)
		}
	}
	// One step off the tangency the section is two lobes and always was; the tangent case must match it.
	near, err := geom.NewPlane(math.P3(0, 3.01, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	if s2, _ := geom.TorusPlaneSection(tor, near); len(s2) != len(secs) {
		t.Errorf("the tangent section has %d curves and the near-tangent one %d; they must agree", len(secs), len(s2))
	}
}
