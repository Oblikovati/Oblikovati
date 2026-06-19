// SPDX-License-Identifier: GPL-2.0-only

// Package diag is the Script Console editor's live syntax-diagnostics layer. It runs a Lua
// syntax check (compile-only, never executing the script) a short while after typing stops, so
// the editor can underline errors without re-parsing on every keystroke. The actual parse is
// injected as a Checker, so this package stays pure and is unit-tested with a fake checker; the
// gopher-lua-backed implementation lives next to the engine that owns that dependency.
package diag

import "time"

// Diagnostic is one reported problem: a 0-based Line/Col (matching textbuf Positions) and a
// human-readable message. Col is the column the error was detected at.
type Diagnostic struct {
	Line    int
	Col     int
	Message string
}

// Checker parses a source and returns its syntax diagnostics (empty when it is valid).
type Checker interface {
	Check(source string) []Diagnostic
}

// Analyzer debounces checking: it watches the source via Observe and only re-runs the Checker
// once the text has been unchanged for the debounce interval, caching the result. This keeps a
// parse off the hot typing path while still updating promptly when the user pauses.
type Analyzer struct {
	checker   Checker
	debounce  time.Duration
	lastSrc   string
	changedAt time.Time
	checked   bool
	diags     []Diagnostic
}

// NewAnalyzer returns an analyzer that re-checks debounce after the last change.
func NewAnalyzer(checker Checker, debounce time.Duration) *Analyzer {
	return &Analyzer{checker: checker, debounce: debounce, checked: true}
}

// Observe records the current source at time now. When the text changed it resets the debounce
// timer; when it has been stable for the debounce interval it runs the check once and caches the
// diagnostics. Call it every frame with the editor's current text and clock.
func (a *Analyzer) Observe(source string, now time.Time) {
	if source != a.lastSrc {
		a.lastSrc = source
		a.changedAt = now
		a.checked = false
		return
	}
	if !a.checked && now.Sub(a.changedAt) >= a.debounce {
		a.diags = a.checker.Check(source)
		a.checked = true
	}
}

// Diagnostics returns the most recently computed diagnostics (nil until the first check settles).
func (a *Analyzer) Diagnostics() []Diagnostic { return a.diags }
