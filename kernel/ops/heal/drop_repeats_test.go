// SPDX-License-Identifier: GPL-2.0-only

package heal_test

import (
	"testing"

	"oblikovati.org/kernel/ops/internal/retopo"
)

func TestDropRepeats(t *testing.T) {
	t.Parallel()
	got := retopo.DropRepeats([]int{1, 1, 2, 3, 1})
	if len(got) != 3 { // consecutive dup removed + closing 1 dropped
		t.Fatalf("dropRepeats = %v, want 3 unique", got)
	}
}
