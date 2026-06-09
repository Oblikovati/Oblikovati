// SPDX-License-Identifier: GPL-2.0-only

package seq

import "testing"

func TestNextIsStrictlyIncreasingAndNonZero(t *testing.T) {
	a := Next()
	b := Next()
	if a == 0 {
		t.Fatal("Next returned 0, which is reserved for the origin frame")
	}
	if b <= a {
		t.Fatalf("Next not increasing: a=%d b=%d", a, b)
	}
}

func TestBumpRaisesFloorButNeverLowersIt(t *testing.T) {
	high := Next() + 1000
	Bump(high)
	if got := Next(); got <= high {
		t.Fatalf("after Bump(%d), Next=%d; want > %d", high, got, high)
	}
	// A bump below the current floor must not rewind the clock.
	before := Next()
	Bump(1) // far below the floor
	if got := Next(); got <= before {
		t.Fatalf("Bump(1) rewound the clock: before=%d after=%d", before, got)
	}
}
