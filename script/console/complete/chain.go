// SPDX-License-Identifier: GPL-2.0-only

package complete

// parseChain walks left from the caret at rune column col over a dotted identifier chain and
// returns it left-to-right together with the column where the trailing (partial) segment began
// — the span a chosen candidate replaces. The trailing segment is the prefix being typed and
// may be empty (caret just after a '.'), in which case the engine lists everything under the
// preceding namespace. Examples on `oblikovati.sketch.rect`:
//
//	col at end      -> chain {"oblikovati","sketch","rect"}, start at the 'r'
//	col after a '.' -> chain {"oblikovati","sketch",""},     start at the caret
func parseChain(line []rune, col int) (chain []string, replaceStart int) {
	col = clampInt(col, 0, len(line))
	j := scanIdentLeft(line, col)
	chain = []string{string(line[j:col])}
	replaceStart = j
	for j > 0 && line[j-1] == '.' {
		dot := j - 1
		m := scanIdentLeft(line, dot)
		seg := string(line[m:dot])
		if seg == "" {
			break // a leading '.' or `..` concat — not a member chain, stop here
		}
		chain = append([]string{seg}, chain...)
		j = m
	}
	return chain, replaceStart
}

// scanIdentLeft returns the column of the start of the identifier run ending just before col
// (col itself is exclusive), or col when the rune left of col is not an identifier rune.
func scanIdentLeft(line []rune, col int) int {
	j := col
	for j > 0 && isIdentPart(line[j-1]) {
		j--
	}
	return j
}

// isIdentPart reports whether r can appear in a Lua identifier ([A-Za-z0-9_]). Kept local to
// the completion boundary detection (it is a parsing concern distinct from the lexer's token
// scanning).
func isIdentPart(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return r == '_'
	}
}

// clampInt constrains v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
