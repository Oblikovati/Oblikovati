// SPDX-License-Identifier: GPL-2.0-only

package editor

import "oblikovati.org/script/console/textbuf"

// openToClose / closeToOpen pair the bracket kinds the editor matches. Lua uses (), [] and {}.
var (
	openToClose = map[rune]rune{'(': ')', '[': ']', '{': '}'}
	closeToOpen = map[rune]rune{')': '(', ']': '[', '}': '{'}
)

// MatchingBracket returns the positions of a bracket pair to highlight when the caret is on or
// just after a bracket, scanning with nesting depth so nested pairs match correctly. ok is false
// when the caret is not by a bracket or the bracket is unbalanced. The result is unordered only
// in that `a` is whichever bracket the caret sits by and `b` is its partner.
func (m *Model) MatchingBracket() (a, b textbuf.Position, ok bool) {
	for _, p := range []textbuf.Position{m.sel.Caret, m.buf.Left(m.sel.Caret)} {
		r, has := m.buf.RuneAt(p)
		if !has {
			continue
		}
		if close, isOpen := openToClose[r]; isOpen {
			if q, found := m.scanBracket(p, r, close, true); found {
				return p, q, true
			}
		}
		if open, isClose := closeToOpen[r]; isClose {
			if q, found := m.scanBracket(p, open, r, false); found {
				return p, q, true
			}
		}
	}
	return textbuf.Position{}, textbuf.Position{}, false
}

// scanBracket walks from the bracket at `from` (counting it as depth 1) toward its partner,
// forward for an opener and backward for a closer, returning the partner's position. It treats
// every same-kind bracket as nesting; string/comment awareness is a future refinement.
func (m *Model) scanBracket(from textbuf.Position, open, close rune, forward bool) (textbuf.Position, bool) {
	depth := 0
	p := from
	for {
		if r, has := m.buf.RuneAt(p); has {
			depth += bracketDelta(r, open, close)
			if depth == 0 {
				return p, true
			}
		}
		next := m.step(p, forward)
		if next == p { // reached a document boundary without balancing
			return textbuf.Position{}, false
		}
		p = next
	}
}

// bracketDelta returns +1 for an opener, -1 for a closer, 0 otherwise — the nesting contribution
// of rune r regardless of scan direction (the caller seeds depth from the starting bracket).
func bracketDelta(r, open, close rune) int {
	switch r {
	case open:
		return 1
	case close:
		return -1
	default:
		return 0
	}
}

// step advances one rune forward or backward across line boundaries.
func (m *Model) step(p textbuf.Position, forward bool) textbuf.Position {
	if forward {
		return m.buf.Right(p)
	}
	return m.buf.Left(p)
}
