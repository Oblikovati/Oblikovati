// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// "Every kernel PR reports the net change in recognizers, tolerance constants, fallback sites,
// and type assertions. The net is ≤ 0 unless an ADR says why." (kernel ground rules.)
//
// Nothing counted any of the four (#2184), so "net ≤ 0" was an intention rather than a fact.
// This is the reporting layer: four numbers, pinned, and a failure on ANY change — up OR down.
// A rise is not forbidden, it is made DELIBERATE: you edit the pin, and the diff to the pin is
// the report the rule asks for. A fall is caught too, because a floor nobody lowers stops being
// a floor.
//
// Three of the four are already measured by a dedicated ratchet in this package, and this test
// reads those rather than counting again — one mechanism per quantity:
//
//	tolerance constants  toleranceDebt      (TestNoUnjustifiedAbsoluteEpsilons, #2189)
//	type assertions      geomSwitchDebt     (TestGeometryKindSwitchesLiveInGeom, #2188)
//	recognizers          the dispatch tables TestNoFirstFitDispatchLadders pins (#2186)
//	fallback sites       diag.Code declarations under kernel/
//
// Fallback sites are counted as declared diag.Code kinds because the rules require a degradation
// to BE one: "Never degrade silently. A fallback, approximation, or dropped element is a
// diag.Defect that reaches feature health, the API, and the UI." That makes the count honest in
// a way a grep for the word "fallback" would not be — but it also means a rise can be an
// IMPROVEMENT, when a previously-silent degradation is finally reported. That is exactly why
// this test fails on any change instead of only on a rise: the number is a prompt to explain,
// not a limit to obey.

// kernelNetDeltaPin is the checked-in baseline. Update it in the same commit as the change that
// moves it, and say in the PR which direction each moved and why.
var kernelNetDeltaPin = map[string]int{
	"tolerance-constants": 233,
	"type-assertions":     774,
	"recognizers":         37, // 26 curvedExactPaths + 11 specialCurvedMeshers
	"fallback-sites":      28,
}

func TestKernelNetDelta(t *testing.T) {
	t.Parallel()
	got := map[string]int{
		"tolerance-constants": sumInts(toleranceDebt),
		"type-assertions":     sumInts(geomSwitchDebt),
		"recognizers":         countRecognizers(t),
		"fallback-sites":      countDiagCodes(t),
	}
	var moved []string
	for _, k := range []string{"tolerance-constants", "type-assertions", "recognizers", "fallback-sites"} {
		want, have := kernelNetDeltaPin[k], got[k]
		if want == have {
			continue
		}
		dir := "ROSE"
		if have < want {
			dir = "fell"
		}
		moved = append(moved, "  "+k+": "+strconv.Itoa(want)+" → "+strconv.Itoa(have)+
			"  ("+dir+" by "+strconv.Itoa(abs(have-want))+")")
	}
	if len(moved) > 0 {
		t.Errorf("kernel net delta moved — report it and update kernelNetDeltaPin in this commit. "+
			"A RISE needs a reason in the PR, and an ADR if it is a new engine, recognizer or "+
			"tolerance rather than a reported degradation; a FALL just needs the pin lowered so it "+
			"holds:\n%s", strings.Join(moved, "\n"))
	}
}

// countRecognizers counts the entries of the ordered dispatch tables — each entry is one
// analytic recognizer, and the count is what "generality over special cases" is measured by.
func countRecognizers(t *testing.T) int {
	t.Helper()
	fset := token.NewFileSet()
	n := 0
	for file := range dispatchLadders {
		f, err := parser.ParseFile(fset, filepath.Join("..", file), nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		ast.Inspect(f, func(node ast.Node) bool {
			cl, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			at, ok := cl.Type.(*ast.ArrayType)
			if !ok || at.Len != nil {
				return true
			}
			if _, isFunc := at.Elt.(*ast.FuncType); isFunc {
				n += len(cl.Elts)
			}
			return true
		})
	}
	if n == 0 {
		t.Fatal("counted no recognizers — the dispatch tables moved; update dispatchLadders")
	}
	return n
}

// countDiagCodes counts the declared diag.Code kinds under kernel/: one per way the kernel can
// degrade and say so.
func countDiagCodes(t *testing.T) int {
	t.Helper()
	fset := token.NewFileSet()
	n := 0
	err := filepath.WalkDir(filepath.Join("..", "kernel"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", p, err)
		}
		ast.Inspect(f, func(node ast.Node) bool {
			vs, ok := node.(*ast.ValueSpec)
			if !ok || vs.Type == nil {
				return true
			}
			sel, ok := vs.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Code" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "diag" {
				n += len(vs.Names)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/: %v", err)
	}
	return n
}

// sumInts totals a debt map's counts.
func sumInts(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
