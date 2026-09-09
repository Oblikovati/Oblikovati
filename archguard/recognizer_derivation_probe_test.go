// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The probe for the recognizer derivation (Oblikovati/Oblikovati#3522).
//
// The derivation's number is the "recognizers" ratchet, and a later slice drives it down by deleting
// arms. So "the walk sees every recognizer" has to be a MEASURED claim, not a hope: the first cut of
// the walk resolved a callee only when it was a bare identifier, and a verdict only when it was spelled
// `bool` or `(<something>Trim, bool)`. Both readings were calibrated to the code in front of them.
// A recognizer written in any other shape moved the count by zero, and no test in this package failed.
//
// Every shape the widened walk claims to see is therefore PLANTED here, as real source in a package of
// its own, and the derivation is run over it. A shape nobody plants is a shape nobody tested — which is
// the same bargain TestTheOwnershipGuardBitesEveryConstructionForm strikes for #3520's ownership guard.
//
// Each row is measured against the CONTROL, whose classification reads nothing and derives zero. Every
// row was run through BOTH walks. Seven of the eight positive rows derived nothing under the first cut
// and derive exactly one name now — those are the blind shapes. The eighth,
// preservedVerdictShapes, derived its name under both and is planted to prove the widening did not lose
// it. The three negative rows are the boundary the widening must NOT cross: a chart a recognizer uses is
// not a verdict, a call into a package outside the classification's tree is not a local call of the same
// name, and a duplicate name is refused rather than guessed.

// probeImportPath is what the planted classification is imported as, so its own subpackage can be told
// apart from `math` in exactly the way the kernel's tree is told apart from geom.
const probeImportPath = "example.test/probe"

// probeRoot is the planted classification: the same shape the kernel's has — a verdict struct whose
// FIELDS are the payload set, and a classifyCurvedTrim that reads recognizers. The first %s is the row's
// body, the second its own declarations.
const probeRoot = `package tessellate

import (
	stdmath "math"

	"example.test/probe/trim"
)

type curvedTrimKind int
type coneApexTrim struct{}
type sphereCapRecognition struct{}
type sphereChart struct{}
type verdictBool bool
type prober struct{}

// curvedTrim is the verdict, and its fields are the payload set the derivation reads. Note that
// sphereChart is NOT among them: a recognizer may use a chart, but no arm's recognition is one.
type curvedTrim struct {
	kind  curvedTrimKind
	cone  coneApexTrim
	cap   sphereCapRecognition
	moved trim.MovedTrim
}

func classifyCurvedTrim() curvedTrim {
%s
	return curvedTrim{}
}

%s
`

// probeSub is a helper package of the planted tree — where a recognizer moved out of the classification
// file, but not out of its tree, would live.
const probeSub = `package trim

type MovedTrim struct{}

%s
`

// TestTheDerivationSeesEveryCalleeAndVerdictShape plants each shape the first cut was blind to and
// checks the derived set, so "the walk sees it" is measured against a control that derives zero.
func TestTheDerivationSeesEveryCalleeAndVerdictShape(t *testing.T) {
	t.Parallel()
	for _, row := range recognizerProbeRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			got := derivedRecognizersIn(t, probeTree(t, row.body, row.decls, row.sub))
			if strings.Join(got, ",") != strings.Join(row.want, ",") {
				t.Errorf("planted %q: the derivation reads %v, want %v — %s", row.name, got, row.want, row.why)
			}
		})
	}
}

// recognizerProbeRow is one planted classification and the recognizers it must derive.
type recognizerProbeRow struct {
	name, body, decls, sub, why string
	want                        []string
}

// recognizerProbeRows are the shapes the derivation claims to see, and the three it must not.
func recognizerProbeRows() []recognizerProbeRow {
	rows := []recognizerProbeRow{{
		name: "control: a classification that reads nothing",
		why:  "the control must derive zero, or every other row's number means nothing",
	}}
	rows = append(rows, blindCalleeShapes()...)
	rows = append(rows, blindVerdictShapes()...)
	rows = append(rows, preservedVerdictShapes()...)
	return append(rows, walkBoundaryShapes()...)
}

// blindCalleeShapes are the calls the first cut could not resolve, because plainCallee accepted only a
// bare identifier: a METHOD, and a call into a package of the classification's own tree.
func blindCalleeShapes() []recognizerProbeRow {
	return []recognizerProbeRow{{
		name:  "a recognizer called as a method",
		body:  "\tp := prober{}\n\tif c, ok := p.coneApexTrimOf(); ok {\n\t\treturn curvedTrim{cone: c}\n\t}",
		decls: "func (p prober) coneApexTrimOf() (coneApexTrim, bool) { return coneApexTrim{}, true }",
		want:  []string{"coneApexTrimOf"},
		why:   "`x.f(…)` is a call like any other; refusing it lets an arm hide a recognizer behind a receiver",
	}, {
		name: "a boolean gate moved into a package of the tree",
		body: "\tif trim.MovedGateHolds() {\n\t\treturn curvedTrim{}\n\t}",
		sub:  "func MovedGateHolds() bool { return true }",
		want: []string{"trim.MovedGateHolds"},
		why:  "moving a gate one directory down is not deleting a shape, so the count must not fall for it",
	}, {
		name: "a payload recognizer moved into a package of the tree",
		body: "\tif m, ok := trim.MovedTrimOf(); ok {\n\t\treturn curvedTrim{moved: m}\n\t}",
		sub:  "func MovedTrimOf() (MovedTrim, bool) { return MovedTrim{}, true }",
		want: []string{"trim.MovedTrimOf"},
		why:  "the verdict struct carries trim.MovedTrim, so trim.MovedTrim IS one of the payloads",
	}}
}

// blindVerdictShapes are the verdicts the first cut could not read, because isVerdictFunc knew two
// literal spellings rather than a result-type set.
func blindVerdictShapes() []recognizerProbeRow {
	return []recognizerProbeRow{{
		name:  "a payload renamed off the \"Trim\" suffix",
		body:  "\tif c, ok := sphereCapRecognitionOf(); ok {\n\t\treturn curvedTrim{cap: c}\n\t}",
		decls: "func sphereCapRecognitionOf() (sphereCapRecognition, bool) { return sphereCapRecognition{}, true }",
		want:  []string{"sphereCapRecognitionOf"},
		why:   "a payload is what the verdict struct CARRIES, not what its name ends in",
	}, {
		name:  "a payload returned by pointer",
		body:  "\tif c, ok := conePointerTrimOf(); ok {\n\t\treturn curvedTrim{cone: *c}\n\t}",
		decls: "func conePointerTrimOf() (*coneApexTrim, bool) { return nil, false }",
		want:  []string{"conePointerTrimOf"},
		why:   "*coneApexTrim hands back the same recognition coneApexTrim does",
	}, {
		name:  "a verdict with a third result",
		body:  "\tif c, k, ok := coneAndKindOf(); ok {\n\t\treturn curvedTrim{cone: c, kind: k}\n\t}",
		decls: "func coneAndKindOf() (coneApexTrim, curvedTrimKind, bool) { return coneApexTrim{}, 0, true }",
		want:  []string{"coneAndKindOf"},
		why:   "the two-spellings reading accepted exactly two results, so a third hid the recognizer",
	}, {
		name:  "a verdict returned as a named bool",
		body:  "\tif namedBoolGateHolds() {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func namedBoolGateHolds() verdictBool { return true }",
		want:  []string{"namedBoolGateHolds"},
		why:   "`type verdictBool bool` is a bool; only its spelling differs",
	}}
}

// preservedVerdictShapes is the one row here that is NOT a blind shape the widening fixed: the first
// cut already accepted `(found, ok bool)`, though by accident — it counted result FIELDS, saw a single
// field of type bool, and read it as the bare `bool` spelling. The result-type set reads it as two
// results and reaches the same answer for the right reason. It is planted because the reading changed
// underneath it, so "unchanged" is a measured claim rather than an assumption.
func preservedVerdictShapes() []recognizerProbeRow {
	return []recognizerProbeRow{{
		name:  "a verdict whose two results are declared in one field",
		body:  "\tif _, ok := foundAndOK(); ok {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func foundAndOK() (found, ok bool) { return true, true }",
		want:  []string{"foundAndOK"},
		why:   "results are counted one per RESULT, not one per field; `(found, ok bool)` is two",
	}}
}

// walkBoundaryShapes are the edges the widening must NOT cross. They are the reason the walk stops at a
// geometric helper instead of following it into the mesher, and the reason a library predicate is not
// counted as a bespoke recognizer.
func walkBoundaryShapes() []recognizerProbeRow {
	return []recognizerProbeRow{{
		name:  "a chart a recognizer USES is not a verdict",
		body:  "\tif _, ok := chooseSphereChart(); ok {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func chooseSphereChart() (sphereChart, bool) { return sphereChart{}, true }",
		why: "sphereChart is no field of the verdict struct, so it is no payload — this is what keeps " +
			"the walk out of the mesher, and widening to \"anything with a trailing bool\" would break it",
	}, {
		name:  "a call into a package OUTSIDE the tree keeps its own identity",
		body:  "\tif stdmath.Signbit(-1) {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func Signbit(f float64) bool { return f < 0 }",
		why: "resolving `stdmath.Signbit` by its bare selector would count the local Signbit as a " +
			"recognizer the classification never reads",
	}}
}

// TestTheDerivationRefusesAnAmbiguousVerdictName plants the one shape the index cannot resolve — two
// declarations answering to one key, either of which could be the verdict a read reaches — and the one
// that looks like it but is not: an ordinary String() on two types.
func TestTheDerivationRefusesAnAmbiguousVerdictName(t *testing.T) {
	t.Parallel()
	twoVerdicts := "func dup() bool { return true }\n\nfunc (p prober) dup() bool { return true }"
	names := probeTree(t, "", twoVerdicts, "").ambiguousVerdictNames()
	if len(names) != 1 || !strings.HasPrefix(names[0], "dup ") {
		t.Errorf("a package function and a method both named dup, both verdict-shaped, are reported as "+
			"%v; the derivation resolves a read by NAME, so that pair must be refused, not guessed", names)
	}
	twoStrings := "func (k curvedTrimKind) String() string { return \"\" }\n\n" +
		"func (v verdictBool) String() string { return \"\" }"
	if names := probeTree(t, "", twoStrings, "").ambiguousVerdictNames(); len(names) != 0 {
		t.Errorf("String() on two types is reported as ambiguous (%v); neither is a verdict, so neither "+
			"can be a read, and ordinary Go must stay legal", names)
	}
}

// probeTree writes one planted classification and its helper package into a temp directory and returns
// the index over them. It writes files rather than parsing strings because the tree's SHAPE — a root
// package and a package beneath it, resolved through real import paths — is half of what is measured.
func probeTree(t *testing.T, body, decls, sub string) *recognizerIndex {
	t.Helper()
	root := t.TempDir()
	writeProbeFile(t, filepath.Join(root, "classify.go"), fmt.Sprintf(probeRoot, body, decls))
	writeProbeFile(t, filepath.Join(root, "trim", "moved.go"), fmt.Sprintf(probeSub, sub))
	return newRecognizerIndex(t, root, probeImportPath)
}

// writeProbeFile writes one planted source file.
func writeProbeFile(t *testing.T, path, src string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
