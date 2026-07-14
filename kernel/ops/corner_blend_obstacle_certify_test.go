// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestNoFoldOverColumnsMatchesObstacle proves the extracted sweep reproduces obstacleNoFold's
// verdict on the folding + non-folding fixtures (behavior-preserving refactor).
func TestNoFoldOverColumnsMatchesObstacle(t *testing.T) {
	of := newT6Obstacle(t)
	g, ok := obstaclePatchNeighbours(of)
	if !ok {
		t.Fatal("obstaclePatchNeighbours declined the T6 fixture")
	}
	sides := obstacleSides(of, g.wingL, g.wingR, g.wall)
	fill, err := geom.FillSurface(g.c0, g.c1, g.d0, g.d1, sides)
	if err != nil {
		t.Fatalf("FillSurface: %v", err)
	}
	fill, err = pinFillBoundary(fill, g.c0, g.c1, g.d0, g.d1)
	if err != nil {
		t.Fatalf("pinFillBoundary: %v", err)
	}
	v0, v1 := fill.VDomain()
	if noFoldOverColumns(fill, v0, v1, blendScale()) != obstacleNoFold(fill, blendScale()) {
		t.Fatal("noFoldOverColumns disagrees with obstacleNoFold on T6")
	}
}

// TestLoopRibLenMatchesValence proves the unified rib length equals the old per-valence helpers.
func TestLoopRibLenMatchesValence(t *testing.T) {
	q := quarterCylLoop(t, 4)
	if loopRibLen(q) != coons4RibLenLegacyForTest(q) {
		t.Fatalf("loopRibLen(valence4) = %g, want legacy value", loopRibLen(q))
	}
}

// coons4RibLenLegacyForTest coops the pre-refactor coons4RibLen formula so the test pins equality
// (named helper, deleted with the test once the refactor is proven).
func coons4RibLenLegacyForTest(loop RailLoop) float64 {
	a, b, c, d := loopCorners(loop)
	return ResolutionForPoints([]math.Point3{a, b, c, d}).Size() * ribbonSpanFactor
}
