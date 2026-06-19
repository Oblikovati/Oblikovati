// SPDX-License-Identifier: GPL-2.0-only

package textbuf

import "strings"

// Advance returns the Position reached by walking text forward from p without touching a
// buffer: every "\n" drops to the next line at column 0 and any other run advances the
// column by its rune count. It is how undo/redo computes the end of a recorded span (the text
// that was removed or inserted) so it can be replaced in reverse. Example:
//
//	Advance(Position{1, 2}, "ab\ncd") == Position{2, 2}
func Advance(p Position, text string) Position {
	nl := strings.Count(text, "\n")
	if nl == 0 {
		return Position{p.Line, p.Col + len([]rune(text))}
	}
	lastSeg := text[strings.LastIndexByte(text, '\n')+1:]
	return Position{p.Line + nl, len([]rune(lastSeg))}
}
