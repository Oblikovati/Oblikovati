// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The BODY-level half of the corpus row kernel/geom pins as TestTheSectionThatBeatTheSampler. It lives
// beside boolean_torus_skew_test.go's corpus and reuses its helpers; it is a file of its own because
// that file reached the 500-line limit (Oblikovati/Oblikovati#3515, review round 4, Minor-1).
// TestTheSectionThatBeatTheSamplerBuildsARightBody is the BODY-level half of the corpus row
// kernel/geom pins as TestTheSectionThatBeatTheSampler (Oblikovati/Oblikovati#3515, review round 2).
//
// The geom row asserts the section's arcs lie on the tool. This one asserts the thing a user would
// actually notice, and the thing the review found wrong: that the BODIES are right. The failure it
// guards was not a crash and not an invalid solid — it was a valid, closed, manifold body carrying many
// times the correct volume, with no diagnostic recorded, on a pair the wave base had refused by name.
// `ops.Validate` cannot see that, and neither can Requicha alone; only an oracle that does not share the
// kernel's geometry can.
//
// So the row is certified three ways: the analytic volumes must satisfy V(−) + V(∩) = V(ring), each must
// match an independent membership integral, and the operation must record NOTHING — a body this exact is
// not a degradation and must not be reported as one.
func TestTheSectionThatBeatTheSamplerBuildsARightBody(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~15s): `make test-corpus`")
	}
	t.Parallel()
	ring, rod := samplerBeatingPair(t)
	const ringR, ringr = 9.775632, 2.621941
	inRing := func(x, y, z float64) bool { d := stdmath.Hypot(x, y) - ringR; return d*d+z*z <= ringr*ringr }
	inRod := insideFiniteCylinder(samplerBeatingRodBase(), samplerBeatingRodAxis(), 6.285269, samplerBeatingRodLength)
	cut := samplerBeatingVolume(t, ops.Cut, ring, rod, inRing, inRod)
	lens := samplerBeatingVolume(t, ops.Intersect, ring, rod, inRing, inRod)
	assertRelative(t, "V(−) + V(∩)", cut+lens, 2*stdmath.Pi*stdmath.Pi*ringR*ringr*ringr, requichaTol)
}

// samplerBeatingVolume runs one operation on the row, asserts it is a valid closed manifold solid with an
// exact boundary and no diagnostic, checks it against the independent membership integral, and returns
// its analytic volume.
func samplerBeatingVolume(t *testing.T, op ops.PartFeatureOperation, ring, rod *topo.Body,
	inRing, inRod func(x, y, z float64) bool,
) float64 {
	t.Helper()
	var rec diag.Recorder
	res, err := ops.BooleanWithDiagnostics(op, ring, rod, &rec)
	if err != nil {
		t.Fatalf("%v: the pair the base refused must build now: %v", op, err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("%v: not a valid closed manifold solid: %+v", op, r)
	}
	if n := rec.Count(diag.Defect); n != 0 {
		t.Errorf("%v: an exact body recorded %d defect(s): %v", op, n, rec.Records())
	}
	box := ring.RangeBox().Union(rod.RangeBox())
	if op == ops.Intersect {
		box = overlapBox(ring.RangeBox(), rod.RangeBox())
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	assertRelative(t, op.String()+" against the membership integral", got,
		skewMembershipVolume(op, inRing, inRod, box), membershipTol)
	return got
}

// samplerBeatingPair is the ring and rod of the row. Its section carries four full-period branches, and
// at tube angle 2.953097 its station's quartic loses two of its four roots to the chart's pole — which is
// why the wave base refused the pair and why rounds 1 and 2 built it with an arc 26 units off the rod.
func samplerBeatingPair(t *testing.T) (ring, rod *topo.Body) {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 9.775632, 2.621941, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	rod, err = brep.SolidCylinder(samplerBeatingRodBase(), samplerBeatingRodAxis(), 6.285269, samplerBeatingRodLength)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	return ring, rod
}

// samplerBeatingRodAxis and samplerBeatingRodBase place the rod so it crosses the ring skew, both caps
// clear of it. The axis is the one the random sweep drew; the base is walked back half the length along
// it so the row reads the same way the sweep's infinite cylinder did.
func samplerBeatingRodAxis() math.Vector3 {
	return math.V3(-0.9065753377189599, 0.33242730045605945, -0.2600254736583529)
}

func samplerBeatingRodBase() math.Point3 {
	unit, _ := math.UnitVector3FromVector(samplerBeatingRodAxis())
	return math.P3(3.360585805574098, -1.7574864466724531, 0.3390622878733773).
		TranslateBy(unit.AsVector().Scale(-samplerBeatingRodLength / 2))
}

// samplerBeatingRodLength is long enough that both caps stand clear of the ring's reach (its farthest
// point is 12.4 from the axis), so the only contact in the row is the one under test.
const samplerBeatingRodLength = 60.0
