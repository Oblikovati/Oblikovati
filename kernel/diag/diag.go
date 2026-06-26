// SPDX-License-Identifier: GPL-2.0-only

// Package diag is the kernel's diagnostic-collection channel: the way a geometry operation reports,
// instead of silently swallowing, that it could not do the ideal thing — a cap it saturated, an exact
// path it declined, a fallback it took. A [Recorder] threaded into an operation collects the typed
// [Diagnostic]s it emits, so callers and tests can SEE what degraded and WHY rather than discovering it
// later when a downstream feature or export breaks (Oblikovati/Oblikovati#1407, #1412).
//
// The channel is deliberately tiny: pass a *Recorder into an operation (or nil to discard), and emit
// with rec.Record(...). Record is safe on a nil receiver, so an emission site never branches on whether
// anyone is listening, and the Recorder is safe for the concurrent face tessellation the mesher runs.
package diag

import (
	"fmt"
	"sync"
)

// Code is a stable, greppable diagnostic identifier — e.g. "boolean.csg-fallback",
// "tessellate.cap-saturated". Use a dotted "domain.what" form so a category greps as a prefix.
type Code string

// Severity ranks a diagnostic. Defect is the load-bearing one: a tracked degradation (a CSG fallback, a
// saturated cap) that produced a possibly-wrong result and should ultimately be eliminated, not an
// expected outcome — the discipline that turns "exact or silent soup" into a visible, countable signal.
type Severity int

const (
	// Info is a benign note (a path was taken, nothing degraded).
	Info Severity = iota
	// Warning is a result that is probably fine but worth surfacing (a sampled tolerance near its limit).
	Warning
	// Defect is a degraded/fallback result that is a tracked defect: it should not happen and is counted.
	Defect
)

// String renders the severity for messages and logs.
func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Defect:
		return "defect"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// Diagnostic is one structured report: a typed [Code] for searching/counting, a [Severity], and a human
// Detail that — per the CLAUDE.md exception-message rule — names the offending value or configuration
// (which operands, which face, which cap value), so the reader can reproduce it without guessing.
type Diagnostic struct {
	Code     Code
	Severity Severity
	Detail   string
}

// String renders the diagnostic as "severity code: detail", the form used in logs and test failures.
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s %s: %s", d.Severity, d.Code, d.Detail)
}

// Recorder collects the diagnostics emitted during one operation. The zero value is ready to use, and a
// nil *Recorder is a valid sink that discards — so a caller that does not care passes nil and every
// emission site stays branch-free. It is safe for concurrent use (the mesher records from parallel face
// goroutines into one shared recorder).
type Recorder struct {
	mu      sync.Mutex
	records []Diagnostic
}

// Record appends d. It is a no-op on a nil receiver, so emission sites call rec.Record(...) whether or
// not the caller supplied a recorder.
func (r *Recorder) Record(d Diagnostic) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, d)
}

// Recordf is Record with a formatted Detail, the common emission shape.
func (r *Recorder) Recordf(code Code, sev Severity, format string, args ...any) {
	if r == nil {
		return
	}
	r.Record(Diagnostic{Code: code, Severity: sev, Detail: fmt.Sprintf(format, args...)})
}

// Records returns a copy of the collected diagnostics in emission order (a copy, so the caller cannot
// mutate the recorder's slice). nil receiver returns nil.
func (r *Recorder) Records() []Diagnostic {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Diagnostic(nil), r.records...)
}

// Count returns how many recorded diagnostics have the given severity — e.g. Count(diag.Defect) is the
// tracked-defect count a test asserts is zero on a clean run.
func (r *Recorder) Count(sev Severity) int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, d := range r.records {
		if d.Severity == sev {
			n++
		}
	}
	return n
}

// Has reports whether any recorded diagnostic carries code — the searchable assertion a handler test
// uses to confirm its decline emitted a diagnostic rather than a silent result.
func (r *Recorder) Has(code Code) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range r.records {
		if d.Code == code {
			return true
		}
	}
	return false
}
