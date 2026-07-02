// SPDX-License-Identifier: GPL-2.0-only

package textbuf

import "fmt"

// lineRangeError builds the panic value for an out-of-range line index, naming the offending
// index and the valid bound per the project's exception-message rule.
func lineRangeError(i, count int) string {
	return fmt.Sprintf("textbuf: line index %d out of range [0,%d)", i, count)
}

// insertLines returns lines with mid spliced in at index at, without aliasing mid into the
// destination (the caller may reuse its slice). at is assumed in [0, len(lines)].
func insertLines(lines [][]rune, at int, mid [][]rune) [][]rune {
	out := make([][]rune, 0, len(lines)+len(mid))
	out = append(out, lines[:at]...)
	out = append(out, mid...)
	out = append(out, lines[at:]...)
	return out
}
