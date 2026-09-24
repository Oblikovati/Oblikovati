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
// Each row is measured against the CONTROL, whose classification reads nothing and derives zero.
//
// PROVENANCE of the "was it blind before?" column: the first cut is `fff94140`, and its walk is
// recoverable as `git show fff94140:archguard/recognizer_derivation_test.go` — the whole derivation
// lived in that one file, so it can be run over these same planted trees by anyone re-checking the
// claim. Under it, every row of blindCalleeShapes, blindVerdictShapes and blindGenericShapes derives
// NOTHING; preservedVerdictShapes and preservedGenericShapes derive their name under both walks and are
// planted to prove the widening did not lose what it already had.
//
// The boundary rows are what the widening must NOT cross: a chart a recognizer uses is not a verdict, a
// call into a package outside the classification's tree is not a local call of the same name, a method
// on a foreign receiver is not an in-tree method of the same name, and a duplicate name is refused
// rather than guessed.

// probeImportPath is what the planted classification is imported as, so its own subpackage can be told
// apart from `math` in exactly the way the kernel's tree is told apart from geom.
const probeImportPath = "example.test/probe"

// probeRoot is the planted classification: the same shape the kernel's has — a verdict struct whose
// FIELDS are the payload set, and a classifyCurvedTrim that reads recognizers. The first %s is the row's
// body, the second its own declarations.
const probeRoot = `package tessellate

import (
	stdmath "math"
	stdtime "time"

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
	for _, group := range []func() []recognizerProbeRow{blindCalleeShapes, blindVerdictShapes,
		blindGenericShapes, preservedVerdictShapes, preservedGenericShapes, walkBoundaryShapes} {
		rows = append(rows, group()...)
	}
	return rows
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
		name: "a recognizer called through a FIELD of a stated receiver",
		body: "\tb := coverProbe{}\n\tif b.inner.holdsShape() {\n\t\treturn curvedTrim{}\n\t}",
		decls: "type coverProbe struct{ inner prober }\n\n" +
			"func (p prober) holdsShape() bool { return true }",
		want: []string{"holdsShape"},
		why: "`b.r.covers(u, v)` is how the chart mesher is written, and the index has already parsed " +
			"the struct that declares r. Refusing to resolve a field made seven such call sites in real " +
			"kernel source unattributable (review N1), and the remedy the refusal prescribed was to " +
			"rewrite clean code to appease a guard",
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

// blindGenericShapes is the callee shape #3536's precedent demanded be either handled or named: a
// generic recognizer called with its type argument WRITTEN OUT. `f[T](…)` is an *ast.IndexExpr and
// `f[T1, T2](…)` an *ast.IndexListExpr, and the first cut resolved neither — while seeing the very same
// recognizer when the type argument was inferred (preservedGenericShapes). A blindness that depends on
// how the caller spells the call is exactly the calibrated-to-today's-code reading this task removes.
func blindGenericShapes() []recognizerProbeRow {
	return []recognizerProbeRow{{
		name:  "a generic recognizer called with an explicit type argument",
		body:  "\tif explicitGateHolds[int]() {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func explicitGateHolds[T any]() bool { return true }",
		want:  []string{"explicitGateHolds"},
		why:   "`f[T](…)` instantiates f; the instantiation is not a different function",
	}, {
		name:  "a generic recognizer called with two explicit type arguments",
		body:  "\tif pairGateHolds[int, string]() {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func pairGateHolds[A any, B any]() bool { return true }",
		want:  []string{"pairGateHolds"},
		why:   "*ast.IndexListExpr is the two-argument spelling of the same thing",
	}}
}

// preservedGenericShapes is the counterpart the blindness was measured against: the SAME recognizer with
// its type argument inferred was always visible, which is what made the explicit spelling a blind spot
// rather than a decision about generics.
func preservedGenericShapes() []recognizerProbeRow {
	return []recognizerProbeRow{{
		name:  "a generic recognizer with its type argument inferred",
		body:  "\tif inferredGateHolds(1) {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func inferredGateHolds[T any](v T) bool { return true }",
		want:  []string{"inferredGateHolds"},
		why:   "an inferred call is a bare identifier and was never blind",
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
	}, {
		name:  "a method on a receiver from OUTSIDE the tree keeps its own identity",
		body:  "\tvar when stdtime.Time\n\tif when.IsZero() {\n\t\treturn curvedTrim{}\n\t}",
		decls: "func IsZero() bool { return false }",
		why: "this row guards a regression this task itself introduced and then removed: claiming every " +
			"bare selector as a method let `when.IsZero()` answer to the local IsZero and INFLATE the " +
			"pin by a recognizer the classification never reads (review I1). An inflated base is worse " +
			"than a missed recognizer, because the fall measured from it looks real",
	}, {
		name:  "a payload returned as a SLICE is not resolved",
		body:  "\tif c, ok := manyConeApexTrimsOf(); ok {\n\t\treturn curvedTrim{cone: c[0]}\n\t}",
		decls: "func manyConeApexTrimsOf() ([]coneApexTrim, bool) { return nil, false }",
		why: "a documented limit, planted so it is a known one (review M4). Unwrapping a slice to its " +
			"element would also turn `func f() []bool` into a verdict. MEASURED over the whole index: " +
			"the unwrap makes 4 existing declarations verdicts (ConsistentOutwardFlips, floodInside, " +
			"frustratedFaces, and seamEndMask, which returns ([]bool, bool)) and catches 0 recognizers, " +
			"because no ([]payload, bool) declaration exists in the tree. Inflation 4, catch 0",
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

// TestTheDerivationRefusesAMethodCallItCannotAttribute plants the residue of the fix for review I1:
// a method call whose receiver comes back from a CALL, so the AST states no type for it, and whose name
// reaches a verdict of the tree. The index cannot tell that read from a call on a foreign value of the
// same method name, and both guesses move the pin — so it is refused, not guessed.
func TestTheDerivationRefusesAMethodCallItCannotAttribute(t *testing.T) {
	t.Parallel()
	call := "\tp := makeProber()\n\tif p.holdsShape() {\n\t\treturn curvedTrim{}\n\t}"
	decls := "func makeProber() prober { return prober{} }\n\nfunc (p prober) holdsShape() bool { return true }"
	unstated := probeTree(t, call, decls, "")
	sites := unresolvableReads(unstated, reachableFromClassification(t, unstated))
	if len(sites) != 1 || !strings.HasPrefix(sites[0], "holdsShape at ") {
		t.Errorf("a receiver whose type the AST does not state, calling a verdict-shaped in-tree name, "+
			"is reported as %v; it must be refused with its site named", sites)
	}
	assertAStatedReceiverIsAttributed(t, decls)
}

// assertAStatedReceiverIsAttributed is the other half of the refusal: a receiver the AST DOES state is
// read normally. A guard that refused both would forbid the method-call recognizer this task exists to
// see.
func assertAStatedReceiverIsAttributed(t *testing.T, decls string) {
	t.Helper()
	stated := probeTree(t, "\tp := prober{}\n\tif p.holdsShape() {\n\t\treturn curvedTrim{}\n\t}", decls, "")
	if sites := unresolvableReads(stated, reachableFromClassification(t, stated)); len(sites) != 0 {
		t.Errorf("`p := prober{}` states p's type, so p.holdsShape() is attributable; it was refused: %v", sites)
	}
	if got := derivedRecognizersIn(t, stated); strings.Join(got, ",") != "holdsShape" {
		t.Errorf("a method on a stated in-tree receiver derives %v, want [holdsShape]", got)
	}
}

// TestATreeImportIsNamedByItsPackageNotItsDirectory plants review M7: a package of the tree whose
// DIRECTORY and PACKAGE names differ, imported without an alias. The importer writes the PACKAGE name,
// so an index that guessed the local name from the last path element would resolve the call to nothing
// and drop the recognizer behind it — a silent fall in the pin, in the one path the real tree cannot
// exercise because it has no subpackages today.
func TestATreeImportIsNamedByItsPackageNotItsDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeProbeFile(t, filepath.Join(root, "classify.go"), renamedDirRoot)
	writeProbeFile(t, filepath.Join(root, "trim", "moved.go"), renamedDirSub)
	got := derivedRecognizersIn(t, newRecognizerIndex(t, root, probeImportPath))
	if strings.Join(got, ",") != "trimpkg.MovedGateHolds" {
		t.Errorf("a tree package in directory \"trim\" declaring `package trimpkg`, imported without an "+
			"alias, derives %v; want [trimpkg.MovedGateHolds] — the local name of an unaliased import is "+
			"the PACKAGE name, never the directory", got)
	}
}

// renamedDirSub is a package in a directory called "trim" that declares itself `package trimpkg`.
const renamedDirSub = `package trimpkg

func MovedGateHolds() bool { return true }
`

// renamedDirRoot is a classification importing a tree package whose directory and package names differ.
const renamedDirRoot = `package tessellate

import "example.test/probe/trim"

type curvedTrimKind int

type curvedTrim struct {
	kind curvedTrimKind
}

func classifyCurvedTrim() curvedTrim {
	if trimpkg.MovedGateHolds() {
		return curvedTrim{}
	}
	return curvedTrim{}
}
`

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
