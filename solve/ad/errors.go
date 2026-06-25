// SPDX-License-Identifier: GPL-2.0-only

package ad

import "fmt"

// indexError reports a seed index outside the variable system's range — a wiring
// bug in a constraint (its Partials referenced a variable it did not declare).
type indexError struct {
	index, count int
}

func (e indexError) Error() string {
	return fmt.Sprintf("ad: variable index %d out of range for a %d-variable system", e.index, e.count)
}

func adIndexError(i, n int) error { return indexError{index: i, count: n} }
