// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// FR4 — the corner-host retrim is oblique-runout-aware. These tests pin the weld-identity the FR3 review
// flagged as unguarded: the host-side corner rail must be the SAME curve OBJECT the arm face's bundle
// already built on that host (not a re-derived congruent curve), so an oblique arm's foot lands ON the
// host loop and the two sides can never drift apart. The D5 meridian arm (the DRAWEXE-pinned oblique
// torus arm, fillet_intersect_arm_capping_test.go) is the fixture.

// d5ObliqueBundle builds the D5 meridian arm's rail bundle through the real oblique far-runout engine, so
// bundle.hostA is the sphere-side host rail RE-TERMINATED on the runout foot (outer end on the host rim).
func d5ObliqueBundle(t *testing.T) (armRails, armRunout, geom.Sphere, geom.Torus) {
	t.Helper()
	tor, sphere, lonPlane, cap := d5MeridianArm(t)
	ef := d5EdgeFillet(t, tor, sphere, lonPlane)
	capFace := stubCapFace(t, cap)
	res := opstol.ForSize(300)
	h0 := contactArcRail(t, sphereContactCircleOf(t, sphere, tor, res), tor)
	h1 := contactArcRail(t, capContactCircleOf(t, lonPlane, tor, res), tor)
	h0p, h1p, run, ok, reason := obliqueRunout(ef, capFace, h0, h1, 10, res)
	if !ok {
		t.Fatalf("obliqueRunout declined the D5 oblique arm: %s", reason)
	}
	setback := endSeg{from: h0p.to, to: h1p.to} // stub setback rail t0→t1 (only hostA/hostB are read here)
	return closeArmRails(h0p, h1p, setback, run), run, sphere, tor
}

// TestArmContactRail_ConsumesObliqueBundleRail is the MANDATORY weld-identity witness. With the arm's
// bundle supplied (the production path), the corner retrim's contact rail on the sphere host IS the arm
// bundle's host rail — the SAME endSeg/curve value — and its outer end is the oblique runout FOOT, exact
// on the host rim. The mutation: re-deriving the rail via the OLD constructor (curvedHostArc, the pre-FR4
// retrim) lands the outer end at the full-arc P0, OFF the rim by the oblique gap — a DIFFERENT curve that
// fails the identity. This is the regression witness that the two weld sides stay one object.
func TestArmContactRail_ConsumesObliqueBundleRail(t *testing.T) {
	t.Parallel()
	bundle, run, sphere, tor := d5ObliqueBundle(t)
	res := opstol.ForSize(300)
	tol := res.Weld() * 10

	// The bundle branch reads only the rail, so host is nil here (bundleContactRail never dereferences it).
	ha := cornerHostArm{set: armSetback{arm: tor}, rail: bundle.hostA, hasRail: true}
	rail, outer, ok := armContactRail(nil, ha, bundle.hostA.to, math.Point3{}, nil, cornerWeld{}, res, tol)
	if !ok {
		t.Fatal("armContactRail declined the supplied oblique bundle rail")
	}
	// (1) OBJECT identity: the retrim rail IS the arm bundle's host rail (same value + same curve object).
	if rail != bundle.hostA || rail.curve != bundle.hostA.curve {
		t.Fatalf("retrim rail is not the SAME object as the arm bundle rail:\n rail  = %+v\n hostA = %+v", rail, bundle.hostA)
	}
	// (2) Its outer end is the oblique runout foot ON the host rim (exact, from the FR3 engine).
	if d := float64(outer.DistanceTo(run.feet[0])); d > 1e-9 {
		t.Fatalf("bundle rail outer %v is not the runout foot %v (off by %.2e)", outer, run.feet[0], d)
	}

	// (3) MUTATION: the OLD constructor (curvedHostArc) rebuilds the FULL 0→φ* arc, whose PointAt(0) sits
	// at the far-vertex azimuth — OFF the rim by the oblique gap, NOT the foot — so it fails the identity.
	w := cornerWeld{radius: 10, arms: []armSetback{{arm: tor, station: 1}}}
	mut, okMut := curvedHostArc(sphere, tor, w, res)
	if !okMut {
		t.Fatal("precondition: curvedHostArc must build the (mutant) full arc on the D5 sphere host")
	}
	if gap := float64(mut.PointAt(0).DistanceTo(run.feet[0])); gap < 0.5 {
		t.Fatalf("mutation witness too weak: curvedHostArc P0 %v is only %.3f from the foot %v — the old constructor must miss the rim by the oblique gap",
			mut.PointAt(0), gap, run.feet[0])
	}
}
