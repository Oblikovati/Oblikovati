// SPDX-License-Identifier: GPL-2.0-only

// Package cmdline implements the Command Window's headless engine (M26): a rolling
// message scrollback, an AutoCAD-style command/alias vocabulary, and an input parser.
// It is pure Go — no UI, no cgo, no dependency on the app package — so the whole
// command-line behaviour is unit-tested; the head renders a view of it and the public
// API drives it. See architecture/mapping/autocad-command-map.md for the vocabulary.
package cmdline

// Severity classifies a scrollback line so the head can colour it and callers can filter
// it. Echo is the user's own input mirrored back (AutoCAD echoes typed commands); Prompt
// is an active command's step prompt ("Specify first point").
type Severity int

const (
	Info Severity = iota
	Echo
	Prompt
	Warning
	Error
)

// Line is one entry in the rolling history: the text plus its severity.
type Line struct {
	Text     string
	Severity Severity
}

// Scrollback is a bounded ring of output lines plus the recallable command history. The
// ring caps memory over a long session (oldest lines drop first); the command history is
// the list of submitted command lines that ↑/↓ recall walks.
//
//	sb := NewScrollback(500)
//	sb.Append("Specify first point", Prompt)
//	sb.RecordCommand("LINE")
type Scrollback struct {
	lines   []Line
	max     int
	history []string
}

// defaultMaxLines is the ring size used when NewScrollback is given a non-positive max.
const defaultMaxLines = 1000

// NewScrollback returns a scrollback holding at most max output lines (≤0 ⇒ a sensible
// default), so a long-running session never grows the buffer without bound.
func NewScrollback(max int) *Scrollback {
	if max <= 0 {
		max = defaultMaxLines
	}
	return &Scrollback{max: max}
}

// Append adds one output line, dropping the oldest when the ring is full.
func (s *Scrollback) Append(text string, sev Severity) {
	s.lines = append(s.lines, Line{Text: text, Severity: sev})
	if len(s.lines) > s.max {
		s.lines = s.lines[len(s.lines)-s.max:]
	}
}

// Lines returns a copy of the current output lines, oldest first.
func (s *Scrollback) Lines() []Line {
	out := make([]Line, len(s.lines))
	copy(out, s.lines)
	return out
}

// Len reports how many output lines are currently held.
func (s *Scrollback) Len() int { return len(s.lines) }

// Clear discards the output lines but keeps the command history (matching a shell's
// "clear", which leaves ↑ recall intact).
func (s *Scrollback) Clear() { s.lines = nil }

// RecordCommand appends a submitted command line to the recall history, collapsing an
// immediate duplicate of the last entry (a shell does not stack identical repeats).
func (s *Scrollback) RecordCommand(cmd string) {
	if cmd == "" {
		return
	}
	if n := len(s.history); n > 0 && s.history[n-1] == cmd {
		return
	}
	s.history = append(s.history, cmd)
}

// History returns a copy of the recall history, oldest first.
func (s *Scrollback) History() []string {
	out := make([]string, len(s.history))
	copy(out, s.history)
	return out
}

// LastCommand returns the most recently recorded command and true, or "" and false when
// the history is empty — the source for AutoCAD's "press Enter to repeat".
func (s *Scrollback) LastCommand() (string, bool) {
	if len(s.history) == 0 {
		return "", false
	}
	return s.history[len(s.history)-1], true
}
