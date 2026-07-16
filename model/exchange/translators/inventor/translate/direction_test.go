// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
)

// TestDirectionOfCombinesDirWithReversed pins the rule that `reversed` is meaningless alone: it
// flips the DirectionAxis's OWN vector, so only the two together say which way the extrude grows.
//
// The "reversed ⇒ NegativeDir" reading looks obvious and is wrong. Measured on the 175-part corpus
// it broke parts that were already exact (the actuator screw 1.01x → 1.16x) for a net loss, because
// a reversed extrude usually names an ALREADY-negative dir that nets back to positive — see
// ipt.TestReversedPairsWithAnAlreadyFlippedDir, where all three reversed extrudes carry dir = -Z.
func TestDirectionOfCombinesDirWithReversed(t *testing.T) {
	up, down := [3]float64{0, 0, 1}, [3]float64{0, 0, -1}
	cases := []struct {
		name string
		ex   ipt.Extrude
		want feature.ExtentDirection
	}{
		{"plain +Z", ipt.Extrude{Dir: up, DirOK: true}, feature.PositiveDir},
		{"plain -Z grows negative", ipt.Extrude{Dir: down, DirOK: true}, feature.NegativeDir},
		{"reversed -Z nets back to positive", ipt.Extrude{Dir: down, DirOK: true, Reversed: true}, feature.PositiveDir},
		{"reversed +Z grows negative", ipt.Extrude{Dir: up, DirOK: true, Reversed: true}, feature.NegativeDir},
		{"midplane straddles regardless", ipt.Extrude{Dir: down, DirOK: true, Reversed: true, Midplane: true}, feature.SymmetricDir},
		{"no direction stated keeps the default", ipt.Extrude{Reversed: true}, feature.PositiveDir},
	}
	for _, c := range cases {
		if got := directionOf(c.ex); got != c.want {
			t.Errorf("%s: directionOf = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestExtentOfKeepsTwoSidedExtrudePositive pins that a two-sided extrude's second length IS its
// negative side, so the extent must stay PositiveDir — pairing Distance2 with NegativeDir would
// grow both spans the same way and double one side.
func TestExtentOfKeepsTwoSidedExtrudePositive(t *testing.T) {
	e := extentOf(ipt.Extrude{
		Distance: 3, Distance2: 2, Dir: [3]float64{0, 0, -1}, DirOK: true,
	})
	if e.Direction != feature.PositiveDir {
		t.Errorf("two-sided extent direction = %v, want PositiveDir", e.Direction)
	}
	if e.Distance2 == nil || e.Distance2() != 2 {
		t.Errorf("two-sided extent lost its second distance")
	}
	if e.Distance == nil || e.Distance() != 3 {
		t.Errorf("two-sided extent lost its primary distance")
	}
}
