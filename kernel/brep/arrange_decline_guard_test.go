// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"strings"
	"testing"
)

// The scanner behind TestEveryArrangingSplitReportsANonConvergentArrangement, proven on in-memory
// sources so the proof needs no edit to a production file (review round 4). Each call form a new
// site would plausibly use is parsed three ways: guarded by a real recordArrangementDecline call,
// unguarded, and "guarded" only by a comment that names the function — which the AST scan must
// NOT accept, since the round-3 text scan did.

// arrangingCallForms are the shapes a trimByImprint / splitFace call can take in a function body.
// The round-3 regex accepted only the first.
var arrangingCallForms = map[string]string{
	"define":       "faces, _, err := trimByImprint(c, wf, s, imp, m)\n\t_ = faces\n\treturn err",
	"return":       "return trimByImprint(c, wf, s, imp, m)",
	"selector":     "p.faces, _, err = trimByImprint(c, wf, s, imp, m)\n\treturn err",
	"nested-call":  "out = append(out, splitFace(f, imp))\n\treturn out",
	"nested-split": "pieces, converged := splitFace(f, imp)\n\t_ = converged\n\treturn pieces",
}

// snippetSource wraps one call form in a function, with the given statement before it.
func snippetSource(before, call string) []byte {
	return []byte("package brep\n\nfunc site(rec *recorder) error {\n\t" + before + "\n\t" + call + "\n}\n")
}

func TestTheGuardFlagsEveryUnguardedCallForm(t *testing.T) {
	t.Parallel()
	for name, form := range arrangingCallForms {
		src := snippetSource("prepare()", form)
		unguarded, total := unguardedArrangingCalls(t, name+".go", src)
		if total != 1 {
			t.Errorf("%s: scanner saw %d arranging calls, want 1", name, total)
		}
		if len(unguarded) != 1 || !strings.HasPrefix(unguarded[0], name+".go:") {
			t.Errorf("%s: an unguarded %q was not flagged (got %v)", name, form, unguarded)
		}
	}
}

func TestTheGuardAcceptsEveryCallFormWithARealDecline(t *testing.T) {
	t.Parallel()
	for name, form := range arrangingCallForms {
		src := snippetSource("if err := prepare(); err != nil {\n\t\trecordArrangementDecline(rec, siteWallTrim, err)\n\t}", form)
		unguarded, total := unguardedArrangingCalls(t, name+".go", src)
		if total != 1 || len(unguarded) != 0 {
			t.Errorf("%s: a call beside a recordArrangementDecline call was flagged: %v (total %d)", name, unguarded, total)
		}
	}
}

// A comment that names the decline is not a decline. The round-3 substring scan accepted it.
func TestTheGuardIgnoresADeclineThatIsOnlyAComment(t *testing.T) {
	t.Parallel()
	for name, form := range arrangingCallForms {
		src := snippetSource("// recordArrangementDecline(rec, siteWallTrim, err) is the caller's job", form)
		unguarded, _ := unguardedArrangingCalls(t, name+".go", src)
		if len(unguarded) != 1 {
			t.Errorf("%s: a comment naming recordArrangementDecline satisfied the guard (got %v)", name, unguarded)
		}
	}
}

// A function without a recorder declines by RETURNING the named refusal — the public imprint entry's
// shape — and that counts; a bare `err` returned from the call, or an unconvergedArrangement call
// whose result is assigned and dropped, does not.
func TestTheGuardAcceptsAReturnedUnconvergedArrangementOnly(t *testing.T) {
	t.Parallel()
	returned := snippetSource("if !ready() {\n\t\treturn unconvergedArrangement(len(imp))\n\t}", arrangingCallForms["define"])
	if unguarded, _ := unguardedArrangingCalls(t, "returned.go", returned); len(unguarded) != 0 {
		t.Errorf("a returned unconvergedArrangement was flagged: %v", unguarded)
	}
	dropped := snippetSource("_ = unconvergedArrangement(len(imp))", arrangingCallForms["define"])
	if unguarded, _ := unguardedArrangingCalls(t, "dropped.go", dropped); len(unguarded) != 1 {
		t.Errorf("an unconvergedArrangement built and dropped passed as a decline: %v", unguarded)
	}
}

// The decline must be in the SAME function body: one in a sibling function does not cover a call.
func TestTheGuardScopesTheDeclineToTheFunctionBody(t *testing.T) {
	t.Parallel()
	src := []byte("package brep\n\nfunc other(rec *recorder, err error) { recordArrangementDecline(rec, siteWallTrim, err) }\n\n" +
		"func site() error {\n\tfaces, _, err := trimByImprint(c, wf, s, imp, m)\n\t_ = faces\n\treturn err\n}\n")
	unguarded, total := unguardedArrangingCalls(t, "sibling.go", src)
	if total != 1 || len(unguarded) != 1 {
		t.Errorf("a decline in a sibling function covered the call: unguarded=%v total=%d", unguarded, total)
	}
}

// A method or a function literal called through a selector is not the package function.
func TestTheGuardMatchesOnlyThePlainCallee(t *testing.T) {
	t.Parallel()
	src := snippetSource("prepare()", "faces, _, err := c.trimByImprint(wf, s, imp, m)\n\t_ = faces\n\treturn err")
	if _, total := unguardedArrangingCalls(t, "method.go", src); total != 0 {
		t.Errorf("a method named trimByImprint was counted as the package function (total %d)", total)
	}
}
