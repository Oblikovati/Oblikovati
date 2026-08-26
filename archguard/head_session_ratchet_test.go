// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Audit I5: head/ui is god-coupled to the concrete *app.Session — every widget takes the
// whole session, so every widget can touch anything, and grep cannot tell load-bearing
// uses from incidental ones. The policy (architecture/notes/head-ui-consumer-interfaces.md)
// is boy-scout, not big-bang: any widget touched from now on declares its own consumer-side
// interface of ≤6 methods (the arrowSession pattern) instead of taking *app.Session. This
// ratchet pins the current coupling count so a new *app.Session parameter fails CI; when a
// conversion lowers it, the pin is lowered with it — the number only goes down.

// headSessionParamPin is the number of *app.Session references in head/ui code (excluding
// the compile-time interface assertions that PROVE a consumer interface, and doc comments
// that merely mention the type). Lower it when a conversion removes references; never raise
// it — a raise means a new widget took the whole session instead of a slim interface.
const headSessionParamPin = 436

func TestHeadSessionCouplingRatchet(t *testing.T) {
	got := countSessionCoupling(t)
	if got > headSessionParamPin {
		t.Fatalf("head/ui *app.Session coupling rose to %d (pinned %d): a widget took the whole "+
			"*app.Session — declare a ≤6-method consumer interface instead (audit I5, arrowSession pattern)",
			got, headSessionParamPin)
	}
	if got < headSessionParamPin {
		t.Fatalf("head/ui *app.Session coupling fell to %d (pinned %d): good — lower headSessionParamPin "+
			"to %d so the ratchet holds the new floor", got, headSessionParamPin, got)
	}
}

// countSessionCoupling counts *app.Session occurrences in head/ui code, skipping full-line
// doc comments (prose mentions of the type) and the `var _ Iface = (*app.Session)(nil)`
// assertions that are the seam's proof, not its coupling.
func countSessionCoupling(t *testing.T) int {
	t.Helper()
	files, err := filepath.Glob("../head/ui/*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("globbing head/ui: %v (found %d)", err, len(files))
	}
	total := 0
	for _, src := range readGoSources(t, "../head/ui/*.go") {
		total += countSessionInSource(src)
	}
	return total
}

// toolDialogSignature matches a tool property dialog's declaration. Its parameter type is fixed
// by the toolDialogDraw callback the registry stores, not by the dialog's own appetite for the
// session — so every NEW tool dialog raised the ratchet by one and the rule "never raise it"
// made shipping a new feature's dialog impossible without an unrelated conversion to pay for it
// (hit while adding the Unwrap, Angle Plane and Delete Body dialogs, #2047/#2044/#2046).
//
// Excluding the signature line keeps the guard's actual intent — no WIDGET may take the whole
// session — while letting the registry's own contract through. Everything a dialog then does
// with the session still counts.
var toolDialogSignature = regexp.MustCompile(`^func draw[A-Z]\w*Dialogs?\(s \*app\.Session\)`)

// countSessionInSource counts *app.Session in one file's source under the ratchet rules.
func countSessionInSource(src string) int {
	n := 0
	for line := range strings.SplitSeq(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.Contains(line, "(*app.Session)(nil)") {
			continue
		}
		if toolDialogSignature.MatchString(trimmed) {
			continue // the registry's callback contract, not the widget's choice
		}
		n += strings.Count(line, "*app.Session")
	}
	return n
}
